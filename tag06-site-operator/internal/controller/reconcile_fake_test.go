package controller

import (
	"context"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	webv1alpha1 "example.com/site-operator/api/v1alpha1"
)

// Der fake client kennt weder Defaults noch Validierung – ideal für Fehlerpfade.
func TestReconcileContentMissing(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = webv1alpha1.AddToScheme(scheme)

	site := &webv1alpha1.StaticSite{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: webv1alpha1.StaticSiteSpec{Replicas: 1,
			ContentFrom: &webv1alpha1.ContentSource{
				ConfigMapRef: &webv1alpha1.ConfigMapReference{Name: "fehlt"},
			}},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(site).WithStatusSubresource(site).Build()
	rec := events.NewFakeRecorder(10)
	r := &StaticSiteReconciler{Client: c, Scheme: scheme, Recorder: rec}

	key := client.ObjectKeyFromObject(site)
	res, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: key})
	if err != nil || res.RequeueAfter != 30*time.Second {
		t.Fatalf("Reconcile = %+v, %v; want RequeueAfter 30s, nil", res, err)
	}

	got := &webv1alpha1.StaticSite{}
	if err := c.Get(context.Background(), key, got); err != nil {
		t.Fatal(err)
	}
	cond := meta.FindStatusCondition(got.Status.Conditions,
		webv1alpha1.ConditionDegraded)
	if cond == nil || cond.Reason != webv1alpha1.ReasonContentMissing {
		t.Fatalf("Degraded = %+v, want Reason ContentMissing", cond)
	}

	close(rec.Events)
	var warnings []string
	for e := range rec.Events {
		if strings.HasPrefix(e, "Warning ContentMissing") {
			warnings = append(warnings, e)
		}
	}
	if len(warnings) != 1 {
		t.Errorf("Warning-Events = %q, want genau eins", warnings)
	}
}
