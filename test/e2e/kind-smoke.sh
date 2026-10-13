#!/usr/bin/env bash
set -euo pipefail

kubectl port-forward deployment/health-bridge 18080:8080 >/tmp/bridge-port-forward.log 2>&1 &
forward_pid=$!
trap 'kill "$forward_pid" 2>/dev/null || true' EXIT

for attempt in $(seq 1 30); do
  if curl --silent --fail http://127.0.0.1:18080/healthz >/dev/null; then break; fi
  sleep 1
done

assert_code() {
  local expected="$1" url="$2" actual
  actual=$(curl --silent --output /tmp/bridge-response --write-out '%{http_code}' "$url")
  if [[ "$actual" != "$expected" ]]; then
    echo "Expected HTTP $expected for $url; got $actual"
    cat /tmp/bridge-response
    exit 1
  fi
}

echo "Verify process liveness and missing-consumer readiness"
assert_code 200 http://127.0.0.1:18080/healthz
assert_code 503 http://127.0.0.1:18080/readyz
assert_code 200 http://127.0.0.1:18080/status
assert_code 200 http://127.0.0.1:18080/metrics

echo "Stop upstream NATS and verify bridge does not enter a restart storm"
before=$(kubectl get pod -l app=health-bridge -o jsonpath='{.items[0].status.containerStatuses[0].restartCount}')
kubectl scale deployment/nats --replicas=0
kubectl rollout status deployment/nats --timeout=60s
sleep 5
assert_code 200 http://127.0.0.1:18080/healthz
assert_code 503 http://127.0.0.1:18080/readyz
after=$(kubectl get pod -l app=health-bridge -o jsonpath='{.items[0].status.containerStatuses[0].restartCount}')
if [[ "$after" != "$before" ]]; then
  echo "Bridge restarted during NATS outage: before=$before after=$after"
  exit 1
fi
echo "Kind probe smoke test passed"
