package scale

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var cmOnce sync.Once

// ensureConfigMaps legt einmalig 2000 ConfigMaps mit 30 Schlüsseln in "mem" an.
func ensureConfigMaps() {
	cmOnce.Do(func() {
		createSites("mem", 0)
		data := map[string]string{}
		for k := range 30 {
			data[fmt.Sprint("key-", k)] = "wert"
		}
		for i := range 2000 {
			cm := &corev1.ConfigMap{Data: data}
			cm.Name, cm.Namespace = fmt.Sprint("cm-", i), "mem"
			Expect(k8sClient.Create(ctx, cm)).To(Succeed())
		}
	})
}

func heapKiB() int64 {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return int64(m.HeapAlloc) / 1024
}

// cacheCost startet einen ConfigMap-Cache und misst den belegten Heap.
func cacheCost(name string, opts cache.Options) {
	time.Sleep(time.Second) // vorherige Caches sicher beenden lassen
	before := heapKiB()
	opts.Scheme = scheme.Scheme
	c, err := cache.New(cfg, opts)
	Expect(err).NotTo(HaveOccurred())
	var l corev1.ConfigMapList
	go func() { _ = c.Start(ctx) }()
	Expect(c.WaitForCacheSync(ctx)).To(BeTrue())
	Expect(c.List(ctx, &l, client.InNamespace("mem"))).To(Succeed()) // lädt Informer
	GinkgoWriter.Printf("%-16s %4d Objekte %6d KiB\n", name, len(l.Items),
		heapKiB()-before)
	runtime.KeepAlive(c)
}

var _ = Describe("Cache-Speicher", func() {
	It("misst den Heap je Cache-Variante", func() {
		ensureConfigMaps()
		cacheCost("alles", cache.Options{})
		cacheCost("ohne managedF.", cache.Options{
			DefaultTransform: cache.TransformStripManagedFields()})
		cacheCost("nur managed-by", cache.Options{ByObject: map[client.Object]cache.ByObject{
			&corev1.ConfigMap{}: {Label: labels.SelectorFromSet(labels.Set{
				"app.kubernetes.io/managed-by": "site-operator"})}}})
	})
})

// pages listet die 2000 ConfigMaps aus "mem" in Seiten zu 500.
func pages(r client.Reader) error {
	for cont, n := "", 1; ; n++ {
		var l corev1.ConfigMapList
		err := r.List(ctx, &l, client.InNamespace("mem"), client.Limit(500),
			client.Continue(cont))
		if err != nil {
			return err
		}
		GinkgoWriter.Printf("Seite %d: %3d Objekte, continue=%q\n", n, len(l.Items),
			l.Continue[:min(12, len(l.Continue))])
		if cont = l.Continue; cont == "" {
			return nil
		}
	}
}

var _ = Describe("Pagination", Ordered, func() {
	It("liest vom API-Server seitenweise", func() {
		ensureConfigMaps()
		Expect(pages(k8sClient)).To(Succeed())
	})
	It("scheitert am Cache", func() {
		c, err := cache.New(cfg, cache.Options{Scheme: scheme.Scheme})
		Expect(err).NotTo(HaveOccurred())
		go func() { _ = c.Start(ctx) }()
		Expect(c.WaitForCacheSync(ctx)).To(BeTrue())
		err = pages(c)
		GinkgoWriter.Println("Cache:", err)
		Expect(err).To(HaveOccurred())
	})
})
