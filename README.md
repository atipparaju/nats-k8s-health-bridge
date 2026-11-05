# nats-k8s-health-bridge

> Kubernetes-native liveness/readiness probe bridge for NATS JetStream consumers.

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![NATS project maturity](https://img.shields.io/badge/NATS-CNCF%20Graduated-blue)](https://www.cncf.io/projects/nats/)

NATS graduated in the CNCF on **November 2, 2026**. This maturity designation refers to NATS itself, not to this independently maintained bridge. The bridge remains an **unreleased private development project**. Its historical 0.1.0 entry does not imply a public release or package. See [release history and compatibility](docs/release-versioning.md) and the [release-readiness checklist](docs/release-readiness-checklist.md).

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

See the [portable Docker first-run guide](docs/docker-quickstart.md) for step-by-step commands for **Windows PowerShell and Linux Bash**. It creates a disposable NATS stream and consumer, builds the bridge locally rather than pulling a potentially unavailable private GHCR package, uses a dedicated Docker network, and injects temporary diagnostic tokens safely.

The corresponding configuration is [examples/docker/bridge-config.yaml](examples/docker/bridge-config.yaml). Do not rely on an unversioned GHCR `:latest` image or the multi-consumer generic configuration for first-run testing.

### Kubernetes sidecar injection

The Helm chart currently renders a **ConfigMap and a non-deploying sidecar
example only**: it does not install a workload or create Secrets. The example
defaults to a **locally built and preloaded** image
`nats-k8s-health-bridge:local` with `imagePullPolicy: Never`; do not assume
an existing or public GHCR `:latest` image. To inspect the sample:

```sh
make docker-build
helm template bridge helm/nats-k8s-health-bridge
python3 scripts/test_helm_templates.py
```

Copy the generated `sidecar-example.yaml` container and volumes into your
own Pod template after review. See [Helm sidecar and Secret integration](docs/helm-sidecar.md)
for credentials/NKey/TLS mounts, diagnostic tokens, probes, secure defaults,
resource sizing and limitations. This example does **not** publish an image.


## Configuration

Configuration is loaded from a YAML file with **environment variable overrides** (12-factor style).

**Environment variable prefix:** `NKHB_`  
**Separator:** `_` (dots in YAML keys become underscores)

### Environment Variables Reference

| Variable | YAML equivalent | Default | Description |
|---|---|---|---|
| `NKHB_LOG_LEVEL` | `log_level` | `info` | `debug`, `info`, `warn`, `error` |
| `NKHB_POLL_INTERVAL` | `poll_interval` | `10s` | How often to poll JetStream |
| `NKHB_POLL_TIMEOUT` | `poll_timeout` | `8s` | Max duration of one polling cycle; checks run with up to four workers |
| `NKHB_MAX_OBSERVATION_AGE` | `max_observation_age` | `30s` | Maximum age of last successful consumer observation; must exceed poll interval |
| `NKHB_SERVER_PORT` | `server.port` | `8080` | HTTP server port |
| `NKHB_SERVER_BIND_ADDRESS` | `server.bind_address` | `0.0.0.0` | IP address on which the HTTP server listens |
| `NKHB_SERVER_METRICS_TOKEN` | `server.metrics_token` | unset | Bearer token for `/metrics` |
| `NKHB_SERVER_STATUS_TOKEN` | `server.status_token` | unset | Bearer token for `/status` |
| `NKHB_SERVER_ALLOW_UNAUTHENTICATED_DIAGNOSTICS` | `server.allow_unauthenticated_diagnostics` | `false` | Explicit insecure opt-in for isolated tests |
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

### Credential-safe NATS connection diagnostics

The bridge logs **sanitized NATS host:port endpoints** rather than raw NATS URLs on initial connection and reconnect. Credentials, URL-encoded userinfo, token-style usernames, URL paths and query parameters are omitted. Connection errors report a safe category such as server unavailable, authorization failure, or timeout. They do not include raw NATS transport error text because it may contain connection credentials. Wrapped errors retain typed `errors.Is` behavior for code handling. Consumer metadata errors surfaced through `/status` are likewise protected.

Use an authenticated Secret mount and restrict diagnostic endpoints; sanitization is defense in depth, not a reason to put real passwords in URLs or enable verbose logs on untrusted networks.

### NATS credential and TLS configuration

NATS credentials and TLS settings may be provided in YAML or through the `NKHB_NATS_*` environment variables listed above. **Environment-only values are supported**, including Kubernetes Secret-mounted file paths; when both YAML and an environment value are present, the environment value wins.

- Use either `nats.credentials_file` (a NATS `.creds` file) **or** `nats.nkey_file` (an NKey seed file), never both.
- Client-certificate authentication requires **both** `nats.tls_cert` and `nats.tls_key`. A standalone `nats.tls_ca` is valid for server trust without a client certificate.
- File-path values identify mounted credentials inside the bridge container; mounting a Kubernetes Secret is still required. Do not pass credential contents in environment variables or commit private keys.
- Invalid combinations fail config validation before connecting, rather than silently falling back to an unauthenticated connection. Missing or unreadable credential files must cause connection setup to fail; review the error securely and check the mount.

The application and bridge use separate NATS connections, even if they share a mounted Secret. For sensitive NATS URL logging and redaction hardening, see tracked Issue #41.

### Full config.yaml example

See [`config.yaml`](config.yaml) for a fully annotated example.

### NATS cold-start resilience

The bridge does **not** need NATS to be available when its pod starts. The Go NATS client uses `RetryOnFailedConnect(true)`; when the initial server pool is unavailable, the connection enters the background reconnect loop instead of making the sidecar exit. During that interval `/healthz` returns 200 for a responsive bridge process, `/readyz` returns 503, and consumer observations remain unknown. Once the connection becomes available, normal polling and readiness recover automatically without a pod restart.

The initial retry uses the existing `nats.reconnect_wait` and `nats.max_reconnects` configuration. Keep `max_reconnects: -1` (the default) for unlimited retries throughout prolonged outages; a finite retry budget can be exhausted, leaving the bridge unready until restarted. Genuine configuration errors, invalid credentials or unreadable TLS files are **not** suppressed; only transient server unavailability is handled through reconnection. Shutdown closes even a never-connected client to stop background retries.

The Kind smoke test intentionally starts the bridge while NATS has zero replicas, then starts NATS and verifies readiness recovery and stable restart counts. This is a single-node functional test, not a full multi-node availability benchmark.

### Consumer-info retries

The bridge retries transient JetStream consumer-info errors (up to two retries with backoff), but returns immediately on typed missing-stream, missing-consumer, authorization and cancellation errors. The next periodic poll still rechecks missing resources, so a subsequently created consumer can recover without restarting the bridge. Per-request timeouts remain retryable when the parent cycle context is still valid. This avoids unnecessary delays for permanent errors.


### Bounded consumer polling

Consumer checks run through a fixed pool of at most four workers per cycle. A shared `poll_timeout` (default `8s`) cancels a slow batch without waiting for every configured consumer's individual retry sequence. A cancelled/not-yet-observed consumer remains stale or unknown, rather than being reported healthy. Set the timeout to fit request latency, retry behavior, and `max_observation_age`. The loop waits for workers to stop before starting the next cycle; it never overlaps cycles. This does not change readiness policy or how consumer lag thresholds are evaluated.

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

For rate-based alerting, instrument message delivery attempts and acknowledgments in the consumer application or another reliable event source. Derive redelivery ratios from observed events over an explicit window. `/status` still includes the current `num_redelivered` snapshot; the current `nats_consumer_redelivered` Prometheus metric is a gauge, not a cumulative counter. Remove `max_redelivery_rate` and `eval_window` from existing configurations before upgrading.

### Securing HTTP endpoints

**Secure-by-default change:** `/metrics` and `/status` now reject unauthenticated requests with HTTP 401 when no bearer token is configured. Configure diagnostic tokens using Kubernetes Secrets or the environment variables below before deployment. For strictly isolated test environments only, you may explicitly set `server.allow_unauthenticated_diagnostics: true`; do not use this option on publicly reachable or shared networks. `/healthz` and `/readyz` remain unauthenticated for kubelet compatibility. Existing unauthenticated Prometheus scrape targets must configure bearer authentication or explicitly opt into insecure local-only access.

**Security fix:** `NKHB_SERVER_METRICS_TOKEN` and `NKHB_SERVER_STATUS_TOKEN` are explicitly registered with Viper so Kubernetes Secret-injected, environment-only token values are included during configuration unmarshaling. Earlier builds could silently leave `/metrics` and `/status` unauthenticated despite those environment variables being present. Rebuild/redeploy the bridge after this fix and verify that requests without or with invalid bearer tokens return HTTP 401. Until verified, restrict diagnostic endpoints at the network level.


The bridge now supports `server.bind_address` and bearer-token authentication for `/metrics` and `/status`. Configure `NKHB_SERVER_METRICS_TOKEN` and `NKHB_SERVER_STATUS_TOKEN` from Kubernetes Secrets; do not commit credentials to ConfigMaps or repository files. Supply `Authorization: Bearer <token>` from your monitoring client. `/healthz` and `/readyz` remain unauthenticated for Kubernetes probes. The heartbeat endpoint remains restricted to loopback callers; pod-local containers are trusted by this mechanism, not cryptographically authenticated.

**Deployment:** `0.0.0.0` remains the default for compatibility and Kubernetes HTTP probes, which normally contact the pod IP. Binding to `127.0.0.1` can prevent kubelet HTTP probes from reaching the container. Use Kubernetes NetworkPolicies, avoid public Services/Ingress, and restrict access to the monitoring namespace. NetworkPolicies do not reliably isolate containers within the same pod. Tokens and explicit access controls are not a substitute for TLS or network isolation; use TLS termination through a trusted proxy if traffic leaves the pod. Prometheus scraping needs a matching authorization configuration when metrics authentication is enabled.

## Prometheus Metrics

| Metric | Type | Labels | Description |
|---|---|---|---|
| `nats_consumer_up` | Gauge | `consumer_name`, `stream`, `consumer` | `1` = healthy, `0` = degraded |
| `nats_consumer_lag` | Gauge | `consumer_name`, `stream`, `consumer` | `num_pending` messages |
| `nats_consumer_ack_pending` | Gauge | `consumer_name`, `stream`, `consumer` | `num_ack_pending` |
| `nats_consumer_redelivered` | Gauge | `consumer_name`, `stream`, `consumer` | Current JetStream redelivery snapshot (not a cumulative total) |
| `nats_connected` | Gauge | — | `1` if connected to NATS |

### Helm configuration parity

The Helm chart now exposes `pollInterval`, `pollTimeout`, `maxObservationAge`, `server.bindAddress`, `server.readinessFailOnDegraded`, and per-consumer `heartbeatTimeout`. Optional diagnostic bearer tokens must come from Kubernetes Secrets. The chart generates a ConfigMap and a commented sidecar example, **not a ready-to-run Deployment**; uncomment and adapt the environment-variable Secret references for your workload. Do not put bearer tokens in Helm values or ConfigMaps. Validate chart rendering with `helm lint helm/nats-k8s-health-bridge` and `helm template health-bridge helm/nats-k8s-health-bridge` before deployment.

### Kubernetes CI smoke test

The chart supports consumers with omitted `thresholds`; Helm renders zero values (disabled thresholds) rather than failing with a nil-pointer template error. This is exercised by the smoke workflow's minimal consumer values.

The separate `.github/workflows/kind-smoke.yml` workflow creates a Kind cluster, lints and renders the Helm chart, builds the bridge image, deploys NATS and the bridge, and tests HTTP probe behavior. The fixture deliberately references a **missing** JetStream consumer: `/readyz` must return 503 while `/healthz` remains 200, including after NATS is scaled down. The test also checks that the bridge container does not restart during the outage. To keep the negative-case fixture running, its Kubernetes readiness probe targets `/healthz`; **production manifests should target `/readyz`**. The expanded smoke test additionally creates a real disposable stream/consumer, checks positive readiness after a pod-local heartbeat, tests heartbeat expiration, and re-seeds after NATS restarts. It still does not simulate real application processing or multi-node failures.

Run locally with Kind, Docker, Helm, kubectl and curl: build `nats-k8s-health-bridge:e2e`, load it into a Kind cluster named `health-bridge-ci`, apply `test/e2e/kind-smoke.yaml`, then run `bash test/e2e/kind-smoke.sh`.

### Container vulnerability scanning

The `.github/workflows/container-security.yml` workflow builds the image from the PR or main branch with `docker build --pull`, scans OS and Go library vulnerabilities with Trivy, reports **HIGH and CRITICAL** findings, and blocks merges for **fixable CRITICAL** findings. Unfixed vulnerabilities are not blocking in this initial gate; they still require risk assessment. The job is separate from the image-publishing workflow and does not publish images or require registry credentials.

**Production security checklist:** Use a supported, patched Go toolchain and base image; review HIGH findings and exceptions; restrict container capabilities and use `runAsNonRoot`, `readOnlyRootFilesystem`, `allowPrivilegeEscalation: false`, and `capabilities.drop: [ALL]` where supported; apply NetworkPolicies; keep credentials in Secrets; and use immutable image digests in production. A clean scan does not prove that the image or deployment is secure.

This workflow is a vulnerability gate, **not** SBOM publication, signed provenance, SLSA attestation, or a comprehensive dependency-review policy. Those remain follow-up work.
### Reproducible container toolchains

Development and CI examples use the NATS `2.15.1-alpine3.22` image instead of a floating `latest` tag; the builder uses Go `1.26.10-alpine3.23` (a supported Go toolchain) even though the module's minimum Go version is lower. These are specific version tags, not immutable digests: production builds should pin reviewed multi-architecture image digests and record their source. Validate toolchain compatibility and renovate version pins explicitly.

### Multi-platform container builds

The Dockerfile uses BuildKit `TARGETOS` and `TARGETARCH` to cross-compile for the requested image platform rather than always producing an amd64 executable. Build and verify both architectures before release:

```bash
docker buildx build --platform linux/amd64,linux/arm64 --tag example/nats-k8s-health-bridge:test .
```

The build command validates compilation for each target but does not by itself prove successful execution on both architectures. Validate the published image manifest and run an arm64 smoke test before declaring multi-platform support production-ready.

## Local Kubernetes validation

See [Local Kubernetes Validation Report](docs/local-kubernetes-validation.md) for the portable Windows PowerShell and Linux Bash lab procedure, observed functional and security test results, authentication regression analysis, rolling-update findings, and outstanding release-readiness work. The report uses generic resource names and does not disclose workstation or cluster-specific identifiers.
## Health evaluation architecture

The consumer threshold evaluator is isolated behind a `ConsumerEvaluator` interface; the current `LegacyEvaluator` remains the default, preserving existing behavior. See [jsm.go evaluator compatibility study](docs/jsm-evaluator-compatibility.md) for the upstream NATS monitor parity matrix and proposed adapter design. The upstream library is **not yet integrated**.

### Container publication controls

The CI workflow does **not** automatically publish GHCR images on pushes to `main` or version tags. Publishing requires a manual GitHub Actions **CI → Run workflow** invocation against a `v*` tag reference with the `publish_image` boolean explicitly selected. PR and normal push workflows continue to run tests but skip the publication job. Before publishing, verify the linked GHCR package's visibility independently in GitHub Packages; a private source repository does not guarantee that its associated container packages are private. This project deliberately does not modify package visibility settings automatically.

Do not select the publish option before code review, a green test matrix, a reviewed tag, package visibility review, and explicit approval. The former `make docker-push` and `setup.sh` paths have been removed: `make docker-build` builds only `nats-k8s-health-bridge:local`, and `make release-dry-run` checks publication guardrails offline without using Docker or contacting a registry. The approved release workflow emits full-version and commit-specific tags (no `latest` or floating major/minor tag). See [release publishing safety](docs/release-publishing.md).

### Maintainer architecture reference

The [developer architecture guide](CLAUDE.md) documents current readiness, authentication, redelivery and evaluator semantics. For upstream monitoring reuse, see [jsm.go compatibility](docs/jsm-evaluator-compatibility.md).

## Building

```bash
# Build binary
make build

# Run tests
make test

# Build a local Docker image (not GHCR)
make docker-build

# Verify release safeguards without publishing
make release-dry-run

# Run locally (requires NATS running on localhost:4222)
make run-local
```

## License

Apache 2.0 — see [LICENSE](LICENSE).

---

*Independently maintained project developed for [NATS](https://nats.io) (a [CNCF Graduated project](https://www.cncf.io/projects/nats/)) and [Kubernetes](https://kubernetes.io) (CNCF Graduated). This bridge is not an official CNCF or NATS project and does not claim their endorsement.*
