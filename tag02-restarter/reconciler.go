package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"maps"
	"slices"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	annoConfigMap = "restarter.lab.example/configmap" // am Deployment
	annoHash      = "restarter.lab.example/hash"      // am Pod-Template
)

// Restarter startet Deployments neu, wenn sich ihre ConfigMap ändert.
type Restarter struct {
	client.Client
}

// Reconcile bekommt nur Namespace/Name – den Rest liest es selbst (Level-based).
func (r *Restarter) Reconcile(ctx context.Context, req ctrl.Request) (
	ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var dep appsv1.Deployment
	if err := r.Get(ctx, req.NamespacedName, &dep); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err) // gelöscht: fertig
	}
	cmName := dep.Annotations[annoConfigMap]
	if cmName == "" {
		return ctrl.Result{}, nil
	}

	var cm corev1.ConfigMap
	err := r.Get(ctx, types.NamespacedName{Namespace: dep.Namespace,
		Name: cmName}, &cm)
	if apierrors.IsNotFound(err) {
		logger.Info("ConfigMap fehlt, warte", "configmap", cmName)
		return ctrl.Result{}, nil // der ConfigMap-Watch weckt uns
	}
	if err != nil {
		return ctrl.Result{}, err // controller-runtime wiederholt mit Backoff
	}

	hash := dataHash(cm.Data)
	if dep.Spec.Template.Annotations[annoHash] == hash {
		return ctrl.Result{}, nil // aktuell: nichts schreiben
	}
	patch := client.MergeFrom(dep.DeepCopy())
	if dep.Spec.Template.Annotations == nil {
		dep.Spec.Template.Annotations = map[string]string{}
	}
	dep.Spec.Template.Annotations[annoHash] = hash // neues Template = Rollout
	if err := r.Patch(ctx, &dep, patch); err != nil {
		return ctrl.Result{}, err
	}
	logger.Info("Rollout ausgelöst", "configmap", cmName, "hash", hash)
	return ctrl.Result{}, nil
}

// dataHash bildet einen stabilen Hash über die sortierten Keys und Werte.
func dataHash(data map[string]string) string {
	h := sha256.New()
	for _, k := range slices.Sorted(maps.Keys(data)) {
		h.Write([]byte(k + "=" + data[k] + "\n"))
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// deploymentsFor übersetzt "ConfigMap geändert" in "diese Deployments prüfen".
func (r *Restarter) deploymentsFor(ctx context.Context,
	obj client.Object) []reconcile.Request {
	var list appsv1.DeploymentList
	if err := r.List(ctx, &list, client.InNamespace(obj.GetNamespace())); err != nil {
		log.FromContext(ctx).Error(err, "Deployments auflisten")
		return nil
	}
	var reqs []reconcile.Request
	for _, d := range list.Items {
		if d.Annotations[annoConfigMap] == obj.GetName() {
			reqs = append(reqs, reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: d.Namespace, Name: d.Name}})
		}
	}
	return reqs
}

// SetupWithManager verdrahtet Watches: Deployments direkt, ConfigMaps gemappt.
func (r *Restarter) SetupWithManager(mgr ctrl.Manager) error {
	annotated := predicate.NewPredicateFuncs(func(o client.Object) bool {
		_, ok := o.GetAnnotations()[annoConfigMap]
		return ok
	})
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}, builder.WithPredicates(annotated)).
		Watches(&corev1.ConfigMap{},
			handler.EnqueueRequestsFromMapFunc(r.deploymentsFor)).
		Named("restarter").
		Complete(r)
}
