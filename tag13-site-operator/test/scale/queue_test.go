package scale

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("MaxConcurrentReconciles", func() {
	It("arbeitet 20 langsame Reconciles mit 1, 4 und 10 Workern ab", func() {
		createSites("workers", 20)
		for _, w := range []int{1, 4, 10} {
			var done, active, peak atomic.Int32
			start := time.Now()
			stop := runController(fmt.Sprint("workers-", w), "workers",
				controller.Options{MaxConcurrentReconciles: w},
				func(context.Context, reconcile.Request) (reconcile.Result, error) {
					n := active.Add(1)
					for p := peak.Load(); n > p; p = peak.Load() {
						if peak.CompareAndSwap(p, n) {
							break
						}
					}
					time.Sleep(200 * time.Millisecond) // langsamer API-Call
					active.Add(-1)
					done.Add(1)
					return reconcile.Result{}, nil
				})
			Eventually(done.Load).WithTimeout(10 * time.Second).Should(BeEquivalentTo(20))
			stop()
			GinkgoWriter.Printf("Worker=%2d  Dauer=%-6v  parallel max=%d\n",
				w, time.Since(start).Round(10*time.Millisecond), peak.Load())
			Expect(peak.Load()).To(BeNumerically("<=", w))
		}
	})
})

type limiter = workqueue.TypedRateLimiter[reconcile.Request]

// gaps lässt einen immer scheiternden Reconciler 8 s laufen und liefert
// die Abstände zwischen den Versuchen.
func gaps(name string, rl limiter) []time.Duration {
	var mu sync.Mutex
	var calls []time.Time
	stop := runController(name, "backoff", controller.Options{RateLimiter: rl},
		func(context.Context, reconcile.Request) (reconcile.Result, error) {
			mu.Lock()
			defer mu.Unlock()
			calls = append(calls, time.Now())
			return reconcile.Result{}, errors.New("registrar nicht erreichbar")
		})
	time.Sleep(8 * time.Second) // Messfenster, kein Warten auf einen Zustand
	stop()
	mu.Lock()
	defer mu.Unlock()
	var d []time.Duration
	for i := 1; i < len(calls); i++ {
		d = append(d, calls[i].Sub(calls[i-1]).Round(time.Millisecond))
	}
	return d
}

var _ = Describe("Backoff", func() {
	It("vergleicht Standard- und eigenen Rate-Limiter", func() {
		createSites("backoff", 1)
		std := gaps("backoff-std", nil)
		GinkgoWriter.Printf("Standard  : %2d Versuche %v\n", len(std)+1, std)
		own := gaps("backoff-own", workqueue.
			NewTypedItemExponentialFailureRateLimiter[reconcile.Request](
			500*time.Millisecond, 30*time.Second))
		GinkgoWriter.Printf("0,5s..30s : %2d Versuche %v\n", len(own)+1, own)
		Expect(len(own)).To(BeNumerically("<", len(std)))
	})
})
