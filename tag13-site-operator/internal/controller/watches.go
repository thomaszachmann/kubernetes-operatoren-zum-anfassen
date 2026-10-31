package controller

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	webv1 "example.com/site-operator/api/v1"
)

// configMapRefIndex ist der Name des Field-Index im Cache.
const configMapRefIndex = ".spec.source.configMapRef.name"

// indexConfigMapRef liefert die ConfigMap, auf die eine StaticSite zeigt.
func indexConfigMapRef(obj client.Object) []string {
	site := obj.(*webv1.StaticSite)
	if site.Spec.Source == nil || site.Spec.Source.ConfigMapRef == nil {
		return nil
	}
	return []string{site.Spec.Source.ConfigMapRef.Name}
}

// sitesForConfigMap übersetzt ein ConfigMap-Ereignis in Requests für alle
// StaticSites im selben Namespace, die auf diese ConfigMap zeigen.
func (r *StaticSiteReconciler) sitesForConfigMap(ctx context.Context,
	cm client.Object) []reconcile.Request {
	var sites webv1.StaticSiteList
	err := r.List(ctx, &sites, client.InNamespace(cm.GetNamespace()),
		client.MatchingFields{configMapRefIndex: cm.GetName()})
	if err != nil {
		logf.FromContext(ctx).Error(err, "StaticSites nicht lesbar")
		return nil
	}
	reqs := make([]reconcile.Request, 0, len(sites.Items))
	for _, s := range sites.Items {
		key := client.ObjectKeyFromObject(&s)
		reqs = append(reqs, reconcile.Request{NamespacedName: key})
	}
	return reqs
}
