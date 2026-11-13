#!/usr/bin/env bash
# Local-only Linux Docker smoke test. Builds and runs images; never authenticates to or pushes to a registry.
set -Eeuo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/../.."
for cmd in docker go curl openssl python3; do
  command -v "$cmd" >/dev/null || { echo "Missing prerequisite: $cmd" >&2; exit 1; }
done

# The multi-platform Dockerfile uses BuildKit automatic platform arguments.
# Without Buildx, Docker may fall back to the legacy builder and fail with
# 'failed to parse platform : ""' after NATS and the seed have already started.
if ! docker buildx version >/dev/null 2>&1; then
  echo "Missing Docker Buildx: the bridge Dockerfile requires BuildKit/Buildx." >&2
  echo "Check: docker buildx version" >&2
  echo "On Ubuntu (Ubuntu archive): sudo apt install docker-buildx" >&2
  echo "On Docker's official apt repository: install docker-buildx-plugin instead." >&2
  echo "See docs/docker-quickstart.md#troubleshooting-linux-builds for alternatives." >&2
  exit 1
fi

id="${GITHUB_RUN_ID:-$$}"
network="nkhb-47-${id}"
nats_container="nkhb-47-nats-${id}"
bridge_container="nkhb-47-bridge-${id}"
seed_bin="$(mktemp)"
cleanup() {
  docker rm -f "$bridge_container" "$nats_container" >/dev/null 2>&1 || true
  docker network rm "$network" >/dev/null 2>&1 || true
  rm -f "$seed_bin"
  unset NKHB_SERVER_STATUS_TOKEN NKHB_SERVER_METRICS_TOKEN
}
trap cleanup EXIT

# An alias preserves the nats-demo hostname used by the checked-in demo config.
docker network create "$network" >/dev/null
docker run -d --name "$nats_container" --network "$network" \
  --network-alias nats-demo -p 127.0.0.1:14222:4222 \
  nats:2.15.1-alpine3.22 -js >/dev/null

go build -o "$seed_bin" ./test/release/seed
seeded=false
for attempt in $(seq 1 20); do
  if "$seed_bin" --server nats://127.0.0.1:14222 >/dev/null 2>&1; then
    seeded=true
    break
  fi
  sleep 1
done
if [[ "$seeded" != "true" ]]; then
  echo "FAIL: unable to seed NATS; check Docker NATS logs" >&2
  exit 1
fi

# All build tags are local, and no Docker login or push is performed.
docker build -t nats-k8s-health-bridge:local . >/dev/null
export NKHB_SERVER_STATUS_TOKEN="$(openssl rand -hex 16)"
export NKHB_SERVER_METRICS_TOKEN="$(openssl rand -hex 16)"
config_path="$PWD/examples/docker/bridge-config.yaml"
docker run -d --name "$bridge_container" --network "$network" \
  -p 127.0.0.1:18080:8080 \
  --env NKHB_SERVER_STATUS_TOKEN --env NKHB_SERVER_METRICS_TOKEN \
  --mount "type=bind,source=$config_path,target=/etc/bridge/config.yaml,readonly" \
  nats-k8s-health-bridge:local --config /etc/bridge/config.yaml >/dev/null

base="http://127.0.0.1:18080"
ready=false
for attempt in $(seq 1 40); do
  if curl --silent --fail "$base/readyz" >/dev/null; then
    ready=true
    break
  fi
  sleep 1
done
if [[ "$ready" != "true" ]]; then
  echo "FAIL: bridge did not become ready; check local bridge container logs" >&2
  exit 1
fi

test "$(curl -s -o /dev/null -w '%{http_code}' "$base/healthz")" = 200
test "$(curl -s -o /dev/null -w '%{http_code}' "$base/readyz")" = 200

# Tokens are not printed or committed. Diagnostics fail closed without valid bearer headers.
for endpoint in status metrics; do
  test "$(curl -s -o /dev/null -w '%{http_code}' "$base/$endpoint")" = 401
  test "$(curl -s -o /dev/null -w '%{http_code}' -H 'Authorization: Bearer invalid-token' "$base/$endpoint")" = 401
done

status_file="$(mktemp)"
metrics_file="$(mktemp)"
trap 'rm -f "$status_file" "$metrics_file"; cleanup' EXIT
curl --silent --show-error --fail -H "Authorization: Bearer $NKHB_SERVER_STATUS_TOKEN" "$base/status" > "$status_file"
curl --silent --show-error --fail -H "Authorization: Bearer $NKHB_SERVER_METRICS_TOKEN" "$base/metrics" > "$metrics_file"

python3 - "$status_file" "$metrics_file" <<'PY'
import json
import pathlib
import sys
payload = json.loads(pathlib.Path(sys.argv[1]).read_text())
assert payload["nats_connected"] is True, "NATS not connected"
assert payload["ready"] is True, "bridge not ready"
assert any(item["name"] == "lab-worker" and item["status"] == "healthy"
           for item in payload["consumers"]), "expected healthy LAB_WORKER"
metrics = pathlib.Path(sys.argv[2]).read_text()
assert "nats_consumer_lag" in metrics, "missing lag metric"
assert "nats_connected" in metrics, "missing connection metric"
PY

echo "PASS: Linux Docker quickstart, real JetStream consumer, readiness, protected diagnostics, and cleanup."
