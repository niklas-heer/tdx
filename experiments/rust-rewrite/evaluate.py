"""Correctness-gated full-application Go/Rust comparison; timings are observations."""
import argparse
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import platform
import shutil
import sqlite3
import statistics
import subprocess
import sys
import tempfile
import threading
import time

from contracts import ROOT, Terminal, cli_contracts, replay_contracts, terminal_contracts, pty_module
from evidence import fingerprint
from parity_contracts import compare as action_parity
from application_contracts import compare as application_parity
from history_contracts import history_contracts, recovery_terminal, verify_snapshots

OUT = ROOT / "dist/rust-rewrite"
MANIFEST = ROOT / "experiments/rust-rewrite/Cargo.toml"


def command(args, env=None):
    return subprocess.check_output(args, cwd=ROOT, env=env, text=True, timeout=600).strip()


def build(args, env, label):
    started = time.perf_counter()
    result = subprocess.run(args, cwd=ROOT, env=env, capture_output=True, text=True, timeout=600)
    (OUT / f"{label}.log").write_text(result.stdout + result.stderr)
    result.check_returncode()
    return time.perf_counter() - started


def process(args, env):
    with tempfile.TemporaryFile() as stdout, tempfile.TemporaryFile() as stderr:
        started = time.perf_counter()
        child = subprocess.Popen([str(a) for a in args], cwd=ROOT, env=env, stdout=stdout, stderr=stderr)
        timer = threading.Timer(15, child.kill)
        timer.start()
        try:
            _, status, usage = os.wait4(child.pid, 0)
            child.returncode = os.waitstatus_to_exitcode(status)
        finally:
            timer.cancel()
            timer.join()
        elapsed = time.perf_counter() - started
        stdout.seek(0); stderr.seek(0)
        output, error = stdout.read(), stderr.read().decode()
        if child.returncode:
            raise RuntimeError(f"{args}: {error}")
        rss = usage.ru_maxrss if sys.platform == "darwin" else usage.ru_maxrss * 1024
        return output, {"wall_ms": elapsed * 1000, "cpu_ms": (usage.ru_utime + usage.ru_stime) * 1000, "rss_mib": rss / 1048576}


def summary(trials):
    return {key: {"median": statistics.median(t[key] for t in trials), "min": min(t[key] for t in trials),
                  "max": max(t[key] for t in trials)} for key in trials[0]}


def source(count):
    return "# Project\n\n" + "".join(f"- [ ] Task {i} café #work !p2 @due(2027-01-12)\n" for i in range(count))


def terminal_timings(binaries, output, rounds=3, actions=20):
    """Key send -> observable file replacement, including polling/parent read overhead.

    This is not key-to-frame or durable-save completion. A short untimed drain
    after every action allows terminal rendering/directory sync to finish.
    """
    records = []
    output.mkdir(parents=True, exist_ok=True)
    for count, operation in ((count, op) for count in (100, 1000, 10000) for op in ("toggle", "distinct-edit")):
        for trial in range(rounds):
            names = list(binaries) if trial % 2 == 0 else list(reversed(binaries))
            for name in names:
                with tempfile.TemporaryDirectory(prefix="tdx-live-timing-") as tmp:
                    base = Path(tmp); path = base / "tasks.md"
                    initial = source(count).encode(); path.write_bytes(initial)
                    terminal = Terminal(binaries[name], path, base / "config")
                    try:
                        terminal.until(lambda: b"Task 0" in terminal.output, "timing initial render")
                        terminal.pump(0.1)
                        rss_start = int(command(["ps", "-o", "rss=", "-p", str(terminal.pid)])) / 1024
                        trials = []; contents = [initial]; current = initial
                        for i in range(actions):
                            if operation == "toggle":
                                expected = initial.replace(b"[ ] Task 0", b"[x] Task 0", 1) if i % 2 == 0 else initial
                                keys = b" "
                            else:
                                lines = current.splitlines(keepends=True)
                                suffix = f" edit-{i}".encode()
                                lines[2] = lines[2].rstrip(b"\n") + suffix + b"\n"
                                expected = b"".join(lines)
                                keys = b"e\x1b[200~" + suffix + b"\x1b[201~\r"
                            started = time.perf_counter(); terminal.send(keys)
                            while path.read_bytes() != expected:
                                terminal.pump(0.001)
                                if time.perf_counter() - started > 10:
                                    raise AssertionError(f"{name}: toggle did not persist")
                            trials.append({"observed_save_ms": (time.perf_counter() - started) * 1000})
                            terminal.pump(0.05)
                            contents.append(expected); current = expected
                        rss_end = int(command(["ps", "-o", "rss=", "-p", str(terminal.pid)])) / 1024
                        terminal.close()
                        config = base / "config"
                        versions = verify_snapshots(config, path, contents)
                        with sqlite3.connect(config / "tdx/versions.sqlite") as db:
                            compressed_bytes = db.execute("SELECT sum(length(content)) FROM file_versions").fetchone()[0]
                        records.append({"engine": name, "tasks": count, "operation": operation, "trial": trial, "actions": actions,
                                        "history_versions": versions, "compressed_snapshot_bytes": compressed_bytes,
                                        "database_bytes": (config / "tdx/versions.sqlite").stat().st_size,
                                        "rss_start_mib": rss_start, "rss_end_mib": rss_end,
                                        "session_cpu_ms": (terminal.usage.ru_utime + terminal.usage.ru_stime) * 1000,
                                        "session_peak_rss_mib": terminal.usage.ru_maxrss / (1048576 if sys.platform == "darwin" else 1024),
                                        "summary": summary(trials), "samples": trials})
                    finally:
                        terminal.cleanup(output / f"{name}-{count}-{operation}-{trial}.ansi")
    return records



