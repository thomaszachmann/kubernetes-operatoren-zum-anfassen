// labenv startet envtest (etcd + kube-apiserver) mit unseren CRDs und
// schreibt eine Admin-Kubeconfig. Nur fürs Lab – Strg+C beendet alles.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func main() {
	env := &envtest.Environment{
		CRDDirectoryPaths:     []string{"config/crd/bases"},
		ErrorIfCRDPathMissing: true,
	}
	if _, err := env.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "envtest:", err)
		os.Exit(1)
	}
	defer func() { _ = env.Stop() }()

	user, err := env.AddUser(envtest.User{
		Name: "lab-admin", Groups: []string{"system:masters"},
	}, nil)
	if err == nil {
		var kc []byte
		if kc, err = user.KubeConfig(); err == nil {
			err = os.WriteFile("labenv.kubeconfig", kc, 0o600)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "kubeconfig:", err)
		return
	}
	fmt.Println("envtest läuft, KUBECONFIG=labenv.kubeconfig")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
}
