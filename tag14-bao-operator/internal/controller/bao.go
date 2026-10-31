package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/openbao/openbao/api/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// TokenSecretName liegt im Operator-Namespace, Keys: address, token.
const TokenSecretName = "bao-operator-token"

// baoClient baut bei jedem Reconcile einen Client aus dem Secret –
// ein rotiertes Token wirkt so ohne Neustart.
func (r *BaoPolicyReconciler) baoClient(ctx context.Context) (*api.Client, error) {
	var s corev1.Secret
	key := types.NamespacedName{Namespace: r.OperatorNamespace, Name: TokenSecretName}
	if err := r.Get(ctx, key, &s); err != nil {
		return nil, fmt.Errorf("secret %s: %w", key, err)
	}
	addr, token := string(s.Data["address"]), string(s.Data["token"])
	if addr == "" || token == "" {
		return nil, fmt.Errorf("secret %s: address oder token fehlt", key)
	}
	cfg := api.DefaultConfig()
	cfg.Address = addr
	cfg.Timeout = 10 * time.Second
	c, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	c.SetToken(token)
	return c, nil
}
