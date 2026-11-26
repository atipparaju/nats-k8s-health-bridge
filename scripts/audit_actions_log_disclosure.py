#!/usr/bin/env python3
"""Privately inspect GitHub Actions logs for user-entered disclosure terms.

Read-only. No patterns, tokens, matching lines, ZIPs or signed URLs are written
to the repository or printed. Run on a trusted local workstation, not CI.
"""
from __future__ import annotations

import argparse
import getpass
import io
import json
import os
import re
import subprocess
import sys
import urllib.error
import urllib.parse
import urllib.request
import zipfile

DEFAULT_REPO = "atipparaju/nats-k8s-health-bridge"
MAX_ARCHIVE_BYTES = 96 * 1024 * 1024
MAX_ENTRY_BYTES = 48 * 1024 * 1024
MAX_ARCHIVE_ENTRIES = 10000
TIMEOUT_SECONDS = 90
API_BASE = "https://api.github.com"


class AuditError(Exception):
    """A review step cannot safely be considered complete."""


class StopRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, request, fp, code, msg, headers, newurl):
        return None


NO_REDIRECT = urllib.request.build_opener(StopRedirect)


def auth_token():
    token = os.environ.get("GH_TOKEN") or os.environ.get("GITHUB_TOKEN")
    if not token:
        try:
            result = subprocess.run(
                ["gh", "auth", "token"], check=True, stdout=subprocess.PIPE,
                stderr=subprocess.DEVNULL, text=True, timeout=15,
            )
            token = result.stdout.strip()
        except (FileNotFoundError, subprocess.CalledProcessError, subprocess.TimeoutExpired):
            raise AuditError("GitHub auth unavailable: sign in using 'gh auth login' or set GH_TOKEN") from None
    if not token:
        raise AuditError("GitHub auth token is empty")
    return token


def api_headers(token):
    return {
        "Authorization": "Bearer " + token,
        "Accept": "application/vnd.github+json",
        "X-GitHub-Api-Version": "2022-11-28",
        "User-Agent": "nkhb-private-disclosure-audit",
    }


def api_json(repo, token, page):
    url = f"{API_BASE}/repos/{repo}/actions/runs?per_page=100&page={page}"
    try:
        with urllib.request.urlopen(
            urllib.request.Request(url, headers=api_headers(token)),
            timeout=TIMEOUT_SECONDS,
        ) as response:
            return json.load(response)
    except (urllib.error.URLError, ValueError, TimeoutError):
        raise AuditError(f"Could not retrieve workflow metadata page {page}") from None


def list_runs(repo, token):
    runs, ids = [], set()
    page, total = 1, None
    while True:
        data = api_json(repo, token, page)
        if not isinstance(data, dict) or not isinstance(data.get("workflow_runs"), list):
            raise AuditError("Unexpected GitHub workflow-run response")
        if total is None:
            total = data.get("total_count")
            if not isinstance(total, int):
                raise AuditError("GitHub run total unavailable")
        batch = data["workflow_runs"]
        for run in batch:
            run_id, attempts = run.get("id"), run.get("run_attempt", 1)
            if not isinstance(run_id, int) or not isinstance(attempts, int) or attempts < 1:
                raise AuditError("Invalid run ID or attempt count")
            if run_id not in ids:
                ids.add(run_id)
                runs.append(run)
        if not batch or len(batch) < 100:
            break
        page += 1
        if page > 100:
            raise AuditError("Run pagination exceeded safety limit")
    if len(runs) != total:
        raise AuditError("Workflow-run count changed or pagination was incomplete; retry")
    return runs


def validate_download_host(url):
    parsed = urllib.parse.urlparse(url)
    host = (parsed.hostname or "").lower()
    trusted_suffixes = (
        "githubusercontent.com", "blob.core.windows.net",
        "amazonaws.com", "github.com",
    )
    if (parsed.scheme != "https" or not host or parsed.username or parsed.password or
            not any(host == s or host.endswith("." + s) for s in trusted_suffixes)):
        raise AuditError("GitHub log redirect target was not an approved HTTPS host")


