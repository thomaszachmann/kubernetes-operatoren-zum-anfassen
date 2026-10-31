package v1alpha1

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	webv1alpha1 "example.com/site-operator/api/v1alpha1"
)

func site(name string) *webv1alpha1.StaticSite {
	return &webv1alpha1.StaticSite{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec:       webv1alpha1.StaticSiteSpec{Content: "<h1>Hallo</h1>"},
	}
}

var _ = Describe("Defaulter (Unit)", func() {
	DescribeTable("normalisiert spec.path",
		func(in, want string) {
			s := site("x")
			s.Spec.Path = in
			Expect((&StaticSiteDefaulter{}).Default(ctx, s)).To(Succeed())
			Expect(s.Spec.Path).To(Equal(want))
		},
		Entry("leer", "", "/"),
		Entry("ohne Slashes", "docs", "/docs/"),
		Entry("schon normal", "/docs/", "/docs/"),
		Entry("verschachtelt", "a/b", "/a/b/"),
	)
})

var _ = Describe("Webhooks im API-Server (envtest)", func() {
	It("normalisiert den Pfad beim Anlegen", func() {
		s := site("mutate")
		s.Spec.Path = "docs"
		Expect(k8sClient.Create(ctx, s)).To(Succeed())
		Expect(s.Spec.Path).To(Equal("/docs/")) // Antwort enthält den Patch
	})

	It("lehnt fremde Registries ab", func() {
		s := site("evil")
		s.Spec.Image = "evil.example/nginx:1.0"
		err := k8sClient.Create(ctx, s)
		Expect(apierrors.IsInvalid(err)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("spec.image"))
	})

	It("warnt bei fehlender ConfigMap, lehnt aber nicht ab", func() {
		var got warnings
		cfgW := rest.CopyConfig(cfg)
		cfgW.WarningHandlerWithContext = &got
		c, err := client.New(cfgW, client.Options{Scheme: k8sClient.Scheme()})
		Expect(err).NotTo(HaveOccurred())

		s := site("later")
		s.Spec.Content = ""
		s.Spec.ContentFrom = &webv1alpha1.ContentSource{
			ConfigMapRef: &webv1alpha1.ConfigMapReference{Name: "kommt-noch"}}
		Expect(c.Create(ctx, s)).To(Succeed())
		Expect(got).To(ContainElement(ContainSubstring("kommt-noch fehlt")))
	})
})

// warnings sammelt Warning-Header des API-Servers.
type warnings []string

func (w *warnings) HandleWarningHeaderWithContext(_ context.Context, _ int,
	_ string, text string) {
	*w = append(*w, text)
}
