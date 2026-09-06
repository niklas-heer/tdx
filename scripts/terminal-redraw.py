#!/usr/bin/env python3
"""Assert final terminal cells, including stale rows and preserved shell context."""
import argparse
import json
from pathlib import Path
import runpy
import tempfile

import pyte

Terminal = runpy.run_path(str(Path(__file__).with_name("usage-pty.py")))["Terminal"]


def run(binary, output):
    with tempfile.TemporaryDirectory(prefix="tdx-redraw-") as tmp:
        base = Path(tmp)
        path = base / "tasks.md"
        original = "# Todos\n\n- [ ] Alpha\n- [x] Beta\n  - [ ] Gamma\n"
        path.write_text(original)
        screen = pyte.Screen(100, 32)
        terminal = Terminal(binary, path, base / "config", context="shell-context\n", screen=screen)
        snapshots = []

        def content():
            return "\n".join(screen.display)

        def expect_list(label, rezero=False):
            terminal.pump(0.1)
            text = content()
            snapshots.append({"stage": label, "screen": text})
            assert screen.display[0].rstrip() == "shell-context", f"{label}: shell context moved or erased"
            assert "check-all" not in text and "Review readiness" not in text, f"{label}: stale command text"
            assert not any(c in text for c in "┌┐└┘"), f"{label}: stale palette border"
            for title in ("Alpha", "Beta", "Gamma"):
                assert text.count(title) == 1, f"{label}: missing or duplicated {title}"
            if rezero:
                assert screen.display[1].strip() == "# Todos", f"{label}: list origin drifted"
            else:
                assert "Alpha" in screen.display[1], f"{label}: list origin drifted"
                assert "REZERO" not in text, f"{label}: stale Rezero status"

        try:
            terminal.until(lambda: "Gamma" in content(), "initial screen")
            expect_list("initial")
            # Opening without a query reproduces the exact taller-palette
            # transition from the reported screenshot. Repeat to catch drift.
            for round_number in range(3):
                terminal.send(b":")
                terminal.until(lambda: "check-all" in content(), "unfiltered palette")
                terminal.send(b"\r")
                terminal.until(lambda: "REZERO REVIEW" in content(), "review screen")
                expect_list(f"review-{round_number}", True)
                terminal.send(b" \r")
                terminal.until(lambda: "REZERO WORK" in content(), "work screen")
                expect_list(f"work-{round_number}", True)
                terminal.send(b"r")
                terminal.until(lambda: "CONTINUE" in content(), "inline continuation")
                terminal.send("\x1b[200~ café 🦀\x1b[201~".encode())
                terminal.until(lambda: "café 🦀" in content(), "Unicode input cells")
                terminal.send(b"\x1b")
                terminal.until(lambda: "CONTINUE" not in content(), "cancel continuation")
                expect_list(f"cancel-input-{round_number}", True)
                terminal.send(b"\x1b")
                terminal.until(lambda: "REZERO" not in content(), "return to normal list")
                expect_list(f"exit-{round_number}")
                terminal.send(b":")
                terminal.until(lambda: "check-all" in content(), "palette before filtering")
                terminal.send(b"rezero")
                terminal.until(lambda: "check-all" not in content(), "filtered palette")
                terminal.send(b"\x1b")
                terminal.until(lambda: "Review readiness" not in content(), "cancel palette")
                expect_list(f"cancel-palette-{round_number}")
            for width in (40, 80, 24, 100):
                terminal.resize(width, 32)
                terminal.pump(0.2)
                expect_list(f"resize-{width}")
            terminal.send(b"e")
            terminal.until(lambda: "EDIT" in content(), "edit after resizing")
            terminal.send("\x1b[200~ café 🦀\x1b[201~\r".encode())
            terminal.until(lambda: "café 🦀" in path.read_text(), "Unicode edit persisted")
            terminal.send(b"u")
            terminal.until(lambda: path.read_text() == original, "undo after resizing")
            expect_list("undo")
            terminal.close()
            assert path.read_text() == original, "visual transitions changed document"
            assert b"\x1b[?1049h" not in terminal.output, "entered alternate screen"
            return {"status": "passed", "screen_assertions": len(snapshots)}
        finally:
            terminal.cleanup(output / "redraw.ansi")
            (output / "redraw-screens.json").write_text(json.dumps(snapshots, indent=2)+"\n")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path, default=Path("tdx"))
    parser.add_argument("--output", type=Path, default=Path("dist/usage/pty"))
    args = parser.parse_args()
    args.output.mkdir(parents=True, exist_ok=True)
    report_path = args.output / "redraw-report.json"
    try:
        report = run(args.binary.resolve(), args.output)
    except Exception as exc:
        report_path.write_text(json.dumps({"status": "failed", "error": str(exc)})+"\n")
        raise
    report_path.write_text(json.dumps(report, indent=2)+"\n")
    print(json.dumps(report))


if __name__ == "__main__":
    main()
