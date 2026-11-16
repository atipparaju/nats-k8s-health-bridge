# Issue #47 — Private pre-publication evidence and owner decision record

**Updated:** 2026-11-09 · **Repository status:** private · **GHCR status:** owner previously confirmed Private; recheck before publication.  
**Decision:** **NOT APPROVED TO PUBLISH.** This file is a plan and an evidence ledger, not a signed certification.

## Evidence reconciliation — 2026-11-13 (Issue #64)

**Current source baseline:** PR #62 merged into `main` as commit `25dfcee5396df3e52b2f647cb775a2a8833876be` on November 13, 2026. The last fully validated PR head was `c2b5135672026da571e96efea8f2d0a3eee5a817`. The manual Windows/Linux tests below were performed on earlier commit `acc39612feda9a5de04c35d2e0a30c029b9e6279`, before PR #62; the Linux Buildx preflight was also tested separately on the PR branch. Do not treat these as one identical run.

### Recorded owner-operated host tests

| Test | Result and scope | Environment | Evidence limitations |
| --- | --- | --- | --- |
| Windows Docker Desktop local smoke (2026-11-11) | **PASS**: disposable real JetStream consumer, readiness and protected diagnostics | Windows/amd64; Docker Desktop 4.99.0 (245822); Docker Engine 29.9.3; Buildx 0.38.1; PowerShell 7.6.6; Go 1.26.2 | Owner-provided terminal output and Desktop screenshot; exact Windows OS version, portable full logs and automated artifact absent |
| Native Linux Docker smoke on OpenVillage AI (2026-11-11) | **PASS**: real JetStream consumer, readiness, protected diagnostics and reported cleanup | Ubuntu 26.04.1 LTS, linux/amd64; Docker Engine 29.1.3; Buildx 0.30.1; Go 1.26.8 | Owner-provided terminal output; no independent archived full log |
| Linux Buildx fix PR #62 (2026-11-12) | **PASS** owner-reported positive smoke with Buildx; **PASS** simulated missing-Buildx early failure with actionable error | Native OVAI Linux | Actual Docker resource absence after simulated negative case was not separately inspected; branch-specific commit not captured in owner terminal output |

