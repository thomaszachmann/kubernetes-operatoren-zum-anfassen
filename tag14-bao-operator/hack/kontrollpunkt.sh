#!/usr/bin/env bash
# Kontrollpunkt Tag 14 – erwartet labenv, bao dev und laufenden Manager.
export KUBECONFIG=$PWD/labenv.kubeconfig BAO_ADDR=http://127.0.0.1:18200
export BAO_TOKEN=lab-root   # Lab-Wert
ok() { echo "✔ $1"; }
nok() { echo "✘ $1"; }
check() { if eval "$2" >/dev/null 2>&1; then ok "$1"; else nok "$1"; fi; }

check "OpenBao Dev-Server erreichbar" "bao status"
check "CRD baopolicies installiert" \
  "kubectl get crd baopolicies.bao.lab.example"
check "Token-Secret vorhanden" \
  "kubectl -n bao-operator-system get secret bao-operator-token"
check "webshop-read ist Ready" "kubectl wait -n shop bp/webshop-read \
  --for=condition=Ready --timeout=10s"
check "Policy steht in OpenBao" \
  "bao policy read k8s.shop.webshop-read | grep -q 'secret/data/webshop'"
check "Finalizer gesetzt" "kubectl get bp -n shop webshop-read \
  -o jsonpath='{.metadata.finalizers}' | grep -q bao.lab.example/finalizer"
check "Metrik bao_operator_policy_sync_total" \
  "curl -s 127.0.0.1:18080/metrics | grep -q bao_operator_policy_sync_total"
check "CEL lehnt policy+rules ab" "! kubectl apply --dry-run=server -f - <<'Y'
apiVersion: bao.lab.example/v1alpha1
kind: BaoPolicy
metadata: {name: x, namespace: shop}
spec: {policy: 'path \"x\" {}', rules: [{path: x, capabilities: [read]}]}
Y"
