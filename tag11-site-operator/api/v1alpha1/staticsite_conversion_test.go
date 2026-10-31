package v1alpha1

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	webv1 "example.com/site-operator/api/v1"
)

// Spoke -> Hub -> Spoke muss verlustfrei sein, sonst verliert jedes
// Lesen über die andere Version Daten.
func TestRoundTrip(t *testing.T) {
	ref := &ConfigMapReference{Name: "html"}
	tests := map[string]StaticSiteSpec{
		"content":      {Content: "<h1>Hallo</h1>", Replicas: 2, Path: "/docs/"},
		"contentFrom":  {ContentFrom: &ContentSource{ConfigMapRef: ref}},
		"leere Quelle": {ContentFrom: &ContentSource{}},
	}
	for name, spec := range tests {
		t.Run(name, func(t *testing.T) {
			in := &StaticSite{ObjectMeta: metav1.ObjectMeta{Name: "rt"}, Spec: spec}
			in.Status.URL = "http://rt.default.svc"
			hub := &webv1.StaticSite{}
			if err := in.ConvertTo(hub); err != nil {
				t.Fatal(err)
			}
			out := &StaticSite{}
			if err := out.ConvertFrom(hub); err != nil {
				t.Fatal(err)
			}
			if !equality.Semantic.DeepEqual(in, out) {
				t.Errorf("Round-Trip verändert:\n in %+v\nout %+v", in.Spec, out.Spec)
			}
		})
	}
}
