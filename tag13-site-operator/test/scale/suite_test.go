// Package scale misst Worker, Backoff, Konflikte, Cache und Pagination
// gegen envtest. Die Specs prüfen Verhalten und geben Messwerte aus.
package scale

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	webv1 "example.com/site-operator/api/v1"
)

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
	ctx       context.Context
	cancel    context.CancelFunc
)

func TestScale(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scale Suite")
}

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.Background())
	Expect(webv1.AddToScheme(scheme.Scheme)).To(Succeed())
	testEnv = &envtest.Environment{ErrorIfCRDPathMissing: true,
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")}}
	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	cancel()
	Expect(testEnv.Stop()).To(Succeed())
})

// createSites legt einen Namespace mit n StaticSites an.
func createSites(ns string, n int) {
	Expect(k8sClient.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: ns}})).To(Succeed())
	for i := range n {
		Expect(k8sClient.Create(ctx, &webv1.StaticSite{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprint("s", i), Namespace: ns},
			Spec:       webv1.StaticSiteSpec{Content: "<h1>last</h1>"},
		})).To(Succeed())
	}
}

// runController startet einen eigenen Manager mit einem Mess-Reconciler
// für die StaticSites in ns. Stoppen über das zurückgegebene cancel.
func runController(name, ns string, opts controller.Options,
	r reconcile.Func) context.CancelFunc {
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme.Scheme,
		Metrics: metricsserver.Options{BindAddress: "0"}})
	Expect(err).NotTo(HaveOccurred())
	only := func(c context.Context, req reconcile.Request) (reconcile.Result, error) {
		if req.Namespace != ns {
			return reconcile.Result{}, nil
		}
		return r(c, req)
	}
	Expect(ctrl.NewControllerManagedBy(mgr).For(&webv1.StaticSite{}).
		Named(name).WithOptions(opts).Complete(reconcile.Func(only))).To(Succeed())
	c, stop := context.WithCancel(ctx)
	go func() {
		defer GinkgoRecover()
		Expect(mgr.Start(c)).To(Succeed())
	}()
	return stop
}
