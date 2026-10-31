package controller

import (
	"fmt"
	"strconv"
	"strings"

	baov1alpha1 "example.com/bao-operator/api/v1alpha1"
)

// policyName ist eindeutig: Namespaces enthalten keine Punkte.
func policyName(bp *baov1alpha1.BaoPolicy) string {
	return "k8s." + bp.Namespace + "." + bp.Name
}

// desiredPolicy liefert das HCL, das in OpenBao stehen soll.
func desiredPolicy(spec baov1alpha1.BaoPolicySpec) string {
	if spec.Policy != "" {
		return spec.Policy
	}
	return renderRules(spec.Rules)
}

// renderRules erzeugt deterministisches HCL; Quote verhindert HCL-Injection.
func renderRules(rules []baov1alpha1.PolicyRule) string {
	var b strings.Builder
	for i, r := range rules {
		if i > 0 {
			b.WriteString("\n")
		}
		caps := make([]string, len(r.Capabilities))
		for j, c := range r.Capabilities {
			caps[j] = strconv.Quote(string(c))
		}
		fmt.Fprintf(&b, "path %s {\n  capabilities = [%s]\n}\n",
			strconv.Quote(r.Path), strings.Join(caps, ", "))
	}
	return b.String()
}
