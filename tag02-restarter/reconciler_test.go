package main

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func deploy(name, cm string) *appsv1.Deployment {
	d := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Namespace: "lab", Name: name}}
	if cm != "" {
		d.Annotations = map[string]string{annoConfigMap: cm}
	}
	return d
}

func configMap(name, value string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Namespace: "lab", Name: name},
		Data:       map[string]string{"setting": value},
	}
}

// reconcileHash ruft Reconcile auf und liefert den Hash am Pod-Template.
func reconcileHash(t *testing.T, c client.Client, name string) string {
	t.Helper()
	ctx := context.Background()
	key := types.NamespacedName{Namespace: "lab", Name: name}
	r := &Restarter{Client: c}
	if _, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: key}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	var d appsv1.Deployment
	if err := c.Get(ctx, key, &d); err != nil {
		t.Fatal(err)
	}
	return d.Spec.Template.Annotations[annoHash]
}

func TestReconcileSetztUndAendertHash(t *testing.T) {
	cm := configMap("app-config", "a")
	c := fake.NewClientBuilder().WithObjects(deploy("web", "app-config"), cm).Build()

	first := reconcileHash(t, c, "web")
	if first == "" {
		t.Fatal("kein Hash gesetzt")
	}
	if again := reconcileHash(t, c, "web"); again != first {
		t.Errorf("Hash ohne Änderung neu: %s != %s", again, first)
	}

	cm.Data["setting"] = "b"
	if err := c.Update(context.Background(), cm); err != nil {
		t.Fatal(err)
	}
	if changed := reconcileHash(t, c, "web"); changed == first {
		t.Error("Hash nach ConfigMap-Änderung unverändert")
	}
}

func TestReconcileOhneAnnotationOderConfigMap(t *testing.T) {
	c := fake.NewClientBuilder().
		WithObjects(deploy("plain", ""), deploy("waiting", "fehlt")).Build()
	for _, name := range []string{"plain", "waiting"} {
		if h := reconcileHash(t, c, name); h != "" {
			t.Errorf("%s: unerwarteter Hash %q", name, h)
		}
	}
}

func TestDeploymentsFor(t *testing.T) {
	c := fake.NewClientBuilder().WithObjects(deploy("a", "app-config"),
		deploy("b", "andere"), deploy("c", "app-config")).Build()
	r := &Restarter{Client: c}
	reqs := r.deploymentsFor(context.Background(), configMap("app-config", "x"))
	if len(reqs) != 2 {
		t.Fatalf("got %d requests, want 2: %v", len(reqs), reqs)
	}
}

func TestDataHashStabil(t *testing.T) {
	a := dataHash(map[string]string{"x": "1", "y": "2"})
	b := dataHash(map[string]string{"y": "2", "x": "1"})
	if a != b || len(a) != 16 {
		t.Errorf("hash instabil: %s vs %s", a, b)
	}
}

func TestDeploymentsForNurEigenerNamespace(t *testing.T) {
	fremd := deploy("x", "app-config")
	fremd.Namespace = "anderer"
	c := fake.NewClientBuilder().WithObjects(deploy("a", "app-config"), fremd).Build()
	reqs := (&Restarter{Client: c}).deploymentsFor(context.Background(),
		configMap("app-config", "neu"))
	if len(reqs) != 1 || reqs[0].Namespace != "lab" {
		t.Errorf("reqs = %v, want nur lab/a", reqs)
	}
}
