package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Capability ist eine OpenBao-Capability.
// +kubebuilder:validation:Enum=create;read;update;patch;delete;list;sudo;deny
type Capability string

// PolicyRule beschreibt einen path-Block der Policy.
type PolicyRule struct {
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=512
	Path string `json:"path"`
	// +kubebuilder:validation:MinItems=1
	// +listType=set
	Capabilities []Capability `json:"capabilities"`
}

// BaoPolicySpec: genau eins von policy (HCL) oder rules.
// +kubebuilder:validation:XValidation:rule="has(self.policy) != has(self.rules)"
type BaoPolicySpec struct {
	// HCL-Text der Policy.
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=65536
	Policy string `json:"policy,omitempty"`
	// Strukturierte Regeln; der Operator rendert daraus HCL.
	// +optional
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=64
	// +listType=atomic
	Rules []PolicyRule `json:"rules,omitempty"`
}

// BaoPolicyStatus ist der beobachtete Zustand.
type BaoPolicyStatus struct {
	// Name der Policy in OpenBao: k8s.<namespace>.<name>
	// +optional
	PolicyName string `json:"policyName,omitempty"`
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=bp
// +kubebuilder:printcolumn:name="Policy",type=string,JSONPath=`.status.policyName`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// BaoPolicy verwaltet eine ACL-Policy in OpenBao.
type BaoPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`

	Spec   BaoPolicySpec   `json:"spec"`
	Status BaoPolicyStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// BaoPolicyList enthält eine Liste von BaoPolicy.
type BaoPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []BaoPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &BaoPolicy{}, &BaoPolicyList{})
		return nil
	})
}
