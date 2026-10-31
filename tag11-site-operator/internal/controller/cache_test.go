package controller

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Cache mit Label-Selektor", func() {
	It("sieht fremde Deployments nicht", func(ctx SpecContext) {
		ns := newNamespace(ctx)
		labels := map[string]string{"app": "fremd"}
		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "fremd", Namespace: ns},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{MatchLabels: labels},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: labels},
					Spec: corev1.PodSpec{Containers: []corev1.Container{
						{Name: "c", Image: "nginxinc/nginx-unprivileged:1.31-alpine"}}},
				},
			},
		}
		Expect(k8sClient.Create(ctx, dep)).To(Succeed())
		key := client.ObjectKeyFromObject(dep)

		Expect(k8sClient.Get(ctx, key, &appsv1.Deployment{})).To(Succeed())
		Consistently(func() bool {
			return apierrors.IsNotFound(cached.Get(ctx, key, &appsv1.Deployment{}))
		}, time.Second).Should(BeTrue())
	})
})
