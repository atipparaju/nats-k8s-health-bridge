# Default-branch CI enforcement and GHCR publication access

**Snapshot:** November 4, 2026 · [Issue #42](https://github.com/atipparaju/nats-k8s-health-bridge/issues/42)  
**Scope:** Solo-maintainer repository governance, required PR checks, branch bypass, and independent container-package visibility.

This is an **action plan**, not a claim that repository protection has been changed. The GitHub connector can inspect rulesets but does not expose an administrator action for editing rulesets or GitHub Container Registry package settings. The owner must confirm the settings directly. Do not publish an image or change visibility as part of this work.

## Verified repository state

- The default branch is `main`, and the source repository is **private**.
- Two **active** branch rulesets named `main` (id `14083325`) and `main_rule` (id `14978586`) target the default branch.
- Both already contain `required_status_checks` rules with `strict_required_status_checks_policy: true`, **but each has `required_status_checks: []`**. An empty check list does not require any named workflow to pass.
- Both rulesets block deletion and non-fast-forward updates. `main` also requires one approving review, code-owner review, and squash merges.
- Both currently have a **repository-role bypass actor** with `bypass_mode: always`. The exact repository-role identity and authorized bypass process must be reviewed by the owner; the connector output is not proof of who used a bypass.
- `.github/CODEOWNERS` assigns every file to `@atipparaju`. GitHub does not permit a PR author to approve their own PR; a solo-maintainer workflow needs an explicitly considered review/bypass policy.
- GitHub CI has the following **actual job names** (not workflow file names):
  - `Test & Lint` — workflow `CI`; runs on PRs and pushes to `main`.
  - `Integration Test` — workflow `CI`; runs on PRs and pushes to `main`.
  - `Container vulnerability gate` — workflow `Container Security`; runs on PRs and pushes to `main`.
  - `Kind / Probe Smoke` — workflow `Kubernetes Smoke Test`; runs on PRs, and after this PR is merged, pushes to `main`.
- `Publish approved release image` is **skipped** on ordinary PRs/pushes; **do not require** this release-only job for merging.

## A. Owner steps: required GitHub Actions checks

1. Open the repository **Settings → Rules → Rulesets**. Inspect `main` and `main_rule` and verify both target `~DEFAULT_BRANCH`; select **one authoritative ruleset** for required checks. Prefer consolidating duplicate/overlapping rules after safely verifying the effective policy; do not delete either during this PR.
2. In that ruleset, find **Require status checks to pass**. Add exactly these four completed GitHub Actions check names (case-sensitive):
   - [ ] `Test & Lint`
   - [ ] `Integration Test`
   - [ ] `Container vulnerability gate`
   - [ ] `Kind / Probe Smoke`
3. Keep **Require branches to be up to date before merging** (strict checks) enabled, since the current repository process rebases/reconciles each issue branch with the latest `main`.
4. Where GitHub offers a check-source selection, select **GitHub Actions** if offered and confirmed as the provider of the check; do not guess an app identity or accept a similarly named unrelated status.
5. Save the ruleset and read it back. Verify four nonempty required contexts. Record **only names and pass/fail**, not credentials or privileged screenshots.
6. Avoid requiring release-only or skipped jobs, such as `Publish approved release image`. Never fabricate check runs or use a workflow that only reports success without exercising tests.

**Why strict matters:** A passing check from before the branch's last update or before a newer `main` change is insufficient. Update the PR branch and rerun checks.

## B. Owner steps: review and bypass policy

- [ ] Review the role-based bypass in **both** active rulesets. Document which role is allowed, why, and who can approve emergency use; narrow or remove overly broad bypass only after verifying there is a workable merge path.
- [ ] Decide how to handle the existing `CODEOWNERS` self-ownership and mandatory approving review. GitHub PR authors **cannot approve their own PRs**. Options: add a trusted independent reviewer with sufficient access, or establish a strictly controlled owner-admin bypass for solo-maintainer releases and record the explicit review/CI evidence each time. Do **not** claim the bypass represents an independent approval.
- [ ] Confirm required checks apply to normal contributors and any actors not permitted to bypass, including non-fast-forward restrictions.
- [ ] Test enforcement using a **safe, disposable PR** after manual ruleset edits: before required checks finish, confirm the merge UI blocks a non-bypass actor; after checks pass on an up-to-date branch, confirm the gate clears. No intentional production-breaking commit or automatic merge is needed.
- [ ] Review audit logs or bypass history, if available, for unplanned bypasses.

## C. Owner steps: GHCR package existence, visibility, and actions access

1. Open the account's **Packages** tab and look for the `nats-k8s-health-bridge` container. If it does **not exist**, record **not published / package absent** and **do not publish it to test**.
2. If it exists, open **Package settings**. Record **private/public** as actually displayed, and inspect whether access is inherited from the repository or set independently. A private source repository does not guarantee a private container package.
3. Check **Manage Actions access** for the repository and grant no broader `packages:write` access than needed for an owner-approved release workflow.
4. Keep any public-visibility toggle, `Change visibility` action, or GHCR publishing **untouched** without a separate explicit release approval.
5. Do not put tokens, settings exports, or package permission screenshots containing sensitive identifiers in this repository.

GitHub official references:
- [Rulesets and required checks](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets)
- [Troubleshooting status check names](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/troubleshooting-rules)
- [Container package visibility and access](https://docs.github.com/en/packages/learn-github-packages/configuring-a-packages-access-control-and-visibility)

## D. PR and issue closure checklist

- [x] Source rulesets inspected and four required CI job names identified.
- [x] Repo remains private; workflows are not configured to auto-publish.
- [x] Kind smoke workflow extended to run on pushes to `main` as a post-merge regression check.
- [ ] All four contexts are actually required in an active rule targeting `main` (**owner-admin configuration**).
- [ ] A safe PR confirms check enforcement for a non-bypass actor.
- [ ] Solo-maintainer approval/bypass path is explicitly reviewed and documented.
- [ ] GHCR package existence and actual visibility verified from its settings.
- [ ] Owner accepts the final ruleset policy and records approval for Issue #42.

**Completion rule:** A green documentation/workflow PR is not sufficient to close Issue #42. Keep this issue open until the required status checks and GHCR visibility have been verified in the GitHub UI. No release or visibility change is authorized here.
