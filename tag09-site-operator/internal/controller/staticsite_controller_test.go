package controller

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	webv1 "example.com/site-operator/api/v1"
)

var _ = Describe("StaticSite Controller", func() {
	var (
		ns   string
		key  client.ObjectKey
		site *webv1.StaticSite
	)

	BeforeEach(func(ctx SpecContext) {
		ns = newNamespace(ctx)
		key = client.ObjectKey{Namespace: ns, Name: "demo"}
		site = newStaticSite(ns, "demo")
	})

	// getDeployment wartet, bis der Operator das Deployment angelegt hat.
	getDeployment := func(ctx SpecContext) *appsv1.Deployment {
		GinkgoHelper()
		dep := &appsv1.Deployment{}
		Eventually(func() error { return k8sClient.Get(ctx, key, dep) }).
			WithContext(ctx).Should(Succeed())
		return dep
	}

	It("erzeugt ConfigMap, Deployment und Service mit Owner", func(ctx SpecContext) {
		Expect(k8sClient.Create(ctx, site)).To(Succeed())

		dep := getDeployment(ctx)
		Expect(metav1.IsControlledBy(dep, site)).To(BeTrue())
		Expect(dep.Spec.Template.Annotations).
			To(HaveKey("web.lab.example/content-hash"))
		Expect(dep.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort).
			To(Equal(int32(8080)))

		svc := &corev1.Service{}
		Expect(k8sClient.Get(ctx, key, svc)).To(Succeed())
		Expect(svc.Spec.Ports[0].Port).To(Equal(int32(80)))

		cm := &corev1.ConfigMap{}
		cmKey := client.ObjectKey{Namespace: ns, Name: "demo-content"}
		Expect(k8sClient.Get(ctx, cmKey, cm)).To(Succeed())
		Expect(cm.Data).To(HaveKeyWithValue("index.html", "<h1>Test</h1>"))
	})

	It("dreht manuelle Änderungen zurück", func(ctx SpecContext) {
		Expect(k8sClient.Create(ctx, site)).To(Succeed())
		dep := getDeployment(ctx)
		dep.Spec.Replicas = ptr.To[int32](5)
		Expect(k8sClient.Update(ctx, dep)).To(Succeed())

		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, key, dep)).To(Succeed())
			g.Expect(*dep.Spec.Replicas).To(Equal(int32(1)))
		}).WithContext(ctx).Should(Succeed())
	})

	It("rollt bei neuem Inhalt mit neuem Hash aus", func(ctx SpecContext) {
		Expect(k8sClient.Create(ctx, site)).To(Succeed())
		hash := getDeployment(ctx).Spec.Template.Annotations[contentHashAnnotation]

		Expect(k8sClient.Get(ctx, key, site)).To(Succeed())
		site.Spec.Content = "<h1>Version 2</h1>"
		Expect(k8sClient.Update(ctx, site)).To(Succeed())

		Eventually(func() string {
			return getDeployment(ctx).Spec.Template.Annotations[contentHashAnnotation]
		}).WithContext(ctx).ShouldNot(Equal(hash))
	})

	It("meldet Available erst mit bereiten Pods", func(ctx SpecContext) {
		Expect(k8sClient.Create(ctx, site)).To(Succeed())
		available := condition(ctx, key, webv1.ConditionAvailable)

		Eventually(available).WithContext(ctx).Should(haveCondition(
			metav1.ConditionFalse, webv1.ReasonReplicasNotReady))
		Consistently(available).WithContext(ctx).WithTimeout(time.Second).
			Should(HaveField("Status", metav1.ConditionFalse))

		markDeploymentReady(ctx, key, 1)
		Eventually(available).WithContext(ctx).Should(haveCondition(
			metav1.ConditionTrue, webv1.ReasonAllReplicasReady))
		Expect(k8sClient.Get(ctx, key, site)).To(Succeed())
		Expect(site.Status.ObservedGeneration).To(Equal(site.Generation))
		Expect(site.Status.URL).To(Equal("http://demo." + ns + ".svc"))
	})

	It("schreibt im Ruhezustand nichts mehr (kein Hotloop)", func(ctx SpecContext) {
		Expect(k8sClient.Create(ctx, site)).To(Succeed())
		getDeployment(ctx)
		markDeploymentReady(ctx, key, 1)
		Eventually(condition(ctx, key, webv1.ConditionAvailable)).
			WithContext(ctx).Should(HaveField("Status", metav1.ConditionTrue))

		resourceVersions := func() []string {
			s, d := &webv1.StaticSite{}, &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, key, s)).To(Succeed())
			Expect(k8sClient.Get(ctx, key, d)).To(Succeed())
			return []string{s.ResourceVersion, d.ResourceVersion}
		}
		before := resourceVersions()
		Consistently(resourceVersions).WithContext(ctx).
			WithTimeout(2 * time.Second).Should(Equal(before))
	})

	It("meldet ContentMissing und erholt sich", func(ctx SpecContext) {
		site.Spec.Content = ""
		site.Spec.Source = &webv1.ContentSource{
			ConfigMapRef: &webv1.ConfigMapReference{Name: "html"},
		}
		Expect(k8sClient.Create(ctx, site)).To(Succeed())
		degraded := condition(ctx, key, webv1.ConditionDegraded)
		Eventually(degraded).WithContext(ctx).Should(haveCondition(
			metav1.ConditionTrue, webv1.ReasonContentMissing))

		Expect(k8sClient.Create(ctx, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "html", Namespace: ns},
			Data:       map[string]string{"index.html": "<h1>CM</h1>"},
		})).To(Succeed())
		// Bis Tag 10 beobachtet der Operator fremde ConfigMaps nicht:
		// eine Annotation weckt ihn, statt 30 s auf RequeueAfter zu warten.
		Expect(k8sClient.Get(ctx, key, site)).To(Succeed())
		site.Annotations = map[string]string{"test/poke": "1"}
		Expect(k8sClient.Update(ctx, site)).To(Succeed())

		Eventually(degraded).WithContext(ctx).Should(haveCondition(
			metav1.ConditionFalse, webv1.ReasonAsExpected))
	})
})
