package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	webv1 "example.com/site-operator/api/v1"
)

var (
	// childWrites zählt, was CreateOrUpdate mit den Kindobjekten gemacht hat.
	// Steigt "updated" ständig, kämpft jemand mit dem Operator (Tag 13).
	childWrites = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "site_operator_child_writes_total",
		Help: "Abgleich der Kindobjekte nach Art und Ergebnis",
	}, []string{"kind", "operation"})

	// siteAvailable ist 1, wenn die Condition Available=True ist, sonst 0.
	siteAvailable = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "site_operator_site_available",
		Help: "Available-Condition je StaticSite (1 = True)",
	}, []string{"namespace", "name"})
)

func init() {
	// Die Registry des Managers – alles darin erscheint unter /metrics.
	metrics.Registry.MustRegister(childWrites, siteAvailable)
}

func countWrite(kind string, op controllerutil.OperationResult) {
	childWrites.WithLabelValues(kind, string(op)).Inc()
}

func setAvailableMetric(site *webv1.StaticSite) {
	v := 0.0
	if meta.IsStatusConditionTrue(site.Status.Conditions,
		webv1.ConditionAvailable) {
		v = 1
	}
	siteAvailable.WithLabelValues(site.Namespace, site.Name).Set(v)
}
