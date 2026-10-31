package v1

import (
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	webv1 "example.com/site-operator/api/v1"
)

// nolint:unused
// log is for logging in this package.
var staticsitelog = logf.Log.WithName("staticsite-resource")

// SetupStaticSiteWebhookWithManager registers the webhook for StaticSite in the manager.
func SetupStaticSiteWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &webv1.StaticSite{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
