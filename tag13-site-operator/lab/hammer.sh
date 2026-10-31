# 100 Runden: Spec ändern und gleichzeitig Deployment/Service annotieren
for i in $(seq 1 100); do
  kubectl patch staticsite demo --type=merge \
    -p "{\"spec\":{\"replicas\":$((i % 3 + 1))}}" >/dev/null &
  kubectl annotate deploy demo hammer=$i --overwrite >/dev/null &
  kubectl annotate svc demo hammer=$i --overwrite >/dev/null &
done
wait