def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--trials", type=int, default=9)
    parser.add_argument("--skip-clean-build", action="store_true", help="development iteration only; omit for a published baseline")
    args = parser.parse_args()
    if args.trials < 3 or sys.platform not in ("darwin", "linux"):
        raise SystemExit("Requires macOS/Linux and at least three trials")
    OUT.mkdir(parents=True, exist_ok=True)
    binaries = {"go": OUT / "tdx-go", "rust": OUT / "tdx-rust"}
    report = {"schema": 3, "measured_at": datetime.now(timezone.utc).isoformat(), "host": platform.platform(),
              "cpu": command(["sysctl", "-n", "machdep.cpu.brand_string"]) if sys.platform == "darwin" else platform.processor(),
              "toolchains": {"go": command(["go", "version"]), "rust": command(["rustc", "--version"]), "python": platform.python_version()},
              "git_parent": command(["git", "rev-parse", "HEAD"]), "working_tree_dirty": bool(command(["git", "status", "--porcelain"])),
              "source": fingerprint(), "trials": args.trials, "build_seconds": {}, "cli": [],
              "limits": ["Full application comparison with matching tested document, CLI, TUI, configuration, recent-file and history services; not an isolated language comparison.",
                         "list --json has equivalent tested output; Go still initializes configuration/styles.",
                         "Both writers capture canonical-path SHA256/zstd SQLite history with retention100, WAL/NORMAL and shutdown checkpoint; library and UI implementations differ.",
                         "Go uses pure-Go SQLite/zstd; Rust uses bundled native SQLite/zstd C libraries. Both load themes and user configuration.",
                         "CLI wall time includes process startup/output capture; RSS is whole-child peak, not live heap.",
                         "PTY measures key-to-observed-replacement, not render latency or completed durable save; includes parent polling/read overhead.",
                         "Fresh builds exclude downloads; Go compiles stdlib, Rust ships precompiled stdlib; single build observations.",
                         "Timing observations describe this host only; native platform correctness is recorded separately."]}
    def save(): (OUT / "full-parity-results.json").write_text(json.dumps(report, indent=2) + "\n")
    command(["go", "mod", "download"])
    command(["cargo", "fetch", "--locked", "--manifest-path", str(MANIFEST)])
    go_build = ["go", "build", "-trimpath", "-ldflags=-s -w", "-o", str(binaries["go"]), "./cmd/tdx"]
    rust_build = ["cargo", "build", "--release", "--locked", "--manifest-path", str(MANIFEST), "--bin", "tdx-rust"]
    if args.skip_clean_build:
        env = {**os.environ, "CARGO_TARGET_DIR": str(OUT / "target")}
        build(go_build, os.environ, "go-build"); build(rust_build, env, "rust-build")
        shutil.copy2(OUT / "target/release/tdx-rust", binaries["rust"])
    else:
        print("Measuring fresh-cache and no-op builds (dependencies already downloaded).", flush=True)
        with tempfile.TemporaryDirectory(prefix="build-", dir=OUT) as tmp:
            go_env = {**os.environ, "GOCACHE": str(Path(tmp) / "go")}
            rust_env = {**os.environ, "CARGO_TARGET_DIR": str(Path(tmp) / "rust")}
            report["build_seconds"] = {"go_fresh_cache": build(go_build, go_env, "go-cold"),
                                       "rust_fresh_cache": build(rust_build, rust_env, "rust-cold"),
                                       "go_noop": build(go_build, go_env, "go-noop"),
                                       "rust_noop": build(rust_build, rust_env, "rust-noop")}
            shutil.copy2(Path(tmp) / "rust/release/tdx-rust", binaries["rust"])
    report["binary_bytes"] = {name: binary.stat().st_size for name, binary in binaries.items()}
    report["binary_sha256"] = {name: hashlib.sha256(binary.read_bytes()).hexdigest() for name, binary in binaries.items()}
    harness = OUT / "tdx-usage"
    build(["go", "build", "-o", str(harness), "./cmd/tdx-usage"], os.environ, "harness-build")
    adapter = OUT / "go-history"
    build(["go", "build", "-o", str(adapter), "./experiments/rust-rewrite/go-history"], os.environ, "history-adapter-build")
    go_adapter = OUT / "go-parity"
    rust_adapter = OUT / "target/release/parity"
    build(["go", "build", "-o", str(go_adapter), "./experiments/rust-rewrite/go-parity"], os.environ, "go-parity-build")
    build(["cargo", "build", "--release", "--locked", "--manifest-path", str(MANIFEST), "--bin", "parity"], {**os.environ, "CARGO_TARGET_DIR": str(OUT / "target")}, "rust-parity-build")
    actions = action_parity(go_adapter, rust_adapter, OUT / "verified-actions.json")
    application = application_parity(binaries, OUT / "verified-application.json")
    assert not actions["failures"] and not application["failures"]
    print("Gating measurements on CLI, identical replay, interoperable history and PTY recovery.", flush=True)
    report["correctness"] = {"document_actions": {"cases": actions["cases"], "steps": actions["steps"]}, "application": application, "fixtures": cli_contracts(binaries),
                             "replays": replay_contracts(binaries, harness, OUT / "verified-replay"),
                             "terminal": terminal_contracts(binaries, OUT / "verified-terminal"),
                             "history": history_contracts(binaries, adapter),
                             "recovery": recovery_terminal(binaries["rust"], OUT / "verified-recovery")}
    for name, binary in binaries.items():
        dest = OUT / "verified-full-terminal" / name; dest.mkdir(parents=True, exist_ok=True)
        report["correctness"][f"full_terminal_{name}"] = pty_module.run(binary, dest)
    save()
    print("Correctness passed. Measuring alternating CLI trials.", flush=True)
    with tempfile.TemporaryDirectory(prefix="tdx-cli-timing-") as tmp:
        base = Path(tmp)
        for count in (0, 100, 1000, 10000):
            path = base / "tasks.md"; original = source(count).encode()
            commands = {"list-json": ["list", "--json"]}
            if count: commands.update({"toggle-with-save": ["toggle", "1"], "edit-with-save": ["edit", "1", "Changed café #next !p1"]})
            for operation, cli in commands.items():
                records = {name: [] for name in binaries}
                expected_output = None
                expected_versions = {name: [original] for name in binaries}
                for trial in range(-2, args.trials):
                    names = list(binaries) if trial % 2 == 0 else list(reversed(binaries))
                    for name in names:
                        path.write_bytes(original)
                        config = base / f"{name}-{count}-{operation}"
                        (config / "tdx").mkdir(parents=True, exist_ok=True)
                        (config / "tdx/config.toml").write_text("[versioning]\nmax_versions=100\n")
                        env = {**os.environ, "XDG_CONFIG_HOME": str(config)}
                        if operation == "edit-with-save":
                            cli = ["edit", "1", f"Changed café #next !p1 trial{trial}"]
                        output, result = process([binaries[name], "--file", path, *cli], env)
                        if operation == "list-json":
                            actual = json.loads(output)
                            if expected_output is None: expected_output = actual
                            assert actual == expected_output
                        elif operation == "toggle-with-save":
                            assert path.read_bytes() == original.replace(b"[ ] Task 0", b"[x] Task 0", 1)
                        else:
                            got, _ = process([binaries[name], "--file", path, "list", "--json"], env)
                            assert json.loads(got)[0]["text"] == f"Changed café #next !p1 trial{trial}"
                        if operation != "list-json":
                            expected_versions[name].append(path.read_bytes())
                        if trial >= 0: records[name].append(result)
                for name in binaries:
                    config = base / f"{name}-{count}-{operation}"
                    if operation == "list-json":
                        assert not (config / "tdx/versions.sqlite").exists()
                    else:
                        verify_snapshots(config, path, expected_versions[name])
                report["cli"].extend({"engine": name, "tasks": count, "operation": operation, "summary": summary(samples), "samples": samples}
                                     for name, samples in records.items())
                save()
    print("Measuring persistent terminal sessions (20 toggles and 20 distinct edits, three sessions per workload).", flush=True)
    report["terminal"] = terminal_timings(binaries, OUT / "timing-terminal")
    report["source_unchanged_during_run"] = report["source"] == fingerprint()
    assert report["source_unchanged_during_run"], "source changed while measuring; rerun"
    save()
    print(f"Wrote {OUT / 'full-parity-results.json'}", flush=True)


if __name__ == "__main__":
    main()
