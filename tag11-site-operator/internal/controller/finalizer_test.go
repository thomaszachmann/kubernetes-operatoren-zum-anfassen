package controller

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"

	webv1 "example.com/site-operator/api/v1"
)

var _ = Describe("Finalizer und Registrar", func() {
	var (
		site   *webv1.StaticSite
		record string
	)

	BeforeEach(func(ctx SpecContext) {
		site = newStaticSite(newNamespace(ctx), "dns")
		record = "dns." + site.Namespace
		Expect(k8sClient.Create(ctx, site)).To(Succeed())
		Eventually(func() bool { return fakeReg.has(record) }).Should(BeTrue())
	})

	// gone ist erfüllt, wenn die StaticSite endgültig weg ist.
	gone := func(ctx SpecContext) func() bool {
		return func() bool {
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(site), site)
			return apierrors.IsNotFound(err)
		}
	}

	It("setzt den Finalizer und räumt beim Löschen ab", func(ctx SpecContext) {
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(site), site)).
			To(Succeed())
		Expect(site.Finalizers).To(ContainElement("web.lab.example/finalizer"))

		Expect(k8sClient.Delete(ctx, site)).To(Succeed())
		Eventually(gone(ctx)).Should(BeTrue())
		Expect(fakeReg.has(record)).To(BeFalse())
	})

	It("hält die Site fest, solange der Registrar ausfällt", func(ctx SpecContext) {
		fakeReg.down.Store(true)
		DeferCleanup(func() { fakeReg.down.Store(false) })

		Expect(k8sClient.Delete(ctx, site)).To(Succeed())
		Consistently(gone(ctx), 2*time.Second).Should(BeFalse())
		Expect(site.DeletionTimestamp).NotTo(BeNil()) // Terminating

		fakeReg.down.Store(false)
		Eventually(gone(ctx), 30*time.Second).Should(BeTrue())
		Expect(fakeReg.has(record)).To(BeFalse())
	})

	It("meldet Degraded, wenn der Eintrag nicht klappt", func(ctx SpecContext) {
		fakeReg.down.Store(true)
		DeferCleanup(func() { fakeReg.down.Store(false) })

		patch := client.MergeFrom(site.DeepCopy()) // Patch: kein Konflikt
		site.Spec.Replicas = 2                     // neue Generation -> Reconcile
		Expect(k8sClient.Patch(ctx, site, patch)).To(Succeed())
		Eventually(func() string {
			_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(site), site)
			c := meta.FindStatusCondition(site.Status.Conditions,
				webv1.ConditionDegraded)
			if c == nil || c.Status != "True" {
				return ""
			}
			return c.Message
		}).Should(ContainSubstring("HTTP 503"))
	})
})
