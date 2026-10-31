ok()  { echo "✔ $1"; }
nok() { echo "✘ $1"; }
cond() { kubectl get staticsite "$1" \
  -o jsonpath="{.status.conditions[?(@.type==\"$2\")].status}"; }
[ "$(kubectl get crd staticsites.web.lab.example \
  -o jsonpath='{.spec.versions[0].additionalPrinterColumns[*].name}')" \
  = "Ready Available URL Age" ] && ok "Printer-Columns" || nok "Printer-Columns"
[ "$(cond demo Available)" = True ] && ok "demo Available=True" || nok "demo"
[ "$(kubectl get staticsite demo -o jsonpath='{.status.observedGeneration}')" \
  = "$(kubectl get staticsite demo -o jsonpath='{.metadata.generation}')" ] \
  && ok "observedGeneration aktuell" || nok "observedGeneration"
[ "$(cond extern Degraded)" = False ] && ok "extern erholt" || nok "extern"
kubectl events --for staticsite/extern --types=Warning | grep -q ContentMissing \
  && ok "Warning-Event ContentMissing" || nok "Event fehlt"
rv=$(kubectl get staticsite demo -o jsonpath='{.metadata.resourceVersion}')
sleep 5
[ "$rv" = "$(kubectl get staticsite demo -o jsonpath='{.metadata.resourceVersion}')" ] \
  && ok "kein Status-Hotloop" || nok "Status ändert sich ständig"
