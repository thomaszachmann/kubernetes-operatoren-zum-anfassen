package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ConfigMapReference zeigt auf eine ConfigMap im selben Namespace.
type ConfigMapReference struct {
	// name der ConfigMap.
	// +kubebuilder:validation:MinLength=1
	// +required
	Name string `json:"name"`
}

// ContentSource beschreibt, woher der HTML-Inhalt kommt.
type ContentSource struct {
	// configMapRef liest den Inhalt aus dem Key index.html einer ConfigMap.
	// +optional
	ConfigMapRef *ConfigMapReference `json:"configMapRef,omitempty"`
}

// StaticSiteSpec ist der Wunschzustand einer StaticSite.
// +kubebuilder:validation:ExactlyOneOf=content;contentFrom
type StaticSiteSpec struct {
	// content ist der HTML-Inhalt der Seite (alternativ zu contentFrom).
	// +kubebuilder:validation:MaxLength=65536
	// +optional
	Content string `json:"content,omitempty"`

	// contentFrom verweist auf eine vorhandene ConfigMap (alternativ zu content).
	// +optional
	ContentFrom *ContentSource `json:"contentFrom,omitempty"`

	// replicas ist die Anzahl der nginx-Pods.
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// image ist das Webserver-Image.
	// +kubebuilder:default="nginxinc/nginx-unprivileged:1.31-alpine"
	// +kubebuilder:validation:XValidation:rule="!self.endsWith(':latest')"
	// +optional
	Image string `json:"image,omitempty"`

	// path ist der URL-Pfad, unter dem die Seite ausgeliefert wird.
	// +kubebuilder:default="/"
	// +kubebuilder:validation:XValidation:rule="self == oldSelf"
	// +optional
	Path string `json:"path,omitempty"`
}

// StaticSiteStatus ist der beobachtete Zustand einer StaticSite.
type StaticSiteStatus struct {
	// readyReplicas ist die Zahl der bereiten Pods des Deployments.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// url ist die clusterinterne Adresse, z. B. http://demo.default.svc.
	// +optional
	URL string `json:"url,omitempty"`

	// observedGeneration ist die zuletzt verarbeitete metadata.generation.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// conditions: Available, Progressing, Degraded.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:deprecatedversion:warning="v1alpha1 ist veraltet, nutze v1"
// +kubebuilder:resource:shortName=site
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.readyReplicas`
// +kubebuilder:printcolumn:name="Available",type=string,JSONPath=`.status.conditions[?(@.type=="Available")].status`
// +kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.status.url`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// StaticSite liefert eine statische Webseite mit nginx aus.
type StaticSite struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec StaticSiteSpec `json:"spec"`

	// +optional
	Status StaticSiteStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// StaticSiteList enthält eine Liste von StaticSites.
type StaticSiteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []StaticSite `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &StaticSite{}, &StaticSiteList{})
		return nil
	})
}
