ok()  { echo "✔ $1"; }
nok() { echo "✘ $1"; }
SA=system:serviceaccount:site-operator-system:site-operator-controller-manager
sleep 4
[ "$(kubectl auth can-i watch services --as=$SA)" = yes ] \
  && ok "RBAC: Services beobachten" || nok "RBAC fehlt"
[ "$(kubectl auth can-i delete deployments --as=$SA)" = no ] \
  && ok "RBAC: kein delete" || nok "RBAC zu weit"
curl -s 'localhost:8081/readyz?verbose' | grep -q '\[+\]informers ok' \
  && ok "readyz mit Cache-Check" || nok "readyz"
T=$(kubectl create token prometheus)
curl -sk -H "Authorization: Bearer $T" https://localhost:8443/metrics \
  | grep -q '^site_operator_site_available' && ok "eigene Metriken" || nok "Metriken"
[ "$(curl -sk -o /dev/null -w '%{http_code}' https://localhost:8443/metrics)" = 401 ] \
  && ok "Metriken ohne Token gesperrt" || nok "Metriken offen"
grep -q 'leader-election-namespace' cmd/main.go && ok "Lease-Namespace-Flag" \
  || nok "Flag fehlt"
