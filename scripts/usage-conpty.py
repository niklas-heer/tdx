#!/usr/bin/env python3
"""Windows ConPTY smoke: render, resize, edit, undo, manual-save isolation, exit.

Requires pywinpty==3.0.2. Runs on Windows only and explicitly selects ConPTY.
This does not claim parity with the POSIX suite's symlink/history/conflict checks
or validate clipboard integration with a desktop terminal.
"""
import argparse
import json
import os
from pathlib import Path
import select
import tempfile
import time


def run(binary, output):
    from winpty import PtyProcess
    from winpty.enums import Backend

    results = []
    for manual in (False, True):
        with tempfile.TemporaryDirectory(prefix="tdx-conpty-") as tmp:
            base = Path(tmp)
            path = base / "tasks.md"
            original = b"# Work\n\n- [ ] Task\n"
            path.write_bytes(original)
            env = dict(os.environ, XDG_CONFIG_HOME=str(base / "config"),
                       TERM="xterm-256color", PYWINPTY_BACKEND="0")
            args = [str(binary), "--file", str(path)]
            if manual:
                args.append("--read-only")
            # pywinpty 3.0.2 treats integer 0 as an omitted backend; its public
            # spawn adapter accepts the string form and then converts to int.
            process = PtyProcess.spawn(args, env=env, dimensions=(25, 100),
                                       backend=str(Backend.ConPTY))
            transcript = []
            queries = ""

            def pump():
                nonlocal queries
                # pywinpty forwards ConPTY output to a socket; fd is that
                # socket's fileno, supported by Windows select (not a pipe).
                if select.select([process.fd], [], [], 0.05)[0]:
                    try:
                        data = process.read(65536)
                    except EOFError:
                        return
                    transcript.append(data)
                    queries += data
                    for query, reply in (("\x1b[6n", "\x1b[1;1R"), ("\x1b[c", "\x1b[?1;2c")):
                        if query in queries:
                            process.write(reply)
                            queries = queries.replace(query, "")
                    queries = queries[-8:]

            def until(predicate, label):
                deadline = time.monotonic() + 10
                while time.monotonic() < deadline:
                    pump()
                    if predicate():
                        return
                raise AssertionError(f"timeout: {label}; output={''.join(transcript)[-600:]!r}")

            label = "manual-save" if manual else "editing"
            try:
                until(lambda: "Task" in "".join(transcript), "initial render")
                process.setwinsize(20, 40)
                process.write(" ")
                if not manual:
                    until(lambda: b"[x] Task" in path.read_bytes(), "checkbox saved")
                    process.write("u")
                    until(lambda: path.read_bytes() == original, "undo exact bytes")
                    process.write("e\x1b[200~ edited\x1b[201~\r")
                    until(lambda: b"Task edited" in path.read_bytes(), "text edited")
                    process.write("u")
                    until(lambda: path.read_bytes() == original, "edit undo")
                process.setwinsize(30, 120)
                process.write("\x04")
                until(lambda: not process.isalive(), "clean exit")
                assert process.exitstatus == 0, process.exitstatus
                assert path.read_bytes() == original, "unsaved edits changed disk"
                results.append({"scenario": label, "passed": True})
            finally:
                (output / f"{label}.ansi").write_text("".join(transcript), encoding="utf-8")
                process.close(force=True)
    return results


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path, default=Path("tdx.exe"))
    parser.add_argument("--output", type=Path, default=Path("dist/usage/conpty"))
    args = parser.parse_args()
    if os.name != "nt":
        parser.error("ConPTY smoke tests require a native Windows runner")
    args.output.mkdir(parents=True, exist_ok=True)
    report = {"backend": "ConPTY", "status": "running"}
    try:
        report["scenarios"] = run(args.binary.resolve(), args.output)
        report["status"] = "passed"
    except Exception as exc:
        report.update(status="failed", error=str(exc))
        raise
    finally:
        (args.output / "report.json").write_text(json.dumps(report, indent=2)+"\n")
    print(json.dumps(report, indent=2))


if __name__ == "__main__":
    main()
