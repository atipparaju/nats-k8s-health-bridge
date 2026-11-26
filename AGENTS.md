# AGENTS.md

Guidance for coding agents and automated contributors working in this repository.

## Scope and source of truth

- This is an independently maintained, early-stage Go sidecar that translates NATS JetStream consumer observations into Kubernetes readiness/liveness endpoints and Prometheus metrics.
- Before changing behavior, read `README.md`, `CONTRIBUTING.md`, the relevant source and tests, and `docs/docker-quickstart.md` for end-to-end usage.
- Treat **current source, `go.mod`, and passing tests** as authoritative if documentation becomes stale. `CLAUDE.md` is a tool-specific entry point to these shared instructions; keep it consistent rather than duplicating changing facts. Do not repeat historical version, release, or deployment claims without checking them.
- Keep changes focused. Explain the problem, expected behavior, compatibility impact, and validation in the pull request.

## Code map

- `cmd/sidecar/`: CLI and application wiring
- `internal/config/`: YAML/environment configuration and validation
- `internal/nats/`: NATS connection lifecycle and JetStream consumer observations
- `internal/health/`: polling, evaluation, heartbeats, freshness, readiness
- `internal/metrics/`: Prometheus instrumentation
- `internal/server/`: HTTP probes, status and diagnostic authentication
- `helm/nats-k8s-health-bridge/`: Kubernetes sidecar configuration examples
- `test/release/`: local Windows/Linux smoke-test harnesses

## Behavior to preserve

- `/healthz` measures the responsive bridge process; NATS disconnection must **not** cause liveness failures/restart loops.
- `/readyz` requires usable, fresh consumer observations and any configured application heartbeat. Threshold degradation fails readiness **only when explicitly enabled**.
- `/metrics` and `/status` reject unauthenticated access by default. Never silently enable public diagnostics.
- Do not log raw NATS URLs, credentials, tokens, NKeys, TLS keys or unfiltered connection errors. Maintain sanitization and failure-path tests.
- Keep polling bounded, propagate `context.Context`, and handle reconnection/cancellation without masking permanent configuration errors.
- Preserve published metric names/labels and JSON response shapes unless a deliberate, tested compatibility change is documented.
- `nats.go/jetstream` is the active client API; `jsm.go` is **not** currently integrated. See `docs/jsm-evaluator-compatibility.md` before proposing an evaluator adapter. Preserve strict `>` threshold semantics unless a behavior change is explicitly reviewed.

## Implementation conventions

- Use idiomatic Go, `gofmt`, structured `log/slog` logging, explicit errors with `%w`, and dependency injection rather than mutable package globals.
- Keep I/O cancellable, use appropriate synchronization, and set HTTP headers before writing status codes.
- Prefer table-driven tests and existing test fixtures. Cover healthy, degraded, unknown, disconnected, stale, unauthorized and recovery paths when relevant.
- Maintain license headers on new Go files as established by existing source.

## Validation

Run the smallest relevant tests while developing, then the applicable full checks before proposing merge (run `go mod tidy` only when dependency/module changes warrant it):

```sh
go test ./...
go test -race -count=1 -timeout=60s ./...
go vet ./...
go mod tidy  # only when updating dependency/module metadata
git diff --exit-code -- go.mod go.sum
```

For runtime/integration changes, run NATS integration tests with a local JetStream-enabled server, and validate Helm and documented Docker/Kind behavior as appropriate. See `CONTRIBUTING.md` and CI workflows for precise setup and current toolchain versions. On Windows, use PowerShell or the documented Windows smoke-test script instead of assuming Bash/Make is available.

Do not claim a check passed unless you ran it or verified the corresponding CI result.

## Public repository hygiene

- Issues and pull requests should contain **technical, public-safe information only**: reproducible bugs, concrete enhancements, tests and design tradeoffs.
- Never include private planning notes, personal data, workplace details, credentials, internal URLs, local machine paths or raw security scan matches in public issues, comments, commits, fixtures or artifacts.
- Use synthetic examples; review generated diffs, logs and screenshots before posting.
- Do not introduce copied third-party code/assets without checking licensing and attribution.
- Do not change repository visibility, publish images or packages, push release tags, modify branch protections or rewrite history as part of routine code work. Such actions require explicit maintainer direction.
- The source preview does not imply a supported binary, public GHCR image, or official NATS/CNCF endorsement.

When uncertain, propose the smallest reversible change and identify open technical questions in the PR rather than making assumptions.
