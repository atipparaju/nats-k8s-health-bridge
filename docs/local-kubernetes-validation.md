# Local Kubernetes Validation Report

**Scope:** Manual single-node NATS JetStream + Health Bridge validation in a disposable Kubernetes namespace.  
**Outcome:** Functional, resilience, heartbeat and authentication scenarios passed, with one security defect corrected and one deployment limitation identified.  
**Portability:** Windows PowerShell 7+ and Linux Bash. No workstation names, private IPs, actual tokens, or unique pod identifiers are recorded.

## Prerequisites and setup

Use Docker Desktop Kubernetes, Kind, k3d or another local Kubernetes cluster; kubectl, Helm, Docker, Git and the NATS CLI (via nats-box). Use a dedicated namespace, a suitable StorageClass, and disposable messages. Never inject failures into shared or production environments.

Example resource names: namespace `nats-health-lab`, NATS Helm release/service `nats`, stream `LAB_EVENTS`, pull consumer `LAB_WORKER`, subject `lab.events.test`, bridge Deployment `health-bridge`, ConfigMap `health-bridge-config`, Secret `health-bridge-auth`.

### Verify the cluster (both shells)

```sh
kubectl config current-context
kubectl get nodes -o wide
kubectl get storageclass
kubectl get pods -A
helm version
docker version
```

Metrics Server is optional. Prefer a kubectl client within one minor version of the Kubernetes API server.

### Deploy official NATS chart

```sh
helm repo add nats https://nats-io.github.io/k8s/helm/charts/
helm repo update
helm pull nats/nats --untar
```

Example `nats-values.yaml` (replace `hostpath` with the local StorageClass):

```yaml
config:
  cluster:
    enabled: false
  jetstream:
    enabled: true
    fileStore:
      enabled: true
      pvc:
        enabled: true
        size: 1Gi
        storageClassName: hostpath
  monitor:
    enabled: true
container:
  resources:
    requests: {cpu: 100m, memory: 256Mi}
    limits: {cpu: "1", memory: 512Mi}
natsBox:
  enabled: true
```

```sh
helm lint ./nats -f ./nats-values.yaml
kubectl create namespace nats-health-lab --dry-run=client -o yaml | kubectl apply -f -
helm upgrade --install nats ./nats -n nats-health-lab -f ./nats-values.yaml --wait --timeout 5m
kubectl get pods,pvc,svc -n nats-health-lab
```

`helm lint nats/nats` incorrectly treats that chart reference as a local directory; lint the extracted `./nats` chart. Pin the chart version for reproducibility. `nats server info` may require system-account permissions; `nats account info` verifies ordinary JetStream access.

### Locate NATS Box

PowerShell:

```powershell
$ns = "nats-health-lab"
$natsBox = kubectl get pods -n $ns -l app.kubernetes.io/component=nats-box -o jsonpath='{.items[0].metadata.name}'
```

Linux Bash:

```bash
NS=nats-health-lab
NATSBOX=$(kubectl get pods -n "$NS" -l app.kubernetes.io/component=nats-box -o jsonpath='{.items[0].metadata.name}')
```

The selector `app.kubernetes.io/name=nats-box` may return no pods: use the component label or inspect `kubectl get pods --show-labels`.

Run these commands **inside NATS Box** (for either OS, first enter with `kubectl exec -it -n nats-health-lab <nats-box-pod> -- sh`):

```sh
nats --server nats://nats:4222 account info
nats --server nats://nats:4222 stream add LAB_EVENTS --subjects 'lab.events.>' --storage file --retention limits --replicas 1 --defaults
nats --server nats://nats:4222 consumer add LAB_EVENTS LAB_WORKER --pull --ack explicit --deliver all --max-deliver 5 --defaults
nats --server nats://nats:4222 consumer info LAB_EVENTS LAB_WORKER
```

### Bridge setup

Build locally with `docker build -t nats-k8s-health-bridge:local .`. Deploy a ConfigMap and bridge workload (see repository README) using:

```yaml
log_level: debug
poll_interval: 5s
max_observation_age: 20s
server:
  port: 8080
  bind_address: 0.0.0.0
  readiness_fail_on_degraded: false
  shutdown_timeout: 10s
nats:
  url: nats://nats:4222
consumers:
  - name: lab-worker
    stream: LAB_EVENTS
    consumer: LAB_WORKER
    thresholds:
      max_lag: 100
      max_ack_pending: 50
```

Mount it as a config file and pass `--config <mounted-path>`. Set liveness `/healthz`, readiness `/readyz`; a pod-reachable bind address is needed for kubelet HTTP probes. `imagePullPolicy: Never` avoids registry pulls but local-image visibility depends on the Kubernetes distribution. Port-forward a specific pod when multiple revisions coexist.

## Manual test evidence

