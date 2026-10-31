// envtest-up startet eine envtest-Control-Plane (etcd + kube-apiserver),
// schreibt eine Kubeconfig und läuft, bis du Ctrl+C drückst.
// Lab-Werkzeug: keine Controller, kein Scheduler, keine Nodes.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func main() {
	home, _ := os.UserHomeDir()
	out := flag.String("out",
		filepath.Join(home, "op-lab", "envtest.kubeconfig"), "Ziel der Kubeconfig")
	crds := flag.String("crds", "", "CRD-Verzeichnisse, kommagetrennt (optional)")
	flag.Parse()

	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		log.Fatal("KUBEBUILDER_ASSETS ist leer – erst ~/op-lab/env.sh sourcen")
	}
	// Signale ab sofort abfangen – auch während Start und Stop, damit ein
	// zweites Ctrl+C keine verwaisten etcd/kube-apiserver-Prozesse hinterlässt.
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer stop()

	env := &envtest.Environment{
		ErrorIfCRDPathMissing: true,
		// Hängt der kube-apiserver beim Stoppen, beendet envtest ihn per SIGKILL.
		ControlPlaneStopTimeout: 10 * time.Second,
	}
	if *crds != "" {
		env.CRDDirectoryPaths = strings.Split(*crds, ",")
	}
	cfg, err := env.Start()
	if err != nil {
		log.Fatalf("envtest start: %v", err)
	}
	// Ab hier muss die Control Plane in jedem Fall wieder gestoppt werden.
	if err := writeKubeconfig(env, *out); err != nil {
		log.Print(err)
	} else {
		fmt.Printf("envtest läuft: %s\n", cfg.Host)
		fmt.Printf("export KUBECONFIG=%s\n", *out)
		fmt.Println("Beenden mit Ctrl+C")
		<-ctx.Done()
	}
	fmt.Println("stoppe envtest ...")
	_ = os.Remove(*out)
	if err := env.Stop(); err != nil {
		log.Printf("envtest stop: %v (Prozess hart beendet)", err)
	}
	fmt.Println("envtest gestoppt, Kubeconfig gelöscht")
}

// writeKubeconfig legt einen Admin-User an und schreibt dessen Kubeconfig.
func writeKubeconfig(env *envtest.Environment, path string) error {
	admin, err := env.AddUser(envtest.User{
		Name:   "lab-admin",
		Groups: []string{"system:masters"},
	}, nil)
	if err != nil {
		return fmt.Errorf("user anlegen: %w", err)
	}
	data, err := admin.KubeConfig()
	if err != nil {
		return fmt.Errorf("kubeconfig erzeugen: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
