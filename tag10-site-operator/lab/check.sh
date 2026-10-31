ok()  { echo "✔ $1"; }
nok() { echo "✘ $1"; }
P=./internal/controller/
env -u KUBEBUILDER_ASSETS go test $P -count=1 \
  -run 'TestMutateDeployment|TestReconcileContentMissing' >/dev/null 2>&1 \
  && ok "Unit-Tests ohne envtest" || nok "Unit-Tests"
out=$(go test $P -count=1 -v -ginkgo.no-color -coverprofile=cover.out 2>&1)
echo "$out" | grep -q '^SUCCESS!' && ok "envtest-Suite grün" || nok "Suite rot"
n=$(echo "$out" | sed -nE 's/^Ran ([0-9]+) of.*/\1/p')
[ "${n:-0}" -ge 14 ] && ok "$n Specs" || nok "zu wenige Specs (${n:-0})"
cov=$(go tool cover -func=cover.out | awk '/^total:/ {print int($3)}')
[ "${cov:-0}" -ge 80 ] && ok "Coverage ${cov}%" || nok "Coverage ${cov:-0}%"
grep -l 'time.Sleep' internal/controller/*_test.go >/dev/null \
  && nok "time.Sleep in Tests" || ok "kein time.Sleep – nur Eventually"
