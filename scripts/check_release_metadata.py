#!/usr/bin/env python3
"""Offline source metadata checks; no registry or GitHub operations."""
from pathlib import Path
import re
import sys

root = Path(__file__).resolve().parents[1]
def load(p):
    return (root / p).read_text(encoding="utf-8")

changelog = load("CHANGELOG.md")
security = load("SECURITY.md")
chart = load("helm/nats-k8s-health-bridge/Chart.yaml")
license_text = load("LICENSE")
notes = load("docs/release-versioning.md")

checks = [
    ("historical baseline explicitly unreleased", "Internal development baseline 0.1.0" in changelog and "not released" in changelog),
    ("non-existent tag/release links removed", "releases/tag/v0.1.0" not in changelog and "compare/v0.1.0" not in changelog),
    ("no supported 0.1.x claim", "No released version is currently supported" in security and "white_check_mark" not in security),
    ("Helm version separated from dev appVersion", bool(re.search(r'(?m)^version: 0\.1\.0$', chart)) and bool(re.search(r'(?m)^appVersion: "dev"$', chart))),
    ("canonical Apache-2.0 appendix restored", "END OF TERMS AND CONDITIONS" in license_text and "Copyright [yyyy] [name of copyright owner]" in license_text and "Copyright 2026 Ati Tipparaju" not in license_text),
    ("attribution and license recheck recorded", "NOASSERTION" in notes and "Ati Tipparaju" in notes and "Anantha Tipparaju" in notes),
    ("approval and compatibility matrix documented", "Compatibility / release evidence matrix" in notes and "NOT APPROVED" in notes),
]
for name, ok in checks:
    print(("PASS" if ok else "FAIL") + ": " + name)
if not all(ok for _, ok in checks):
    sys.exit(1)
print("Offline release metadata checks passed (no publishing).")
