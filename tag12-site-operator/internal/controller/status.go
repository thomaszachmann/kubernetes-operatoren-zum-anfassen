package controller

import (
	"context"
	"errors"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	webv1 "example.com/site-operator/api/v1"
)

// updateStatus leitet den Status aus Deployment und Reconcile-Fehler ab und
// schreibt ihn nur, wenn sich etwas geändert hat (sonst: Hotloop-Gefahr).
func (r *StaticSiteReconciler) updateStatus(ctx context.Context,
	site *webv1.StaticSite, dep *appsv1.Deployment, recErr error) error {
	orig := site.DeepCopy()
	st := &site.Status
	st.ObservedGeneration = site.Generation
	st.URL = fmt.Sprintf("http://%s.%s.svc", site.Name, site.Namespace)

	set := func(typ string, s metav1.ConditionStatus, reason, msg string) {
		meta.SetStatusCondition(&st.Conditions, metav1.Condition{
			Type: typ, Status: s, Reason: reason, Message: msg,
			ObservedGeneration: site.Generation,
		})
	}

	switch {
	case errors.Is(recErr, errContentMissing):
		set(webv1.ConditionDegraded, metav1.ConditionTrue,
			webv1.ReasonContentMissing, recErr.Error())
	case recErr != nil:
		set(webv1.ConditionDegraded, metav1.ConditionTrue,
			webv1.ReasonReconcileError, recErr.Error())
	default:
		set(webv1.ConditionDegraded, metav1.ConditionFalse,
			webv1.ReasonAsExpected, "alle Objekte abgeglichen")
	}

	if dep == nil { // Deployment existiert (noch) nicht
		st.ReadyReplicas = 0
		set(webv1.ConditionAvailable, metav1.ConditionFalse,
			webv1.ReasonReplicasNotReady, "kein Deployment")
	} else {
		setReplicaConditions(site, dep, set)
	}

	setAvailableMetric(site)
	if equality.Semantic.DeepEqual(orig.Status, site.Status) {
		return nil
	}
	r.recordTransitions(orig, site)
	// MergeFrom: nur der Status-Diff geht an den API-Server.
	return r.Status().Patch(ctx, site, client.MergeFrom(orig))
}

func setReplicaConditions(site *webv1.StaticSite, dep *appsv1.Deployment,
	set func(string, metav1.ConditionStatus, string, string)) {
	want, ds := site.Spec.Replicas, dep.Status
	site.Status.ReadyReplicas = ds.ReadyReplicas
	msg := fmt.Sprintf("%d/%d Pods bereit", ds.ReadyReplicas, want)
	if ds.ReadyReplicas >= want {
		set(webv1.ConditionAvailable, metav1.ConditionTrue,
			webv1.ReasonAllReplicasReady, msg)
	} else {
		set(webv1.ConditionAvailable, metav1.ConditionFalse,
			webv1.ReasonReplicasNotReady, msg)
	}
	// Rollout fertig: Deployment-Controller hat die aktuelle Generation gesehen
	// und alle Pods sind aktualisiert und bereit.
	done := ds.ObservedGeneration >= dep.Generation &&
		ds.UpdatedReplicas >= want && ds.ReadyReplicas >= want
	if done {
		set(webv1.ConditionProgressing, metav1.ConditionFalse,
			webv1.ReasonRolloutComplete, msg)
	} else {
		set(webv1.ConditionProgressing, metav1.ConditionTrue,
			webv1.ReasonRolloutInProgress, msg)
	}
}

// recordTransitions erzeugt Events nur bei Zustandswechseln, nicht bei jedem Lauf.
func (r *StaticSiteReconciler) recordTransitions(orig, site *webv1.StaticSite) {
	if r.Recorder == nil { // z. B. im Test ohne Recorder
		return
	}
	for _, typ := range []string{
		webv1.ConditionAvailable, webv1.ConditionDegraded,
	} {
		before := meta.FindStatusCondition(orig.Status.Conditions, typ)
		now := meta.FindStatusCondition(site.Status.Conditions, typ)
		if now == nil || (before != nil && before.Status == now.Status) {
			continue
		}
		evType := corev1.EventTypeNormal
		if typ == webv1.ConditionDegraded && now.Status == metav1.ConditionTrue {
			evType = corev1.EventTypeWarning
		}
		r.Recorder.Eventf(site, nil, evType, now.Reason, "Reconcile",
			"%s=%s: %s", typ, now.Status, now.Message)
	}
}
