# Private repository release-readiness audit and checklist

**Audit date:** 2026-11-03  
**Audited branch/commit:** `main` @ `578b0bb4c9d309198b71d9b5b1521f55d8170360`  
**Scope:** Repository documents, Go source, tests, Docker, Helm, GitHub Actions, reported repository rulesets, security posture, publication controls and code-comment quality.  
**Release decision:** **Do not make the repository or GHCR image public yet.** Resolve release-blocking findings and obtain explicit owner approval.

This is a **source and connected-repository audit**, not a penetration test, production certification, runtime assessment on every operating system, or inspection of private settings inaccessible to the connector. It uses only generic file paths and public upstream URLs: no workstation names, addresses, real tokens, cluster identities, or internal infrastructure data.

## Evidence and boundaries

- Repository metadata confirmed the repository is **private** on the audit date.
- All eight prior audit remediation PRs (#30–#37) had been merged into `main`.
- The latest `main` commit passed its **CI** and **Container Security** push workflows. The expanded **Kind Smoke Test** passed on the last integrated PR before its merge; it is configured for pull requests and manual runs, rather than regular pushes to `main`.
- The container security gate reports HIGH/CRITICAL with unfixed findings excluded, and blocks **fixable CRITICAL** vulnerabilities; a passing gate is not a complete vulnerability-absence statement.
- Two default-branch rulesets were visible: non-fast-forward/deletion protections, plus PR/code-owner review in one. Neither visible ruleset required named CI status checks. Admin-only branch-protection and GHCR package visibility could not be independently verified.
- The GitHub Releases listing was empty and a `v0.1.0` tag reference was not found. Existing `CHANGELOG.md`, `SECURITY.md` and chart version claims require reconciliation; do not assume a public 0.1.0 release occurred.
- Official [CNCF NATS project page](https://www.cncf.io/projects/nats/) confirms **NATS graduated on 2026-11-02**. Graduation belongs to **NATS**, not to this independent bridge. It does not imply official endorsement, certification or graduation of this repository.

## 1. Verified strengths to preserve

- [x] Non-root, multi-stage distroless container build.
- [x] JetStream consumer lag, acknowledgement, redelivery snapshot and connection metrics.
- [x] Distinct liveness and readiness semantics; threshold degradation is observational by default.
- [x] Maximum observation age with stale-state handling and bounded concurrent polling.
- [x] Typed fail-fast classification for permanent consumer errors.
- [x] Optional application heartbeat and HTTP endpoint regression tests.
- [x] Diagnostic bearer tokens fail closed by default, with test-only explicit anonymous opt-in.
- [x] Token environment-only binding regression fix previously verified.
- [x] Kind tests include a real consumer, readiness recovery, heartbeat expiry and NATS interruption after startup.
- [x] GitHub Actions GHCR publishing is explicitly gated on manual, version-tag-scoped dispatch.
- [x] Apache-2.0 license text and contribution/security reporting documents are present.

These observations must remain covered in regression tests as remediation proceeds.

## 2. Must-resolve technical and security issues

| Priority | Finding | Evidence | Issue | Exit criterion |
| --- | --- | --- | --- | --- |
| P1 | Optional NATS auth/TLS environment-only keys may be silently omitted by Viper unmarshaling | `internal/config/config.go` binds only diagnostic tokens | [#39](https://github.com/atipparaju/nats-k8s-health-bridge/issues/39) | Field-specific YAML/env precedence tests and real credential/TLS integration validation |
| P1 | NATS unavailable during initial startup causes process failure before HTTP probes start | `cmd/sidecar/main.go` creates NATS client before HTTP server | [#40](https://github.com/atipparaju/nats-k8s-health-bridge/issues/40) | Fail-operational startup policy and Kind NATS-down-first/recovery test |
| P1 | Error messages/log fields may expose credentials embedded in configured NATS URLs | `internal/nats/client.go` formats full `cfg.NATS.URL` into errors | [#41](https://github.com/atipparaju/nats-k8s-health-bridge/issues/41) | Redacted URL handling and synthetic credential-leak regression tests |
| P1 | Visible default-branch rulesets lack required CI status checks | Repository ruleset GET | [#42](https://github.com/atipparaju/nats-k8s-health-bridge/issues/42) | Repo-admin verifies required checks, bypass scope and package privacy |
| P1 | `:latest` and incomplete Secret wiring remain in Helm/sidecar examples | `README.md`, Helm values/template | [#43](https://github.com/atipparaju/nats-k8s-health-bridge/issues/43) | Correct template guidance, image provenance and Secret-mount tests |
| P1 | Legacy setup/push commands could bypass documented publication discipline | `setup.sh`, `Makefile` | [#45](https://github.com/atipparaju/nats-k8s-health-bridge/issues/45) | Remove/guard obsolete publication paths; no actual publishing |
| P1 | Release verification gaps remain (cross-platform, secret-history, licenses, multi-arch runtime, SBOM) | Current CI/release process | [#47](https://github.com/atipparaju/nats-k8s-health-bridge/issues/47) | Captured evidence, security review and owner sign-off |

The findings above are **potential risks and confirmed process/code gaps**, not claims of actual leaked credentials, a compromise, or a production incident.

## 3. Documentation, contributor experience and code readability

- [x] Update NATS's maturity badge and clarify independent project status in the README — [#38](https://github.com/atipparaju/nats-k8s-health-bridge/issues/38) (documentation PR).
- [ ] Reconcile `SECURITY.md` supported versions, `CHANGELOG.md`, Helm `appVersion`, unavailable `v0.1.0` public release and GitHub license detection — [#44](https://github.com/atipparaju/nats-k8s-health-bridge/issues/44).
- [ ] Remove assumptions that a private/unpublished GHCR image is pullable as `:latest` — [#43](https://github.com/atipparaju/nats-k8s-health-bridge/issues/43).
- [ ] Verify full Docker quickstart in **PowerShell 7+ on Windows** and **Bash on Linux**, including shell quoting, bind mount, authorized diagnostics and cleanup — [#47](https://github.com/atipparaju/nats-k8s-health-bridge/issues/47).
- [ ] Remove `setup.sh`, local GHCR push commands and stale release comments; verify the offline dry-run — [#45](https://github.com/atipparaju/nats-k8s-health-bridge/issues/45), [publication safety procedure](release-publishing.md). Pending PR merge and owner approval.
- [ ] Consistently apply `gofmt` and enforce formatting in CI — [#46](https://github.com/atipparaju/nats-k8s-health-bridge/issues/46).
- [ ] Correct inaccurate/misattached comments (notably `IsReady` vs `RecordHeartbeat` in `checker.go`; /healthz comment in `server.go`) — [#46](https://github.com/atipparaju/nats-k8s-health-bridge/issues/46).
- [ ] Comment **why** the worker pool is bounded, why observations become stale, why heartbeat is a readiness-only signal, and why a NATS outage does not fail liveness — [#46](https://github.com/atipparaju/nats-k8s-health-bridge/issues/46).
- [ ] Preserve concise GoDoc for public types and important operations; avoid comments that merely repeat a statement in the next line.
- [ ] Review README, `CLAUDE.md`, security policy, code of conduct and examples for publication-safe wording, working links and unintended personal/contact information before release.

**Commenting guideline:** Document exported API behavior, non-obvious invariants, failure semantics, and security tradeoffs. Do not add comments to every declaration or obvious conditional. The implementation is already relatively well structured; this should be a focused clarity/formatting cleanup, not a rewrite.

## 4. Release supply chain and repository settings

- [ ] Require named PR checks for Test & Lint, Integration Test, Container vulnerability gate, and Kind / Probe Smoke in default-branch rules, with appropriate administrator exceptions — [#42](https://github.com/atipparaju/nats-k8s-health-bridge/issues/42). See [repository ruleset and GHCR verification runbook](repository-security-controls.md). **Owner-admin settings verification still pending; an empty required-checks rule does not enforce green CI.**
- [ ] Confirm GHCR image/package visibility **in GitHub's own package settings**; do not infer it from repository privacy — [#42](https://github.com/atipparaju/nats-k8s-health-bridge/issues/42). If the container package does not exist, record **not published**, rather than creating one solely for validation.
- [ ] Ensure manual release/tag approval and image publication stay separate from routine CI — [#45](https://github.com/atipparaju/nats-k8s-health-bridge/issues/45).
- [ ] Review upstream container image digests, build provenance and provenance metadata for Linux amd64 and arm64 — [#47](https://github.com/atipparaju/nats-k8s-health-bridge/issues/47).
- [ ] Obtain an SBOM and verify license compatibility/attributions; run dependency, container and Git history secret scans — [#47](https://github.com/atipparaju/nats-k8s-health-bridge/issues/47).
- [ ] Validate image signature/attestation strategy and a minimal release artifact manifest — [#47](https://github.com/atipparaju/nats-k8s-health-bridge/issues/47).
- [ ] Validate that any production-ready claims are based on independent testing, rather than on passing source and Kind tests alone — [#47](https://github.com/atipparaju/nats-k8s-health-bridge/issues/47).
- [ ] Confirm publication ownership/IP approvals and any other required external approvals before making any part of the project public — [#47](https://github.com/atipparaju/nats-k8s-health-bridge/issues/47).

## 5. Suggested remediation order

**Gate A — Security and reliability:** #39, #41, #40. Re-run Go/race/integration and Kind tests after every merged PR.

**Gate B — Delivery safety:** #42 (settings review), #45 (manual publishing guard), #43 (Helm/README deployment correctness).

**Gate C — Maintainer experience:** #38 (NATS project status; documentation-only), #46 (comments/formatting), #44 (release metadata).

**Gate D — Release decision:** #47 (actual Windows/Linux steps, secret history, license/IP and artifact provenance review). Close all release-blocking issues and document acceptance before publication.

## 6. PR and release approval protocol

1. Use one focused issue and PR per substantive finding; avoid unrelated behavioral changes.
2. Update each branch against the **latest main** after earlier PRs merge; resolve conflicts explicitly.
3. Require complete CI, container security and Kind checks on the new integrated commit.
4. Include test coverage and migration/rollback notes for runtime changes.
5. Record evidence in each PR, avoid declaring tests green before the provider reports success.
6. Leave all PR merges, GHCR publication, repository visibility and release tagging to explicit owner approval.

**Final release status:** **Not yet approved for public release.** This checklist is the starting point for evidence-based closure.
