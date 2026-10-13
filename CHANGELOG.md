# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- Fail readiness on stale consumer observations rather than reusing outdated successful poll results; report stale status in `/status`.

### Added
- Configurable HTTP bind address and optional bearer-token protection for `/metrics` and `/status`.
- Optional per-consumer application processing heartbeat (`heartbeat_timeout`) and loopback-only `POST /heartbeat/{consumer}` endpoint to detect stalled consumers.
- Configurable `max_observation_age` (default `30s`) and `last_successful` consumer timestamp.

- Removed the misleading redelivery-rate health threshold, which inferred delivery rates from non-monotonic JetStream consumer snapshots.

### Changed
- Removed `max_redelivery_rate` and `eval_window` configuration keys; use application-level delivery/ack instrumentation for rate-based monitoring.


## [0.1.0] - 2026-04-12

### Added

- NATS JetStream consumer health monitoring sidecar for Kubernetes
- HTTP health probe endpoints: `/healthz` (liveness), `/readyz` (readiness), `/status` (debug JSON)
- Prometheus metrics endpoint at `/metrics` with 5 consumer metrics (`nats_consumer_up`, `nats_consumer_lag`, `nats_consumer_ack_pending`, `nats_consumer_redelivered_total`, `nats_connected`)
- YAML + environment variable configuration (12-factor style, `NKHB_` prefix)
- Configurable thresholds: `max_lag`, `max_ack_pending`, `max_redelivery_rate` with sliding window evaluation
- NATS connection management with automatic reconnect and retry logic
- Graceful shutdown on SIGTERM/SIGINT with configurable timeout
- Multi-stage Dockerfile using distroless base image (nonroot)
- Helm chart with ConfigMap and sidecar injection example
- GitHub Actions CI: test, lint, integration test, Docker build and push to GHCR
- TLS and NKey/credentials authentication support for NATS connections
- Structured JSON logging via `log/slog`

[Unreleased]: https://github.com/atipparaju/nats-k8s-health-bridge/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/atipparaju/nats-k8s-health-bridge/releases/tag/v0.1.0
