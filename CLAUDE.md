# CLAUDE.md — nats-k8s-health-bridge

> Read this file fully before writing any code. It contains the complete project
> context, architecture, coding standards, and current implementation state.

---

## Project Purpose

`nats-k8s-health-bridge` is a pre-release Go sidecar intended for open-source release that bridges NATS JetStream
consumer health metrics to Kubernetes-native liveness and readiness probe HTTP endpoints,
plus a Prometheus /metrics endpoint.

**Problem it solves:** NATS JetStream consumers running in Kubernetes pods have no native
way to signal their health to Kubernetes. A consumer can be lagging, stuck, or disconnected
while the pod still shows `Ready`. This sidecar runs alongside the consumer container and
translates JetStream consumer state into K8s probe responses.

**Ecosystem positioning:** This project focuses on Kubernetes policy, freshness, and application
heartbeat integration for JetStream consumers. Related NATS monitoring and autoscaling tools
already exist; do not claim uniqueness or NATS/CNCF endorsement without independent evidence.

**NATS maturity:** The upstream NATS project graduated in the CNCF on November 2, 2026 (https://www.cncf.io/projects/nats/). This private bridge is independently maintained; do not imply CNCF membership, certification, sponsorship or NATS endorsement for this repository.

**GitHub repo:** https://github.com/atipparaju/nats-k8s-health-bridge (private)
**Go module:** `github.com/atipparaju/nats-k8s-health-bridge`
**License:** Apache 2.0
**Phase:** 1 — Sidecar only. Operator (CRD-based) is Phase 2.

---

## Architecture

```
┌─────────────────────────── Pod ────────────────────────────────┐
│                                                                 │
│  ┌───────────────────┐        ┌──────────────────────────────┐ │
│  │  Your Consumer    │        │  nats-k8s-health-bridge      │ │
│  │  Application      │        │  (sidecar)                   │ │
│  │                   │        │                              │ │
│  │  - Reads from     │        │  Poll loop (every N secs)    │ │
│  │    JetStream      │        │    └─► js.ConsumerInfo()     │ │
│  │  - Processes msgs │        │         └─► threshold eval   │ │
│  └───────────────────┘        │                              │ │
│                               │  HTTP endpoints:             │ │
│                               │    /healthz  → 200/503       │ │
│  Both containers share        │    /readyz   → 200/503       │ │
│  the same NATS connection     │    /metrics  → Prometheus    │ │
│  credentials (via volume)     │    /status   → JSON debug    │ │
│                               └──────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
         │                                    │
         ▼                                    ▼
   NATS JetStream                      Kubernetes kubelet
   (stream + consumer)                 (liveness/readiness)
```

---

## Representative File Tree

```
nats-k8s-health-bridge/
├── CLAUDE.md                        ← this file
├── README.md                        ← user-facing docs
├── LICENSE                          ← Apache 2.0
├── go.mod                           ← Go module definition, pinned deps
├── go.sum                           ← dependency checksums (generated)
├── config.yaml                      ← annotated example configuration
├── Dockerfile                       ← multi-stage distroless build
├── Makefile                         ← build/test/docker/helm targets
├── .gitignore
├── .github/
│   └── workflows/
│       ├── ci.yml                   ← tests and container release pipeline
│       ├── container-security.yml   ← container vulnerability scanning
│       └── kind-smoke.yml           ← Kubernetes smoke tests
├── cmd/
│   └── sidecar/
│       └── main.go                  ← entrypoint: cobra CLI, signal handling, wiring
├── internal/
│   ├── config/
│   │   └── config.go                ← viper YAML+env config, validation
│   ├── nats/
│   │   └── client.go                ← NATS connection, reconnect, GetConsumerInfo()
│   ├── health/
│   │   └── checker.go               ← polling loop, threshold eval, state management
│   ├── metrics/
│   │   └── prometheus.go            ← all Prometheus metric definitions
│   └── server/
│       └── server.go                ← HTTP server: /healthz /readyz /metrics /status
└── helm/
    └── nats-k8s-health-bridge/
        ├── Chart.yaml
        ├── values.yaml
        └── templates/
            ├── _helpers.tpl
            └── configmap.yaml
```

---

For the complete current tree, inspect the repository. The listing below highlights core source files and does not enumerate all tests and documentation.

## Dependency Versions (pinned in go.mod)

| Package | Version | Purpose |
|---|---|---|
| `github.com/nats-io/nats.go` | v1.49.0 | NATS client + JetStream API |
| `github.com/prometheus/client_golang` | v1.21.1 | Prometheus metrics |
| `github.com/spf13/cobra` | v1.9.1 | CLI flags |
| `github.com/spf13/viper` | v1.20.1 | YAML + env var config |

Go module minimum version: **1.24** (see `go.mod`; CI/toolchain pins may be newer).

---

## Key API Details

**Connection-log security invariant:** Never write raw configured NATS URLs, `ConnectedUrl()`, or unfiltered transport errors to logs or HTTP diagnostics. Use `safeNATSEndpoints` and `safeNATSError` for host-only metadata and categorized errors; preserve `errors.Is` via wrapping. Add synthetic credential-leak tests for any new connection lifecycle path.

### NATS JetStream (nats.go v1.49.0)

The project uses the **new JetStream API** (not the legacy `nc.JetStream()` context):

```go
import "github.com/nats-io/nats.go/jetstream"

js, err := jetstream.New(nc)
consumer, err := js.Consumer(ctx, streamName, consumerName)
info, err := consumer.Info(ctx)
```

**ConsumerInfo fields used:**
- `info.NumPending`     — undelivered messages (lag)
- `info.NumAckPending`  — delivered but not yet acked
- `info.NumRedelivered` — current redelivery snapshot, not a lifetime counter
- `info.NumWaiting`     — pull consumers: waiting fetch requests
- `info.PushBound`      — push consumer has active subscriber

**Connection state:**
```go
nc.IsConnected() // bool — NATS availability contributes to /readyz, not /healthz liveness
```

---

## Configuration System

12-factor style: YAML file + environment variable overrides.

**Env var prefix:** `NKHB_`
**Key mapping:** dots → underscores (e.g. `nats.url` → `NKHB_NATS_URL`)

**Environment-only NATS Secret configuration:** Viper `AutomaticEnv` does not reliably include keys absent from YAML during `Unmarshal`. Explicit `BindEnv` registration is required for `nats.credentials_file`, `nats.nkey_file`, `nats.tls_cert`, `nats.tls_key`, and `nats.tls_ca`, as well as the diagnostic bearer tokens. The two NATS identity file settings are mutually exclusive; the client certificate/key must be set together; a standalone CA file is valid. Keep credential contents out of logs and source files.

Config is loaded in `internal/config/config.go` via viper:
1. Reads `config.yaml` (or path from `--config` flag)
2. Also checks `/etc/nats-k8s-health-bridge/config.yaml`
3. Env vars override any YAML values
4. Validates required fields and sets defaults

**Full Config struct:**
```
Config
├── LogLevel       string
├── PollInterval       time.Duration
├── PollTimeout        time.Duration (default 8s, bounded polling cycle)
├── MaxObservationAge  time.Duration
├── Server
│   ├── Port, BindAddress, ShutdownTimeout
│   ├── ReadinessFailOnDegraded
│   ├── AllowUnauthenticatedDiagnostics (false by default)
│   └── MetricsToken, StatusToken
├── NATS
│   ├── URL, CredentialsFile, NKeyFile
│   ├── TLSCert, TLSKey, TLSCA
│   ├── ConnectTimeout, ReconnectWait, MaxReconnects
└── Consumers []
    ├── Name, Stream, Consumer
    ├── HeartbeatTimeout time.Duration (optional)
    └── Thresholds
        ├── MaxLag        uint64
        └── MaxAckPending uint64
```

---

## HTTP Endpoints

| Endpoint | Probe type | Logic |
|---|---|---|
| `GET /healthz` | Liveness | Process liveness independent of NATS connectivity |
| `GET /readyz` | Readiness | Requires NATS, fresh consumer observations and enabled heartbeat; threshold degradation fails readiness only with strict opt-in |
| `GET /metrics` | Prometheus | HTTP 401 without configured bearer token, unless local-test anonymous opt-in |
| `GET /status` | Debug | HTTP 401 without configured bearer token, unless local-test anonymous opt-in; historical message counts may be stale |

---

## Prometheus Metrics

All metrics registered in `internal/metrics/prometheus.go`:

| Metric | Type | Labels |
|---|---|---|
| `nats_consumer_up` | Gauge | `consumer_name`, `stream`, `consumer` |
| `nats_consumer_lag` | Gauge | `consumer_name`, `stream`, `consumer` |
| `nats_consumer_ack_pending` | Gauge | `consumer_name`, `stream`, `consumer` |
| `nats_consumer_redelivered` | Gauge | `consumer_name`, `stream`, `consumer` |
| `nats_connected` | Gauge | _(none)_ |

Plus standard Go runtime and process metrics via `prometheus.NewGoCollector()`.

---

## Coding Standards

These rules apply to every file in this project. Do not deviate.

### Error handling
```go
// Always wrap errors with context
return fmt.Errorf("connecting to NATS: %w", err)

// Never ignore errors silently — log and handle or propagate
if err != nil {
    slog.Warn("non-fatal error", "error", err)
}
```

### Logging (slog)
```go
// Always use structured key-value pairs, never fmt.Sprintf in log args
slog.Info("consumer polled", "consumer", name, "lag", lag)
slog.Warn("threshold exceeded", "consumer", name, "lag", lag, "max", max)
slog.Error("fatal error", "error", err)

// Debug-level for hot paths (polling loop output)
slog.Debug("http request", "method", r.Method, "status", rw.status)
```

### Context propagation
```go
// All functions that do I/O must accept ctx context.Context as first argument
func (c *Client) GetConsumerInfo(ctx context.Context, stream, consumer string) (*ConsumerInfo, error)
```

### No package-level globals
```go
// BAD
var globalChecker *health.Checker

// GOOD — pass dependencies via constructors
func NewChecker(client *natsclient.Client, cfg *config.Config, ...) *Checker
```

### Concurrency
```go
// Protect shared state with sync.RWMutex
// Use RLock/RUnlock for reads, Lock/Unlock for writes
c.mu.RLock()
defer c.mu.RUnlock()
```

### HTTP responses
```go
// Always set Content-Type before WriteHeader
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(status)
json.NewEncoder(w).Encode(body)
```

---

## Build Commands

```bash
# Download dependencies
go mod tidy

# Build binary
make build
# or: go build -o bin/nats-k8s-health-bridge ./cmd/sidecar

# Run all tests
make test
# or: go test -race -count=1 ./...

# Run locally (needs NATS on localhost:4222)
make run-local

# Build Docker image
make docker-build

# Push Docker image to GHCR — restricted to explicitly approved release operations
# DO NOT run without owner approval, package visibility review and successful CI.
# make docker-push
```

---

## Running Tests Locally

```bash
# Start NATS with JetStream for integration tests (use a reviewed, pinned version)
docker run -d --name nats-test -p 4222:4222 nats:2.15.1-alpine3.22 -js

# Run tests
go test -race -count=1 ./...

# Stop NATS
docker stop nats-test && docker rm nats-test
```

Unit tests should be placed alongside source files: `internal/health/checker_test.go` etc.

**Test priorities (implement in this order):**
1. `internal/config` — config loading, validation, env var overrides
2. `internal/health` — threshold evaluation logic (mock the NATS client)
3. `internal/server` — HTTP handler responses (httptest.NewRecorder)
4. `internal/nats` — integration test with real NATS (use build tag `//go:build integration`)

---

## Testing with a Local NATS Server

```bash
# Start NATS with JetStream (use a reviewed, pinned version)
docker run -d --name nats-dev -p 4222:4222 -p 8222:8222 nats:2.15.1-alpine3.22 -js

# Create a test stream and consumer (requires nats CLI)
nats stream add ORDERS --subjects "orders.>" --storage memory --retention limits --max-msgs -1 --max-bytes -1 --max-age -1 --replicas 1 --defaults
nats consumer add ORDERS order-processor --pull --defaults

# Run the sidecar with example config
make run-local

# In another terminal — check endpoints
curl http://localhost:8080/healthz   # should return {"status":"ok"}
curl http://localhost:8080/readyz    # may return 503 until consumer is polled
curl http://localhost:8080/status    # 401 unless a bearer token is configured or isolated-test anonymous access is explicitly enabled
curl http://localhost:8080/metrics   # 401 under the same default policy

# Publish some test messages to create lag
nats pub orders.new "test message" --count=100

# Watch lag with an authenticated status request (or local-only explicit anonymous opt-in)
# curl -H "Authorization: Bearer $NKHB_SERVER_STATUS_TOKEN" http://localhost:8080/status | jq .
```

---

## Current Implementation Status

**Readiness policy:** Consumer lag and acknowledgement threshold breaches are observational by default, not automatic pod eviction. An optional application heartbeat is supported; see README for its limitations. A pure `LegacyEvaluator` currently implements health thresholds, while reuse of `jsm.go` is documented as a future compatibility study.


### ✅ Complete (Phase 1)
- [x] Project scaffold (go.mod, Dockerfile, Makefile, CI workflow)
- [x] Config system (YAML + env vars via viper)
- [x] NATS client wrapper with reconnect handlers
- [x] JetStream ConsumerInfo polling
- [x] Health checker with threshold evaluation
- [x] Prometheus metrics (5 metrics)
- [x] HTTP server (/healthz, /readyz, /metrics, /status)
- [x] Graceful shutdown (SIGTERM/SIGINT)
- [x] Helm chart (ConfigMap + values)
- [x] README with K8s sidecar injection example

### ✅ Complete (Phase 1 — continued)
- [x] `go.sum` — generated via `go mod tidy`
- [x] `LICENSE` — Apache 2.0 text added
- [x] Unit tests for config, health, server, and metrics packages
- [x] Module path set to `github.com/atipparaju/nats-k8s-health-bridge`
- [x] Integration test with real NATS (build tag: `integration`)
- [x] Removed invalid redelivery-rate inference; redelivery remains available as a snapshot gauge
- [x] Retry logic for NATS operations
- [x] CONTRIBUTING.md
- [x] Docker config: embedded default config.yaml, env-var-only mode
- [x] Interface extraction for testability

### 🔲 Phase 2 (Operator — do NOT start until Phase 1 tests pass)
- [ ] `NATSConsumerHealth` CRD definition
- [ ] Kubernetes operator using controller-runtime
- [ ] Pod annotation updates based on consumer health
- [ ] KEDA ScaledObject integration
- [ ] Argo Rollouts readiness gate

---

## Module Path

All source files use `github.com/atipparaju/nats-k8s-health-bridge`.
