package controller

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus/testutil"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	baov1alpha1 "example.com/bao-operator/api/v1alpha1"
)

var _ = Describe("BaoPolicy", Ordered, func() {
	const timeout, tick = 10 * time.Second, 200 * time.Millisecond
	key := client.ObjectKey{Namespace: "default", Name: "app-read"}
	const name = "k8s.default.app-read"
	readPolicy := func() (string, error) { return baoc.Sys().GetPolicy(name) }

	It("lehnt policy und rules gleichzeitig ab (CEL)", func() {
		bp := &baov1alpha1.BaoPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: "beides", Namespace: "default"},
			Spec: baov1alpha1.BaoPolicySpec{Policy: "x", Rules: []baov1alpha1.PolicyRule{
				{Path: "a", Capabilities: []baov1alpha1.Capability{capRead}},
			}},
		}
		err := k8sClient.Create(ctx, bp)
		Expect(apierrors.IsInvalid(err)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("has(self.policy) != has(self.rules)"))
	})

	It("meldet TokenUnavailable ohne Secret", func() {
		bp := &baov1alpha1.BaoPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace},
			Spec: baov1alpha1.BaoPolicySpec{Rules: []baov1alpha1.PolicyRule{{
				Path:         "secret/data/app/*",
				Capabilities: []baov1alpha1.Capability{capRead},
			}}},
		}
		Expect(k8sClient.Create(ctx, bp)).To(Succeed())
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, key, bp)).To(Succeed())
			c := meta.FindStatusCondition(bp.Status.Conditions, "Synced")
			g.Expect(c).NotTo(BeNil())
			g.Expect(c.Reason).To(Equal("TokenUnavailable"))
		}, timeout, tick).Should(Succeed())
	})

	It("schreibt die Policy, sobald das Secret da ist", func() {
		Expect(k8sClient.Create(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: TokenSecretName, Namespace: opNS},
			StringData: map[string]string{"address": baoAddr, "token": baoRoot},
		})).To(Succeed())
		Eventually(readPolicy, timeout, tick).Should(ContainSubstring(
			`path "secret/data/app/*"`))
		bp := &baov1alpha1.BaoPolicy{}
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, key, bp)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(bp.Status.Conditions, "Ready")).
				To(BeTrue())
			g.Expect(bp.Status.PolicyName).To(Equal(name))
			g.Expect(bp.Status.ObservedGeneration).To(Equal(bp.Generation))
		}, timeout, tick).Should(Succeed())
		Expect(bp.Finalizers).To(ContainElement(finalizerName))
	})

	It("übernimmt Spec-Änderungen", func() {
		bp := &baov1alpha1.BaoPolicy{}
		Expect(k8sClient.Get(ctx, key, bp)).To(Succeed())
		bp.Spec.Rules[0].Capabilities = append(bp.Spec.Rules[0].Capabilities, "list")
		Expect(k8sClient.Update(ctx, bp)).To(Succeed())
		Eventually(readPolicy, timeout, tick).Should(ContainSubstring(`"read", "list"`))
	})

	It("repariert Drift beim Resync", func() {
		before := testutil.ToFloat64(policySyncTotal.WithLabelValues("drift"))
		Expect(baoc.Sys().PutPolicy(name, `path "*" { capabilities = ["sudo"] }`)).
			To(Succeed())
		Eventually(readPolicy, timeout, tick).Should(ContainSubstring(`"read", "list"`))
		Expect(testutil.ToFloat64(policySyncTotal.WithLabelValues("drift"))).
			To(BeNumerically(">", before))
	})

	It("löscht die Policy über den Finalizer", func() {
		bp := &baov1alpha1.BaoPolicy{}
		Expect(k8sClient.Get(ctx, key, bp)).To(Succeed())
		Expect(k8sClient.Delete(ctx, bp)).To(Succeed())
		Eventually(func() bool {
			return apierrors.IsNotFound(k8sClient.Get(ctx, key, bp))
		}, timeout, tick).Should(BeTrue())
		Expect(readPolicy()).To(BeEmpty())
	})
})
