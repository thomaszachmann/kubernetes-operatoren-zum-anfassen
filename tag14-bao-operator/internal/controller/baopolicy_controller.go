package controller

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	baov1alpha1 "example.com/bao-operator/api/v1alpha1"
)

const finalizerName = "bao.lab.example/finalizer"

// BaoPolicyReconciler synchronisiert BaoPolicy-Objekte nach OpenBao.
type BaoPolicyReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	Recorder          events.EventRecorder
	OperatorNamespace string
	ResyncInterval    time.Duration
}

// +kubebuilder:rbac:groups=bao.lab.example,resources=baopolicies,verbs=get;list
// +kubebuilder:rbac:groups=bao.lab.example,resources=baopolicies,verbs=watch;update
// +kubebuilder:rbac:groups=bao.lab.example,resources=baopolicies/status,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=events.k8s.io,resources=events,verbs=create;patch

func (r *BaoPolicyReconciler) Reconcile(
	ctx context.Context, req ctrl.Request,
) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	var bp baov1alpha1.BaoPolicy
	if err := r.Get(ctx, req.NamespacedName, &bp); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	name := policyName(&bp)

	if !bp.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, r.finalize(ctx, &bp, name)
	}
	if controllerutil.AddFinalizer(&bp, finalizerName) {
		if err := r.Update(ctx, &bp); err != nil {
			return ctrl.Result{}, err
		}
	}

	before := bp.Status.DeepCopy()
	bp.Status.PolicyName = name
	result, reason, err := r.sync(ctx, &bp, name)
	policySyncTotal.WithLabelValues(result).Inc()
	if err != nil {
		log.Error(err, "Sync fehlgeschlagen", "policy", name)
		r.setConditions(&bp, metav1.ConditionFalse, reason, err.Error())
		r.Recorder.Eventf(&bp, nil, corev1.EventTypeWarning, reason,
			"SyncPolicy", "%v", err)
	} else {
		bp.Status.ObservedGeneration = bp.Generation
		r.setConditions(&bp, metav1.ConditionTrue, reason, "Policy "+name+" aktiv")
	}
	if !equality.Semantic.DeepEqual(before, &bp.Status) {
		if uerr := r.Status().Update(ctx, &bp); uerr != nil {
			return ctrl.Result{}, uerr
		}
	}
	if err != nil {
		return ctrl.Result{}, err
	}
	log.V(1).Info("Policy synchronisiert", "policy", name, "result", result)
	return ctrl.Result{RequeueAfter: r.ResyncInterval}, nil
}

// SetupWithManager: BaoPolicy plus das Token-Secret beobachten.
func (r *BaoPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	isToken := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetNamespace() == r.OperatorNamespace &&
			o.GetName() == TokenSecretName
	})
	return ctrl.NewControllerManagedBy(mgr).
		For(&baov1alpha1.BaoPolicy{}).
		Watches(&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.allPolicies),
			builder.WithPredicates(isToken)).
		Named("baopolicy").
		Complete(r)
}

// allPolicies: neues/rotiertes Token -> alle Policies neu abgleichen.
func (r *BaoPolicyReconciler) allPolicies(
	ctx context.Context, _ client.Object,
) []reconcile.Request {
	var list baov1alpha1.BaoPolicyList
	if err := r.List(ctx, &list); err != nil {
		logf.FromContext(ctx).Error(err, "BaoPolicies auflisten")
		return nil
	}
	reqs := make([]reconcile.Request, 0, len(list.Items))
	for _, bp := range list.Items {
		reqs = append(reqs, reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(&bp),
		})
	}
	return reqs
}
