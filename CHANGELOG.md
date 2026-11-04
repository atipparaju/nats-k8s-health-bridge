# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- Keep bridge HTTP probes live when NATS is unavailable during initial startup; reconnect with the existing client retry policy and verify recovery in Kind.
- Redact NATS URL userinfo, passwords, tokens, paths and query parameters from connection and reconnect logs and errors; retain safe host metadata and typed error handling.
- Explicitly bind environment-only NATS credentials/NKey and TLS file-path configuration; reject mutually exclusive auth files and incomplete client certificate pairs before connecting.
- Reconciled maintainer documentation with actual health, redelivery, authorization and evaluator behavior.
- Removed the misleading redelivery-rate health threshold, which inferred delivery rates from non-monotonic JetStream consumer snapshots.
- Bound JetStream polling with a four-worker pool and configurable cycle timeout to prevent slow consumers from starving other observations.
- Skip retry delays for permanent JetStream missing-stream, missing-consumer and authorization errors while retaining retries for transient failures.
- Disabled automatic GHCR publishing on main/tag pushes; release image publication now requires an explicit manual tag-scoped CI dispatch.
- Register environment-only diagnostic bearer-token keys explicitly in Viper to prevent silently unauthenticated `/metrics` and `/status` endpoints when tokens are injected from Kubernetes Secrets.
- Correct Docker multi-platform builds by compiling Go binaries for BuildKit target architecture.
- Align Helm chart configuration with observation freshness, readiness policy, heartbeat and diagnostic endpoint settings.
- Fail readiness on stale consumer observations rather than reusing outdated successful poll results; report stale status in `/status`.

### Added
- A cross-platform Docker quickstart with a matching real JetStream consumer configuration and local image build.
- Expanded Kind smoke coverage with real consumer creation, healthy readiness, heartbeat expiration, NATS outage and recovery.
- Cross-platform local Kubernetes validation report covering functional, resilience, heartbeat, security and rolling-update testing.
- Container image vulnerability CI with Trivy: report HIGH/CRITICAL and block fixable CRITICAL findings.
- Kind-based Kubernetes smoke CI for Helm rendering, probe behavior, and upstream NATS outage resilience.
- Configurable HTTP bind address and optional bearer-token protection for `/metrics` and `/status`.
- Optional per-consumer application processing heartbeat (`heartbeat_timeout`) and loopback-only `POST /heartbeat/{consumer}` endpoint to detect stalled consumers.
- Configurable `max_observation_age` (default `30s`) and `last_successful` consumer timestamp.

### Changed
- Updated the NATS upstream CNCF maturity badge and independent-project attribution; added the pre-publication audit checklist.
- Pin NATS development/test images and a supported Go builder/CI toolchain to specific version tags; digest pinning remains a release follow-up.
- Diagnostic endpoints now fail closed when tokens are omitted; insecure anonymous access requires an explicit test-only opt-in.
- Extracted consumer threshold evaluation behind an interface while preserving existing strict-threshold semantics; added upstream jsm.go compatibility design documentation.
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
