# Release status, provenance and compatibility

**Verified snapshot, November 4, 2026:** No GitHub Releases or Git tag refs exist; repository metadata reported license `Other` / SPDX `NOASSERTION`. Reverify these facts before a future release.

## Status and version policy

The original changelog contained a dated `0.1.0` entry (2026-04-12) but no corresponding `v0.1.0` tag or published GitHub Release. This is preserved as an *internal development baseline*; entries remain historical notes, not proof of shipped behavior on that date. No retrospective release or nonexistent compare/tag link is created.

Helm `Chart.yaml` `version: 0.1.0` remains an internal **chart packaging** version; `appVersion: "dev"` states that the application is unreleased. A future release must deliberately coordinate the chart and application versions and cut notes only for a real, owner-approved tag.

`Makefile` computes version from `git describe --tags --always --dirty`; the Docker build injects a Git description into `main.version`. Without tags, this is a development commit marker rather than a released semver. Verify `--version`, source SHA, OCI image digest, and release notes against the same immutable commit before approval.

## SPDX / license provenance

The repository contains Apache License 2.0 terms. Its appendix was personalized with `Copyright 2026 Ati Tipparaju`, whereas Go source files contain `Copyright 2026 Anantha Tipparaju`. The license appendix now uses the **canonical Apache-2.0 template placeholders** `Copyright [yyyy] [name of copyright owner]` to improve GitHub license detection. Original attribution persists in Git history; existing source headers have not been rewritten. These differing attributions require owner verification, not an invented transfer of ownership.

**After merging:** Check the repository GitHub API `license.spdx_id` again; `Apache-2.0` is expected, but this is **not yet verified**. If `NOASSERTION` persists, compare the LICENSE to the upstream canonical text and investigate. Conduct rights, employment/IP, third-party code, dependency license and any NOTICE obligations review before distribution. Do not claim legal clearance based on an automated detector.

## Compatibility / release evidence matrix

| Target | Observed evidence | Remaining owner release gate |
| --- | --- | --- |
| Go | `go.mod` requires Go 1.25.0; CI uses Go 1.26.11 | Exact toolchain, dependency hashes and repeatable build |
| NATS JetStream | Integration Test runs `nats:2.15.3-alpine3.22` | Supported server version range and auth/TLS tests |
| Helm / Kubernetes | Helm 3.22.1 chart lint/template tests; Kind smoke | Record Kubernetes/Kind version and production workload compatibility |
| Linux amd64 | Container CI and multi-platform build configuration | Runtime smoke, digest and provenance |
| Linux arm64 | Buildx target declared | Independent ARM64 runtime smoke and digest |
| Windows PowerShell 7+ | Docker quickstart documented | Recorded first-run test |
| Linux Bash / Docker | Docker quickstart documented | Recorded first-run test |
| GHCR image | Manual `v*` tag dispatch with explicit opt-in | Approvals, actual GHCR permissions/privacy, SBOM/attestation (#47) |

Passing repository CI does not certify production compatibility.

## Owner sign-off (NOT APPROVED)

- [ ] Confirm first release version, changelog accuracy and security support window.
- [ ] Confirm rights holder(s), copyright/NOTICE and third-party provenance.
- [ ] Collect OS/CPU/Kubernetes/NATS compatibility evidence and image digests.
- [ ] Verify SBOM, package visibility, Actions access and signing/attestation (#47).
- [ ] Explicitly approve the exact immutable commit, versioned tag and release content.
- [ ] Separately approve registry image publication and repository visibility.

**No release, tag, package publication or visibility change is authorized.**
