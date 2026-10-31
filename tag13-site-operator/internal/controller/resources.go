package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"maps"
	"path"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	webv1 "example.com/site-operator/api/v1"
)

const (
	contentHashAnnotation = "web.lab.example/content-hash"
	contentKey            = "index.html"
	htmlRoot              = "/usr/share/nginx/html"
	containerPort         = 8080
)

// selectorLabels sind unveränderlich: nur der Name, sonst bricht der Selector.
func selectorLabels(site *webv1.StaticSite) map[string]string {
	return map[string]string{"app.kubernetes.io/name": site.Name}
}

// labelsFor hängt an alle erzeugten Objekte dieselben Labels.
func labelsFor(site *webv1.StaticSite) map[string]string {
	l := selectorLabels(site)
	l["app.kubernetes.io/managed-by"] = "site-operator"
	return l
}

func contentConfigMapName(site *webv1.StaticSite) string {
	return site.Name + "-content"
}

// contentHash ändert sich mit dem Inhalt und löst so einen Rollout aus.
func contentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:8])
}

func setLabels(obj *metav1.ObjectMeta, site *webv1.StaticSite) {
	if obj.Labels == nil {
		obj.Labels = map[string]string{}
	}
	maps.Copy(obj.Labels, labelsFor(site))
}

// mutateConfigMap setzt nur die Felder, die dem Operator gehören.
func mutateConfigMap(cm *corev1.ConfigMap, site *webv1.StaticSite,
	content string) {
	setLabels(&cm.ObjectMeta, site)
	cm.Data = map[string]string{contentKey: content}
}

func mutateDeployment(dep *appsv1.Deployment, site *webv1.StaticSite,
	hash string) {
	setLabels(&dep.ObjectMeta, site)
	dep.Spec.Replicas = ptr.To(site.Spec.Replicas)
	if dep.Spec.Selector == nil { // Selector ist nach dem Anlegen unveränderlich
		dep.Spec.Selector = &metav1.LabelSelector{MatchLabels: selectorLabels(site)}
	}
	tpl := &dep.Spec.Template
	setLabels(&tpl.ObjectMeta, site)
	if tpl.Annotations == nil {
		tpl.Annotations = map[string]string{}
	}
	tpl.Annotations[contentHashAnnotation] = hash

	if len(tpl.Spec.Containers) == 0 {
		tpl.Spec.Containers = []corev1.Container{{Name: "nginx"}}
	}
	c := &tpl.Spec.Containers[0] // nur Felder setzen, Server-Defaults behalten
	c.Image = site.Spec.Image
	c.Ports = []corev1.ContainerPort{{
		Name: "http", ContainerPort: containerPort, Protocol: corev1.ProtocolTCP,
	}}
	c.VolumeMounts = []corev1.VolumeMount{{
		Name: "content", MountPath: path.Join(htmlRoot, site.Spec.Path),
		ReadOnly: true,
	}}
	tpl.Spec.Volumes = []corev1.Volume{{
		Name: "content",
		VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: contentConfigMapName(site),
			},
			DefaultMode: ptr.To[int32](0o644), // sonst ergänzt der Server 420
		}},
	}}
}
