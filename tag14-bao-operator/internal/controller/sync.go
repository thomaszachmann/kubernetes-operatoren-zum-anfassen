package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	baov1alpha1 "example.com/bao-operator/api/v1alpha1"
)

// reasonBaoError: OpenBao hat einen Fehler gemeldet.
const reasonBaoError = "BaoError"

// sync schreibt die Policy, wenn sie fehlt oder abweicht (Drift).
func (r *BaoPolicyReconciler) sync(
	ctx context.Context, bp *baov1alpha1.BaoPolicy, name string,
) (result, reason string, err error) {
	bao, err := r.baoClient(ctx)
	if err != nil {
		return resultError, "TokenUnavailable", err
	}
	want := desiredPolicy(bp.Spec)
	got, err := bao.Sys().GetPolicyWithContext(ctx, name)
	if err != nil {
		return resultError, reasonBaoError, err
	}
	switch {
	case got == want:
		return "unchanged", "InSync", nil
	case got == "":
		result, reason = "created", "Created"
	case bp.Generation != bp.Status.ObservedGeneration:
		result, reason = "updated", "Updated"
	default:
		result, reason = "drift", "DriftCorrected"
	}
	if err := bao.Sys().PutPolicyWithContext(ctx, name, want); err != nil {
		return resultError, reasonBaoError, err
	}
	typ, note := corev1.EventTypeNormal, "Policy %s geschrieben"
	if result == "drift" {
		typ, note = corev1.EventTypeWarning, "Drift in %s korrigiert"
	}
	r.Recorder.Eventf(bp, nil, typ, reason, "PutPolicy", note, name)
	return result, reason, nil
}

func (r *BaoPolicyReconciler) setConditions(
	bp *baov1alpha1.BaoPolicy, s metav1.ConditionStatus, reason, msg string,
) {
	ready := "PolicyActive"
	if s == metav1.ConditionFalse {
		ready = "SyncFailed"
	}
	for _, c := range [][2]string{{"Synced", reason}, {"Ready", ready}} {
		meta.SetStatusCondition(&bp.Status.Conditions, metav1.Condition{
			Type: c[0], Status: s, Reason: c[1], Message: msg,
			ObservedGeneration: bp.Generation,
		})
	}
}

// finalize löscht die Policy in OpenBao, bevor das Objekt verschwindet.
func (r *BaoPolicyReconciler) finalize(
	ctx context.Context, bp *baov1alpha1.BaoPolicy, name string,
) error {
	if !controllerutil.ContainsFinalizer(bp, finalizerName) {
		return nil
	}
	bao, err := r.baoClient(ctx)
	if err != nil {
		return err
	}
	if err := bao.Sys().DeletePolicyWithContext(ctx, name); err != nil {
		policySyncTotal.WithLabelValues(resultError).Inc()
		return err
	}
	policySyncTotal.WithLabelValues("deleted").Inc()
	r.Recorder.Eventf(bp, nil, corev1.EventTypeNormal, "Deleted",
		"DeletePolicy", "Policy %s gelöscht", name)
	controllerutil.RemoveFinalizer(bp, finalizerName)
	return r.Update(ctx, bp)
}
