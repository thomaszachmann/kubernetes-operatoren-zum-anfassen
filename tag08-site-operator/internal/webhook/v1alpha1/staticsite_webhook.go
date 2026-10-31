package v1alpha1

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	webv1alpha1 "example.com/site-operator/api/v1alpha1"
)

// AllowedImagePrefixes ist die Registry-Policy des Lab-Clusters.
var AllowedImagePrefixes = []string{"nginxinc/", "registry.lab.example/"}

var siteGK = webv1alpha1.GroupVersion.WithKind("StaticSite").GroupKind()

// SetupStaticSiteWebhookWithManager registriert Defaulter und Validator.
func SetupStaticSiteWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &webv1alpha1.StaticSite{}).
		WithDefaulter(&StaticSiteDefaulter{}).
		WithValidator(&StaticSiteValidator{Reader: mgr.GetAPIReader()}).
		Complete()
}

// StaticSiteDefaulter normalisiert spec.path.
// +kubebuilder:object:generate=false
type StaticSiteDefaulter struct{}

// Default macht aus "docs" oder "/docs" den Pfad "/docs/".
func (d *StaticSiteDefaulter) Default(_ context.Context,
	site *webv1alpha1.StaticSite) error {
	p := strings.Trim(site.Spec.Path, "/")
	if p == "" {
		site.Spec.Path = "/"
		return nil
	}
	site.Spec.Path = "/" + p + "/"
	return nil
}

// StaticSiteValidator prüft, was CEL nicht kann: Policy und Nachbarobjekte.
// +kubebuilder:object:generate=false
type StaticSiteValidator struct {
	Reader client.Reader // liest direkt vom API-Server, ohne Cache
}

func (v *StaticSiteValidator) ValidateCreate(ctx context.Context,
	site *webv1alpha1.StaticSite) (admission.Warnings, error) {
	return v.validate(ctx, site)
}

func (v *StaticSiteValidator) ValidateUpdate(ctx context.Context,
	_, site *webv1alpha1.StaticSite) (admission.Warnings, error) {
	return v.validate(ctx, site)
}

func (v *StaticSiteValidator) ValidateDelete(context.Context,
	*webv1alpha1.StaticSite) (admission.Warnings, error) {
	return nil, nil
}

// validate: Registry-Policy hart durchsetzen, fehlende ConfigMap nur melden.
func (v *StaticSiteValidator) validate(ctx context.Context,
	site *webv1alpha1.StaticSite) (admission.Warnings, error) {
	if !allowedImage(site.Spec.Image) {
		return nil, apierrors.NewInvalid(siteGK, site.Name, field.ErrorList{
			field.Forbidden(field.NewPath("spec", "image"),
				fmt.Sprintf("erlaubt sind nur %v", AllowedImagePrefixes)),
		})
	}
	src := site.Spec.ContentFrom
	if src == nil || src.ConfigMapRef == nil {
		return nil, nil
	}
	key := client.ObjectKey{Namespace: site.Namespace, Name: src.ConfigMapRef.Name}
	err := v.Reader.Get(ctx, key, &corev1.ConfigMap{})
	if apierrors.IsNotFound(err) { // GitOps: Die ConfigMap kommt evtl. später
		return admission.Warnings{"ConfigMap " + key.Name + " fehlt (noch)"}, nil
	}
	return nil, nil // andere Fehler blockieren nicht, der Controller meldet sie
}

func allowedImage(image string) bool {
	for _, p := range AllowedImagePrefixes {
		if strings.HasPrefix(image, p) {
			return true
		}
	}
	return false
}
