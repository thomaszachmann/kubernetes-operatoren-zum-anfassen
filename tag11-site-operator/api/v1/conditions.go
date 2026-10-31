package v1

// Condition-Typen einer StaticSite (Kubernetes-API-Konventionen).
const (
	ConditionAvailable   = "Available"   // genug Pods bereit
	ConditionProgressing = "Progressing" // Rollout läuft
	ConditionDegraded    = "Degraded"    // Operator kann nicht abgleichen
)

// Reasons sind CamelCase, maschinenlesbar und stabil – Alerts hängen daran.
const (
	ReasonAllReplicasReady  = "AllReplicasReady"
	ReasonReplicasNotReady  = "ReplicasNotReady"
	ReasonRolloutInProgress = "RolloutInProgress"
	ReasonRolloutComplete   = "RolloutComplete"
	ReasonContentMissing    = "ContentMissing"
	ReasonReconcileError    = "ReconcileError"
	ReasonAsExpected        = "AsExpected"
)
