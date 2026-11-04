#!/usr/bin/env bash
set -euo pipefail

bridge_pid=""
nats_pid=""
cleanup() {
  if [[ -n "$bridge_pid" ]]; then kill "$bridge_pid" 2>/dev/null || true; fi
  if [[ -n "$nats_pid" ]]; then kill "$nats_pid" 2>/dev/null || true; fi
}
trap cleanup EXIT

kubectl port-forward deployment/health-bridge 18080:8080 >/tmp/bridge-port-forward.log 2>&1 &
bridge_pid=$!

for attempt in $(seq 1 40); do
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

wait_code() {
  local expected="$1" url="$2"
  for attempt in $(seq 1 25); do
    if [[ "$(curl --silent --output /dev/null --write-out '%{http_code}' "$url")" == "$expected" ]]; then return 0; fi
    sleep 1
  done
  echo "Timed out waiting for HTTP $expected at $url"
  return 1
}

start_nats_forward() {
  kubectl port-forward svc/nats 14222:4222 >/tmp/nats-port-forward.log 2>&1 &
  nats_pid=$!
}

seed_consumer() {
  for attempt in $(seq 1 8); do
    if go run ./test/e2e/seed --server nats://127.0.0.1:14222; then return 0; fi
    sleep 2
  done
  echo "Unable to create test stream/consumer"
  return 1
}

send_heartbeat() {
  kubectl exec deployment/health-bridge -c heartbeat-client -- \
    wget -qO- --post-data='' http://127.0.0.1:8080/heartbeat/absent-worker |
    grep -q '"accepted"'
}

# Look up the bridge by container name, not by array position: the pod
# contains a separate heartbeat helper whose restart count is irrelevant.
bridge_restart_count() {
  kubectl get pods -l app=health-bridge -o jsonpath='{.items[0].status.containerStatuses[?(@.name=="bridge")].restartCount}'
}

echo "1. Check bridge cold-start liveness with NATS deliberately absent"
test "$(kubectl get deployment/nats -o jsonpath='{.spec.replicas}')" = "0"
assert_code 200 http://127.0.0.1:18080/healthz
assert_code 503 http://127.0.0.1:18080/readyz
assert_code 200 http://127.0.0.1:18080/status
assert_code 200 http://127.0.0.1:18080/metrics
curl -fsS http://127.0.0.1:18080/status | grep -q '"nats_connected":false'
cold_start_restarts=$(bridge_restart_count)
sleep 5
if [[ "$(bridge_restart_count)" != "$cold_start_restarts" ]]; then
  echo "Bridge restarted while NATS was unavailable at startup"
  exit 1
fi

echo "2. Start NATS after the bridge, then seed real consumer"
kubectl scale deployment/nats --replicas=1
kubectl rollout status deployment/nats --timeout=120s
start_nats_forward
sleep 2
seed_consumer
assert_code 503 http://127.0.0.1:18080/readyz
curl -fsS http://127.0.0.1:18080/status | grep -q 'processing heartbeat missing or expired'

echo "3. Heartbeat restores healthy readiness"
send_heartbeat
wait_code 200 http://127.0.0.1:18080/readyz
curl -fsS http://127.0.0.1:18080/status | grep -q '"nats_connected":true'
curl -fsS http://127.0.0.1:18080/status | grep -q '"status":"healthy"'

echo "4. Heartbeat expiration fails readiness without restarting bridge"
sleep 17
wait_code 503 http://127.0.0.1:18080/readyz
curl -fsS http://127.0.0.1:18080/status | grep -q 'processing heartbeat missing or expired'
before=$(bridge_restart_count)

echo "5. Stop NATS and validate liveness/readiness separation"
kubectl scale deployment/nats --replicas=0
kubectl rollout status deployment/nats --timeout=60s
sleep 5
assert_code 200 http://127.0.0.1:18080/healthz
assert_code 503 http://127.0.0.1:18080/readyz
after=$(bridge_restart_count)
if [[ "$after" != "$before" ]]; then
  echo "Bridge restarted during NATS outage: before=$before after=$after"
  exit 1
fi

echo "6. Restore NATS; re-seed in-memory stream and recover readiness"
if [[ -n "$nats_pid" ]]; then kill "$nats_pid" 2>/dev/null || true; fi
nats_pid=""
kubectl scale deployment/nats --replicas=1
kubectl rollout status deployment/nats --timeout=120s
start_nats_forward
sleep 2
seed_consumer
send_heartbeat
wait_code 200 http://127.0.0.1:18080/readyz
echo "Kind positive/recovery smoke tests passed"
