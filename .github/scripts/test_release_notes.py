"""Offline checks for reviewed release-note selection."""
import importlib.util
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location("release_notes", Path(__file__).with_name("generate_release_notes.py"))
NOTES = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(NOTES)


class ReviewedNotesTests(unittest.TestCase):
    def test_exact_notes_without_credentials_or_network(self):
        import contextlib
        import io
        expected = Path(__file__).resolve().parents[2] / "docs/releases/0.14.0.md"
        stdout = io.StringIO()
        with patch.dict("os.environ", {"CURRENT_TAG": "v0.14.0", "GITHUB_REPOSITORY": "niklas-heer/tdx"}, clear=True):
            with patch.object(NOTES, "generate_release_notes_with_ai", side_effect=AssertionError("unexpected API call")):
                with contextlib.redirect_stdout(stdout):
                    NOTES.main()
        self.assertEqual(stdout.getvalue(), expected.read_text())

    def test_missing_and_unsafe_tags_do_not_read_notes(self):
        for tag in ("v999.999.999", "../RELEASE", "v../../README", "v0.14.0/../../README"):
            self.assertIsNone(NOTES.reviewed_release_notes(tag))

    def test_empty_reviewed_notes_fail(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "docs/releases").mkdir(parents=True)
            (root / "docs/releases/0.14.0.md").write_text(" \n")
            with patch.object(NOTES, "__file__", str(root / ".github/scripts/generate_release_notes.py")):
                with self.assertRaises(ValueError):
                    NOTES.reviewed_release_notes("v0.14.0")


if __name__ == "__main__":
    unittest.main()
