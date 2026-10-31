package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// resultError ist das Label für fehlgeschlagene Syncs.
const resultError = "error"

// policySyncTotal zählt Syncs nach Ergebnis:
// created, updated, drift, unchanged, deleted, error.
var policySyncTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "bao_operator_policy_sync_total",
	Help: "Anzahl der Policy-Syncs gegen OpenBao nach Ergebnis.",
}, []string{"result"})

func init() {
	metrics.Registry.MustRegister(policySyncTotal)
}
