#!/usr/bin/env python3
"""Offline release guardrail test. Never builds, authenticates, pushes, or publishes."""

from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parents[1]
checks = []


def verify(condition: bool, description: str) -> None:
    checks.append((condition, description))


makefile = (ROOT / "Makefile").read_text(encoding="utf-8")
workflow = (ROOT / ".github/workflows/ci.yml").read_text(encoding="utf-8")

# The old setup script and Makefile push target could change the remote
# repository/registry without the tag-scoped manual workflow approval.
verify(not (ROOT / "setup.sh").exists(), "obsolete Git bootstrap/push script removed")
verify(not re.search(r"(?m)^docker-push\s*:", makefile), "no Makefile registry push target")
verify(not re.search(r"\bdocker\s+push\b|\bbuildx\s+build\b[^\n]*--push", makefile),
       "Makefile contains no image publication commands")
verify("IMAGE_LOCAL   := nats-k8s-health-bridge:local" in makefile,
       "local image tag has no GHCR prefix")
verify("docker build --tag $(IMAGE_LOCAL) ." in makefile,
       "local Docker build does not require GHCR credentials")
verify(":latest" not in makefile and "ghcr.io/" not in makefile,
       "local developer targets do not create floating GHCR tags")
verify(re.search(r"(?m)^release-dry-run:\s*\n\tpython3 scripts/check_release_safety\.py\s*$",
                 makefile) is not None,
       "Makefile exposes the offline release-dry-run target")

# Read only the release job, not other if-expressions in test jobs. We assert
# its authorization predicate and the lack of floating release tag patterns.
release = re.search(r"(?ms)^  build-push:\s*\n(.*?)(?=^  [\w-]+:\s*$|\Z)", workflow)
verify(release is not None, "approved release job still exists")
job = release.group(1) if release else ""

expected_gate = (
    "github.event_name == 'workflow_dispatch' && inputs.publish_image == true "
    "&& startsWith(github.ref, 'refs/tags/v')"
)
gate = re.search(r"(?m)^    if: \$\{\{\s*(.*?)\s*\}\}\s*$", job)
verify(gate is not None and gate.group(1) == expected_gate,
       "release publish requires manual dispatch, explicit opt-in and v* tag reference")
verify("publish_image:" in workflow and re.search(r"(?m)^        default: false\s*$", workflow),
       "manual publication input defaults to false")
verify("permissions:\n      contents: read\n      packages: write" in job,
       "registry write permission is scoped to approved release job")
verify("docker/build-push-action" in job and re.search(r"(?m)^          push: true\s*$", job),
       "publishing occurs only inside the guarded build-push job")
verify("type=semver,pattern={{version}}" in job and "type=sha,prefix=sha-" in job,
       "approved releases receive version and commit-specific tags")
verify("type=ref,event=branch" not in job
       and "type=semver,pattern={{major}}.{{minor}}" not in job
       and "type=raw,value=latest" not in job,
       "release job does not generate branch, major/minor, or latest tags")
verify("python3 scripts/check_release_safety.py" in workflow,
       "CI runs this offline release-safety check")

failures = [name for valid, name in checks if not valid]
for valid, name in checks:
    print(f"{'PASS' if valid else 'FAIL'}: {name}")
if failures:
    sys.exit(f"Release dry-run failed ({len(failures)} guardrails).")
print("Release dry-run passed: no network access, image push, or publication performed.")
