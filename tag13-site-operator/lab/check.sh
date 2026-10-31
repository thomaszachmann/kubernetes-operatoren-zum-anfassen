ok()  { echo "✔ $1"; }
nok() { echo "✘ $1"; }
out=$(go test ./test/scale/ -count=1 -v -ginkgo.v -ginkgo.no-color 2>&1)
echo "$out" | grep -q '^SUCCESS!' && ok "Mess-Suite grün" || nok "Mess-Suite rot"
echo "$out" | grep -q 'MergeFrom-Patch .* 0 Konflikte' && ok "Patch ohne 409" \
  || nok "Patch-Konflikte"
go test ./internal/... -count=1 >/dev/null 2>&1 && ok "Controller-Tests grün" \
  || nok "Controller-Tests rot"
grep -q 'WithOptions(r.Options)' internal/controller/staticsite_controller.go \
  && ok "Controller-Optionen verdrahtet" || nok "WithOptions fehlt"
grep -q 'apierrors.IsConflict(recErr)' internal/controller/staticsite_controller.go \
  && ok "Konflikte leise" || nok "Konflikt-Behandlung fehlt"
grep -q 'client.FieldOwner(fieldOwner)' internal/controller/staticsite_controller.go \
  && ok "SSA mit Field Owner" || nok "SSA fehlt"
grep -q 'TransformStripManagedFields' internal/controller/cache.go \
  && ok "Cache-Transform" \
  || nok "Transform fehlt"
grep -q 'LeaderElectionReleaseOnCancel: *true' cmd/main.go && ok "ReleaseOnCancel" \
  || nok "ReleaseOnCancel fehlt"
