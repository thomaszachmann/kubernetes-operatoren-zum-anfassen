package controller

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	webv1alpha1 "example.com/site-operator/api/v1alpha1"
)

// siteFinalizer hält die StaticSite fest, bis der DNS-Eintrag weg ist.
const siteFinalizer = "web.lab.example/finalizer"

// recordName ist der Schlüssel im Registrar: <name>.<namespace>.
func recordName(site *webv1alpha1.StaticSite) string {
	return site.Name + "." + site.Namespace
}

// registerDNS trägt die Site beim Registrar ein (PUT ist idempotent).
func (r *StaticSiteReconciler) registerDNS(ctx context.Context,
	site *webv1alpha1.StaticSite) error {
	if r.Registrar == nil {
		return nil // --registrar-url leer: DNS-Pflege aus
	}
	url := fmt.Sprintf("http://%s.%s.svc", site.Name, site.Namespace)
	return r.Registrar.Upsert(ctx, recordName(site), url)
}

// reconcileDelete räumt extern auf und gibt danach das Objekt frei.
func (r *StaticSiteReconciler) reconcileDelete(ctx context.Context,
	site *webv1alpha1.StaticSite) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(site, siteFinalizer) {
		return ctrl.Result{}, nil // nicht (mehr) unsere Aufgabe
	}
	if r.Registrar != nil {
		if err := r.Registrar.Delete(ctx, recordName(site)); err != nil {
			return ctrl.Result{}, err // Finalizer bleibt, Retry mit Backoff
		}
		logf.FromContext(ctx).Info("DNS-Eintrag entfernt", "record", recordName(site))
	}
	controllerutil.RemoveFinalizer(site, siteFinalizer)
	return ctrl.Result{}, r.Update(ctx, site)
}
