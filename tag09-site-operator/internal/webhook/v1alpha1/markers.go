// Marker für controller-gen: erzeugen config/webhook/manifests.yaml.
package v1alpha1

// +kubebuilder:webhook:path=/mutate-web-lab-example-v1alpha1-staticsite,mutating=true,failurePolicy=fail,sideEffects=None,groups=web.lab.example,resources=staticsites,verbs=create;update,versions=v1alpha1,name=mstaticsite-v1alpha1.kb.io,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/validate-web-lab-example-v1alpha1-staticsite,mutating=false,failurePolicy=fail,sideEffects=None,groups=web.lab.example,resources=staticsites,verbs=create;update,versions=v1alpha1,name=vstaticsite-v1alpha1.kb.io,admissionReviewVersions=v1
