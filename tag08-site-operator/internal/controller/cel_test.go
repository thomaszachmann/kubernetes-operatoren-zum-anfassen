package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	webv1alpha1 "example.com/site-operator/api/v1alpha1"
)

var _ = DescribeTable("CEL-Regeln im API-Server",
	func(ctx SpecContext, mutate siteMutator, msg string) {
		site := newStaticSite(newNamespace(ctx), "cel", mutate)
		Expect(k8sClient.Create(ctx, site)).To(MatchError(ContainSubstring(msg)))
	},
	Entry("content und contentFrom", func(s *webv1alpha1.StaticSite) {
		s.Spec.ContentFrom = &webv1alpha1.ContentSource{
			ConfigMapRef: &webv1alpha1.ConfigMapReference{Name: "html"}}
	}, "exactly one of the fields"),
	Entry("weder noch", func(s *webv1alpha1.StaticSite) { s.Spec.Content = "" },
		"exactly one of the fields"),
	Entry(":latest", func(s *webv1alpha1.StaticSite) { s.Spec.Image = "nginx:latest" },
		"failed rule: !self.endsWith(':latest')"),
)

var _ = It("verbietet, path nachträglich zu ändern", func(ctx SpecContext) {
	site := newStaticSite(newNamespace(ctx), "pfad")
	Expect(k8sClient.Create(ctx, site)).To(Succeed())
	patch := client.MergeFrom(site.DeepCopy()) // Patch statt Update: kein Konflikt
	site.Spec.Path = "/neu/"
	Expect(k8sClient.Patch(ctx, site, patch)).
		To(MatchError(ContainSubstring("failed rule: self == oldSelf")))
})
