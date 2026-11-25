# Private Git-history and publication-metadata review

**Status:** RUNBOOK ONLY — review has **not** been completed against every historic object. Tracking: [Issue #80](https://github.com/atipparaju/nats-k8s-health-bridge/issues/80).

This review is the remaining human security/privacy and intellectual-property gate for making the **existing private repository and its history** public. GitHub code search and green Gitleaks checks provide useful but incomplete coverage. A historical artifact may be exposed even after a file is removed from `main`.

## What is already established

- The CI `Git History Secret Scan` uses a full fetch of branch/tag refs and the Gitleaks scanner in a no-network, read-only container; it has passed on previously validated heads. Three exact fingerprint exceptions are documented in `.gitleaksignore` for synthetic test-only bearer-header text. These should still be inspected, not globally disabled.
- Spot searches of the current indexed repository for confidential company names were inconclusive as a **full-history** check; no current-source hit cannot prove an absence in older commits, removed files, PRs, tags or unreachable objects.
- Current public-facing materials contain author/copyright metadata and a contact email in `CODE_OF_CONDUCT.md`. On November 25, the owner approved disclosure of their own personal commit-author email and the existing Code of Conduct contact address. This **does not** approve employer identifiers or other contributors' private details.
- The root license and configuration snippets are present, but source provenance is not attested by an automated scan.
- Do not post a personal email address, personal commit metadata, organization-specific internal identifiers, credentials or scanner matches in an issue, PR comment or public file to demonstrate review.

## GitHub branch preflight — November 24, 2026

The connected repository currently exposes **five** branch refs. A comparison against `main` found:

| Branch | Ahead of main | Behind main | Review significance |
| --- | ---: | ---: | --- |
| `main` | — | — | Current release candidate |
| `copilot/nats-k8s-health-bridge` | 0 | 44 | No unique commits relative to `main`; historical metadata still visible |
| `feature/oss-hygiene` | 4 | 45 | Unique contributor/governance docs and README history |
| `fix/24-retry-permanent-errors` | 6 | 28 | Unique NATS client and integration-test history |
| `fix/45-guard-release-publication` | 10 | 16 | Unique historical CI, scripts, Makefile and publication-safety documentation |

These ahead/behind values are **snapshots**, not an exhaustive content or credentials audit. The `fix/45-guard-release-publication` branch's current CI file was checked for the image-publication guard; its visible publish job still requires manual dispatch, `publish_image=true`, and a `v*` tag. That **does not** validate all earlier workflow versions or exclude other publication mechanisms.

**Do not remove old branches solely to obtain a green status.** First check whether they hold unique information that must be preserved elsewhere, inspect their full histories and PR attachments for sensitivity, and document owner-approved retention or deletion. Deleting a branch does not necessarily remove GitHub's retained historical objects or old PR-linked commits. The complete review of these branches and GitHub-side metadata has not been attested.

## Retained GitHub pull-request history — coverage gap and remediation

The November 24 remote audit enumerated **53 retained `refs/pull/*/head` references**, of which **50 head SHA values were absent from the 76 commits enumerated across the five named branches**. These PR refs may contain unique commits not covered by a branch/tag-only clone. They do not by themselves prove a secret exposure.

[Issue #84](https://github.com/atipparaju/nats-k8s-health-bridge/issues/84) was closed after merged PR #85 added an explicit private `git fetch` of those retained PR heads to Gitleaks `--all`. The [successful scan run 45548963680](https://github.com/atipparaju/nats-k8s-health-bridge/actions/runs/45548963680) fetched **54 PR-head refs** and reported zero candidates. This establishes improved **reachable-ref secret-scanner coverage**, **not** complete IP, privacy, author-email, Actions-log, attachment, or source-provenance clearance. The job must fail closed if the PR refs cannot be fetched; no secret matches or raw commit metadata should be posted in CI logs or PR comments.

### Remote content preflight (not a complete all-object audit)

- Compared 5 named branches, enumerated the 76 distinct commits reachable from their tips, and inspected the changed-file patches of those 76 commits for a limited set of high-risk strings and file types. This heuristic pass did not prove that historical commits are entirely free of secrets or proprietary content.
- The earlier GitHub search and remote patch preflight identified ordinary public-contact fields and synthetic credential values used by security regression tests; those should be assessed for public disclosure rather than assumed safe solely from their location.
- The API listed 82 GitHub issue/PR discussion records and 37 issue comments at that snapshot. Automated text-screening found no URL pattern matching uploaded GitHub attachments in those particular bodies; it is **not proof** that PR refs, GitHub assets, screenshots, unreachable objects or other provider surfaces have no sensitive attachments.
- The root repository's license metadata still reported `NOASSERTION` despite the Apache-2.0 text in `LICENSE`; interpret this as a recognition/provenance discrepancy, not proof the project lacks a license.
- Do not discard or rewrite branches and PR refs as a substitute for inspection. Retained review links, discussion metadata and reachable Git objects need owner approval before source visibility changes.

## Route 1 historic Actions logs — private employer-identification check

The owner selected Route 1 (preserve and eventually publish the **existing** repository) for preparation only. Before any public visibility change, follow [the local-only Actions-log audit procedure](actions-log-disclosure-audit.md), tracked in [Issue #87](https://github.com/atipparaju/nats-k8s-health-bridge/issues/87). This review must cover log contents, not just run titles. The owner-approved personal and maintainer contact emails are not the reason to block publication; **employer-identifying material is explicitly disallowed**.

The script requests a private runtime-only search expression and prints counts/IDs, never matching values. Unavailable logs remain unresolved, not automatically cleared. Source privacy, branch and pull-request refs, comments, attachments, employer IP provenance and licensing still require independent review.

## Owner-run inspection on a trusted machine

Run from a fresh private clone, using a scratch directory **outside the worktree** with restrictive permissions. Substitute your **own locally defined** sensitive names/domains/identifiers; do not commit that pattern file or record raw matches in GitHub.

```bash
# Only on a trusted workstation with approved repository access.
umask 077
REVIEW_DIR="$(mktemp -d "${TMPDIR:-/tmp}/nkhb-public-review.XXXXXXXX")"
chmod 700 "$REVIEW_DIR"
git fetch --all --tags --prune
git fetch origin '+refs/heads/*:refs/remotes/origin/*'

# Review all names of fetched refs, reachable commit metadata and historical paths.
git for-each-ref --format='%(refname)' > "$REVIEW_DIR/refs.txt"
git log --all --format='%H%x09%an%x09%ae%x09%cn%x09%ce%x09%s' \
  > "$REVIEW_DIR/commit-metadata.txt"
git rev-list --objects --all > "$REVIEW_DIR/reachable-objects-and-paths.txt"
git log --all --name-status --format='%H' \
  > "$REVIEW_DIR/historical-files.txt"

# Search **historical content**, not only the current worktree.
# Keep the locally chosen pattern private and do not paste match lines.
read -r -s -p 'Private review keywords (regex): ' REVIEW_TERMS; echo
git rev-list --all | while read -r commit; do
  git grep -I -n -i -E "$REVIEW_TERMS" "$commit" -- . || true
done > "$REVIEW_DIR/historical-keyword-matches.txt"
unset REVIEW_TERMS
```

Review **every finding**, especially historic configuration, sample credentials, URLs, internal organization references, copied code, binary/large assets, git submodules, CI-generated files, and commit author addresses. The output of `historical-keyword-matches.txt` may contain sensitive content; do not upload it or copy your private scratch directory into the repository. When finished, securely dispose of the working copies using your organization's applicable data-retention process.

### Separate from these local commands

- [ ] Review GitHub **all refs intended to be exposed**: main/branches, tags, PR head refs, old PR comments and attached uploads, GitHub releases/assets, wiki/pages, and connected package surfaces. Some content is not reachable via an ordinary `git fetch --all`.
- [ ] Check whether removed/dangling commit objects and historical PR refs may still be reachable via retained GitHub URLs. If they expose sensitive material, resolve remediation with appropriate maintainers before going public; simply deleting a present-day file is inadequate.
- [ ] Inspect all tracked assets for image-embedded/private information and third-party intellectual property, not only text matches.
- [ ] Verify exact narrow Gitleaks exceptions remain synthetic; rotate/revoke any real exposed credential before considering history rewriting.
- [ ] Confirm author/contributor identity and email disclosure are acceptable for all commits and public documents.
- [ ] Review the proposed source-only public visibility scope separately from GHCR package permissions, signed releases and external security-reporting access.
- [ ] Record pass/fail and the exact reviewed source SHA, refs and review date in the private owner decision ledger; keep sensitive findings **outside** GitHub.

## Clearance criteria

**Green CI alone is not enough.** Declare this gate complete only after:
1. All intended public Git refs, source content, metadata, third-party source and GitHub-side attachments have been inspected.
2. Potential credentials or confidential IP have been investigated and remediated, with no unresolved high-risk finding.
3. Owner explicitly accepts disclosure of contributor/commit metadata, public contact details and the exact source tree.
4. Evidence is dated and associated with a specific immutable commit and visibility decision.

**Current decision: PENDING — no review attestation or public-visibility authorization has been provided.**