Counts are observations from one lab run, **not** performance benchmarks or guarantees.

| ID | Test | Procedure | Observed result |
| --- | --- | --- | --- |
| T01 | Baseline | Inspect durable consumer and bridge | 10 pending, 0 ack pending, 0 redelivered; healthy, ready |
| T02 | Publish tracking | Publish one message | Pending 10 → 11 |
| T03 | Backlog threshold | Publish 100; max_lag 100 | 111 pending; degraded with correct reason; readiness 200 |
| T04 | Recovery | Fetch/ack 20 | Pending 91; healthy |
| T05 | Ack-pending | Fetch 60 without ack; threshold 50 | 60 outstanding; degraded; readiness true |
| T06 | Redelivery | Wait beyond AckWait, fetch one without ack | Delivery attempt 2; redelivery gauge 1 |
| T07 | Gauge stability | Wait across multiple polls without another pull | Gauge remained 1, did not accumulate snapshots |
| T08 | NATS outage | Scale test StatefulSet to zero | Disconnected; unknown/stale; readiness false; bridge restarts 0 |
| T09 | NATS recovery | Restore StatefulSet to one | Reconnected; ready; legitimate ack-pending degradation preserved |
| T10 | Persistence | Inspect stream after restart | File-backed stream retained 112 messages, sequences 1–112, active consumer |
| T11 | Missing heartbeat | Configure heartbeat_timeout 30s and restart | New pod running/unready; 503 with missing-heartbeat reason |
| T12 | Heartbeat recovery | POST heartbeat via pod-local debug container | Accepted; immediate readiness 200 |
| T13 | Heartbeat expiry | Stop heartbeats; wait over 30s | Readiness 503 with expired-heartbeat reason |
| T14 | Initial auth | Inject Kubernetes Secret-backed tokens | **Defect:** unauthenticated /status still returned 200 |
| T15 | Auth regression retest | Rebuild after PR #19 fix | Status: absent/invalid 401, valid 200; metrics: absent 401, valid 200; cross-token 401 |
| T16 | Rolling update with missing heartbeats | Restart without continuous heartbeat producer | New revision unready; rollout timed out, old replica retained |
| T17 | Rollout recovery | Disable heartbeat in standalone lab, restart | Rollout succeeded, one ready pod; Secret references retained; unauthenticated /status still 401 |

### Key interpretations

- Consumer degradation (lag/ack pending) is distinct from Kubernetes readiness by default; strict readiness is opt-in.
- An expired AckWait does not prove redelivery; the subsequent pull showed `tries: 2`. `nats_consumer_redelivered` is a snapshot gauge, not a lifetime counter.
- During disconnection, last-known counts are historical and must not be interpreted as current.
- PVC-backed restart persistence is not proof of recovery from PVC loss.
- An accepted heartbeat proves only receipt of a signal, not genuine worker progress. A worker-integrated watchdog and idle-worker policy are still needed.
- Enabling mandatory heartbeat before the worker starts reporting can stall rolling updates. Consider coordinated activation or startup grace periods.

## Security incident and remediation

**Finding:** Kubernetes Secret references existed for `NKHB_SERVER_STATUS_TOKEN` and `NKHB_SERVER_METRICS_TOKEN`, but an unauthenticated diagnostic request returned 200.

**Cause:** Viper `AutomaticEnv()` supports lookup but may not enumerate environment-only keys for `Unmarshal()`. Absent YAML keys left tokens empty, silently disabling endpoint protection.