def download_archive(repo, token, run_id, attempt):
    """Fetch GitHub's authenticated redirect then download anonymously.

    This prevents forwarding the Authorization header to a signed log-storage URL.
    """
    api_url = f"{API_BASE}/repos/{repo}/actions/runs/{run_id}/attempts/{attempt}/logs"
    request = urllib.request.Request(api_url, headers=api_headers(token))
    try:
        with NO_REDIRECT.open(request, timeout=TIMEOUT_SECONDS):
            raise AuditError("Unexpected non-redirect log archive response")
    except urllib.error.HTTPError as response:
        if response.code in (404, 410):
            raise AuditError("Log archive unavailable or expired") from None
        if response.code not in (301, 302, 303, 307, 308):
            raise AuditError(f"Log archive API returned HTTP {response.code}") from None
        location = response.headers.get("Location", "")
    except (urllib.error.URLError, TimeoutError):
        raise AuditError("Log archive API request failed") from None
    validate_download_host(location)
    try:
        # Intentionally omit Authorization and every other GitHub API header.
        with urllib.request.urlopen(
            urllib.request.Request(location, headers={"User-Agent": "nkhb-private-audit"}),
            timeout=TIMEOUT_SECONDS,
        ) as response:
            data = response.read(MAX_ARCHIVE_BYTES + 1)
    except (urllib.error.URLError, TimeoutError):
        raise AuditError("Signed log archive download failed") from None
    if len(data) > MAX_ARCHIVE_BYTES:
        raise AuditError("Log archive exceeds maximum allowed size")
    return data


def archive_matches(data, pattern):
    """Return match status; never return matching text or private log paths."""
    try:
        with zipfile.ZipFile(io.BytesIO(data)) as archive:
            entries = archive.infolist()
            if not entries or len(entries) > MAX_ARCHIVE_ENTRIES:
                raise AuditError("Log archive is empty or has too many entries")
            matched = False
            for entry in entries:
                if entry.is_dir():
                    continue
                if entry.file_size > MAX_ENTRY_BYTES:
                    raise AuditError("Log entry exceeds maximum allowed size")
                if pattern.search(entry.filename):
                    matched = True
                raw = archive.read(entry)
                if pattern.search(raw.decode("utf-8", errors="replace")):
                    matched = True
            return matched
    except (zipfile.BadZipFile, RuntimeError, OSError, ValueError):
        raise AuditError("Log archive could not be fully inspected") from None


def inspect(repo, token, pattern):
    runs = list_runs(repo, token)
    matched, unresolved = [], []
    scanned_attempts = 0
    for i, run in enumerate(runs, 1):
        run_id = run["id"]
        metadata = " ".join(str(run.get(k, "")) for k in
                            ("name", "display_title", "head_branch", "path"))
        run_match = bool(pattern.search(metadata))
        for attempt in range(1, run.get("run_attempt", 1) + 1):
            try:
                content = download_archive(repo, token, run_id, attempt)
                if archive_matches(content, pattern):
                    run_match = True
                scanned_attempts += 1
            except AuditError:
                unresolved.append((run_id, attempt))
        if run_match:
            matched.append(run_id)
        if i % 25 == 0 or i == len(runs):
            print(f"Reviewed {i}/{len(runs)} workflow-run records (counts only).")
    print(f"Workflow runs enumerated: {len(runs)}")
    print(f"Log attempts inspected: {scanned_attempts}")
    print(f"Runs with potential private-term matches: {len(matched)}")
    print(f"Unreviewed log attempts: {len(unresolved)}")
    if matched:
        print("Potential-match run IDs (inspect privately): " +
              ", ".join(str(i) for i in matched))
    if unresolved:
        print("Unreviewable run/attempt IDs (owner decision required): " +
              ", ".join(f"{i}/{a}" for i, a in unresolved))
    if matched or unresolved:
        print("NO-GO: matched or unavailable logs require private remediation/review.")
        return 2
    print("PASS for the supplied private search terms in all available runs/attempts.")
    print("This does not review GitHub attachments, PR discussion assets, or IP rights.")
    return 0


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", default=DEFAULT_REPO,
                        help="Private GitHub owner/repo to inspect")
    args = parser.parse_args()
    if os.environ.get("GITHUB_ACTIONS", "").lower() == "true":
        print("ERROR: run only on a trusted local workstation, not GitHub Actions.", file=sys.stderr)
        return 3
    if not re.fullmatch(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+", args.repo):
        print("ERROR: repository must be a GitHub owner/repo pair.", file=sys.stderr)
        return 3
    terms = getpass.getpass("Private terms (regex; input hidden, never saved): ")
    if not terms.strip():
        print("ERROR: a nonempty private search expression is required.", file=sys.stderr)
        return 3
    try:
        pattern = re.compile(terms, re.IGNORECASE)
    except re.error:
        print("ERROR: invalid regular expression (expression withheld).", file=sys.stderr)
        return 3
    del terms
    try:
        return inspect(args.repo, auth_token(), pattern)
    except AuditError as exc:
        print(f"ERROR: {exc}. Audit incomplete; NO-GO.", file=sys.stderr)
        return 3


if __name__ == "__main__":
    raise SystemExit(main())
