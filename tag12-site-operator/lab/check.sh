ok()  { echo "✔ $1"; }
nok() { echo "✘ $1"; }
docker build --check . >/dev/null 2>&1 && ok "Dockerfile-Check" || nok "Dockerfile"
[ -s lab/manager.tar ] && ok "ko-Image als Tarball" || nok "ko-Tarball fehlt"
grep -q 'kind: CustomResourceDefinition' dist/install.yaml \
  && ok "install.yaml mit CRD" || nok "install.yaml"
bin/kustomize build config/overlays/prod | grep -q 'replicas: 2' \
  && ok "Prod-Overlay" || nok "Overlay"
helm lint dist/chart >/dev/null 2>&1 && ok "helm lint" || nok "helm lint"
helm template t dist/chart | grep -q 'resource-policy": keep' \
  && ok "CRD bleibt bei uninstall" || nok "CRD-Policy"
kubectl get crd staticsites.web.lab.example >/dev/null 2>&1 \
  && ok "CRD nach uninstall vorhanden" || nok "CRD gelöscht"
