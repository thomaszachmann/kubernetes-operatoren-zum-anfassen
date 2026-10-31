package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	webv1 "example.com/site-operator/api/v1"
)

var _ = Describe("StaticSite API", func() {
	newSite := func(name string) *webv1.StaticSite {
		return &webv1.StaticSite{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
			Spec:       webv1.StaticSiteSpec{Content: "<h1>Test</h1>"},
		}
	}

	It("setzt Defaults im API-Server", func(ctx SpecContext) {
		site := newSite("defaults")
		Expect(k8sClient.Create(ctx, site)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(k8sClient.Delete(ctx, site)).To(Succeed())
		})
		// Create schreibt die Antwort des Servers zurück in site.
		Expect(site.Spec.Replicas).To(Equal(int32(1)))
		Expect(site.Spec.Image).To(Equal("nginxinc/nginx-unprivileged:1.31-alpine"))
		Expect(site.Spec.Path).To(Equal("/"))
	})

	It("lehnt replicas > 10 ab", func(ctx SpecContext) {
		site := newSite("zu-viele")
		site.Spec.Replicas = 11
		err := k8sClient.Create(ctx, site)
		Expect(apierrors.IsInvalid(err)).To(BeTrue(), "err = %v", err)
	})
})

var _ = Describe("StaticSite API – source", func() {
	It("verlangt configMapRef.name", func(ctx SpecContext) {
		site := &webv1.StaticSite{
			ObjectMeta: metav1.ObjectMeta{Name: "ohne-name", Namespace: "default"},
			Spec: webv1.StaticSiteSpec{
				Source: &webv1.ContentSource{
					ConfigMapRef: &webv1.ConfigMapReference{},
				},
			},
		}
		err := k8sClient.Create(ctx, site)
		Expect(apierrors.IsInvalid(err)).To(BeTrue(), "err = %v", err)
		Expect(err.Error()).To(ContainSubstring("spec.source.configMapRef.name"))
	})
})
