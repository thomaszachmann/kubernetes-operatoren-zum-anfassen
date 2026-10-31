package controller

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	webv1 "example.com/site-operator/api/v1"
)

var _ = Describe("Watch auf referenzierte ConfigMaps", func() {
	It("reagiert sofort auf Anlegen und Ändern der Quelle", func(ctx SpecContext) {
		ns := newNamespace(ctx)
		site := newStaticSite(ns, "ref", func(s *webv1.StaticSite) {
			s.Spec.Content = ""
			s.Spec.Source = &webv1.ContentSource{
				ConfigMapRef: &webv1.ConfigMapReference{Name: "html"}}
		})
		Expect(k8sClient.Create(ctx, site)).To(Succeed())

		hash := func() string {
			var dep appsv1.Deployment
			key := client.ObjectKey{Namespace: ns, Name: "ref"}
			if err := k8sClient.Get(ctx, key, &dep); err != nil {
				return ""
			}
			return dep.Spec.Template.Annotations["web.lab.example/content-hash"]
		}
		Consistently(hash, time.Second).Should(BeEmpty()) // Quelle fehlt noch

		src := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "html", Namespace: ns},
			Data:       map[string]string{"index.html": "<h1>v1</h1>"},
		}
		Expect(k8sClient.Create(ctx, src)).To(Succeed())
		Eventually(hash, 5*time.Second).ShouldNot(BeEmpty()) // nicht erst nach 30 s
		before := hash()

		src.Data["index.html"] = "<h1>v2</h1>"
		Expect(k8sClient.Update(ctx, src)).To(Succeed())
		Eventually(hash, 5*time.Second).ShouldNot(Equal(before))
	})
})
