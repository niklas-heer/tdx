"""Executable and actual-terminal contracts shared by Go and Rust."""
import argparse
import importlib.util
import fcntl
import hashlib
import json
import os
from pathlib import Path
from history_contracts import history_contracts, recovery_terminal, snapshots
import subprocess
import tempfile
import time
import termios

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location("usage_pty", ROOT / "scripts/usage-pty.py")
pty_module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(pty_module)
Terminal = pty_module.Terminal


def invoke(binary, path, config, *args, success=True):
    result = subprocess.run([str(binary), "--file", str(path), *args], capture_output=True,
                            env={**os.environ, "XDG_CONFIG_HOME": str(config)}, timeout=10)
    if success:
        assert result.returncode == 0, (args, result.stderr.decode())
    else:
        assert result.returncode != 0, args
    return result


def cli_contracts(binaries):
    fixtures = json.loads((ROOT / "experiments/rust-eval/fixtures.json").read_text())
    results = []
    with tempfile.TemporaryDirectory(prefix="tdx-rust-contract-") as tmp:
        base = Path(tmp)
        for fixture in fixtures:
            outputs = {}
            source = fixture["source"].encode()
            for name, binary in binaries.items():
                path = base / f"{name}.md"
                config = base / name
                path.write_bytes(source)
                outputs[name] = json.loads(invoke(binary, path, config, "list", "--json").stdout)
                for index in range(len(fixture["markers"])):
                    path.write_bytes(source)
                    if fixture["name"] == "frontmatter":
                        invoke(binary, path, config, "toggle", str(index + 1), success=False)
                        assert path.read_bytes() == source
                    else:
                        invoke(binary, path, config, "toggle", str(index + 1))
                        after = path.read_bytes()
                        assert len(after) == len(source) and sum(a != b for a, b in zip(after, source)) == 1
                path.write_bytes(source)
                for args in [("toggle", "0"), ("toggle", "999999"), ("delete", "-1"), ("list", "--status", "invalid"), ("--read-only", "add", "bad")]:
                    invoke(binary, path, config, *args, success=False)
                    assert path.read_bytes() == source
            if "go" in outputs and "rust" in outputs:
                assert outputs["go"] == outputs["rust"], (fixture["name"], outputs)
            results.append({"fixture": fixture["name"], "query": "matched", "byte_preserving_toggle_and_rejections": "passed"})
        source = "- [ ] café #tag #tag #other !p0 !p10 !p2 @due(2027-02-29) @due(2028-02-29)\n- [x] done #tag\n"
        for args in [("list", "--json"), ("list", "--json", "--status", "open"), ("list", "--json", "--tag", "tag", "--tag", "other")]:
            outputs = []
            for name, binary in binaries.items():
                path = base / f"{name}.md"; path.write_text(source)
                outputs.append(json.loads(invoke(binary, path, base / name, *args).stdout))
            assert all(output == outputs[0] for output in outputs)
        # Both executables must honor the SAME Go-compatible advisory lock.
        lock_path = base / "locked.md"
        lock_path.write_text("- [ ] locked\n")
        config = base / "shared-config"
        locks = config / "tdx/locks"
        locks.mkdir(parents=True)
        digest = hashlib.sha256(str(lock_path.resolve()).encode()).hexdigest()
        with (locks / f"{digest}.lock").open("w") as lock:
            fcntl.flock(lock, fcntl.LOCK_EX)
            for binary in binaries.values():
                error = invoke(binary, lock_path, config, "toggle", "1", success=False)
                assert b"busy" in error.stderr
                assert lock_path.read_text() == "- [ ] locked\n"
        results.append({"shared_advisory_lock": "both reject busy target without modifying it"})

    return results


