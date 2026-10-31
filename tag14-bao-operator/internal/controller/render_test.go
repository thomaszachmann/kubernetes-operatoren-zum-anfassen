package controller

import (
	"testing"

	baov1alpha1 "example.com/bao-operator/api/v1alpha1"
)

const capRead baov1alpha1.Capability = "read"

func TestRenderRules(t *testing.T) {
	tests := []struct {
		name  string
		rules []baov1alpha1.PolicyRule
		want  string
	}{
		{"eine Regel", []baov1alpha1.PolicyRule{
			{Path: "secret/data/app/*", Capabilities: []baov1alpha1.Capability{
				capRead, "list"}},
		}, "path \"secret/data/app/*\" {\n  capabilities = [\"read\", \"list\"]\n}\n"},
		{"Anführungszeichen werden escaped", []baov1alpha1.PolicyRule{
			{Path: `a" {`, Capabilities: []baov1alpha1.Capability{"deny"}},
		}, "path \"a\\\" {\" {\n  capabilities = [\"deny\"]\n}\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := renderRules(tt.rules); got != tt.want {
				t.Errorf("renderRules() =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}
