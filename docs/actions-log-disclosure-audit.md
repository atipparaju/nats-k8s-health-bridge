# Route 1: private GitHub Actions log disclosure audit

**Issue:** [#87](https://github.com/atipparaju/nats-k8s-health-bridge/issues/87) · **Parent:** [#80](https://github.com/atipparaju/nats-k8s-health-bridge/issues/80) · **Status:** NOT RUN TO COMPLETION · **Publication:** NO-GO

The owner chose **Route 1 (preserve this repository and its history when eventually made public)**, but has **not approved any visibility change**. The owner accepts disclosure of their personal commit-author email and the existing Code of Conduct contact address, while requiring employer-identifying information to stay private.

GitHub documents that when a private repository becomes public, past GitHub Actions history and logs become visible to everyone. Earlier API screening checked workflow *metadata*, not log *content*. This local-only audit addresses that specific difference.

## Run on a trusted local workstation only

Prerequisites: Python 3, network access to GitHub, and authenticated GitHub CLI with access to **Actions read** for this private repository. Use `gh auth login` or your already authenticated `gh` session; do not paste any access token in commands or arguments.

From a private clone of this repository:

```bash
python3 scripts/audit_actions_log_disclosure.py
```

On Windows PowerShell:

```powershell
py -3 scripts/audit_actions_log_disclosure.py
```

The program prompts for a **hidden, private case-insensitive regular expression**. Enter only identifiers to screen, separated using the regular expression OR operator when needed. Do not add those identifiers to `go.mod`, workflows, GitHub Issues, README, command-line flags, shell history, ChatGPT transcripts, test files or this document.

The script:
1. Reads every currently enumerated GitHub Actions workflow run using pagination; fails if enumeration counts are inconsistent.
2. Checks metadata fields and downloads the log archive for **every reported attempt** of each run, using the Actions API. The authenticated GitHub API returns a short-lived redirect URL; the script downloads it **without forwarding the GitHub Authorization header**.
3. Searches decompressed log text and archive filenames privately. The content exists only in process memory. No ZIP, full text, private search expression or signed download URL is written to disk or echoed.
4. Prints **only progress counts, reviewed attempt counts, numbers of matches and unavailable attempts, and run/attempt IDs**. It does not print matching text, filenames, identities, email addresses or log URLs.
5. Returns nonzero if any match is found or if any historical archive cannot be examined. A log that was deleted, expired or is too large is **UNVERIFIED**, not proof of safety.

The CI step `Test private history log audit (offline, no credentials)` uses synthetic ZIP archives and an entirely local mock. **CI never runs the private-word scan**, never obtains the sensitive search expression, and never inspects actual historical logs.

## Interpret the results

| Outcome | Meaning | Next action |
| --- | --- | --- |
| Zero matches and zero unavailable attempts | The private terms were not detected in **accessible log archives and inspected workflow metadata at that snapshot** | Record run count, attempt count, commit SHA and date in a **private** clearance ledger; still review Git refs, PR attachments, actions artifacts, rights and licenses |
| One or more matched run IDs | A run/attempt contains a possible employer-identifying term | Inspect the flagged run privately; distinguish test examples from real employer disclosure, and decide remediation before publication. Do not paste matches or raw logs into GitHub |
| One or more unavailable run/attempt IDs | The archive could not be checked (expired, removed, oversized, permissions or network failure) | Treat the coverage as incomplete. Confirm whether logs could still become public, and get explicit owner risk disposition before any visibility change |
| GitHub authentication/API error | The audit did not finish | Repair access or connection and rerun; no GO decision |

**No script can establish that every historic GitHub surface is clean.** Run the separate [Git-history and metadata review](public-history-review-runbook.md) for all intended branches/PR refs, issue comments, attachments, retained artifacts, old screenshots, commit messages, identities, copied source files and possible confidential intellectual property. Source and image license/NOTICE obligations and ownership interpretation under the agreement remain independent gates.

## Safety boundary

The script is read-only: it uses the GitHub API to list runs and download available logs, without mutation or repository writes. It enforces a maximum archive size and entry size to avoid uncontrolled extraction, and does not extract ZIP archives to disk. Do not use it inside GitHub Actions, in a shared terminal recording, or on a machine where logs/terminal transcripts are automatically collected externally.

Do not delete workflow runs, rewrite Git history, remove branches or PRs, rotate credentials, change repository visibility, create a public fork, publish GHCR images, or declare GO based solely on this scan. Any remediation should be separately reviewed and expressly authorized.
