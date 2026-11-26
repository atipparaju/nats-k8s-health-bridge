#!/usr/bin/env python3
"""Deterministic public AI-guidance consistency checks; no network or AI calls."""
from pathlib import Path
import re
import sys

REQUIRED = (
    "AGENTS.md",
    "CLAUDE.md",
    ".github/copilot-instructions.md",
    "README.md",
    "CONTRIBUTING.md",
)
ENTRY_POINTS = ("CLAUDE.md", ".github/copilot-instructions.md")
STALE_PHRASES = (
    "repository (private)",
    "this private bridge",
    "private development project",
    "intended for open-source release",
)
VERSION_PIN = re.compile(
    r"\b(?:github\.com/[\w./-]+|golang\.org/[\w./-]+)\s+v\d+\.\d+\.\d+\b"
)
MARKDOWN_LINK = re.compile(r"\[[^\]\n]+\]\(([^)]+)\)")


def check(root: Path) -> list[str]:
    errors = []
    texts = {}
    for name in REQUIRED:
        path = root / name
        if not path.is_file():
            errors.append(f"missing required file: {name}")
        else:
            texts[name] = path.read_text(encoding="utf-8")

    for name in ENTRY_POINTS:
        if name not in texts:
            continue
        content = texts[name]
        # Every assistant-specific file must explicitly direct agents to AGENTS.md.
        if "AGENTS.md" not in content:
            errors.append(f"{name}: must reference AGENTS.md")
        if VERSION_PIN.search(content):
            errors.append(f"{name}: remove pinned module versions; use go.mod")
        for phrase in STALE_PHRASES:
            if phrase in content.lower():
                errors.append(f"{name}: outdated repository-status phrase")

    for name, content in texts.items():
        parent = (root / name).parent
        for target in MARKDOWN_LINK.findall(content):
            target = target.split("#", 1)[0].strip()
            if not target or "://" in target or target.startswith(("mailto:", "#")):
                continue
            # Only check relative repository markdown/document links.
            if Path(target).is_absolute():
                errors.append(f"{name}: invalid absolute link")
                continue
            resolved = (parent / target).resolve()
            if not resolved.is_relative_to(root.resolve()):
                errors.append(f"{name}: relative link escapes repository")
            elif not resolved.exists():
                errors.append(f"{name}: broken relative link to {target}")
    return errors


def main() -> int:
    root = Path(__file__).resolve().parents[1]
    errors = check(root)
    for error in errors:
        print("FAIL:", error)
    if errors:
        return 1
    print("PASS: AI documentation entry points, references, and stale-text checks")
    return 0


if __name__ == "__main__":
    sys.exit(main())
