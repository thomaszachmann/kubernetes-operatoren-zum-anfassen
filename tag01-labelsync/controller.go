package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const labelKey = "team"

// Controller kopiert das Label "team" vom Namespace auf alle Pods darin.
type Controller struct {
	client  kubernetes.Interface
	nsList  listersv1.NamespaceLister
	podList listersv1.PodLister
	synced  []cache.InformerSynced
	queue   workqueue.TypedRateLimitingInterface[string] // Key: Namespace-Name
}

// New verdrahtet Informer, Lister und Workqueue.
func New(client kubernetes.Interface, f informers.SharedInformerFactory) *Controller {
	nsInf := f.Core().V1().Namespaces()
	podInf := f.Core().V1().Pods()
	c := &Controller{
		client:  client,
		nsList:  nsInf.Lister(),
		podList: podInf.Lister(),
		synced: []cache.InformerSynced{
			nsInf.Informer().HasSynced, podInf.Informer().HasSynced,
		},
		queue: workqueue.NewTypedRateLimitingQueue(
			workqueue.DefaultTypedControllerRateLimiter[string]()),
	}
	// Event-Handler machen nichts außer "Namespace X ansehen" in die Queue legen.
	_, _ = nsInf.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj any) { c.enqueueNS(obj) },
		UpdateFunc: func(_, obj any) { c.enqueueNS(obj) },
	})
	_, _ = podInf.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj any) { c.enqueuePod(obj) },
		UpdateFunc: func(_, obj any) { c.enqueuePod(obj) },
	})
	return c
}

func (c *Controller) enqueueNS(obj any) {
	if ns, ok := obj.(*corev1.Namespace); ok {
		c.queue.Add(ns.Name)
	}
}

func (c *Controller) enqueuePod(obj any) {
	if pod, ok := obj.(*corev1.Pod); ok {
		c.queue.Add(pod.Namespace)
	}
}

// Run wartet auf die Caches und startet die Worker, bis ctx endet.
func (c *Controller) Run(ctx context.Context, workers int) error {
	defer c.queue.ShutDown()
	if !cache.WaitForNamedCacheSyncWithContext(ctx, c.synced...) {
		return fmt.Errorf("caches nicht synchron")
	}
	slog.Info("caches synchron, starte worker", "workers", workers)
	for range workers {
		go wait.UntilWithContext(ctx, c.worker, time.Second)
	}
	<-ctx.Done()
	return nil
}

func (c *Controller) worker(ctx context.Context) {
	for c.processNext(ctx) {
	}
}

// processNext holt einen Key, ruft sync und entscheidet über Retry.
func (c *Controller) processNext(ctx context.Context) bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key) // Key wieder freigeben (sonst nie wieder verarbeitet)

	if err := c.sync(ctx, key); err != nil {
		slog.Error("sync fehlgeschlagen", "ns", key, "err", err,
			"retries", c.queue.NumRequeues(key))
		c.queue.AddRateLimited(key) // exponentielles Backoff: 5ms, 10ms, ...
		return true
	}
	c.queue.Forget(key) // Erfolg: Backoff-Zähler zurücksetzen
	return true
}

// sync bringt alle Pods eines Namespace auf den Label-Wert des Namespace.
func (c *Controller) sync(ctx context.Context, nsName string) error {
	ns, err := c.nsList.Get(nsName) // liest aus dem Cache, nicht vom API-Server
	if apierrors.IsNotFound(err) {
		return nil // Namespace weg: nichts zu tun
	}
	if err != nil {
		return err
	}
	want, has := ns.Labels[labelKey]

	pods, err := c.podList.Pods(nsName).List(labels.Everything())
	if err != nil {
		return err
	}
	for _, pod := range pods {
		got, podHas := pod.Labels[labelKey]
		if got == want && podHas == has {
			continue // schon richtig: nichts schreiben (Idempotenz)
		}
		if err := c.patchPod(ctx, pod, want, has); err != nil {
			return err
		}
		slog.Info("label gesetzt", "pod", nsName+"/"+pod.Name,
			labelKey, want, "entfernt", !has)
	}
	return nil
}

// patchPod schreibt nur das eine Label per JSON-Merge-Patch (null = löschen).
func (c *Controller) patchPod(ctx context.Context, pod *corev1.Pod,
	value string, set bool) error {
	var v any // nil wird zu JSON null
	if set {
		v = value
	}
	patch, err := json.Marshal(map[string]any{
		"metadata": map[string]any{"labels": map[string]any{labelKey: v}},
	})
	if err != nil {
		return err
	}
	_, err = c.client.CoreV1().Pods(pod.Namespace).Patch(ctx, pod.Name,
		types.MergePatchType, patch, metav1.PatchOptions{})
	return err
}
