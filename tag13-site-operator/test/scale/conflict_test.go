package scale

import (
	"fmt"
	"sync"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	webv1 "example.com/site-operator/api/v1"
)

// write liest die Site, setzt eine eigene Annotation und schreibt zurück –
// per Update (mit resourceVersion) oder per Merge-Patch (nur der Diff).
func write(i int, patch bool) error {
	var s webv1.StaticSite
	key := client.ObjectKey{Namespace: "conflict", Name: "s0"}
	if err := k8sClient.Get(ctx, key, &s); err != nil {
		return err
	}
	base := s.DeepCopy()
	if s.Annotations == nil {
		s.Annotations = map[string]string{}
	}
	s.Annotations[fmt.Sprint("writer-", i)] = "x"
	if patch {
		return k8sClient.Patch(ctx, &s, client.MergeFrom(base))
	}
	return k8sClient.Update(ctx, &s)
}

// race lässt 20 Goroutinen gleichzeitig schreiben und zählt 409-Fehler.
func race(name string, f func(i int) error) int32 {
	var conflicts atomic.Int32
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Go(func() {
			defer GinkgoRecover()
			if err := f(i); apierrors.IsConflict(err) {
				conflicts.Add(1)
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		})
	}
	wg.Wait()
	GinkgoWriter.Printf("%-24s 20 Schreiber, %2d Konflikte\n", name, conflicts.Load())
	return conflicts.Load()
}

var _ = Describe("Konflikte", func() {
	It("provoziert 409 und vergleicht Gegenmittel", func() {
		createSites("conflict", 1)
		update := func(i int) error { return write(i, false) }
		Expect(race("Update", update)).To(BeNumerically(">", 0))
		race("RetryOnConflict", func(i int) error {
			return retry.RetryOnConflict(retry.DefaultBackoff,
				func() error { return update(i) })
		})
		Expect(race("MergeFrom-Patch", func(i int) error { return write(i, true) })).
			To(BeZero())
	})
})
