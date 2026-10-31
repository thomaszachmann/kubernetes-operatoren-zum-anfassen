package v1alpha1

import (
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	webv1 "example.com/site-operator/api/v1"
)

// ConvertTo wandelt diese Spoke-Version (v1alpha1) in den Hub (v1).
func (src *StaticSite) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*webv1.StaticSite)
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec = webv1.StaticSiteSpec{
		Content:  src.Spec.Content,
		Replicas: src.Spec.Replicas,
		Image:    src.Spec.Image,
		Path:     src.Spec.Path,
	}
	if from := src.Spec.ContentFrom; from != nil { // contentFrom -> source
		dst.Spec.Source = &webv1.ContentSource{}
		if from.ConfigMapRef != nil {
			dst.Spec.Source.ConfigMapRef = &webv1.ConfigMapReference{
				Name: from.ConfigMapRef.Name}
		}
	}
	dst.Status = webv1.StaticSiteStatus(src.Status) // identische Felder
	return nil
}

// ConvertFrom wandelt den Hub (v1) zurück in v1alpha1.
func (dst *StaticSite) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*webv1.StaticSite)
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec = StaticSiteSpec{
		Content:  src.Spec.Content,
		Replicas: src.Spec.Replicas,
		Image:    src.Spec.Image,
		Path:     src.Spec.Path,
	}
	if s := src.Spec.Source; s != nil { // source -> contentFrom
		dst.Spec.ContentFrom = &ContentSource{}
		if s.ConfigMapRef != nil {
			dst.Spec.ContentFrom.ConfigMapRef = &ConfigMapReference{
				Name: s.ConfigMapRef.Name}
		}
	}
	dst.Status = StaticSiteStatus(src.Status)
	return nil
}
