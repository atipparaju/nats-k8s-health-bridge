# Source publication strategy and metadata-disclosure decision

**Checkpoint:** 2026-11-25 · **Parent:** [Issue #80](https://github.com/atipparaju/nats-k8s-health-bridge/issues/80) · **Visibility:** PRIVATE · **Decision:** NO-GO until explicit owner approval

This is a **private, non-authorizing decision record**. Do not use it to change visibility, create a public repository, publish a package, rewrite Git history, or change publication credentials. Retain the existing [GO/NO-GO matrix](pre-publication-go-no-go.md), [history review procedure](public-history-review-runbook.md), and [dependency license evidence](dependency-license-review.md).

## Verified exposure and current limitations

| Surface | Observed state | Risk/next approval |
| --- | --- | --- |
| Commit author metadata | Earlier audit of **76 branch-reachable distinct commits** found **75 records with non-GitHub-noreply author email fields** (two distinct addresses). Counts are a dated snapshot and do not include all PR-only Git history | Owner must decide if public disclosure is acceptable. Do not add raw emails to docs, issues, or CI logs |
| Maintainer contact | `CODE_OF_CONDUCT.md` publishes **one direct email address** | Owner accepts that contact's disclosure; do not infer broader publication approval |
| Historical refs | The repository has five named branches and older divergent histories. Private Gitleaks validation in [run 45548963680](https://github.com/atipparaju/nats-k8s-health-bridge/actions/runs/45548963680) fetched **54 retained PR-head refs** and reported **zero candidates** | Secret scanning is improved, but IP, author metadata, workflow logs and GitHub-side attachments remain independent disclosure checks |
| GitHub Actions history | On November 25, GitHub REST listed **499 workflow runs** in the private repo (a changing count; audit scripts must enumerate again at run time). GitHub states that **Actions history and logs become visible to everyone** when a private repo becomes public | The historic run/attempt log content has **not** all been reviewed. Review them (and retained artifacts) before any visibility change. GitHub also warns that **push rulesets are disabled** on private-to-public conversion; re-establish suitable protection afterwards |
| Repository license detection | Root `LICENSE` includes Apache-2.0 terms, but GitHub REST previously reported `license.spdx_id: NOASSERTION` | Investigate recognition and rights; do not confuse machine detection with legal ownership |
| Source descriptions | `README.md` and `SECURITY.md` explicitly identify a private, unreleased project | Do not silently change these until a particular source publication strategy is authorized |
| Release packaging | Go module is `github.com/atipparaju/nats-k8s-health-bridge`; its import path is embedded throughout the source and tests | Changing repository name/owner requires module path, imports, documentation and CI verification |
| Rights/licensing | Owner selected Option B: believes applicable agreements permit publication without separate approval. 28 pinned Go-module license-file sources identified; redistribution notices and owner-specific provenance not fully cleared | Owner-attested rights interpretation is **not** independent clearance. Both publication options still require the same source/IP/notice and data-sensitivity review |

Official GitHub references:
- [Changing repository visibility](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/managing-repository-settings/setting-repository-visibility)
- [Setting commit author email](https://docs.github.com/en/account-and-profile/how-tos/email-preferences/setting-your-commit-email-address)
- [Removing sensitive data from repository history](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/removing-sensitive-data-from-a-repository)

## Publication choice (owner approval required)

| | **Route 1: make existing repository public** | **Route 2: curated clean-history public repository** |
| --- | --- | --- |
| Public code | Current reviewed source and all public-facing repository surfaces | Only a reviewed source snapshot in a **separate** newly created repository |
| Development history | Original commits, branch/PR history, issues, author metadata, Actions logs become public | Private development repository and its existing Git/PR/Actions history remain private; new public repository begins a separate history |
| Attribution/provenance | Preserves visible development timeline and review provenance | Preserves original private audit evidence, but **does not** automatically publicly demonstrate older historical development |
| Privacy | More risk from earlier content, author emails, contact, legacy logs and PRs | Smaller disclosure surface if source, assets and release documents are explicitly curated; new public project still requires rights and license checks |
| Technical effort | Release-readiness, archive/history review, public metadata updates, ruleset verification | Curated file inventory, correct Go module/import path for the new repository, CI/build/Kind tests, updated documentation/links, fresh LICENSE and valid attribution; package identifiers and GHCR remain separate |
| Existing repository | Becomes public only with explicit owner authorization | Stays private; **never fork or mirror it with complete history to create the snapshot** |
| When to choose | Owner wants original public commit timeline **and accepts full historical disclosure** after clearance | Owner prioritizes publishing a working source preview **without exposing private Git/Actions history** |

**Owner selection (November 25, 2026): Route 1 — prepare the existing repository for public visibility, with its Git history preserved.** Route 2 remains an unused alternative. This choice authorizes **preparation only**, not publication or a GO decision. The owner permits exposure of their personal commit-author email and the existing Code of Conduct contact email, but **does not authorize exposure of employer-identifying information**. See [private historical Actions-log audit](actions-log-disclosure-audit.md).

### Route 2 engineering constraints

- Confirm intended public repository owner/name **before** changing the module path. The current `go.mod` and internal imports use the existing private repository path; using a new public name without updating these creates broken imports and unattributable external fetch behavior.
- Prepare a verified allowlist of code, sample configuration, Helm templates, tests, CI and public-facing docs. **Do not** copy `.git`, old workflow logs/artifacts, internal release evidence or confidential review files.
- Reconcile README, SECURITY, CONTRIBUTING, Code of Conduct reporting contact, versioning, package installation instructions, NOTICE obligations, attribution and JSM/nonendorsement statements with the chosen public scope.
- Test actual build, Go race/integration, Kind, source-only quickstart, secret history on the **new** repository, and Linux/arm64 image behavior when applicable. These are future gates, not completed evidence here.
- Keep the original private repo for existing audit records and development; do not imply that copying its source transfers intellectual-property rights.

### Route 1 engineering/security constraints

- Accept or remediate author email and contact disclosure; changing Git email settings affects **future commits only**, not prior history.
- Review all reachable branches, PR refs, issue comments, screenshots/attachments, and historical Actions logs. The expanded Gitleaks scan is not a review of those non-Git surfaces.
- Do **not** rewrite history casually. Existing PR references and caches can retain older Git objects, while rewrites change commit IDs and can damage review/audit provenance.
- Plan for GitHub's documented visibility-change effects (Actions logs exposed and push rulesets disabled). Verify branch protections, secret access and GHCR package visibility **after any future, separately approved** change.
- Decide whether the historic changelog, development README and Security policy are suitable for public users; record exact approved commit SHA and policy.

## Owner decision gates (all pending)

- [x] **Route 1 selected for preparation only**; existing repository and history will be retained. The public visibility decision is still separate and pending.
- [x] Owner has accepted their own commit-author and existing Code of Conduct contact email disclosure. Other contributors' rights, if any, remain separate.
- [ ] Record rights review rationale for this exact code; privately verify any applicable employment/IP exceptions and third-party file provenance.
- [ ] Finish source-dependency attribution, NOTICE and public metadata checks.
- [ ] If A: review prior GitHub Actions logs/PR attachments and approve existing repository history. If B: approve precise destination, selected file inventory and module namespace migration.
- [ ] Approve exact source SHA and timing for source publication, separately from GHCR image publication or any signed release.

**Decision remains NO-GO.** No public repository, release, tag, GHCR push, history rewrite, public contact change or visibility change has been authorized.
