package v1

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	webv1 "example.com/site-operator/api/v1"
	webv1alpha1 "example.com/site-operator/api/v1alpha1"
)

var _ = Describe("Conversion im API-Server", func() {
	meta := func(name string) metav1.ObjectMeta {
		return metav1.ObjectMeta{Name: name, Namespace: "default"}
	}

	It("liest ein v1alpha1-Objekt als v1", func() {
		old := &webv1alpha1.StaticSite{ObjectMeta: meta("alt"),
			Spec: webv1alpha1.StaticSiteSpec{ContentFrom: &webv1alpha1.ContentSource{
				ConfigMapRef: &webv1alpha1.ConfigMapReference{Name: "html"}}}}
		Expect(k8sClient.Create(ctx, old)).To(Succeed())

		hub := &webv1.StaticSite{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(old), hub)).To(Succeed())
		Expect(hub.Spec.Source.ConfigMapRef.Name).To(Equal("html"))
	})

	It("liest ein v1-Objekt als v1alpha1", func() {
		neu := &webv1.StaticSite{ObjectMeta: meta("neu"),
			Spec: webv1.StaticSiteSpec{Source: &webv1.ContentSource{
				ConfigMapRef: &webv1.ConfigMapReference{Name: "html"}}}}
		Expect(k8sClient.Create(ctx, neu)).To(Succeed())

		spoke := &webv1alpha1.StaticSite{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(neu), spoke)).
			To(Succeed())
		Expect(spoke.Spec.ContentFrom.ConfigMapRef.Name).To(Equal("html"))
		Expect(spoke.Spec.Path).To(Equal("/")) // Default aus dem v1-Schema
	})

	It("warnt bei Nutzung von v1alpha1", func() {
		var got warnings
		cfgW := rest.CopyConfig(cfg)
		cfgW.WarningHandlerWithContext = &got
		c, err := client.New(cfgW, client.Options{Scheme: k8sClient.Scheme()})
		Expect(err).NotTo(HaveOccurred())
		Expect(c.Get(ctx, client.ObjectKey{Namespace: "default", Name: "neu"},
			&webv1alpha1.StaticSite{})).To(Succeed())
		Expect(got).To(ContainElement("v1alpha1 ist veraltet, nutze v1"))
	})
})

// warnings sammelt Warning-Header des API-Servers.
type warnings []string

func (w *warnings) HandleWarningHeaderWithContext(_ context.Context, _ int,
	_ string, text string) {
	*w = append(*w, text)
}
