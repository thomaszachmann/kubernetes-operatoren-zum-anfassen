package controller

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	webv1alpha1 "example.com/site-operator/api/v1alpha1"
)

// newNamespace isoliert jede Spec in einem eigenen Namespace.
// envtest löscht Namespaces nie ganz (kein Namespace-Controller) – egal im Test.
func newNamespace(ctx context.Context) string {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: "site-test-"}}
	Expect(k8sClient.Create(ctx, ns)).To(Succeed())
	return ns.Name
}

// newStaticSite baut eine gültige StaticSite; mutate passt sie je Test an.
func newStaticSite(ns, name string,
	mutate ...func(*webv1alpha1.StaticSite)) *webv1alpha1.StaticSite {
	site := &webv1alpha1.StaticSite{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec:       webv1alpha1.StaticSiteSpec{Content: "<h1>Test</h1>"},
	}
	for _, m := range mutate {
		m(site)
	}
	return site
}

// markDeploymentReady spielt den Deployment-Controller, den envtest nicht hat.
func markDeploymentReady(ctx context.Context, key client.ObjectKey, n int32) {
	GinkgoHelper()
	dep := &appsv1.Deployment{}
	Expect(k8sClient.Get(ctx, key, dep)).To(Succeed())
	patch := client.MergeFrom(dep.DeepCopy())
	dep.Status = appsv1.DeploymentStatus{
		ObservedGeneration: dep.Generation, Replicas: n,
		UpdatedReplicas: n, ReadyReplicas: n, AvailableReplicas: n,
	}
	Expect(k8sClient.Status().Patch(ctx, dep, patch)).To(Succeed())
}

// condition liefert eine Funktion für Eventually/Consistently.
func condition(ctx context.Context, key client.ObjectKey,
	typ string) func() (metav1.Condition, error) {
	return func() (metav1.Condition, error) {
		site := &webv1alpha1.StaticSite{}
		if err := k8sClient.Get(ctx, key, site); err != nil {
			return metav1.Condition{}, err
		}
		c := meta.FindStatusCondition(site.Status.Conditions, typ)
		if c == nil {
			return metav1.Condition{}, fmt.Errorf("condition %s fehlt", typ)
		}
		return *c, nil
	}
}

// haveCondition prüft Status und Reason in einem Matcher.
func haveCondition(status metav1.ConditionStatus, reason string) OmegaMatcher {
	return SatisfyAll(HaveField("Status", status), HaveField("Reason", reason))
}
