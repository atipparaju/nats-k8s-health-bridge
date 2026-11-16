#!/usr/bin/env python3
"""Read-only registry index digest audit for source container images.

Print candidate immutable multi-platform refs. This does not publish, login,
modify any Docker images, or write a lockfile. Verify the CI run before pinning.
"""
import json
import re
import subprocess
import sys

IMAGES = (
    "docker.io/library/golang:1.26.11-alpine3.23",
    "docker.io/library/nats:2.15.3-alpine3.22",
    "gcr.io/distroless/static-debian12:nonroot",
)
REQUIRED = {"linux/amd64", "linux/arm64"}
DIGEST = re.compile(r"(?m)^Digest:\s*(sha256:[0-9a-f]{64})\s*$")


def inspect(ref, raw=False):
    cmd = ["docker", "buildx", "imagetools", "inspect"]
    if raw:
        cmd.append("--raw")
    cmd.append(ref)
    return subprocess.run(cmd, check=True, capture_output=True, text=True, timeout=120).stdout


def audit(ref):
    summary = inspect(ref)
    found = DIGEST.search(summary)
    if not found:
        raise RuntimeError(f"No registry index digest in imagetools output for {ref}")
    index = json.loads(inspect(ref, raw=True))
    manifests = index.get("manifests")
    if not isinstance(manifests, list) or not manifests:
        raise RuntimeError(f"Not a multi-platform OCI/Docker manifest index: {ref}")
    platforms = {
        f"{p.get('os')}/{p.get('architecture')}"
        for m in manifests
        if isinstance(m, dict)
        for p in [m.get("platform")]
        if isinstance(p, dict)
    }
    missing = REQUIRED - platforms
    if missing:
        raise RuntimeError(f"Missing required platforms {sorted(missing)} for {ref}")
    print(f"PASS: {ref}")
    print(f"  Verified platforms: {', '.join(sorted(REQUIRED))}")
    print(f"  Candidate immutable ref: {ref}@{found.group(1)}")


def main():
    try:
        for ref in IMAGES:
            audit(ref)
    except (subprocess.CalledProcessError, subprocess.TimeoutExpired, ValueError, RuntimeError) as err:
        print(f"FAIL: registry digest audit: {type(err).__name__}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
