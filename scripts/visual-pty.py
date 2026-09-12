#!/usr/bin/env python3
"""Check real inline terminal frames with pyte's screen and scrollback model.

Run `mise run test:terminal-visual` for isolated test dependencies, or install
pyte==0.8.2 and wcwidth==0.8.3 in a test environment and invoke this script.
Requires a POSIX PTY (macOS/Linux); no production dependency is introduced.
"""
import argparse
import importlib.util
import json
from pathlib import Path
import re
import tempfile

import pyte


spec = importlib.util.spec_from_file_location(
    "usage_pty", Path(__file__).with_name("usage-pty.py"))
pty_suite = importlib.util.module_from_spec(spec)
spec.loader.exec_module(pty_suite)


class VisualTerminal(pty_suite.Terminal):
    def __init__(self, *args):
        self.screen = pyte.HistoryScreen(100, 32, history=2000)
        self.stream = pyte.ByteStream(self.screen)
        self.fed = 0
        super().__init__(*args)

    def resize(self, width, height):
        self.screen.resize(lines=height, columns=width)
        super().resize(width, height)

    def pump(self, seconds=0.05):
        super().pump(seconds)
        self.stream.feed(bytes(self.output[self.fed:]))
        self.fed = len(self.output)

    def text(self):
        return "\n".join(self.screen.display)

    def history_text(self):
        return "\n".join(
            "".join(line[column].data for column in sorted(line))
            for line in self.screen.history.top)

    def selected_task(self, number):
        return re.search(r"0\s*[➜→]\s*\[ \] Task " + str(number) + r"\b", self.text())


def run_case(binary, output, resize):
    output.mkdir(parents=True, exist_ok=True)
    snapshots = []
    with tempfile.TemporaryDirectory(prefix="tdx-visual-pty-") as tmp:
        base = Path(tmp)
        path = base / "tasks.md"
        content = "---\nshow-headings: true\nfilter-done: true\nmax-visible: 0\n---\n# Project\n"
        for number in range(9):
            content += f"## Old {number}\n- [x] Done {number}\n"
        content += "## Other features\n"
        for number in range(10):
            content += f"- [ ] Task {number} " + "long description with Unicode café 界 " * 4 + "\n"
        path.write_text(content)
        terminal = VisualTerminal(binary, path, base / "config")

        def capture(label, focused=False):
            terminal.pump(0.15)
            text, history = terminal.text(), terminal.history_text()
            (output / f"{label}.txt").write_text(text + "\n--- HISTORY ---\n" + history)
            snapshot = {
                "label": label,
                "old_headings_on_screen": text.count("## Old"),
                "old_headings_in_history": history.count("## Old"),
                "section_banner": "Section: Other features" in text,
                "selected_task": bool(re.search(r"0\s*[➜→]\s*\[ \] Task", text)),
                "history_lines": len(terminal.screen.history.top),
            }
            snapshots.append(snapshot)
            (output / "snapshots.json").write_text(json.dumps(snapshots, indent=2) + "\n")
            assert snapshot["old_headings_in_history"] == 0, f"{label}: headings leaked into scrollback"
            if focused:
                assert snapshot["old_headings_on_screen"] == 0, f"{label}: stale headings remain above focused view"
                assert snapshot["section_banner"], f"{label}: section banner is clipped"
                assert snapshot["selected_task"], f"{label}: selected task is clipped"
            return snapshot

        try:
            terminal.until(lambda: terminal.selected_task(0), "initial selected task")
            initial = capture("initial")
            assert initial["old_headings_on_screen"] > 0, "fixture must exercise headings preceding the first task"
            if resize:
                terminal.resize(80, 24)
            terminal.send(b"G")
            terminal.until(lambda: terminal.selected_task(9), "last task")
            capture("all-bottom")
            terminal.send(b"s")
            terminal.until(lambda: "Sections" in terminal.text(), "sections picker")
            terminal.send(b"G\r")
            terminal.until(lambda: "Section: Other features" in terminal.text(), "section focus")
            capture("focused", focused=True)
            terminal.send(b"kkj")
            terminal.until(lambda: terminal.selected_task(8), "focused task navigation")
            capture("focused-navigated", focused=True)
            if resize:
                terminal.resize(40, 12)
                terminal.until(lambda: terminal.selected_task(8) and "Section: Other features" in terminal.text(), "small focused frame")
                capture("focused-small", focused=True)
                terminal.resize(100, 32)
                terminal.until(lambda: terminal.selected_task(8) and "Section: Other features" in terminal.text(), "grown focused frame")
                capture("focused-grown", focused=True)
            terminal.send(b"S")
            terminal.until(lambda: "Section: Other features" not in terminal.text(), "all sections")
            terminal.send(b"s")
            terminal.until(lambda: "Sections" in terminal.text(), "sections picker again")
            terminal.send(b"G\r")
            terminal.until(lambda: "Section: Other features" in terminal.text(), "section focus again")
            capture("focused-again", focused=True)
            assert path.read_text() == content, "navigation changed Markdown"
            terminal.close()
        finally:
            terminal.cleanup(output / "transcript.ansi")
    return {"scenario": "resize" if resize else "fixed-size", "passed": True, "snapshots": snapshots}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path, default=Path("tdx"))
    parser.add_argument("--output", type=Path, default=Path("dist/usage/visual-pty"))
    args = parser.parse_args()
    args.output.mkdir(parents=True, exist_ok=True)
    report_path = args.output / "report.json"
    report_path.write_text(json.dumps({"status": "running"}) + "\n")
    try:
        scenarios = [run_case(args.binary.resolve(), args.output / name, resize)
                     for name, resize in (("fixed-size", False), ("resize", True))]
    except Exception as exc:
        report_path.write_text(json.dumps({"status": "failed", "error": str(exc)}, indent=2) + "\n")
        raise
    report = {"status": "passed", "scenarios": scenarios}
    report_path.write_text(json.dumps(report, indent=2) + "\n")
    print(json.dumps({"status": "passed", "scenarios": len(scenarios), "output": str(args.output)}))


if __name__ == "__main__":
    main()
