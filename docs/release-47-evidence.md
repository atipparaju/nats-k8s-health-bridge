# Issue #47 — Private pre-publication evidence and owner decision record

**Updated:** 2026-11-09 · **Repository status:** private · **GHCR status:** owner previously confirmed Private; recheck before publication.  
**Decision:** **NOT APPROVED TO PUBLISH.** This file is a plan and an evidence ledger, not a signed certification.

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
