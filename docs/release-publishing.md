# Release publication safety

The repository is still in **private pre-publication review**. No release, GHCR push, visibility change, or publication is authorized by this document.

## Local development cannot publish

Use the following from the repository root:

```sh
make docker-build       # builds nats-k8s-health-bridge:local only
make release-dry-run    # offline release guardrail verification; no Docker needed
```

The old `setup.sh` and `make docker-push` commands have been removed. `make docker-build` never tags a GHCR reference, never logs in to a registry, and never pushes an image. Docker is required only for the **build**, not for the dry-run.

The dry-run uses Python 3 standard library and checks that:
- the obsolete Git bootstrap/push script and Makefile push target are absent;
- local builds use only a local tag;
- `.github/workflows/ci.yml` only permits image publication from an explicitly opted-in `workflow_dispatch` on a `v*` Git tag;
- the publish job retains scoped `packages: write` permission;
- release metadata uses a full semantic-version tag and a commit-specific `sha-` tag, not `:latest`, a branch tag, or a floating major/minor tag;
- CI runs the same guardrail check to detect policy drift.

**This is a static/offline safety test, not an execution of GitHub's authorization system or a proof that every possible contributor or credential cannot push packages.** Review GitHub settings and Actions permissions separately.

The historical chart version is not an application release; see [release versioning and provenance](release-versioning.md).

## Owner-approved release procedure (future only)

Before taking any publication action:

1. Resolve release-blocking issues and record explicit owner approval for the **specific** version, contents, image and destination.
2. Review the release commit on `main`, required CI, Container Security, Kubernetes Smoke Test, licensing, secrets, dependency risk, image provenance and image-digest review.
3. Confirm the GitHub Container Registry package's **actual current visibility** in GitHub Packages and check **Manage Actions access**. The owner verified that the existing package was **Private** during Issue #42; future publication must recheck, because visibility and permissions may change independently of repository visibility.
4. Create or select a reviewed, protected `vX.Y.Z` Git tag that points to the approved commit. Avoid reusing/moving published tags.
5. Only after explicit approval, go to **GitHub Actions → CI → Run workflow**, choose the approved `v*` tag reference, and explicitly set `publish_image=true`. Normal pushes and tag creation do not publish an image.
6. Verify the resulting package's visibility, exact full-version tag, `sha-` tag, immutable manifest digest, amd64/arm64 contents, scanner results and provenance. Document the release in the changelog.

The GitHub Actions job is named **Publish approved release image** and is intentionally *not* a required pull-request status check because it is skipped outside manually approved release dispatches. Only that job receives `packages: write` for a release.

The workflow no longer generates `:latest`, branch, or floating major/minor aliases by default. Semver tags are still mutable in a registry unless registry permissions and Git tag protection are enforced; record the immutable image digest for deployments.

## Private validation and provenance decision

The [Issue #47 evidence matrix](release-47-evidence.md) records what automated checks can prove before a release and what remains owner-only (Windows Docker Desktop, native arm64, licensing/IP, registry digest/attestation, and GHCR permissions). The private `Private Release Validation` workflow creates local images and a private SPDX inventory artifact but **never publishes**. It is not an authorization to run this manual release pipeline.

## Remaining separate release work

- [Issue #43](https://github.com/atipparaju/nats-k8s-health-bridge/issues/43): replace `:latest` in Helm/README deployment examples and wire mounted Secrets.
- [Issue #44](https://github.com/atipparaju/nats-k8s-health-bridge/issues/44): reconcile release version, tag and changelog history.
- [Issue #47](https://github.com/atipparaju/nats-k8s-health-bridge/issues/47): verify GHCR Actions access, actual cross-platform/architecture behavior, SBOM, provenance, and publication/IP approvals.

No user-specific workstation paths, credentials or private infrastructure settings belong in release documentation.
