ok()  { echo "✔ $1"; }
nok() { echo "✘ $1"; }
chk() { if eval "$2" >/dev/null 2>&1; then ok "$1"; else nok "$1"; fi; }
j() { kubectl get "$1" extern -o jsonpath="$2"; }
chk "Deployment gehört StaticSite/extern" \
  '[ "$(j deployment "{.metadata.ownerReferences[0].name}")" = extern ]'
ann='{.spec.template.metadata.annotations.web\.lab\.example/content-hash}'
chk "Pod-Template trägt content-hash" '[ -n "$(j deployment "$ann")" ]'
chk "Service 80 -> 8080" \
  '[ "$(j service "{.spec.ports[0].port}->{.spec.ports[0].targetPort}")" = "80->8080" ]'
chk "ConfigMap extern-content hat index.html" \
  'kubectl get configmap extern-content -o jsonpath="{.data.index\.html}" | grep -q h1'
kubectl scale deployment extern --replicas=5 >/dev/null; sleep 2
chk "Drift korrigiert (replicas=1)" '[ "$(j deployment "{.spec.replicas}")" = 1 ]'
