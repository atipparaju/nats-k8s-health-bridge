# Private Git-history and publication-metadata review

**Status:** RUNBOOK ONLY — review has **not** been completed against every historic object. Tracking: [Issue #80](https://github.com/atipparaju/nats-k8s-health-bridge/issues/80).

This review is the remaining human security/privacy and intellectual-property gate for making the **existing private repository and its history** public. GitHub code search and green Gitleaks checks provide useful but incomplete coverage. A historical artifact may be exposed even after a file is removed from `main`.

## What is already established

- The CI `Git History Secret Scan` uses a full fetch of branch/tag refs and the Gitleaks scanner in a no-network, read-only container; it has passed on previously validated heads. Three exact fingerprint exceptions are documented in `.gitleaksignore` for synthetic test-only bearer-header text. These should still be inspected, not globally disabled.
- Spot searches of the current indexed repository for confidential company names were inconclusive as a **full-history** check; no current-source hit cannot prove an absence in older commits, removed files, PRs, tags or unreachable objects.
- Current public-facing materials contain author/copyright metadata and a contact email in `CODE_OF_CONDUCT.md`. The owner must approve the disclosure of those details.
- The root license and configuration snippets are present, but source provenance is not attested by an automated scan.
- Do not post a personal email address, personal commit metadata, organization-specific internal identifiers, credentials or scanner matches in an issue, PR comment or public file to demonstrate review.

## Owner-run inspection on a trusted machine

Run from a fresh private clone, using a scratch directory with restrictive permissions. Substitute your **own locally defined** sensitive names/domains/identifiers; do not commit that pattern file or record raw matches in GitHub.

```bash
# Only on a trusted workstation with approved repository access.
umask 077
mkdir -p ./private-review
git fetch --all --tags --prune
git fetch origin '+refs/heads/*:refs/remotes/origin/*'

# Review all names of fetched refs, reachable commit metadata and historical paths.
git for-each-ref --format='%(refname)' > ./private-review/refs.txt
git log --all --format='%H%x09%an%x09%ae%x09%cn%x09%ce%x09%s' \
  > ./private-review/commit-metadata.txt
git rev-list --objects --all > ./private-review/reachable-objects-and-paths.txt
git log --all --name-status --format='%H' \
  > ./private-review/historical-files.txt

# Search **historical content**, not only the current worktree.
# Keep the locally chosen pattern private and do not paste match lines.
read -r -s -p 'Private review keywords (regex): ' REVIEW_TERMS; echo
git rev-list --all | while read -r commit; do
  git grep -I -n -i -E "$REVIEW_TERMS" "$commit" -- . || true
done > ./private-review/historical-keyword-matches.txt
unset REVIEW_TERMS
```

Review **every finding**, especially historic configuration, sample credentials, URLs, internal organization references, copied code, binary/large assets, git submodules, CI-generated files, and commit author addresses. The output of `historical-keyword-matches.txt` may contain sensitive content; do not upload it or commit `private-review/`. When finished, securely dispose of the working copies using your organization's applicable data-retention process.

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
