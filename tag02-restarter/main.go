// restarter startet Deployments neu, wenn sich eine referenzierte ConfigMap ändert.
// Tag 2: controller-runtime ohne Kubebuilder (Manager, Reconciler, Builder).
package main

import (
	"flag"
	"os"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

func main() {
	opts := zap.Options{Development: true}
	opts.BindFlags(flag.CommandLine) // -zap-log-level, -zap-devel, ...
	flag.Parse()                     // -kubeconfig registriert controller-runtime
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	setupLog := ctrl.Log.WithName("setup")

	// GetConfig: -kubeconfig, $KUBECONFIG, In-Cluster, ~/.kube/config.
	if os.Getenv("KUBECONFIG") == "" && flag.Lookup("kubeconfig").Value.String() == "" {
		setupLog.Info("KUBECONFIG nicht gesetzt – Abbruch (Lab-Schutz)")
		os.Exit(1)
	}
	cfg := ctrl.GetConfigOrDie()

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Metrics:                metricsserver.Options{BindAddress: "0"}, // aus
		HealthProbeBindAddress: "0",                                     // aus
	})
	if err != nil {
		setupLog.Error(err, "Manager anlegen")
		os.Exit(1)
	}
	if err := (&Restarter{Client: mgr.GetClient()}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "Controller anlegen")
		os.Exit(1)
	}

	setupLog.Info("starte Manager", "host", cfg.Host)
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "Manager beendet")
		os.Exit(1)
	}
}
