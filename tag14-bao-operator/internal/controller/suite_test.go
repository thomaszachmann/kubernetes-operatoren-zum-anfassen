package controller

import (
	"context"
	"net/http"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openbao/openbao/api/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	baov1alpha1 "example.com/bao-operator/api/v1alpha1"
)

const (
	baoAddr = "http://127.0.0.1:18200"
	baoRoot = "lab-root" // Lab-Wert, nur für den Dev-Server
	opNS    = "bao-operator-system"
)

var (
	ctx       context.Context
	cancel    context.CancelFunc
	testEnv   *envtest.Environment
	k8sClient client.Client
	baoCmd    *exec.Cmd
	baoc      *api.Client
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.TODO())

	By("OpenBao-Dev-Server starten")
	if resp, err := http.Get(baoAddr + "/v1/sys/health"); err == nil {
		_ = resp.Body.Close()
		Fail("Port 18200 ist belegt – läuft noch ein bao server -dev?")
	}
	baoCmd = exec.Command("bao", "server", "-dev",
		"-dev-root-token-id="+baoRoot, "-dev-listen-address=127.0.0.1:18200")
	Expect(baoCmd.Start()).To(Succeed())
	Eventually(func() error {
		resp, err := http.Get(baoAddr + "/v1/sys/health")
		if err == nil {
			_ = resp.Body.Close()
		}
		return err
	}, 15*time.Second, 200*time.Millisecond).Should(Succeed())
	cfg := api.DefaultConfig()
	cfg.Address = baoAddr
	var err error
	baoc, err = api.NewClient(cfg)
	Expect(err).NotTo(HaveOccurred())
	baoc.SetToken(baoRoot)

	By("envtest starten")
	Expect(baov1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}
	restCfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	k8sClient, err = client.New(restCfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: opNS},
	})).To(Succeed())

	By("Manager mit dem Reconciler starten")
	mgr, err := ctrl.NewManager(restCfg, ctrl.Options{
		Scheme:  scheme.Scheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
	})
	Expect(err).NotTo(HaveOccurred())
	Expect((&BaoPolicyReconciler{
		Client:            mgr.GetClient(),
		Scheme:            mgr.GetScheme(),
		Recorder:          mgr.GetEventRecorder("bao-operator"),
		OperatorNamespace: opNS,
		ResyncInterval:    2 * time.Second,
	}).SetupWithManager(mgr)).To(Succeed())
	go func() {
		defer GinkgoRecover()
		Expect(mgr.Start(ctx)).To(Succeed())
	}()
})

var _ = AfterSuite(func() {
	cancel()
	if testEnv != nil {
		Expect(testEnv.Stop()).To(Succeed())
	}
	if baoCmd != nil && baoCmd.Process != nil {
		_ = baoCmd.Process.Kill()
		_ = baoCmd.Wait()
	}
})
