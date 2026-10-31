package controller

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// reconciles liest controller_runtime_reconcile_total für unseren Controller.
func reconciles() float64 {
	families, err := metrics.Registry.Gather()
	Expect(err).NotTo(HaveOccurred())
	var sum float64
	for _, f := range families {
		if f.GetName() != "controller_runtime_reconcile_total" {
			continue
		}
		for _, m := range f.GetMetric() {
			for _, l := range m.GetLabel() {
				if l.GetName() == "controller" && l.GetValue() == "staticsite" {
					sum += m.GetCounter().GetValue()
				}
			}
		}
	}
	return sum
}

var _ = Describe("GenerationChangedPredicate", func() {
	It("ignoriert Metadaten, reagiert auf spec", func(ctx SpecContext) {
		site := newStaticSite(newNamespace(ctx), "quiet")
		Expect(k8sClient.Create(ctx, site)).To(Succeed())
		Eventually(func() bool { // warten, bis der Controller zur Ruhe kommt
			n := reconciles()
			time.Sleep(time.Second)
			return n == reconciles()
		}, 20*time.Second).Should(BeTrue())
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(site), site)).
			To(Succeed())
		before := reconciles()

		patch := client.MergeFrom(site.DeepCopy())
		site.Annotations = map[string]string{"team": "web"}
		Expect(k8sClient.Patch(ctx, site, patch)).To(Succeed())
		Consistently(reconciles, 2*time.Second).Should(Equal(before))

		patch = client.MergeFrom(site.DeepCopy())
		site.Spec.Replicas = 2
		Expect(k8sClient.Patch(ctx, site, patch)).To(Succeed())
		Eventually(reconciles).Should(BeNumerically(">", before))
	})
})