[Issue #47 evidence comment](https://github.com/atipparaju/nats-k8s-health-bridge/issues/47#issuecomment-6810701894) records the first two platform tests.

### Verified automated evidence on PR #62 final pre-merge head

| GitHub Actions workflow | Run ID | Recorded result |
| --- | --- | --- |
| CI (Test & Lint, Integration Test) | [43520813166](https://github.com/atipparaju/nats-k8s-health-bridge/actions/runs/43520813166) | **SUCCESS** |
| Container Security | [43520813129](https://github.com/atipparaju/nats-k8s-health-bridge/actions/runs/43520813129) | **SUCCESS**; not proof of absence of all vulnerabilities |
| Kubernetes Smoke Test (Kind / Probe Smoke) | [43520813133](https://github.com/atipparaju/nats-k8s-health-bridge/actions/runs/43520813133) | **SUCCESS** |
| Private Release Validation | [43520813262](https://github.com/atipparaju/nats-k8s-health-bridge/actions/runs/43520813262) | **SUCCESS**: Linux Docker Quickstart, Windows PowerShell syntax (not Docker Desktop runtime), Git History Secret Scan, Local Image SBOM, Linux arm64 runtime smoke under QEMU |

The full-history Gitleaks scan passed with exact, reviewed synthetic-test fingerprint entries documented in Issues #61 and #63 and merged PR #62. It is **not** a guarantee that every external ref or untracked artifact was scanned. The private SPDX artifact `private-candidate-sbom.spdx.json` (artifact ID `14081449878`, run `43520813262`) existed and was not expired when queried on November 13, 2026. Artifact existence is **not** a third-party license review or approval.

**Post-merge verification boundary:** merged `main` was checked for the Buildx preflight and exact historical Gitleaks entries; a separate post-merge CI success has not been independently established in this ledger.

### Remaining owner decisions — do not check off automatically

- [x] Owner-reported real Windows Docker Desktop and native Linux/amd64 first-run smoke results and their available environment metadata recorded
- [x] Pre-merge PR #62 CI, Kind, Container Security, Linux/Windows syntax, Gitleaks, SBOM generation and QEMU arm64 evidence recorded with run IDs
- [ ] Review the **contents** of the private SPDX artifact for dependency license compatibility, required NOTICE attribution and exceptions; inventory generation alone is insufficient
- [ ] Reverify GitHub REST license/SPDX recognition; reconcile source copyright headers and actual rights-holder identity without assuming ownership transfer
- [ ] Verify contributor/employment/other contractual rights and obtain any required written approvals for public distribution
- [ ] Recheck GHCR package visibility and Actions access in provider settings immediately before any contemplated publication
- [ ] Decide precise supported Go/NATS/Kubernetes and operating-system compatibility boundaries; assess whether native ARM64 validation is needed
- [ ] Agree on registry-image digest, signed provenance and attestation strategy; no registry digest/signature exists until an explicitly approved artifact is published
- [ ] Explicitly authorize the exact code commit, version/tag, image destination, license notices, repository/package visibility and publication timing

**Decision: NO-GO for public release.** Automated CI and host smoke tests do not resolve the legal, rights, dependency-attribution, registry-provenance or owner-authorization gates. This document records evidence only and must not trigger publication.

## Image-index digest candidate collection (Issue #78)

The private `Read-only Source Image Digest Audit` job resolves **registry index digests** for the exact Go builder, NATS Server test fixture, and Distroless runtime tags through `docker buildx imagetools inspect`. It requires `linux/amd64` and `linux/arm64` in the returned manifest index and prints candidate `image:tag@sha256:...` references.

**Verified registry index evidence — November 16, 2026:** [Private Release Validation run 43971140088](https://github.com/atipparaju/nats-k8s-health-bridge/actions/runs/43971140088), read-only image digest job succeeded for each reference and confirmed both AMD64 and ARM64:

| Source image | Verified registry index digest |
| --- | --- |
| `golang:1.26.11-alpine3.23` | `sha256:23f198931d4bd83b4f5a613bd75656257ae8bf832dcbc2db80e1050b5219ca2a` |
| `nats:2.15.3-alpine3.22` | `sha256:558394a23dd3c36f2d3a64b51865fd09908010e80a542cea5e75c2af543ed62a` |
| `gcr.io/distroless/static-debian12:nonroot` | `sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab` |

PR #79 adds these `tag@sha256` pins to the Dockerfile, NATS release smoke, Kind, CI and local setup documentation. **These pins require a fresh full PR-head validation before merge**; the earlier audit success is not a runtime test of the new pins.

An immutable registry index reference prevents silent tag movement, but does not alone establish a validated signature, provenance, vulnerability-free content, license clearance or native ARM64 runtime correctness. The audit reads public registry manifests without login or publication. A future version/digest rotation must re-resolve multi-platform index digests and rerun the entire CI suite. See [release publishing safeguards](release-publishing.md).

## Evidence rules

Record the exact commit SHA, CI run URL, tool and version, operating system, architecture, test conditions, outcome, and open findings. Do **not** commit tokens, vulnerability reports containing confidential context, proprietary source, developer machine paths, screenshots showing credentials, or unredacted secret-scan output. "Script exists" is not equivalent to "test passed." A GitHub Actions runner is not a Windows Docker Desktop or native ARM64 machine.

## Test matrix

| Gate | Automated evidence | Owner/manual evidence still required |
| --- | --- | --- |
| Linux Docker / JetStream first run | `Private Release Validation → Linux Docker Quickstart`: builds a local image, uses actual memory-backed stream/consumer, verifies /healthz, /readyz, authenticated /status and /metrics, rejects unauthenticated access, cleans up | Verify Linux Docker Engine version and rerun if targeting a specific distribution |
| Windows PowerShell 7 / Docker Desktop | `PowerShell Syntax (not Docker Desktop)`: parses the PowerShell harness on a GitHub-hosted Windows runner | Execute `pwsh -File test/release/docker-quickstart-windows.ps1` on a real Windows machine with Docker Desktop in Linux-container mode and Go installed; record success/failure and versions |
| History secret scan | `Git History Secret Scan`: Gitleaks v8.30.1 container pinned to a SHA256 digest; full fetched branch/tag history, redacted output withheld from logs and artifacts | If scan fails, inspect candidate matches privately, revoke/rotate confirmed exposed credentials before considering rewriting public history, and rerun. Scan external/missing refs and non-Git files separately if relevant |
| Container SBOM | `Local Image SBOM`: Syft/Anchore SPDX JSON artifact of a **locally built** image, privately retained for 7 days; package count check | Review license metadata and dependencies, required notices, third-party attribution, redistribution terms, and source ownership. Automated inventory is not legal clearance |
| amd64 runtime | Linux Docker quickstart uses the native Linux runner | Validate native amd64 targets as needed |
| arm64 runtime | `Linux arm64 Local Runtime Smoke`: cross-built image, inspected architecture, executed `--version` under QEMU | Verify on **native** arm64 hardware and optionally run full NATS integration on arm64; QEMU is not native |
| Digest / manifest | Docker produces a **local** image ID and private CI evidence; no registry push | After an explicitly approved release, record **registry manifest digest** and verify both amd64/arm64 platform descriptors; local image IDs do not prove registry digests |
| Vulnerabilities | Existing required Container Security gate blocks fixable CRITICAL vulnerabilities and reports HIGH/CRITICAL | Review unfixed HIGH/CRITICAL findings, exceptions/expiration and applicable upstream CVEs |
| Provenance / signatures | No signed public artifact exists. Private source controls and CI results are available | Select and implement a signing/attestation mechanism supported by the actual GitHub plan and registry. Verify the signature and subject digest before any production recommendation |
| Licensing | Apache-2.0 `LICENSE` present, canonical appendix restored in #44 | GitHub REST still reported `license.spdx_id: NOASSERTION` on 2026-11-09. Investigate indexing and reconcile copyright ownership/source headers. Inventory derivative work and NOTICE obligations |
| Security and rights | Required CI checks with no administrative CI bypass (`main_rule`); PR-only administrator bypass on review rule (`main`) | Verify GitHub Packages → Manage Actions access, contributor agreements and all employer/other contractual IP constraints; obtain written authorization before release |
| Release approvals | Release workflow remains explicit `workflow_dispatch` on an approved `v*` tag with `publish_image=true`; no publishing from this evidence workflow | Owner must authorize **the exact commit, tag, destination, visibility, content and timing** separately. No permission is inferred from a green CI run |

## Running the new evidence tests

**Automated:** Open the `Private Release Validation` GitHub Actions run on PR #58 (or the current issue #47 PR). Review all five jobs on the **same head commit**. The job `PowerShell Syntax (not Docker Desktop)` checks syntax only. The `Local Image SBOM` job creates a **private workflow artifact**, not a GitHub Release asset.

**Linux host with Docker + Go:** From a fresh checkout, run:

```bash
bash test/release/docker-quickstart-linux.sh
```

**Windows host with Docker Desktop (Linux containers), PowerShell 7 and Go:**

```powershell
pwsh -File .\test\release\docker-quickstart-windows.ps1
```

The scripts provision ephemeral local stream/consumer objects, build `nats-k8s-health-bridge:local`, use temporary bearer tokens and clean up containers/network. They use host loopback ports **14222** and **18080**; ensure these are free. They do **not** publish, tag a release, change GitHub settings, or alter package visibility. The tested local image itself remains until removed explicitly (e.g. `docker image rm nats-k8s-health-bridge:local`).

## Owner-completed evidence record — do not check prematurely

- [ ] PR #47 GitHub Actions **Linux Docker Quickstart** success, run ID and commit recorded:
- [ ] PR #47 GitHub Actions **Windows PowerShell syntax** success, run ID and commit recorded:
- [ ] **Windows Docker Desktop end-to-end smoke**, OS/Docker Desktop/Go/PowerShell versions, date and synthetic-only result:
- [ ] PR #47 **full-history Gitleaks** pass, run ID and scan scope; private findings resolved:
- [ ] PR #47 **SPDX SBOM** generated, artifact reference, dependencies and license/NOTICE review complete:
- [ ] PR #47 **QEMU arm64 runtime** success and native arm64 applicability assessed:
- [ ] Release image manifests, digests and signature/provenance policy approved (not possible to mark verified without a real approved artifact):
- [ ] GitHub SPDX license recognition investigated and owner copyright / contributor / IP provenance signed:
- [ ] GHCR Actions access and private package visibility rechecked:
- [ ] Documentation, supported version policy, security contacts, rollback and compatibility claims reviewed:
- [ ] Explicit, separate release/publication authorization received for exact commit/version and destinations:

**Current decision: NO-GO for publishing.** Issue #47 must remain open until the manual owner attestations and all applicable tests are completed. No automation should make the repository or package public as part of this issue.

## Source references

- [Docker quickstart](docker-quickstart.md)
- [Versioning and provenance](release-versioning.md)
- [Release publication procedure](release-publishing.md)
- [Original readiness audit](release-readiness-checklist.md)
- [GitHub artifact attestation availability](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations) — availability depends on GitHub plan; private repositories on Free/Pro/Team cannot assume attestations
- [Gitleaks CLI](https://github.com/gitleaks/gitleaks), [Syft SBOM tooling](https://github.com/anchore/syft)
