// labelsync kopiert das Label "team" vom Namespace auf alle Pods darin.
// Tag 1: Operator-Muster von Hand mit client-go (Informer, Lister, Workqueue).
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	kubeconfig := flag.String("kubeconfig", os.Getenv("KUBECONFIG"),
		"Pfad zur Kubeconfig (Pflicht, kein Fallback auf ~/.kube/config)")
	workers := flag.Int("workers", 2, "parallele Worker")
	flag.Parse()

	if err := run(*kubeconfig, *workers); err != nil {
		slog.Error("abbruch", "err", err)
		os.Exit(1)
	}
}

func run(kubeconfig string, workers int) error {
	if kubeconfig == "" {
		return errors.New("-kubeconfig oder KUBECONFIG setzen")
	}
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return err
	}
	slog.Info("verbinde", "host", cfg.Host)
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Resync 10 min: alle Objekte kommen periodisch erneut als Update-Event.
	factory := informers.NewSharedInformerFactory(client, 10*time.Minute)
	c := New(client, factory)
	factory.Start(ctx.Done()) // erst NACH New(): Informer müssen registriert sein
	defer factory.Shutdown()

	err = c.Run(ctx, workers)
	slog.Info("beendet")
	return err
}
