package main

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func ns(name, team string) *corev1.Namespace {
	n := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	if team != "" {
		n.Labels = map[string]string{labelKey: team}
	}
	return n
}

func pod(ns, name string, lbl map[string]string) *corev1.Pod {
	return &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Namespace: ns, Name: name, Labels: lbl}}
}

func TestSync(t *testing.T) {
	tests := []struct {
		name    string
		team    string            // Label am Namespace ("" = keins)
		podLbl  map[string]string // Labels am Pod vorher
		want    string
		wantHas bool
	}{
		{"setzt Label", "blau", nil, "blau", true},
		{"korrigiert Label", "blau", map[string]string{labelKey: "rot"}, "blau", true},
		{"entfernt Label", "", map[string]string{labelKey: "rot"}, "", false},
		{"lässt fremde Pods", "", nil, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			client := fake.NewClientset(ns("shop", tt.team),
				pod("shop", "web-1", tt.podLbl))
			f := informers.NewSharedInformerFactory(client, 0)
			c := New(client, f)
			f.Start(ctx.Done())
			if !cache.WaitForCacheSync(ctx.Done(), c.synced...) {
				t.Fatal("cache nicht synchron")
			}

			if err := c.sync(ctx, "shop"); err != nil {
				t.Fatalf("sync: %v", err)
			}

			p, err := client.CoreV1().Pods("shop").Get(ctx, "web-1",
				metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}
			got, has := p.Labels[labelKey]
			if got != tt.want || has != tt.wantHas {
				t.Errorf("label = %q (%v), want %q (%v)", got, has, tt.want, tt.wantHas)
			}
		})
	}
}

func TestSyncNamespaceWeg(t *testing.T) {
	client := fake.NewClientset()
	c := New(client, informers.NewSharedInformerFactory(client, 0))
	if err := c.sync(context.Background(), "gibtsnicht"); err != nil {
		t.Errorf("sync auf fehlenden Namespace: %v", err)
	}
}

func TestSyncIdempotent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := fake.NewClientset(ns("shop", "blau"), pod("shop", "web-1", nil))
	f := informers.NewSharedInformerFactory(client, 0)
	c := New(client, f)
	f.Start(ctx.Done())
	cache.WaitForCacheSync(ctx.Done(), c.synced...)

	if err := c.sync(ctx, "shop"); err != nil {
		t.Fatal(err)
	}
	// Warten, bis der Informer den gepatchten Pod im Cache hat.
	for {
		p, _ := c.podList.Pods("shop").Get("web-1")
		if p.Labels[labelKey] == "blau" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	client.ClearActions()
	if err := c.sync(ctx, "shop"); err != nil {
		t.Fatal(err)
	}
	for _, a := range client.Actions() {
		if a.GetVerb() == "patch" {
			t.Errorf("zweiter sync patcht erneut: %v", a)
		}
	}
}
