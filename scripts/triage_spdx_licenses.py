#!/usr/bin/env python3
"""Read-only SPDX 2.x package triage. No license classification or legal conclusions."""
import argparse
import collections
import csv
import hashlib
import json
from pathlib import Path
import sys


FIELDS = ("package", "version", "spdx_id", "declared", "concluded",
          "files_analyzed", "classification", "evidence", "obligations",
          "source_disposition", "image_disposition", "reviewer", "review_date")


def classify(package):
    declared = str(package.get("licenseDeclared", "NOASSERTION")).strip()
    concluded = str(package.get("licenseConcluded", "NOASSERTION")).strip()
    if declared in ("", "NOASSERTION", "NONE") or concluded in ("", "NOASSERTION", "NONE"):
        return "REQUIRES_REVIEW"
    if "LicenseRef-" in declared or "LicenseRef-" in concluded:
        return "REQUIRES_REVIEW"
    return "REVIEW_RECORDED_NOT_APPROVED"


def rows_from_spdx(doc):
    if not isinstance(doc, dict) or not str(doc.get("spdxVersion", "")).startswith("SPDX-2."):
        raise ValueError("Expected SPDX 2.x JSON object")
    packages = doc.get("packages")
    if not isinstance(packages, list) or not packages:
        raise ValueError("No SPDX packages found")
    rows = []
    for p in packages:
        if not isinstance(p, dict):
            raise ValueError("Invalid SPDX package")
        rows.append({
            "package": str(p.get("name") or "UNKNOWN"),
            "version": str(p.get("versionInfo") or "UNKNOWN"),
            "spdx_id": str(p.get("SPDXID") or "UNKNOWN"),
            "declared": str(p.get("licenseDeclared") or "NOASSERTION"),
            "concluded": str(p.get("licenseConcluded") or "NOASSERTION"),
            "files_analyzed": str(p.get("filesAnalyzed", "unknown")).lower(),
            "classification": classify(p),
            "evidence": "",
            "obligations": "",
            "source_disposition": "PENDING",
            "image_disposition": "PENDING",
            "reviewer": "",
            "review_date": "",
        })
    return rows


def write_matrix(path, rows):
    destination = Path(path).expanduser().resolve()
    # The output may contain third-party metadata. Keep it out of git and CI artifacts.
    repo_root = Path(__file__).resolve().parents[1]
    if destination == repo_root or repo_root in destination.parents:
        raise ValueError("Matrix destination must be outside the repository worktree")
    if destination.exists():
        raise ValueError("Refusing to overwrite an existing matrix")
    destination.parent.mkdir(parents=True, exist_ok=True)
    with destination.open("x", encoding="utf-8", newline="") as stream:
        writer = csv.DictWriter(stream, fieldnames=FIELDS)
        writer.writeheader()
        writer.writerows(rows)
    try:
        destination.chmod(0o600)
    except OSError:
        pass


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("spdx_json", help="Path to existing locally retrieved SPDX JSON SBOM")
    parser.add_argument("--matrix", help="Optional private CSV output outside repository")
    args = parser.parse_args(argv)
    source = Path(args.spdx_json)
    try:
        raw = source.read_bytes()
        doc = json.loads(raw)
        rows = rows_from_spdx(doc)
        counts = collections.Counter(row["classification"] for row in rows)
        if args.matrix:
            write_matrix(args.matrix, rows)
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        print("FAIL: SBOM triage could not complete:", type(exc).__name__, file=sys.stderr)
        return 2
    # Aggregate-only stdout, no package names, paths, tokens or SPDX source snippets.
    print("SBOM SHA256:", hashlib.sha256(raw).hexdigest())
    print("SPDX package count:", len(rows))
    for key in ("REQUIRES_REVIEW", "REVIEW_RECORDED_NOT_APPROVED"):
        print(key + ":", counts[key])
    print("License obligations, source ownership, NOTICE and artifact redistribution: UNVERIFIED")
    print("This inventory does not authorize publication.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
