package controller

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	webv1alpha1 "example.com/site-operator/api/v1alpha1"
)

// Reiner Unit-Test: kein API-Server, läuft in Millisekunden.
func TestMutateDeployment(t *testing.T) {
	tests := []struct {
		name, path, wantMount string
	}{
		{"Wurzel", "/", "/usr/share/nginx/html"},
		{"Unterpfad", "/docs", "/usr/share/nginx/html/docs"},
		{"ohne führenden Slash", "blog/", "/usr/share/nginx/html/blog"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			site := &webv1alpha1.StaticSite{
				ObjectMeta: metav1.ObjectMeta{Name: "demo"},
				Spec: webv1alpha1.StaticSiteSpec{
					Path: tt.path, Replicas: 2, Image: "nginx:lab",
				},
			}
			dep := &appsv1.Deployment{}
			mutateDeployment(dep, site, "abc")

			mount := dep.Spec.Template.Spec.Containers[0].VolumeMounts[0]
			if mount.MountPath != tt.wantMount {
				t.Errorf("MountPath = %q, want %q", mount.MountPath, tt.wantMount)
			}
			before := dep.DeepCopy()
			mutateDeployment(dep, site, "abc") // zweiter Lauf darf nichts ändern
			if !equality.Semantic.DeepEqual(before, dep) {
				t.Error("mutateDeployment ist nicht idempotent")
			}
		})
	}
}
