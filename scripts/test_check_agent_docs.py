"""Synthetic unit tests for AI docs validator; no repository files modified."""
from pathlib import Path
import importlib.util
import tempfile
import unittest

module_path = Path(__file__).with_name("check_agent_docs.py")
spec = importlib.util.spec_from_file_location("check_agent_docs", module_path)
validator = importlib.util.module_from_spec(spec)
spec.loader.exec_module(validator)


def sample_tree(root):
    for path in validator.REQUIRED:
        target = root / path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text("# Example\n", encoding="utf-8")
    (root / "CLAUDE.md").write_text("[AGENTS.md](AGENTS.md)\n", encoding="utf-8")
    (root / ".github/copilot-instructions.md").write_text(
        "[AGENTS.md](../AGENTS.md)\n", encoding="utf-8"
    )


class AgentDocsTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        sample_tree(self.root)

    def test_valid_tree(self):
        self.assertEqual(validator.check(self.root), [])

    def test_missing_required_file(self):
        (self.root / "AGENTS.md").unlink()
        self.assertTrue(any("missing required" in x for x in validator.check(self.root)))

    def test_missing_agent_reference(self):
        (self.root / "CLAUDE.md").write_text("# No pointer\n", encoding="utf-8")
        self.assertTrue(any("must reference" in x for x in validator.check(self.root)))

    def test_broken_relative_link(self):
        (self.root / "CLAUDE.md").write_text(
            "[AGENTS.md](AGENTS.md) and [missing](missing.md)", encoding="utf-8"
        )
        self.assertTrue(any("broken relative link" in x for x in validator.check(self.root)))

    def test_rejects_escape_from_repository(self):
        (self.root / ".github/copilot-instructions.md").write_text(
            "[AGENTS.md](../../outside.md)\n", encoding="utf-8"
        )
        self.assertTrue(any("escapes repository" in x for x in validator.check(self.root)))

    def test_stale_private_wording(self):
        (self.root / "CLAUDE.md").write_text(
            "[AGENTS.md](AGENTS.md)\nRepository (private)\n", encoding="utf-8"
        )
        self.assertTrue(any("outdated repository-status" in x for x in validator.check(self.root)))

    def test_version_pin(self):
        (self.root / "CLAUDE.md").write_text(
            "[AGENTS.md](AGENTS.md)\ngithub.com/nats-io/nats.go v1.55.0\n",
            encoding="utf-8",
        )
        self.assertTrue(any("pinned module" in x for x in validator.check(self.root)))


if __name__ == "__main__":
    unittest.main()
