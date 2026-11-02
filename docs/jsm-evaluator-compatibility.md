# Evaluator compatibility study: jsm.go monitor

## Scope

This is a design and parity-test plan, not a claim that jsm.go has been integrated. The initial PR introduces a pure `ConsumerEvaluator` interface and retains `LegacyEvaluator` as the active implementation. Kubernetes readiness, heartbeat, observation freshness, NATS connectivity and HTTP authentication remain outside the evaluator.

Upstream references:
- [jsm.go consumer checks](https://github.com/nats-io/jsm.go/blob/main/monitor/consumer.go)
- [jsm.go monitor result](https://github.com/nats-io/jsm.go/blob/main/monitor/result.go)
- [NATS CLI server consumer checks](https://github.com/nats-io/natscli/blob/main/cli/server_consumer_check.go)

## Parity matrix

| Dimension | Current bridge | Upstream jsm.go monitor | Migration requirement |
| --- | --- | --- | --- |
| Unprocessed messages | `NumPending > max_lag` | `UnprocessedCritical` uses `>=` | Preserve strict boundary or explicitly version a behavior change |
| Outstanding acknowledgements | `NumAckPending > max_ack_pending` | `AckOutstandingCritical` uses `>=` | Boundary parity tests |
| Zero threshold | Disabled | Disabled for corresponding critical checks | Preserve |
| Redelivery | Gauge only; no threshold | `RedeliveryCritical` | Optional new feature, not silent default |
| Waiting pull requests | Not evaluated | `WaitingCritical` | Extend snapshot and config |
| Last delivery age | Not evaluated | `LastDeliveryCritical` | Define idle-consumer policy |
| Last acknowledgement age | Not evaluated | `LastAckCritical` | Define idle-consumer policy |
| Pinned client groups | Not evaluated | `Pinned` | Check supported server/client versions |
| Consumer fetch failures | Unknown, unready | Critical/unknown result abstractions | Preserve Kubernetes policy |
| Observation freshness | Last-successful age check | Not equivalent to bridge freshness policy | Keep in bridge |
| Application heartbeat | Pod-local POST, optional | No equivalent application progress heartbeat | Keep in bridge |
| Readiness policy | Degraded remains ready by default | No Kubernetes policy | Keep in bridge |

## Recommended adapter design

1. Keep `ConsumerEvaluator` independent of network access, metrics, Kubernetes probes and polling.
2. Preserve `LegacyEvaluator` as default and run its existing unit/integration tests.
3. In a subsequent PR, pin a compatible `jsm.go` version, evaluate its dependency footprint and API stability, and create a dedicated `JSMEvaluator`.
4. Preserve the full upstream consumer-info fields needed by checks, without silently changing the current public JSON schema.
5. Compare both evaluators against shared fixtures: threshold-1, threshold, threshold+1, idle consumers, no acknowledgements yet, redelivery, pinned consumers, missing metadata, stale observations and connection loss.
6. Normalize upstream severities into bridge-specific health classifications without conflating degraded with Kubernetes unready.
7. Benchmark polling latency, binary size and memory; document any behavioral differences.
8. Consider feature-gating the upstream evaluator before making it the default.

## Acceptance criteria for a future jsm.go adapter

- No regression in existing readiness, liveness, heartbeat, freshness, metrics or auth behavior.
- Boundary semantics are explicitly documented and tested.
- No additional NATS connection per consumer poll unless justified.
- Deterministic, documented handling of idle consumers.
- No silent metric renames or threshold interpretation changes.
- CI, Kind smoke tests and local Windows/Linux simulation remain green.
- Dependency and license review completed before public release.

## Why the bridge remains distinct

The upstream monitor provides established JetStream health checks. The bridge translates these observations into Kubernetes orchestration decisions, integrates application progress heartbeats, enforces freshness, and exposes diagnostics. Reusing upstream checks can improve consistency while preserving this separate operational role.
