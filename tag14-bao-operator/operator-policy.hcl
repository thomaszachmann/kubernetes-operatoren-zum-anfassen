# Der Operator darf nur Policies mit Präfix k8s. verwalten.
path "sys/policies/acl/k8s.*" {
  capabilities = ["create", "read", "update", "delete"]
}
