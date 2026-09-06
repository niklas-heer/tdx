#!/usr/bin/env python3
"""Real terminal contracts for a tdx-compatible executable (macOS/Linux).

Uses isolated files/config/history, bounded observable waits and actual PTYs.
This is a short terminal suite, not an accelerated-hours accounting mechanism.
"""
import argparse
import errno
import fcntl
import hashlib
import json
import os
from pathlib import Path
import pty
import select
import signal
import sqlite3
import struct
import tempfile
import termios
import time


class Terminal:
    def __init__(self, binary, path, config, readonly=False, context="", screen=None):
        self.screen = screen
        if screen is not None:
            import pyte
            self.stream = pyte.ByteStream(screen)
            screen.write_process_input = lambda reply: self.send(reply.encode())
        self.output = bytearray()
        self.queries = bytearray()
        self.reaped = False
        config = Path(config)
        (config / "tdx").mkdir(parents=True, exist_ok=True)
        config_file = config / "tdx/config.toml"
        if not config_file.exists(): config_file.write_text("[versioning]\nmax_versions=100\n")
        self.pid, self.fd = pty.fork()
        if self.pid == 0:
            os.environ.update(XDG_CONFIG_HOME=str(config), TERM="xterm-256color")
            args = [str(binary), "--file", str(path)]
            if readonly:
                args.append("--read-only")
            os.write(1, context.encode())
            os.execv(str(binary), args)
        self.resize(100, 32)

    def resize(self, width, height):
        if self.screen is not None: self.screen.resize(lines=height, columns=width)
        fcntl.ioctl(self.fd, termios.TIOCSWINSZ, struct.pack("HHHH", height, width, 0, 0))
        os.kill(self.pid, signal.SIGWINCH)

    def send(self, data):
        os.write(self.fd, data)

    def pump(self, seconds=0.05):
        end = time.monotonic() + seconds
        while time.monotonic() < end:
            if not select.select([self.fd], [], [], min(0.05, max(0, end-time.monotonic())))[0]:
                continue
            try:
                data = os.read(self.fd, 65536)
            except OSError as exc:
                if exc.errno == errno.EIO:
                    return
                raise
            if not data:
                return
            self.output.extend(data)
            if self.screen is not None:
                self.stream.feed(data)
                continue
            self.queries.extend(data)
            for query, reply in [(b"\x1b[6n", b"\x1b[1;1R"), (b"\x1b[c", b"\x1b[?1;2c")]:
                while query in self.queries:
                    self.send(reply)
                    self.queries = self.queries.replace(query, b"", 1)
            self.queries = self.queries[-8:]

    def until(self, predicate, label, timeout=5):
        end = time.monotonic() + timeout
        while time.monotonic() < end:
            self.pump()
            if predicate():
                return
        raise AssertionError(f"timeout: {label}; terminal tail={self.output[-600:]!r}")

    def close(self):
        self.send(b"\x04")
        end = time.monotonic() + 5
        while time.monotonic() < end:
            self.pump()
            pid, status, usage = os.wait4(self.pid, os.WNOHANG)
            if pid:
                self.reaped = True
                self.usage = usage
                assert os.waitstatus_to_exitcode(status) == 0, status
                return
        raise AssertionError("terminal failed to exit")

    def cleanup(self, transcript):
        transcript.write_bytes(self.output)
        if not self.reaped:
            try:
                os.kill(self.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
            os.waitpid(self.pid, 0)
        os.close(self.fd)


def run(binary, output):
    results = []
    for readonly in (False, True):
        started = time.monotonic()
        with tempfile.TemporaryDirectory(prefix="tdx-pty-") as tmp:
            base = Path(tmp)
            path = base / "tasks.md"
            original = b"# Work\n\n<div>keep</div>\n\n9. [ ] Nine\n   - [x] Child\n"
            path.write_bytes(original)
            alias = base / "alias.md"
            alias.symlink_to(path)
            terminal = Terminal(binary, alias, base / "config", readonly)
            label = "readonly" if readonly else "editing"
            try:
                terminal.until(lambda: b"Nine" in terminal.output, "initial render")
                for width in (24, 80, 120):
                    terminal.resize(width, 20)
                    terminal.send(b" ")
                    if readonly:
                        terminal.pump(0.15)
                        assert path.read_bytes() == original, "read-only toggle wrote to disk"
                    else:
                        terminal.until(lambda: b"9. [x] Nine" in path.read_bytes(), "checkbox save")
                        assert path.read_bytes() == original.replace(b"[ ] Nine", b"[x] Nine"), "source changed"
                    terminal.send(b"u")
                    terminal.until(lambda: path.read_bytes() == original, "undo bytes")
                if not readonly:
                    terminal.send(b"e\x1b[200~ caf\xc3\xa9 \xf0\x9f\xa6\x80\x1b[201~\r")
                    terminal.until(lambda: "Nine café 🦀" in path.read_text(), "Unicode paste and save")
                    terminal.send(b"u")
                    terminal.until(lambda: path.read_bytes() == original, "Unicode undo")
                    # No debounce wait between typing the command and pressing Enter.
                    start = len(terminal.output)
                    terminal.send(b":versions\r")
                    terminal.until(lambda: b"[Enter]" in terminal.output[start:], "version browser via alias")
                    terminal.send(b"\x1b")
                    terminal.pump(0.65)
                    databases = list((base / "config").rglob("versions.sqlite"))
                    assert len(databases) == 1, databases
                    with sqlite3.connect(databases[0]) as db:
                        assert db.execute("select count(*) from file_versions").fetchone()[0] >= 3
                    start = len(terminal.output)
                    terminal.send(b"e")
                    terminal.until(lambda: b"EDIT" in terminal.output[start:], "edit mode before external write")
                    external = original.replace(b"Nine", b"External")
                    path.write_bytes(external)
                    terminal.pump(1.2)  # Exercise at least one real watch tick during input.
                    start = len(terminal.output)
                    terminal.send(b"\x1b[200~ local edit\x1b[201~\r")
                    terminal.until(lambda: b":force-save" in terminal.output[start:], "conflict after external edit during input")
                    assert path.read_bytes() == external, "save overwrote external edit"
                    terminal.send(b"\x1b")
                    terminal.pump(0.65)
                    start = len(terminal.output)
                    terminal.send(b":reload\r")
                    # Incremental terminal redraws may reuse letters from the prior frame.
                    # Verify the loaded revision through history and a subsequent edit.
                    def captured_external():
                        with sqlite3.connect(databases[0]) as db:
                            return db.execute("select count(*) from file_versions where version_hash=?", (hashlib.sha256(external).hexdigest(),)).fetchone()[0] == 1
                    terminal.until(captured_external, "reload external content")
                    terminal.send(b" ")
                    terminal.until(lambda: path.read_bytes() == external.replace(b"[ ] External", b"[x] External"), "edit reloaded revision")
                    terminal.send(b"u")
                    terminal.until(lambda: path.read_bytes() == external, "undo reloaded revision")
                    original = external
                terminal.close()
                assert path.read_bytes() == original, "exit changed file"
                results.append({"scenario": label, "passed": True, "wall_seconds": time.monotonic()-started})
            finally:
                terminal.cleanup(output / f"{label}.ansi")
    return results


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path, default=Path("tdx"))
    parser.add_argument("--output", type=Path, default=Path("dist/usage/pty"))
    args = parser.parse_args()
    args.output.mkdir(parents=True, exist_ok=True)
    report_path = args.output / "report.json"
    report_path.write_text(json.dumps({"status": "running"})+"\n")
    try:
        results = run(args.binary.resolve(), args.output)
    except Exception as exc:
        report_path.write_text(json.dumps({"status": "failed", "error": str(exc)}, indent=2)+"\n")
        raise
    report = {"status": "passed", "scenarios": results}
    report_path.write_text(json.dumps(report, indent=2)+"\n")
    print(json.dumps(report, indent=2))


if __name__ == "__main__":
    main()
