package controller

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	webv1 "example.com/site-operator/api/v1"
)

type siteMutator = func(*webv1.StaticSite)

func replicas(n int32) siteMutator {
	return func(s *webv1.StaticSite) { s.Spec.Replicas = n }
}

var _ = DescribeTable("CRD-Validierung",
	func(ctx SpecContext, mutate siteMutator, valid bool) {
		site := newStaticSite(newNamespace(ctx), "v", mutate)
		err := k8sClient.Create(ctx, site)
		if valid {
			Expect(err).NotTo(HaveOccurred())
			return
		}
		Expect(apierrors.IsInvalid(err)).To(BeTrue(), "err = %v", err)
	},
	Entry("minimal", func(*webv1.StaticSite) {}, true),
	Entry("10 Replikas", replicas(10), true),
	Entry("11 Replikas", replicas(11), false),
	// omitempty: 0 wird gar nicht gesendet, der Server setzt den Default 1.
	Entry("0 Replikas = Default", replicas(0), true),
	Entry("Inhalt über 64 KiB", func(s *webv1.StaticSite) {
		s.Spec.Content = strings.Repeat("x", 64*1024+1)
	}, false),
)
