#!/usr/bin/env python3
"""Isolated regression tests for bootstrap_project_docs.py."""

from __future__ import annotations

import hashlib
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name("bootstrap_project_docs.py")
START = "<!-- DOCS_GOVERNANCE:START -->"


def run_script(root: Path, *args: str, expect: int = 0) -> subprocess.CompletedProcess[str]:
    result = subprocess.run(
        [sys.executable, str(SCRIPT), "--project-root", str(root), *args],
        check=False,
        capture_output=True,
        text=True,
    )
    if result.returncode != expect:
        raise AssertionError(
            f"expected exit {expect}, got {result.returncode}\nstdout:\n{result.stdout}\nstderr:\n{result.stderr}"
        )
    return result


def tree_digest(root: Path) -> str:
    digest = hashlib.sha256()
    for path in sorted(p for p in root.rglob("*") if p.is_file()):
        digest.update(path.relative_to(root).as_posix().encode())
        digest.update(path.read_bytes())
    return digest.hexdigest()


class BootstrapProjectDocsTests(unittest.TestCase):
    def test_new_project_dry_run_write_check_and_idempotency(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            preview = run_script(root, "--mode", "auto", "--dry-run")
            self.assertIn("mode=new", preview.stdout)
            self.assertEqual(list(root.iterdir()), [])

            run_script(root, "--mode", "auto", "--project-name", "Demo", "--owner", "team")
            self.assertTrue((root / "docs/01-product/project-charter.md").is_file())
            self.assertFalse((root / "docs/02-architecture/current-state-baseline.md").exists())
            run_script(root, "--check")

            before = tree_digest(root)
            rerun = run_script(root, "--mode", "new", "--project-name", "Demo", "--owner", "team")
            after = tree_digest(root)
            self.assertEqual(before, after)
            self.assertIn("AGENTS.md=kept", rerun.stdout)

    def test_existing_project_preserves_files_and_agents_content(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            (root / "src").mkdir()
            (root / "docs/00-governance").mkdir(parents=True)
            (root / "package.json").write_text('{"private": true}\n', encoding="utf-8")
            (root / "docs/legacy.md").write_text("# Existing\n", encoding="utf-8")
            principles = root / "docs/00-governance/documentation-principles.md"
            principles.write_text("# Custom principles\n", encoding="utf-8")
            (root / "AGENTS.md").write_text("# Existing rules\n\nKeep me.\n", encoding="utf-8")

            result = run_script(root, "--mode", "auto")
            self.assertIn("mode=existing", result.stdout)
            self.assertEqual((root / "docs/legacy.md").read_text(encoding="utf-8"), "# Existing\n")
            self.assertEqual(principles.read_text(encoding="utf-8"), "# Custom principles\n")
            agents = (root / "AGENTS.md").read_text(encoding="utf-8")
            self.assertIn("Keep me.", agents)
            self.assertEqual(agents.count(START), 1)
            baseline = (root / "docs/02-architecture/current-state-baseline.md").read_text(encoding="utf-8")
            self.assertIn("`package.json`", baseline)
            self.assertIn("`src/`", baseline)
            self.assertIn("`docs/legacy.md`", baseline)
            run_script(root, "--check")

    def test_malformed_agents_markers_fail_before_writes(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            (root / "AGENTS.md").write_text(f"# Broken\n\n{START}\n", encoding="utf-8")
            result = run_script(root, expect=2)
            self.assertIn("不唯一或未闭合", result.stderr)
            self.assertFalse((root / "docs").exists())


if __name__ == "__main__":
    unittest.main()
