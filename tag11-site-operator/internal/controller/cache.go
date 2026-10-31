package controller

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CacheOptions: Deployments und Services nur cachen, wenn sie uns gehören.
// ConfigMaps bleiben ungefiltert – die Quellen der Nutzer tragen kein Label.
func CacheOptions() cache.Options {
	mine := labels.SelectorFromSet(labels.Set{
		"app.kubernetes.io/managed-by": "site-operator",
	})
	return cache.Options{ByObject: map[client.Object]cache.ByObject{
		&appsv1.Deployment{}: {Label: mine},
		&corev1.Service{}:    {Label: mine},
	}}
}
