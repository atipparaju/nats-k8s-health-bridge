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
| `NKHB_SERVER_PORT` | `server.port` | `8080` | HTTP server port |
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

### Readiness versus degradation

By default, `/readyz` requires a NATS connection and successfully retrieved consumer state. A consumer that exceeds `max_lag` or `max_ack_pending` (removed) is reported as **degraded** in `/status` and metrics, but the pod stays ready. This prevents a healthy worker with a large backlog from being removed from service solely because of load.

Set `server.readiness_fail_on_degraded: true` only if your deployment explicitly requires threshold breaches to fail Kubernetes readiness. This is an opt-in compatibility mode; it may reduce capacity during incidents. NATS disconnections and unknown consumer states continue to fail readiness. Readiness is not a substitute for an application-level processing heartbeat.

### Redelivery-rate measurement

JetStream consumer-info `num_redelivered` is a point-in-time count, not a monotonic lifetime counter. The former `max_redelivery_rate` threshold was calculated from changes in `num_redelivered` and `num_ack_pending`; these snapshots cannot provide a reliable delivery denominator or event rate. **That threshold has been removed** rather than producing misleading health decisions. `eval_window` is also removed because it was only used by this calculation.

For rate-based alerting, instrument message delivery attempts and acknowledgments in the consumer application or another reliable event source. Derive redelivery ratios from observed events over an explicit window. `/status` still includes the current `num_redelivered` snapshot; the metric fix is handled separately in the redelivery gauge PR. Remove `max_redelivery_rate` and `eval_window` from existing configurations before upgrading.

## Prometheus Metrics

| Metric | Type | Labels | Description |
|---|---|---|---|
| `nats_consumer_up` | Gauge | `consumer_name`, `stream`, `consumer` | `1` = healthy, `0` = degraded |
| `nats_consumer_lag` | Gauge | `consumer_name`, `stream`, `consumer` | `num_pending` messages |
| `nats_consumer_ack_pending` | Gauge | `consumer_name`, `stream`, `consumer` | `num_ack_pending` |
| `nats_consumer_redelivered` | Gauge | `consumer_name`, `stream`, `consumer` | Current JetStream redelivery snapshot (not a cumulative total) |
| `nats_connected` | Gauge | — | `1` if connected to NATS |

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
