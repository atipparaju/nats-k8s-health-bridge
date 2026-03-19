# Refactoring Plan: Selective Pattern Adoption

**Date:** 2026-03-18
**Goal:** Improve testability and robustness while keeping CNCF-ready simplicity
**Reference:** msvc_irouter hexagonal architecture patterns (selective adoption)

---

## Guiding Principles

1. **CNCF-first**: Minimal, idiomatic Go. Easy to read in 15 minutes.
2. **Interface-driven testability**: Extract interfaces only where they enable testing.
3. **No shared libraries**: Everything self-contained in `internal/`.
4. **No over-abstraction**: No DI containers, builders, or hexagonal layering.
5. **Borrowed patterns, not borrowed code**: Inspired by msvc_irouter, not copied.

---

## Phase 1A: Interface Extraction & Unit Tests (Current Sprint)

### Step 1: Extract `NATSClient` interface in `internal/health/`

**Why:** `health.Checker` currently depends on concrete `*nats.Client`. This makes it
impossible to unit test threshold evaluation without a running NATS server.

**What:**
```go
// internal/health/checker.go
type NATSClient interface {
    IsConnected() bool
    GetConsumerInfo(ctx context.Context, stream, consumer string) (*natsclient.ConsumerInfo, error)
}
```

- `health.Checker` accepts `NATSClient` interface instead of `*natsclient.Client`
- `*natsclient.Client` already satisfies this interface (no changes to nats package)
- Tests can use a mock implementation

**Impact:** Enables full unit test coverage of checker without NATS.

### Step 2: Extract `HealthChecker` interface in `internal/server/`

**Why:** `server.Server` depends on concrete `*health.Checker`. Extracting an interface
allows HTTP handler tests with controlled health states.

**What:**
```go
// internal/server/server.go
type HealthChecker interface {
    IsLive() bool
    IsReady() (bool, []string)
    States() []*health.ConsumerState
}
```

**Impact:** HTTP handler tests become simple table-driven tests with mock checker.

### Step 3: Write unit tests for all packages

**Priority order (per CLAUDE.md):**

1. `internal/config/` — config loading, validation, env var overrides
2. `internal/health/` — threshold evaluation with mock NATSClient
3. `internal/server/` — HTTP handlers with mock HealthChecker (httptest)
4. `internal/metrics/` — metric registration verification

**Target:** >80% coverage (CNCF reviewers expect this).

### Step 4: Implement MaxRedeliveryRate threshold

**What:** Track NumRedelivered delta between consecutive polls over EvalWindow.
Compute rate = delta / total_delivered. Compare to MaxRedeliveryRate threshold.

**Inspired by:** msvc_irouter's metric tracking patterns in domain services.

---

## Phase 1B: Robustness Improvements

### Step 5: Add retry logic to GetConsumerInfo

**Inspired by:** msvc_irouter's `natsx.CreateStreamWithRetry()` pattern.

**What:** Simple retry (2 attempts, 1s backoff) on transient NATS errors in
`GetConsumerInfo`. Not a shared utility — just inline retry in the client method.

### Step 6: Improve graceful shutdown

**Inspired by:** msvc_irouter's `natsx.SetupGracefulShutdown()` and context propagation.

**What:** Ensure all goroutines respect context cancellation cleanly. Add shutdown
logging similar to msvc_irouter's operation-tagged log entries.

---

## Phase 1C: CNCF Readiness Polish

### Step 7: Integration test with build tag

**What:** `internal/nats/client_integration_test.go` with `//go:build integration`
Tests against real NATS server (docker-based in CI).

### Step 8: CI pipeline enhancements

- Add coverage reporting to CI
- Add golangci-lint step
- Badge in README (coverage %, build status)

### Step 9: Documentation review

- Ensure README has clear "Why this exists" section
- Add CONTRIBUTING.md
- Add architecture diagram
- Verify all code examples in README compile

---

## What We Are NOT Doing (and Why)

| Pattern from msvc_irouter | Why we skip it |
|---|---|
| Hexagonal architecture (domain/application/infrastructure) | Over-engineering for a single-purpose sidecar |
| DI Container (`container.go`) | Only 4-5 deps — main.go wiring is clearer |
| Builder pattern | Not enough components to justify |
| Enterprise logging (`pkgs/logging`) | `log/slog` is idiomatic and sufficient |
| Typed domain errors (`pkgs/errors`) | `fmt.Errorf` with `%w` is standard Go |
| Gin HTTP framework | `net/http` is perfect for 4 endpoints |
| Shared `pkgs/` library | CNCF projects must be self-contained |
| Database/cache layers | Not applicable — no persistence |
| envconfig (replacing Viper) | Viper's YAML+env is more user-friendly for sidecar config |

---

## Implementation Order

```
Step 1: Extract NATSClient interface     → enables Step 3
Step 2: Extract HealthChecker interface   → enables Step 3
Step 3: Write unit tests (all packages)   → biggest CNCF value
Step 4: MaxRedeliveryRate threshold       → completes Phase 1 feature set
Step 5: Retry logic in GetConsumerInfo    → robustness
Step 6: Graceful shutdown improvements    → production readiness
Step 7: Integration tests                 → CI confidence
Step 8: CI pipeline enhancements          → CNCF presentation
Step 9: Documentation review              → CNCF submission ready
```

Estimated: Steps 1-4 are the critical path. Steps 5-9 are polish.

---

## TODO Checklist

- [x] Step 1: Extract NATSClient interface in health package
- [x] Step 2: Extract HealthChecker interface in server package
- [x] Step 3: Unit tests for config, health, server, metrics
- [x] Step 4: Implement MaxRedeliveryRate sliding window
- [x] Step 5: Add retry to GetConsumerInfo
- [x] Step 6: Improve graceful shutdown
- [x] Step 7: Integration test with build tag
- [x] Step 8: CI pipeline (coverage, lint, integration tests)
- [x] Step 9: CONTRIBUTING.md added
