# CLAUDE.md — nats-k8s-health-bridge

This is the Claude-specific entry point for the project. **Read [AGENTS.md](AGENTS.md) first** for the shared instructions that also apply to other AI coding assistants. Do not interpret this file as an alternative set of coding standards.

## Project snapshot

- **Repository:** https://github.com/atipparaju/nats-k8s-health-bridge (public)
- **Go module:** `github.com/atipparaju/nats-k8s-health-bridge`
- **License:** Apache-2.0
- **Stage:** Early, unsupported source preview; no released binary or approved public GHCR image.
- **Purpose:** A Kubernetes sidecar that turns NATS JetStream consumer observations and optional application heartbeats into HTTP readiness/liveness, Prometheus metrics, and authenticated diagnostics.
- **Boundaries:** Independently maintained; not an official NATS or CNCF project. The Helm chart provides sidecar configuration examples, not a deploying Kubernetes operator.

## Read before making changes

1. `AGENTS.md` — shared expectations, code map, safeguards, and PR hygiene.
2. `README.md` and `CONTRIBUTING.md` — current user-facing behavior and developer workflow.
3. Relevant `cmd/`, `internal/`, Helm or test sources; use tests to resolve documentation ambiguity.
4. `go.mod`, `Dockerfile`, and `.github/workflows/` — current dependency, platform and CI settings.

Prefer current source and tests when historical documentation conflicts. Do not hardcode dependency versions into this file: obtain exact versions from `go.mod` and CI when needed. As of this update, the current code uses the `github.com/nats-io/nats.go/jetstream` API; the upstream `jsm.go` monitor is a **future compatibility investigation**, not an installed evaluator. See `docs/jsm-evaluator-compatibility.md`.

## Implementation invariants

- **Liveness:** A responsive bridge remains live when NATS is unavailable. Disconnection or backlog must not force a liveness restart.
- **Readiness:** Connection, fresh consumer observation, and configured application heartbeat govern readiness. Lag and ack threshold breaches are observational unless strict readiness is explicitly enabled.
- **Thresholds:** The active `LegacyEvaluator` uses strict `>` comparisons. Do not silently change semantics or switch to `jsm.go`.
- **Diagnostics:** `/metrics` and `/status` require bearer authentication by default. Do not expose or log credentials, full NATS URLs, NKeys, TLS keys, tokens, or unsanitized errors.
- **Metrics:** Preserve metric names and labels, including the distinction between snapshot redelivery gauges and cumulative counters.
- **NATS connection:** Keep initial retry/reconnection, bounded polling, cancellation, error classification, and clean shutdown intact.
- **Compatibility:** Changes to HTTP JSON, configuration or Helm examples require focused tests and user-facing documentation.

## Validation and change delivery

Use idiomatic Go with `gofmt`, `context.Context` for I/O, `log/slog`, error wrapping, and explicit synchronization. Start with the relevant package tests, then run `go test -race -count=1 -timeout=60s ./...`, `go vet ./...`, and applicable integration/Kind/Helm tests. See `CONTRIBUTING.md` for exact commands and `docs/docker-quickstart.md` for Windows PowerShell/Linux Docker instructions.

Work in a focused branch and PR. Describe what changed, behavioral impact, tests actually executed, and any remaining uncertainty. Do not claim tests passed before observing their results.

**Public-safe workflow:** Only put reproducible technical content in public issues, PRs, commits, fixtures and logs. Never include credentials, private correspondence, personal or organization-specific details, local workstation paths or raw diagnostic secrets. Do not publish images, create releases/tags, change repository visibility or rewrite history without explicit maintainer authorization.
