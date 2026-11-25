# Pre-publication GO/NO-GO: source visibility versus artifacts

**Evidence review:** 2026-11-24 · **Tracking:** [Issue #80](https://github.com/atipparaju/nats-k8s-health-bridge/issues/80) · **Repository visibility observed:** private · **Decision:** **NO-GO pending owner approvals**

This is an evidence and approval matrix, not legal advice, a security certification, or permission to publish. It deliberately separates making **source code public**, publishing a **GHCR image**, and announcing a **supported release**. The first action can occur without the others, but every action still requires its own approval.

## 1. Verified current-source observations

| Area | Observed evidence | Limits |
| --- | --- | --- |
| Scope and functionality | `go.mod` depends on `nats.go`, not `jsm.go`; `internal/nats/client.go` uses `jetstream.New`, `Consumer`, and `Info`. `internal/health/checker.go` initializes `LegacyEvaluator{}` | The upstream JSM adapter is a study only; do not advertise it as integrated or imply official NATS/CNCF endorsement |
| Tooling and provenance | Go/NATS test images and Distroless source images are pinned to registry indexes supporting amd64/arm64; verified in merged PR #79, [validation run 43971654081](https://github.com/atipparaju/nats-k8s-health-bridge/actions/runs/43971654081) | Source-image digest is not the final release image manifest, signer identity or release attestation |
| CI and security | Last reviewed PR-head checks include Go tests/race, NATS integration, Kind, Linux Docker, Windows PowerShell syntax, Gitleaks, SPDX SBOM **creation**, QEMU arm64 and container vulnerability scan | Passing CI does not prove full attack-surface, secret-history, native ARM, or dependency-license clearance |
| Host testing | Earlier owner-reported Windows Docker Desktop and native Ubuntu Linux amd64 smoke results are recorded in [release evidence](release-47-evidence.md) | Tests predate later merged dependency/image pins and are not proof of new Windows/native ARM runtime validation |
| Licenses | Root `LICENSE` contains Apache-2.0 terms; source headers name a copyright holder; no separate root `NOTICE` was found | Whether notices are legally required is **undetermined**; ownership and third-party license expressions are not yet independently cleared |
| SBOM | The fresh private [PR #81 validation run](https://github.com/atipparaju/nats-k8s-health-bridge/actions/runs/45495524474) generated SPDX 2.3 artifact ID `14968835381`; its actual 35 package records were examined for license metadata | **29 of 35 packages** report both `licenseDeclared` and `licenseConcluded` as `NOASSERTION`, including Go dependencies and this project. This is missing classification evidence, **not proof of a license violation**. No legal/NOTICE compatibility clearance was completed |
| Registry and governance | GHCR requires separately authorized manual publication; repository remains private | GHCR visibility, current effective rulesets, external report access and attestation support need owner verification |
| Public metadata | `CODE_OF_CONDUCT.md` includes an external reporting email; `README.md` and `SECURITY.md` explicitly describe a private/unreleased project | The contact's permission for public exposure and public-facing descriptions must be reviewed before changing visibility |

**Current open maintenance issues #67 and #72 are not inherently release blockers.** The final publication decision depends on unresolved approvals and findings below, not issue count. Issue #72 can be reconciled administratively after its merged Prometheus PR, but neither that closure nor dependency version currency establishes IP clearance.

### SPDX license-field triage — 2026-11-24

Downloaded and inspected `private-candidate-sbom.spdx.json` (artifact **14968835381**, CI run **45495524474**). It is an image inventory with **35 package entries** and **SPDX 2.3** metadata:

- **29 packages** have `NOASSERTION` for both declared and concluded license, including the compiled Go dependency modules and the project itself. Do not interpret these fields as evidence that they are unlicensed; inspect original package license files and metadata.
- Other package records include `BSD-3-Clause` (Go standard library), `GPL-2.0-only` (`netbase`), a composite GPL/MPL expression (`ca-certificates`), and `LicenseRef-*` values (`base-files`, `media-types`, `tzdata`). These are **scanner observations, not a conclusion** about license compatibility or distribution obligations.
- A stand-alone root `NOTICE` file was not found. Whether a NOTICE and redistributable license texts are required for a particular distributed artifact must be resolved after examining component licensing and packaging.
- The artifact inventory covers the **built image**, not every source, external ref, repository-history artifact, or source-dependency notice obligation. Public Git history and copyright/rights review remain distinct.
- Keep raw SBOM and any sensitive scanner output out of the public repository unless separately reviewed for disclosure; workflow artifacts have limited retention.

**Disposition: license clearance remains BLOCKED / UNVERIFIED** until each unknown record is resolved with independent upstream license evidence, attribution/notice requirements are reviewed, and the appropriate rights holder signs off.

### Preliminary upstream license-source cross-check (2026-11-24)

The 29 `NOASSERTION` SPDX results do **not** mean 29 packages have unlicensed code. A separate *repository-level* inspection found recognizable license texts for several corresponding upstream projects:

| Upstream group / examples | Observed upstream license texts | Verification boundary |
| --- | --- | --- |
| NATS (`nats.go`, `nkeys`, `nuid`) and Prometheus (`client_golang`, `client_model`, `common`, `procfs`) | Apache License 2.0 | Upstream root LICENSE checked; distribution of individual releases and any notices still needs evaluation |
| Cobra, Afero | Apache License 2.0 | Cobra/Afero use `LICENSE.txt`; no assumption that a root `LICENSE` path exists |
| Viper, mapstructure/v2, go-toml, locafero, cast, Sourcegraph conc, gotenv, perks, xxhash | MIT-style license texts | Preserve actual upstream copyright and notice text as required by each license |
| fsnotify, pflag, `golang.org/x/crypto`, `x/sys`, `x/text`, protobuf-go, goautoneg, klauspost/compress | BSD-style redistribution terms, with component-specific headers | Verify exact vendored package scope and applicable license text |
| Go YAML source project | Contains both MIT and Apache license sections | File-level or dual-license implications require specific review |
| Prometheus client_golang NOTICE | **Upstream NOTICE exists** and references additional attribution | Review applicability to the intended binary/source artifact before distribution |

**Exact-tag verification:** The root LICENSE files were checked at the currently pinned versions for `nats.go v1.55.0`, `nkeys v0.4.16`, `nuid v1.0.1`, `client_golang v1.25.0`, `client_model v0.6.3`, `common v0.72.0`, `procfs v0.22.0`, Cobra v1.10.1, Viper v1.21.0, and compress v1.20.1. The four Prometheus modules (`client_golang`, `client_model`, `common`, `procfs`) each also contain a version-tagged upstream `NOTICE`, including third-party SoundCloud attribution. Those are concrete redistribution-notice review inputs; **do not assume a root project NOTICE is automatically sufficient or automatically required without examining the release contents.**

These checks cover ten exact tagged dependencies, **not** the complete version-specific bill of licenses or file-level obligations. Dependencies and OS components in the image include scanner-reported GPL/MPL and LicenseRef values that require separate analysis if distributing an image. The project's missing root `NOTICE` should **not** be filled with invented attribution. Obtain actual required texts and permission/compatibility decisions first.

**Next license clearance action:** independently confirm **the exact tagged dependency versions**, their retained LICENSE/NOTICE files and build contents; then produce a source-only versus binary/image obligation matrix. Record exceptions/uncertainties privately and do not promote the results to legal certification.

### Detailed review artifacts (November 24)

- [Exact-version Go dependency license evidence](dependency-license-review.md): verified upstream license texts for all **28 selected Go modules** (including pseudo-version source commits), identified the four selected Prometheus NOTICE files, and distinguished source-only from binary and container redistribution. This does **not** resolve the image SBOM's OS-package and file-level notice requirements.
- [Private history and metadata review runbook](public-history-review-runbook.md): documents how to inspect fetched refs, older content and commit metadata without publishing matches. The complete owner history review is **not yet attested**.
- [Issue #84](https://github.com/atipparaju/nats-k8s-health-bridge/issues/84) is **closed** after merged PR #85. The expanded redacted private Gitleaks scan fetched **54 retained PR-head refs** and reported zero candidates on [validated run 45548963680](https://github.com/atipparaju/nats-k8s-health-bridge/actions/runs/45548963680). **Gate B remains incomplete** until the owner reviews author/contact metadata, historic Actions logs and GitHub-side attachments and confirms public-disclosure scope.
- Owner Option B (agreements understood to allow publication without separate approval) remains an attested interpretation, not independent legal clearance. This does not change the visibility NO-GO.

### Public-source strategy and metadata exposure (2026-11-25)

The [public-source strategy decision](public-source-strategy.md) distinguishes **Route 1: publish the existing repository with history** from **Route 2: publish a curated, separately validated source snapshot** while retaining the development repository privately. Neither has been selected or authorized.

GitHub's documented private-to-public visibility change exposes historical Actions logs and disables push rulesets. The owner must also decide whether historical Git author email metadata and the direct Code of Conduct contact address are acceptable disclosures. Changing the Git commit email for future commits does not remove email metadata from older commits. Do not merge publication wording or alter existing Git history ahead of that decision.

## 2. Must resolve before making the source repository PUBLIC

| Gate | Evidence required for approval | State |
| --- | --- | --- |
| A — Rights and provenance | Owner selected **B: agreements permit publication without a separate approval** (2026-11-24). This is owner-attested interpretation, **not independently reviewed legal clearance**. Confirm applicability to every committed contribution, employer/inventions provisions, rightsholder identity and historic attribution; document the rationale or obtain qualified review if uncertain | **OWNER-ATTESTED — verification pending** |
| B — Public-safe contents and Git history | Review the exact commit, all refs and entire history intended to be exposed (including earlier deleted content), sample configuration, Helm examples, test fixtures, workflow logs, commit metadata and documentation for credentials, personal data, confidential company/customer references and incompatible borrowed materials. Document the scope and results without putting secrets in this file | **PARTIAL — BLOCKER** |
| C — Dependencies and redistribution | Obtain fresh accessible SPDX/SBOM including source and image dependencies. Investigate unknown/NOASSERTION licenses, copyleft/NOTICE/attribution obligations and bundled license distribution; document review and remediation. `NOTICE` absence by itself proves neither compliance nor noncompliance | **UNVERIFIED — BLOCKER** |
| D — Correct license and author claims | Reconcile copyright ownership, source headers, root Apache-2.0 grant, GitHub license identification and independent project claims; do not misrepresent upstream monitoring algorithms or support maturity | **PARTIAL — BLOCKER** |
| E — Public contact/security disclosure | Confirm intended public email/contact, GitHub security advisories accessibility for external reporters, disclosure instructions, support policy and exact claims about current private/unreleased state | **PARTIAL — BLOCKER** |
| F — Publication decision | Review and explicitly approve **the exact source SHA**, Git history scope, visibility switch and timing. Separately confirm the GHCR package remains private if no image release is approved | **NOT AUTHORIZED — BLOCKER** |

## 3. Before an IMAGE or SUPPORTED VERSION release (separate approvals)

- [ ] Confirm GHCR package's current visibility and GitHub Actions access in package settings. Private repository visibility does not establish package visibility.
- [ ] Determine and verify image signature, attestation and build provenance for the **actual published immutable multi-platform manifest digest**. Source-image index pins and local image IDs are insufficient.
- [ ] Approve supported NATS/Kubernetes/Go/OS versions, severity handling, security update policy, changelog and version/tag.
- [ ] Review unfixed HIGH/CRITICAL findings: the current Trivy job uses `ignore-unfixed: true` and blocks only fixable CRITICAL. A green gate cannot be presented as an absence of serious vulnerabilities.
- [ ] Explicitly approve registry destination, version/tag, public/private package visibility and image publishing **separately from source visibility**.

The optional `jsm.go` evaluator, migration to Go 1.27 / Helm 4, and the Docker build-push-action major upgrade are **not automatic blockers** to a clearly labeled source preview. Critical vulnerability or API compatibility issues found during release review can override that general assessment.

## 4. Operational checklist (owner/compliance sign-off)

- [ ] **Rights:** Owner selected Option B. Privately document why the relevant employment/invention terms permit this *specific* code and project, whether external contributors or reusable work product are covered, and whether any exceptions apply. No separate permission is claimed. Keep contract excerpts outside the repository.
- [ ] **History:** Run full scanner over all intended public refs, inspect scoped exceptions, and manually review tree/history for company or person-identifying material. Document pass/fail only.
- [ ] **Licenses:** Run new CI SBOM (private, short retention), inspect the actual JSON, third-party notices and source reuse; record unknowns and actions.
- [ ] **Metadata:** Confirm public contact address, owner identity, copyright, contribution policy, support/security policy, and README's NATS/JSM statements.
- [ ] **Settings:** Re-read effective GitHub rulesets, required checks, role bypass and package/Actions access rather than relying on older snapshots.
- [ ] **Final:** Record exact authorized SHA and date of source-public decision. Never infer authorization from merged PRs or green CI.

### Decision log

| Field | Current decision |
| --- | --- |
| Source repository visibility | **NO-GO; private** |
| GHCR image publishing | **NO-GO; separately gated** |
| Supported first release | **NO-GO; tag/version/signature policy not approved** |
| JSM integration required before source preview | **No**; current `LegacyEvaluator` behavior is documented, and JSM integration is a future compatibility experiment |
| Owner/legal attestation | **Option B selected: agreements understood to allow publication; no separate permission obtained or independently verified** |

This document is safe to keep in a private PR because it contains only generic outcomes, project filenames and publicly attributable references. Do **not** commit confidential employer agreements, personal identity documents, secret scanner matches, dependency scans with sensitive infrastructure metadata, or private consultation transcripts.