def terminal_contracts(binaries, output):
    output.mkdir(parents=True, exist_ok=True)
    results = []
    for name, binary in binaries.items():
        for readonly in (False, True):
            with tempfile.TemporaryDirectory(prefix="tdx-rust-pty-") as tmp:
                base = Path(tmp); path = base / "tasks.md"
                original = b"# Work\n\n<div>keep</div>\n\n- [ ] One\n- [x] Two\n"
                path.write_bytes(original)
                alias = base / "alias.md"; alias.symlink_to(path)
                terminal = Terminal(binary, alias, base / "config", readonly)
                label = f"{name}-{'readonly' if readonly else 'editing'}"
                try:
                    terminal.until(lambda: b"One" in terminal.output, "initial render")
                    for width in (24, 80, 120) * (1 if readonly else 10):
                        terminal.resize(width, 20)
                        terminal.send(b" ")
                        if readonly:
                            terminal.pump(0.1); assert path.read_bytes() == original
                        else:
                            terminal.until(lambda: path.read_bytes() == original.replace(b"[ ] One", b"[x] One"), "toggle exact bytes")
                        terminal.send(b"u")
                        terminal.until(lambda: path.read_bytes() == original, "undo exact bytes")
                    if not readonly:
                        terminal.send(b"e\x1b[200~ caf\xc3\xa9 \xf0\x9f\xa6\x80\x1b[201~\r")
                        terminal.until(lambda: "One café 🦀" in path.read_text(), "Unicode edit")
                        terminal.send(b"u"); terminal.until(lambda: path.read_bytes() == original, "Unicode undo")
                        terminal.send(b"eignored\x1b"); terminal.pump(0.1)
                        assert path.read_bytes() == original
                        terminal.send(b"N" + b"\x1b[200~New task\x1b[201~\r")
                        terminal.until(lambda: b"New task" in path.read_bytes(), "add")
                        terminal.send(b"u"); terminal.until(lambda: path.read_bytes() == original, "undo add")
                        terminal.send(b"kkkjd")
                        terminal.until(lambda: b"Two" not in path.read_bytes(), "delete second task")
                        terminal.send(b"u"); terminal.until(lambda: path.read_bytes() == original, "undo delete")
                        terminal.send(b"kkke")
                        start = len(terminal.output)
                        terminal.until(lambda: b"EDIT" in terminal.output[start:], "pending edit")
                        external = original.replace(b"One", b"External")
                        path.write_bytes(external)
                        terminal.pump(0.8)  # Give Go's real watcher a tick during input.
                        terminal.send(b" local\r")
                        start = len(terminal.output)
                        marker = b":reload or :force-save" if name == "go" else b"externally"
                        terminal.until(lambda: marker in terminal.output[start:], "save conflict")
                        assert path.read_bytes() == external
                        terminal.send(b"\x1b"); terminal.pump(0.65)
                        terminal.send(b":reload\r")
                        terminal.until(lambda: hashlib.sha256(external).hexdigest() in {h for _, h in snapshots(base / "config", path)}, "reload captured external revision")
                        terminal.send(b" ")
                        terminal.until(lambda: path.read_bytes() == external.replace(b"[ ] External", b"[x] External"), "toggle reloaded revision")
                        terminal.send(b"u")
                        terminal.until(lambda: path.read_bytes() == external, "undo reloaded revision")
                        assert path.read_bytes() == external
                    assert alias.is_symlink()
                    terminal.close()
                    assert termios.tcgetattr(terminal.fd)[3] & termios.ICANON, "terminal raw mode was not restored"
                    results.append({"engine": name, "read_only": readonly, "status": "passed", "immediate_resize_toggle_undo_cycles": 3 if readonly else 30})
                finally:
                    terminal.cleanup(output / f"{label}.ansi")
    return results


def replay_contracts(binaries, harness, output, sessions=2, steps=100):
    reports = {}
    for name, binary in binaries.items():
        dest = output / name
        subprocess.run([str(harness), "-driver", "cli", "-binary", str(binary), "-sessions", str(sessions),
                        "-steps", str(steps), "-output", str(dest)], check=True, timeout=300)
        reports[name] = json.loads((dest / "report.json").read_text())
    assert [r["trace_sha256"] for r in reports["go"]] == [r["trace_sha256"] for r in reports["rust"]]
    return reports


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--go", type=Path, default=ROOT / "dist/rust-rewrite/tdx-go")
    parser.add_argument("--rust", type=Path, default=ROOT / "dist/rust-rewrite/target/release/tdx-rust")
    parser.add_argument("--harness", type=Path, default=ROOT / "dist/rust-rewrite/tdx-usage")
    parser.add_argument("--history-adapter", type=Path, default=ROOT / "dist/rust-rewrite/go-history")
    parser.add_argument("--output", type=Path, default=ROOT / "dist/rust-rewrite/contracts")
    args = parser.parse_args()
    binaries = {"go": args.go.resolve(), "rust": args.rust.resolve()}
    args.output.mkdir(parents=True, exist_ok=True)
    result = {"cli": cli_contracts(binaries), "replays": replay_contracts(binaries, args.harness.resolve(), args.output / "replay"),
              "terminal": terminal_contracts(binaries, args.output / "terminal"),
              "history": history_contracts(binaries, args.history_adapter.resolve()),
              "recovery": recovery_terminal(binaries["rust"], args.output / "recovery")}
    for name, binary in binaries.items():
        dest = args.output / "full-terminal" / name; dest.mkdir(parents=True, exist_ok=True)
        result[f"full_terminal_{name}"] = pty_module.run(binary, dest)
    (args.output / "report.json").write_text(json.dumps(result, indent=2) + "\n")
    print("CLI, replay, history interoperability and terminal recovery contracts passed.")


if __name__ == "__main__":
    main()
