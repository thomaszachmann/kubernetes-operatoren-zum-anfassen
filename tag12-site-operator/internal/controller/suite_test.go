package controller

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	webv1 "example.com/site-operator/api/v1"
	"example.com/site-operator/internal/registrar"
)

var (
	ctx       context.Context
	cancel    context.CancelFunc
	testEnv   *envtest.Environment
	k8sClient client.Client // direkt, ohne Cache: sieht immer den aktuellen Stand
	fakeReg   *fakeRegistrar
	cached    client.Client // Client des Managers: liest aus dem Cache
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	SetDefaultEventuallyTimeout(10 * time.Second)
	SetDefaultEventuallyPollingInterval(100 * time.Millisecond)
	// Mit WithContext(ctx) gilt der Default-Timeout sonst nicht – Test hinge ewig.
	EnforceDefaultTimeoutsWhenUsingContexts()
	ctx, cancel = context.WithCancel(context.Background())
	Expect(webv1.AddToScheme(scheme.Scheme)).To(Succeed())

	By("envtest starten (Binaries aus KUBEBUILDER_ASSETS)")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
		},
		ErrorIfCRDPathMissing: true,
	}
	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())

	By("Manager mit dem echten Reconciler starten")
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:  scheme.Scheme,
		Cache:   CacheOptions(),
		Metrics: metricsserver.Options{BindAddress: "0"}, // kein Port im Test
	})
	Expect(err).NotTo(HaveOccurred())
	cached = mgr.GetClient()
	fakeReg = newFakeRegistrar()
	Expect((&StaticSiteReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Recorder:  mgr.GetEventRecorder("site-operator"),
		Registrar: registrar.New(fakeReg.URL),
	}).SetupWithManager(mgr)).To(Succeed())
	go func() {
		defer GinkgoRecover()
		Expect(mgr.Start(ctx)).To(Succeed())
	}()
})

var _ = AfterSuite(func() {
	By("Manager und envtest stoppen")
	cancel()
	fakeReg.Close()
	Expect(testEnv.Stop()).To(Succeed())
})