**Fix:** Explicitly bind both token environment keys before unmarshaling; add regression tests for environment-only values and environment-over-YAML precedence. See [PR #19](https://github.com/atipparaju/nats-k8s-health-bridge/pull/19).

**Retest:** Missing/invalid status tokens → 401, valid status token → 200; missing metrics token → 401, valid metrics token → 200; status token used for metrics → 401. After rollout recovery, unauthenticated status still returned 401.

**Security guidance:** Versions before the fix may be insecure when protection relies only on Secret-backed environment values. Restrict diagnostic network access until patched and verified. Never include real credentials in commits, command history, screenshots or test reports. Loopback-only heartbeat is not a security boundary between containers in the same pod.

## Cross-platform command recipes

### Publish 100 messages

PowerShell:

```powershell
$ns = "nats-health-lab"
$natsBox = kubectl get pods -n $ns -l app.kubernetes.io/component=nats-box -o jsonpath='{.items[0].metadata.name}'
1..100 | ForEach-Object {
  kubectl exec -n $ns $natsBox -- nats --server nats://nats:4222 pub lab.events.test "backlog-test-$_" | Out-Null
}
```

Linux Bash:

```bash
NS=nats-health-lab
NATSBOX=$(kubectl get pods -n "$NS" -l app.kubernetes.io/component=nats-box -o jsonpath='{.items[0].metadata.name}')
for i in $(seq 1 100); do
  kubectl exec -n "$NS" "$NATSBOX" -- nats --server nats://nats:4222 pub lab.events.test "backlog-test-$i" >/dev/null
done
```

### Consumer delivery and redelivery

Inside NATS Box:

```sh
nats --server nats://nats:4222 consumer next LAB_EVENTS LAB_WORKER --count 20 --ack
nats --server nats://nats:4222 consumer next LAB_EVENTS LAB_WORKER --count 60 --no-ack
nats --server nats://nats:4222 consumer info LAB_EVENTS LAB_WORKER
```

Use disposable messages only. `AckWait` in the observed run was 30 seconds. Consult `nats consumer next --help` for version-specific flags.

### Status and HTTP codes

PowerShell 7+:

```powershell
(Invoke-WebRequest http://localhost:8080/status -SkipHttpErrorCheck).StatusCode
Invoke-RestMethod http://localhost:8080/status | ConvertTo-Json -Depth 10
```

Linux Bash:

```bash
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/status
curl -fsS http://localhost:8080/status
```

### NATS outage and recovery (lab only)

```sh
kubectl scale statefulset nats --replicas=0 -n nats-health-lab
kubectl get pods -n nats-health-lab -l app=health-bridge
kubectl scale statefulset nats --replicas=1 -n nats-health-lab
kubectl rollout status statefulset/nats -n nats-health-lab --timeout=120s
```

### Secret-based authentication

Create a Kubernetes Secret with locally generated **temporary** credentials; do not use literal example tokens in shared scripts. Inject through `secretKeyRef`:

```yaml
env:
  - name: NKHB_SERVER_STATUS_TOKEN
    valueFrom:
      secretKeyRef: {name: health-bridge-auth, key: status-token}
  - name: NKHB_SERVER_METRICS_TOKEN
    valueFrom:
      secretKeyRef: {name: health-bridge-auth, key: metrics-token}
```

Use `Authorization: Bearer <token>` for each diagnostic endpoint; test absent, invalid, valid, and cross-endpoint tokens. Health probes intentionally remain unauthenticated.

### Pod-local heartbeat

A distroless bridge image may have no shell. A disposable ephemeral container can call loopback:

```sh
kubectl debug -n nats-health-lab -it pod/<bridge-pod> --image=busybox:1.36 --container=heartbeat-check -- sh -c "wget -qO- --post-data='' http://127.0.0.1:8080/heartbeat/lab-worker; echo; wget -qO- http://127.0.0.1:8080/readyz"
```

Use a new debug-container name for each attempt; exited ephemeral containers cannot be restarted with `kubectl exec`. A port-forward request may not appear as loopback to the bridge. A 30-second timeout requires an immediate readiness check or a sustained worker heartbeat.

### Windows PowerShell YAML application

Do not use Linux `/dev/stdin` with native Windows PowerShell. For a multi-document lab YAML file with ConfigMap first, isolate the ConfigMap:

```powershell
$yaml = Get-Content .\health-bridge-lab.yaml -Raw
$configMapYaml = ($yaml -split '(?m)^---\s*$')[0]
$configMapYaml | kubectl apply -n nats-health-lab -f -
```

The pipeline through `ConvertFrom-Json` / `ConvertTo-Json` is not a reliable way to select one YAML document from a multi-document manifest. Verify local source files before applying and restart the bridge to reload config when hot reload is unavailable.

## Outstanding validation and release gaps

- A real worker in the same pod, with continuous heartbeat, idle semantics, deadlock detection and graceful shutdown.
- Successful rolling update with heartbeat enabled and a coordinated worker; startup grace-period policy.
- Multi-node JetStream quorum, network partition, failover, volume loss, and restore testing.
- TLS, NATS credentials/NKeys, Kubernetes NetworkPolicy, authenticated Prometheus scraping, token rotation and secret lifecycle.
- Healthy-consumer recovery in automated Kind tests (current smoke test is deliberately a negative-case fixture).
- ARM64 runtime smoke test, SBOM/provenance, vulnerability review and release compatibility matrix.
- Reproducible automated test evidence with captured timestamps, versions, CI links and logs, without sensitive infrastructure metadata.

## Cleanup

After exporting required evidence, delete only the disposable namespace:

```sh
kubectl delete namespace nats-health-lab
```

**Warning:** This deletes the lab's PVC and JetStream data when the StorageClass reclaim policy is Delete. Do not execute until all test data and evidence are no longer needed.

## Conclusion

The observed manual tests support functional correctness for the scenarios above, including a discovered and fixed authentication defect. They do **not** establish production certification, comprehensive security assurance, or field-level adoption. The next milestone is to convert the remaining scenarios into repeatable integration tests and complete release-readiness gates.
