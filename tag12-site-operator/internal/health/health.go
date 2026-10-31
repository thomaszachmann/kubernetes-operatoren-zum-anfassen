// Package health enthält Readiness-Checks für den Manager.
package health

import (
	"context"
	"errors"
	"net/http"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

// CacheSynced meldet "bereit" erst, wenn alle Informer ihre erste Liste haben.
// Fehlt z. B. RBAC für einen beobachteten Typ, bleibt der Pod NotReady.
func CacheSynced(c cache.Cache) healthz.Checker {
	return func(req *http.Request) error {
		ctx, cancel := context.WithTimeout(req.Context(), time.Second)
		defer cancel()
		if !c.WaitForCacheSync(ctx) {
			return errors.New("informer-cache nicht synchron")
		}
		return nil
	}
}
