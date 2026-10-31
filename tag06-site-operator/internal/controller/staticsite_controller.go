package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	webv1alpha1 "example.com/site-operator/api/v1alpha1"
)

// errContentMissing: die referenzierte ConfigMap fehlt (noch).
var errContentMissing = errors.New("content source not found")

// StaticSiteReconciler gleicht StaticSites mit ConfigMap, Deployment und Service ab.
type StaticSiteReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder events.EventRecorder
}

// +kubebuilder:rbac:groups=web.lab.example,resources=staticsites,verbs=get;list;watch
// +kubebuilder:rbac:groups=web.lab.example,resources=staticsites,verbs=update;patch
// +kubebuilder:rbac:groups=web.lab.example,resources=staticsites/status,verbs=get;patch
// +kubebuilder:rbac:groups=web.lab.example,resources=staticsites/status,verbs=update
// +kubebuilder:rbac:groups=web.lab.example,resources=staticsites/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=configmaps;services,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps;services,verbs=create;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=create;update;patch
// +kubebuilder:rbac:groups=events.k8s.io,resources=events,verbs=create;patch

func (r *StaticSiteReconciler) Reconcile(ctx context.Context,
	req ctrl.Request) (ctrl.Result, error) {
	var site webv1alpha1.StaticSite
	if err := r.Get(ctx, req.NamespacedName, &site); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err) // gelöscht: nichts zu tun
	}

	dep, recErr := r.reconcileResources(ctx, &site)
	// Status immer schreiben – gerade im Fehlerfall will der Nutzer wissen, warum.
	if err := r.updateStatus(ctx, &site, dep, recErr); err != nil {
		return ctrl.Result{}, errors.Join(recErr, err)
	}
	if errors.Is(recErr, errContentMissing) {
		logf.FromContext(ctx).Info("Inhalt fehlt, neuer Versuch in 30s",
			"error", recErr.Error())
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}
	return ctrl.Result{}, recErr // nil oder Backoff durch die Workqueue
}

// reconcileResources gleicht ConfigMap, Deployment und Service ab.
func (r *StaticSiteReconciler) reconcileResources(ctx context.Context,
	site *webv1alpha1.StaticSite) (*appsv1.Deployment, error) {
	content, err := r.resolveContent(ctx, site)
	if err != nil {
		return nil, err
	}
	hash, err := r.reconcileConfigMap(ctx, site, content)
	if err != nil {
		return nil, err
	}
	dep, err := r.reconcileDeployment(ctx, site, hash)
	if err != nil {
		return nil, err
	}
	return dep, r.reconcileService(ctx, site)
}

// resolveContent liefert spec.content oder index.html aus der referenzierten ConfigMap.
func (r *StaticSiteReconciler) resolveContent(ctx context.Context,
	site *webv1alpha1.StaticSite) (string, error) {
	src := site.Spec.ContentFrom
	if src == nil || src.ConfigMapRef == nil {
		return site.Spec.Content, nil
	}
	var cm corev1.ConfigMap
	key := client.ObjectKey{Namespace: site.Namespace, Name: src.ConfigMapRef.Name}
	if err := r.Get(ctx, key, &cm); err != nil {
		if apierrors.IsNotFound(err) {
			return "", fmt.Errorf("configmap %s: %w", key.Name, errContentMissing)
		}
		return "", err
	}
	html, ok := cm.Data[contentKey]
	if !ok {
		return "", fmt.Errorf("configmap %s ohne %s: %w", key.Name, contentKey,
			errContentMissing)
	}
	return html, nil
}

func (r *StaticSiteReconciler) reconcileConfigMap(ctx context.Context,
	site *webv1alpha1.StaticSite, content string) (string, error) {
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name: contentConfigMapName(site), Namespace: site.Namespace,
	}}
	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		mutateConfigMap(cm, site, content)
		return controllerutil.SetControllerReference(site, cm, r.Scheme)
	})
	if err != nil {
		return "", fmt.Errorf("configmap: %w", err)
	}
	logf.FromContext(ctx).Info("ConfigMap abgeglichen", "operation", op)
	return contentHash(content), nil
}

func (r *StaticSiteReconciler) reconcileDeployment(ctx context.Context,
	site *webv1alpha1.StaticSite, hash string) (*appsv1.Deployment, error) {
	dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Name: site.Name, Namespace: site.Namespace,
	}}
	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, dep, func() error {
		mutateDeployment(dep, site, hash)
		return controllerutil.SetControllerReference(site, dep, r.Scheme)
	})
	if err != nil {
		return nil, fmt.Errorf("deployment: %w", err)
	}
	logf.FromContext(ctx).Info("Deployment abgeglichen", "operation", op)
	return dep, nil
}

func (r *StaticSiteReconciler) reconcileService(ctx context.Context,
	site *webv1alpha1.StaticSite) error {
	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{
		Name: site.Name, Namespace: site.Namespace,
	}}
	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		mutateService(svc, site)
		return controllerutil.SetControllerReference(site, svc, r.Scheme)
	})
	if err != nil {
		return fmt.Errorf("service: %w", err)
	}
	logf.FromContext(ctx).Info("Service abgeglichen", "operation", op)
	return nil
}

// SetupWithManager: Owns() sorgt dafür, dass Änderungen an den Kindern
// den Besitzer (die StaticSite) erneut in die Queue legen.
func (r *StaticSiteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webv1alpha1.StaticSite{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Named("staticsite").
		Complete(r)
}
