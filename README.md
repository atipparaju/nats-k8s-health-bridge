# nats-k8s-health-bridge

> Kubernetes-native liveness/readiness probe bridge for NATS JetStream consumers.

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![CNCF Ecosystem](https://img.shields.io/badge/CNCF-NATS%20Incubating-blue)](https://www.cncf.io/projects/nats/)

## What it does

NATS JetStream consumers running inside Kubernetes pods have no native way to signal their health to the Kubernetes control plane. A consumer can be lagging, stuck in a redelivery loop, or disconnected — and the pod still shows `Ready`.

`nats-k8s-health-bridge` runs as a **sidecar** alongside your consumer pod and exposes:

| Endpoint | Use | Returns |
|---|---|---|
| `GET /healthz` | Kubernetes **liveness** probe | `200` while the bridge HTTP process is responsive (independent of NATS connectivity) |
| `GET /readyz` | Kubernetes **readiness** probe | `200` if NATS is connected and consumer status is available; threshold breaches only fail readiness when explicitly enabled |
| `GET /metrics` | Prometheus scrape | Consumer lag, ack pending, redelivered, connection state |
| `GET /status` | Debug JSON | Full state of all monitored consumers |
| `POST /heartbeat/{consumer}` | Optional application heartbeat | `200` for configured consumer from loopback; `403` for non-loopback |

## Quickstart

### Run locally with Docker

```bash
# Start a local NATS server with JetStream enabled
docker run -d -p 4222:4222 -p 8222:8222 nats:latest -js

# Run the sidecar
docker run --rm \
  -e NKHB_NATS_URL=nats://host.docker.internal:4222 \
  -e NKHB_LOG_LEVEL=debug \
  -p 8080:8080 \
  ghcr.io/atipparaju/nats-k8s-health-bridge:latest \
  --config /etc/nats-k8s-health-bridge/config.yaml

# Check health
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
curl http://localhost:8080/status
```

### Kubernetes sidecar injection

Add this to your existing pod template:

```yaml
containers:
  - name: your-consumer
    image: your-image:tag
    # ... your existing config

  - name: nats-health-bridge
    image: ghcr.io/atipparaju/nats-k8s-health-bridge:latest
    args: ["--config", "/etc/nats-health-bridge/config.yaml"]
    ports:
      - containerPort: 8080
        name: health
    volumeMounts:
      - name: health-bridge-config
        mountPath: /etc/nats-health-bridge
    livenessProbe:
      httpGet:
        path: /healthz
        port: health
      initialDelaySeconds: 5
      periodSeconds: 10
    readinessProbe:
      httpGet:
        path: /readyz
        port: health
      initialDelaySeconds: 10
      periodSeconds: 10
    resources:
      requests:
        cpu: 10m
        memory: 32Mi
      limits:
        cpu: 100m
        memory: 64Mi

volumes:
  - name: health-bridge-config
    configMap:
      name: my-health-bridge-config
```

## Configuration

Configuration is loaded from a YAML file with **environment variable overrides** (12-factor style).

**Environment variable prefix:** `NKHB_`  
**Separator:** `_` (dots in YAML keys become underscores)

### Environment Variables Reference

| Variable | YAML equivalent | Default | Description |
|---|---|---|---|
| `NKHB_LOG_LEVEL` | `log_level` | `info` | `debug`, `info`, `warn`, `error` |
| `NKHB_POLL_INTERVAL` | `poll_interval` | `10s` | How often to poll JetStream |
| `NKHB_MAX_OBSERVATION_AGE` | `max_observation_age` | `30s` | Maximum age of last successful consumer observation; must exceed poll interval |
| `NKHB_SERVER_PORT` | `server.port` | `8080` | HTTP server port |
| `NKHB_SERVER_BIND_ADDRESS` | `server.bind_address` | `0.0.0.0` | IP address on which the HTTP server listens |
| `NKHB_SERVER_METRICS_TOKEN` | `server.metrics_token` | unset | Optional bearer token for `/metrics` |
| `NKHB_SERVER_STATUS_TOKEN` | `server.status_token` | unset | Optional bearer token for `/status` |
| `NKHB_SERVER_READINESS_FAIL_ON_DEGRADED` | `server.readiness_fail_on_degraded` | `false` | Opt into threshold-based readiness failures |
| `NKHB_SERVER_SHUTDOWN_TIMEOUT` | `server.shutdown_timeout` | `10s` | Graceful shutdown wait |
| `NKHB_NATS_URL` | `nats.url` | `nats://localhost:4222` | NATS server URL |
| `NKHB_NATS_CREDENTIALS_FILE` | `nats.credentials_file` | — | Path to `.creds` file |
| `NKHB_NATS_NKEY_FILE` | `nats.nkey_file` | — | Path to NKey seed file |
| `NKHB_NATS_TLS_CERT` | `nats.tls_cert` | — | Path to TLS client cert |
| `NKHB_NATS_TLS_KEY` | `nats.tls_key` | — | Path to TLS client key |
| `NKHB_NATS_TLS_CA` | `nats.tls_ca` | — | Path to CA cert |
| `NKHB_NATS_CONNECT_TIMEOUT` | `nats.connect_timeout` | `5s` | Initial connection timeout |
| `NKHB_NATS_MAX_RECONNECTS` | `nats.max_reconnects` | `-1` | `-1` = infinite |

### Full config.yaml example

See [`config.yaml`](config.yaml) for a fully annotated example.

### Observation freshness

The bridge tracks both `last_checked` (most recent poll attempt) and `last_successful` (most recent successful JetStream consumer-info response). If a consumer has not been observed successfully within `max_observation_age` (default `30s`, with a `10s` poll interval), `/readyz` returns `503` even if the last recorded state was healthy. `/status` exposes an effective `unknown` state and a stale reason, without modifying the stored last observation. Set the timeout above the poll interval with sufficient margin for retries and network latency. A zero value disables freshness enforcement only for programmatically constructed configurations; file-based configurations default to `30s`.

### Application processing heartbeat (opt-in)

JetStream consumer metadata cannot prove that an application is actively processing work. To enable cooperative heartbeat monitoring, configure `heartbeat_timeout: 30s` on an individual consumer and have the application container call `POST http://127.0.0.1:8080/heartbeat/order-processor` periodically (for example, every 10 seconds). The route only accepts loopback callers, and rejects unknown consumer names. Kubernetes containers in the same pod share a network namespace, so a local call can reach the sidecar.

An absent or expired heartbeat fails `/readyz` and is reported as `unknown` in `/status`; a fresh heartbeat restores the normal JetStream-based readiness evaluation. Disabled by default (zero timeout). **Security:** loopback is not an authentication boundary between containers sharing the pod network namespace. Only use with trusted pod-local workloads. Do not forward this endpoint through an ingress or service proxy. A heartbeat proves only that the application is reporting progress; for stuck-work detection, emit it from a processing-progress watchdog rather than an unrelated timer.

### Readiness versus degradation

By default, `/readyz` requires a NATS connection and successfully retrieved consumer state. A consumer that exceeds `max_lag` or `max_ack_pending` is reported as **degraded** in `/status` and metrics, but the pod stays ready. This prevents a healthy worker with a large backlog from being removed from service solely because of load.

Set `server.readiness_fail_on_degraded: true` only if your deployment explicitly requires threshold breaches to fail Kubernetes readiness. This is an opt-in compatibility mode; it may reduce capacity during incidents. NATS disconnections and unknown consumer states continue to fail readiness. Readiness is not a substitute for an application-level processing heartbeat.

### Redelivery-rate measurement

JetStream consumer-info `num_redelivered` is a point-in-time count, not a monotonic lifetime counter. The former `max_redelivery_rate` threshold was calculated from changes in `num_redelivered` and `num_ack_pending`; these snapshots cannot provide a reliable delivery denominator or event rate. **That threshold has been removed** rather than producing misleading health decisions. `eval_window` is also removed because it was only used by this calculation.

For rate-based alerting, instrument message delivery attempts and acknowledgments in the consumer application or another reliable event source. Derive redelivery ratios from observed events over an explicit window. `/status` still includes the current `num_redelivered` snapshot; the metric fix is handled separately in the redelivery gauge PR. Remove `max_redelivery_rate` and `eval_window` from existing configurations before upgrading.

### Securing HTTP endpoints

**Security fix:** `NKHB_SERVER_METRICS_TOKEN` and `NKHB_SERVER_STATUS_TOKEN` are explicitly registered with Viper so Kubernetes Secret-injected, environment-only token values are included during configuration unmarshaling. Earlier builds could silently leave `/metrics` and `/status` unauthenticated despite those environment variables being present. Rebuild/redeploy the bridge after this fix and verify that requests without or with invalid bearer tokens return HTTP 401. Until verified, restrict diagnostic endpoints at the network level.


The bridge now supports `server.bind_address` and optional bearer-token authentication for `/metrics` and `/status`. Configure `NKHB_SERVER_METRICS_TOKEN` and `NKHB_SERVER_STATUS_TOKEN` from Kubernetes Secrets; do not commit credentials to ConfigMaps or repository files. Supply `Authorization: Bearer <token>` from your monitoring client. `/healthz` and `/readyz` remain unauthenticated for Kubernetes probes. The heartbeat endpoint remains restricted to loopback callers; pod-local containers are trusted by this mechanism, not cryptographically authenticated.

**Deployment:** `0.0.0.0` remains the default for compatibility and Kubernetes HTTP probes, which normally contact the pod IP. Binding to `127.0.0.1` can prevent kubelet HTTP probes from reaching the container. Use Kubernetes NetworkPolicies, avoid public Services/Ingress, and restrict access to the monitoring namespace. NetworkPolicies do not reliably isolate containers within the same pod. Tokens are optional and not a substitute for TLS or network isolation; use TLS termination through a trusted proxy if traffic leaves the pod. Prometheus scraping needs a matching authorization configuration when metrics authentication is enabled.

## Prometheus Metrics

| Metric | Type | Labels | Description |
|---|---|---|---|
| `nats_consumer_up` | Gauge | `consumer_name`, `stream`, `consumer` | `1` = healthy, `0` = degraded |
| `nats_consumer_lag` | Gauge | `consumer_name`, `stream`, `consumer` | `num_pending` messages |
| `nats_consumer_ack_pending` | Gauge | `consumer_name`, `stream`, `consumer` | `num_ack_pending` |
| `nats_consumer_redelivered` | Gauge | `consumer_name`, `stream`, `consumer` | Current JetStream redelivery snapshot (not a cumulative total) |
| `nats_connected` | Gauge | — | `1` if connected to NATS |

### Helm configuration parity

The Helm chart now exposes `pollInterval`, `maxObservationAge`, `server.bindAddress`, `server.readinessFailOnDegraded`, and per-consumer `heartbeatTimeout`. Optional diagnostic bearer tokens must come from Kubernetes Secrets. The chart generates a ConfigMap and a commented sidecar example, **not a ready-to-run Deployment**; uncomment and adapt the environment-variable Secret references for your workload. Do not put bearer tokens in Helm values or ConfigMaps. Validate chart rendering with `helm lint helm/nats-k8s-health-bridge` and `helm template health-bridge helm/nats-k8s-health-bridge` before deployment.

### Kubernetes CI smoke test

The chart supports consumers with omitted `thresholds`; Helm renders zero values (disabled thresholds) rather than failing with a nil-pointer template error. This is exercised by the smoke workflow's minimal consumer values.

The separate `.github/workflows/kind-smoke.yml` workflow creates a Kind cluster, lints and renders the Helm chart, builds the bridge image, deploys NATS and the bridge, and tests HTTP probe behavior. The fixture deliberately references a **missing** JetStream consumer: `/readyz` must return 503 while `/healthz` remains 200, including after NATS is scaled down. The test also checks that the bridge container does not restart during the outage. To keep the negative-case fixture running, its Kubernetes readiness probe targets `/healthz`; **production manifests should target `/readyz`**. This is a bounded smoke test, not yet a complete consumer-creation, recovery, heartbeat-expiry, or fault-injection suite.

Run locally with Kind, Docker, Helm, kubectl and curl: build `nats-k8s-health-bridge:e2e`, load it into a Kind cluster named `health-bridge-ci`, apply `test/e2e/kind-smoke.yaml`, then run `bash test/e2e/kind-smoke.sh`.

### Container vulnerability scanning

The `.github/workflows/container-security.yml` workflow builds the image from the PR or main branch with `docker build --pull`, scans OS and Go library vulnerabilities with Trivy, reports **HIGH and CRITICAL** findings, and blocks merges for **fixable CRITICAL** findings. Unfixed vulnerabilities are not blocking in this initial gate; they still require risk assessment. The job is separate from the image-publishing workflow and does not publish images or require registry credentials.

**Production security checklist:** Use a supported, patched Go toolchain and base image; review HIGH findings and exceptions; restrict container capabilities and use `runAsNonRoot`, `readOnlyRootFilesystem`, `allowPrivilegeEscalation: false`, and `capabilities.drop: [ALL]` where supported; apply NetworkPolicies; keep credentials in Secrets; and use immutable image digests in production. A clean scan does not prove that the image or deployment is secure.

This workflow is a vulnerability gate, **not** SBOM publication, signed provenance, SLSA attestation, or a comprehensive dependency-review policy. Those remain follow-up work.
### Multi-platform container builds

The Dockerfile uses BuildKit `TARGETOS` and `TARGETARCH` to cross-compile for the requested image platform rather than always producing an amd64 executable. Build and verify both architectures before release:

```bash
docker buildx build --platform linux/amd64,linux/arm64 --tag example/nats-k8s-health-bridge:test .
```

The build command validates compilation for each target but does not by itself prove successful execution on both architectures. Validate the published image manifest and run an arm64 smoke test before declaring multi-platform support production-ready.

## Health evaluation architecture

The consumer threshold evaluator is isolated behind a `ConsumerEvaluator` interface; the current `LegacyEvaluator` remains the default, preserving existing behavior. See [jsm.go evaluator compatibility study](docs/jsm-evaluator-compatibility.md) for the upstream NATS monitor parity matrix and proposed adapter design. The upstream library is **not yet integrated**.

## Building

```bash
# Build binary
make build

# Run tests
make test

# Build Docker image
make docker-build

# Run locally (requires NATS running on localhost:4222)
make run-local
```

## License

Apache 2.0 — see [LICENSE](LICENSE).

---

*Part of the [CNCF](https://cncf.io) ecosystem — built to complement [NATS](https://nats.io) (CNCF Incubating) and [Kubernetes](https://kubernetes.io) (CNCF Graduated).*
