"""Compare full Go executable with the bounded Rust CLI/TUI; timings are observations."""
import argparse
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import platform
import shutil
import statistics
import subprocess
import sys
import tempfile
import threading
import time

from contracts import ROOT, Terminal, cli_contracts, replay_contracts, terminal_contracts

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
    for count in (100, 1000, 10000):
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
                        trials = []
                        for i in range(actions):
                            expected = initial.replace(b"[ ] Task 0", b"[x] Task 0", 1) if i % 2 == 0 else initial
                            started = time.perf_counter(); terminal.send(b" ")
                            while path.read_bytes() != expected:
                                terminal.pump(0.001)
                                if time.perf_counter() - started > 10:
                                    raise AssertionError(f"{name}: toggle did not persist")
                            trials.append({"observed_save_ms": (time.perf_counter() - started) * 1000})
                            terminal.pump(0.05)
                        rss_end = int(command(["ps", "-o", "rss=", "-p", str(terminal.pid)])) / 1024
                        terminal.close()
                        records.append({"engine": name, "tasks": count, "trial": trial, "actions": actions,
                                        "rss_start_mib": rss_start, "rss_end_mib": rss_end,
                                        "summary": summary(trials), "samples": trials})
                    finally:
                        terminal.cleanup(output / f"{name}-{count}-{trial}.ansi")
    return records


def fingerprint():
    paths = sorted(set(list((ROOT / "cmd").rglob("*.go")) + list((ROOT / "cmd/tdx/themes").glob("*.toml"))
                       + list((ROOT / "internal").rglob("*.go")) + list(MANIFEST.parent.glob("*.py"))
                       + list((MANIFEST.parent / "src").glob("*.rs"))
                       + [MANIFEST, MANIFEST.with_name("Cargo.lock"), ROOT / "go.mod", ROOT / "go.sum",
                          ROOT / "scripts/usage-pty.py", ROOT / "experiments/rust-eval/fixtures.json"]))
    return {"sha256": hashlib.sha256(b"".join(p.relative_to(ROOT).as_posix().encode() + b"\0" + p.read_bytes() for p in paths)).hexdigest(),
            "files": [p.relative_to(ROOT).as_posix() for p in paths]}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--trials", type=int, default=9)
    parser.add_argument("--skip-clean-build", action="store_true", help="development iteration only; omit for a published baseline")
    args = parser.parse_args()
    if args.trials < 3 or sys.platform not in ("darwin", "linux"):
        raise SystemExit("Requires macOS/Linux and at least three trials")
    OUT.mkdir(parents=True, exist_ok=True)
    binaries = {"go": OUT / "tdx-go", "rust": OUT / "tdx-rust"}
    report = {"schema": 1, "measured_at": datetime.now(timezone.utc).isoformat(), "host": platform.platform(),
              "cpu": command(["sysctl", "-n", "machdep.cpu.brand_string"]) if sys.platform == "darwin" else platform.processor(),
              "toolchains": {"go": command(["go", "version"]), "rust": command(["rustc", "--version"]), "python": platform.python_version()},
              "git_parent": command(["git", "rev-parse", "HEAD"]), "working_tree_dirty": bool(command(["git", "status", "--porcelain"])),
              "source": fingerprint(), "trials": args.trials, "build_seconds": {}, "cli": [],
              "limits": ["Full Go executable versus a feature-incomplete Rust prototype; not an isolated language comparison.",
                         "list --json has equivalent tested output; Go still initializes configuration/styles.",
                         "Writes and TUI include Go SQLite history; Rust has no history. These are unmatched services.",
                         "CLI wall time includes process startup/output capture; RSS is whole-child peak, not live heap.",
                         "PTY measures key-to-observed-replacement, not render latency or completed durable save; includes parent polling/read overhead.",
                         "Fresh builds exclude downloads; Go compiles stdlib, Rust ships precompiled stdlib; single build observations.",
                         "macOS/Linux runner; this snapshot validates one host only. No claim of Windows parity."]}
    def save(): (OUT / "results.json").write_text(json.dumps(report, indent=2) + "\n")
    command(["go", "mod", "download"])
    command(["cargo", "fetch", "--locked", "--manifest-path", str(MANIFEST)])
    go_build = ["go", "build", "-trimpath", "-ldflags=-s -w", "-o", str(binaries["go"]), "./cmd/tdx"]
    rust_build = ["cargo", "build", "--release", "--locked", "--manifest-path", str(MANIFEST)]
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
    print("Gating measurements on CLI, identical replay and basic PTY contracts.", flush=True)
    report["correctness"] = {"fixtures": cli_contracts(binaries),
                             "replays": replay_contracts(binaries, harness, OUT / "verified-replay"),
                             "terminal": terminal_contracts(binaries, OUT / "verified-terminal")}
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
                for trial in range(-2, args.trials):
                    names = list(binaries) if trial % 2 == 0 else list(reversed(binaries))
                    for name in names:
                        path.write_bytes(original)
                        env = {**os.environ, "XDG_CONFIG_HOME": str(base / name)}
                        output, result = process([binaries[name], "--file", path, *cli], env)
                        if operation == "list-json":
                            actual = json.loads(output)
                            if expected_output is None: expected_output = actual
                            assert actual == expected_output
                        elif operation == "toggle-with-save":
                            assert path.read_bytes() == original.replace(b"[ ] Task 0", b"[x] Task 0", 1)
                        else:
                            got, _ = process([binaries[name], "--file", path, "list", "--json"], env)
                            assert json.loads(got)[0]["text"] == "Changed café #next !p1"
                        if trial >= 0: records[name].append(result)
                report["cli"].extend({"engine": name, "tasks": count, "operation": operation, "summary": summary(samples), "samples": samples}
                                     for name, samples in records.items())
                save()
    print("Measuring persistent terminal sessions (20 toggles, three sessions per workload).", flush=True)
    report["terminal"] = terminal_timings(binaries, OUT / "timing-terminal")
    report["source_unchanged_during_run"] = report["source"] == fingerprint()
    assert report["source_unchanged_during_run"], "source changed while measuring; rerun"
    save()
    print(f"Wrote {OUT / 'results.json'}", flush=True)


if __name__ == "__main__":
    main()
