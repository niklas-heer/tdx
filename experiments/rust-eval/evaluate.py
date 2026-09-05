"""Reproducible, bounded parser experiment. Writes only ignored dist artifacts."""
from datetime import datetime, timezone
import hashlib
import json
import os
import platform
import shutil
import statistics
import subprocess
import sys
import tempfile
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
EXPERIMENT = Path(__file__).resolve().parent
OUT = ROOT / "dist" / "rust-eval"
MANIFEST = EXPERIMENT / "rust-probe" / "Cargo.toml"
TRIALS = 7


def command(args, env=None):
    return subprocess.check_output(args, cwd=ROOT, env=env, text=True).strip()


def build(args, env, label):
    started = time.perf_counter()
    result = subprocess.run(args, cwd=ROOT, env=env, capture_output=True, text=True, timeout=600)
    (OUT / f"{label}.log").write_text(result.stdout + result.stderr)
    result.check_returncode()
    return time.perf_counter() - started


def process(args, env=None):
    """wait4 returns CPU and peak RSS for this child, not a cumulative child maximum."""
    with tempfile.TemporaryFile() as stdout, tempfile.TemporaryFile() as stderr:
        started = time.perf_counter()
        child = subprocess.Popen([str(a) for a in args], cwd=ROOT, env=env, stdout=stdout, stderr=stderr)
        _, status, usage = os.wait4(child.pid, 0)
        child.returncode = os.waitstatus_to_exitcode(status)
        elapsed = time.perf_counter() - started
        stdout.seek(0)
        stderr.seek(0)
        out, error = stdout.read(), stderr.read().decode()
        rss_bytes = usage.ru_maxrss if sys.platform == "darwin" else usage.ru_maxrss * 1024
        return {"exit": child.returncode, "stdout": out.decode(), "stderr": error,
                "wall_ms": elapsed * 1000, "cpu_ms": (usage.ru_utime + usage.ru_stime) * 1000,
                "rss_mib": rss_bytes / (1024 * 1024)}


def invoke(binary, operation, path, n):
    result = process([binary, operation, path, n])
    if result["exit"]:
        raise RuntimeError(f"{operation}: {result['stderr']}")
    return result


def summarize(trials):
    keys = ["wall_ms", "cpu_ms", "rss_mib"]
    if "ns_per_op" in trials[0]:
        keys.append("ns_per_op")
    return {key: {"median": statistics.median(t[key] for t in trials),
                  "min": min(t[key] for t in trials), "max": max(t[key] for t in trials)} for key in keys}


def profile_rust(report):
    """Sample a separate symbolized release build; never time this build against Go."""
    if sys.platform != "darwin" or not Path("/usr/bin/sample").exists():
        return {"status": "unavailable", "reason": "Stack sampling currently uses macOS sample; CPU/RSS timings remain available."}
    env = {**os.environ, "CARGO_TARGET_DIR": str(OUT / "profile-target"),
           "CARGO_PROFILE_RELEASE_DEBUG": "1", "CARGO_PROFILE_RELEASE_STRIP": "none"}
    build(["cargo", "build", "--release", "--locked", "--manifest-path", str(MANIFEST)], env, "rust-profile-build")
    timing = next(b for b in report["benchmarks"] if b["engine"] == "rust" and b["tasks"] == 1000 and b["operation"] == "scan")
    iterations = max(1, int(8_000_000_000 / timing["summary"]["ns_per_op"]["median"]))
    args = [str(OUT / "profile-target" / "release" / "tdx-rust-probe"), "scan", str(OUT / "fixtures" / "generated-1000.md"), str(iterations)]
    ready = OUT / "rust-profile-ready"
    ready.unlink(missing_ok=True)
    with (OUT / "rust-profile-run.log").open("w") as log:
        child = subprocess.Popen(args, cwd=ROOT, stdout=log, stderr=log, env={**env, "TDX_PROBE_PROFILE_READY": str(ready)})
        try:
            deadline = time.monotonic() + 15
            while not ready.exists():
                if child.poll() is not None or time.monotonic() > deadline:
                    raise RuntimeError("Rust profile did not reach the measured loop")
                time.sleep(0.02)
            sampled = subprocess.run(["/usr/bin/sample", str(child.pid), "1", "1", "-file", str(OUT / "rust-sample.txt")],
                                     capture_output=True, text=True, timeout=15)
            child.wait(timeout=30)
        finally:
            if child.poll() is None:
                child.terminate()
                child.wait(timeout=5)
    (OUT / "rust-sample.log").write_text(sampled.stdout + sampled.stderr)
    return {"status": "captured" if sampled.returncode == 0 and child.returncode == 0 else "failed",
            "profiler": "macOS sample, 1 second at 1 ms", "operation": "scan", "tasks": 1000,
            "iterations": iterations, "symbols": "separate release build with debug=1 and strip=none"}


def main():
    if not hasattr(os, "wait4") or sys.platform not in ("darwin", "linux"):
        raise SystemExit("The RSS/CPU experiment currently supports macOS and Linux only.")
    OUT.mkdir(parents=True, exist_ok=True)
    fixtures = json.loads((EXPERIMENT / "fixtures.json").read_text())
    sources = (sorted((ROOT / "internal").rglob("*.go")) + sorted(EXPERIMENT.rglob("*.go"))
               + sorted(EXPERIMENT.rglob("*.rs")) + [ROOT / "go.mod", ROOT / "go.sum",
               MANIFEST, MANIFEST.with_name("Cargo.lock"), EXPERIMENT / "fixtures.json", Path(__file__)])
    fingerprint = hashlib.sha256(b"".join(p.relative_to(ROOT).as_posix().encode() + b"\0" + p.read_bytes() for p in sources)).hexdigest()
    report = {"measured_at": datetime.now(timezone.utc).isoformat(), "host": platform.platform(), "machine": platform.machine(), "cpu": platform.processor(),
              "go": command(["go", "version"]), "rust": command(["rustc", "--version"]),
              "python": platform.python_version(), "commit": command(["git", "rev-parse", "HEAD"]),
              "working_tree_dirty": bool(command(["git", "status", "--porcelain"])),
              "source_sha256": fingerprint, "trials": TRIALS, "correctness": [], "benchmarks": []}
    if sys.platform == "darwin":
        report["cpu"] = command(["sysctl", "-n", "machdep.cpu.brand_string"])
    command(["go", "mod", "download"])
    command(["cargo", "fetch", "--locked", "--manifest-path", str(MANIFEST)])
    print("Dependencies cached; measuring fresh build caches (Rust ships a prebuilt stdlib).", flush=True)
    # Fresh caches avoid silently comparing a warm Go build with a cold Rust build.
    with tempfile.TemporaryDirectory(prefix="build-", dir=OUT) as cache:
        env_go = {**os.environ, "GOCACHE": str(Path(cache) / "go")}
        env_rust = {**os.environ, "CARGO_TARGET_DIR": str(Path(cache) / "rust")}
        go = OUT / "go-probe"
        rust = OUT / "rust-probe"
        go_build = ["go", "build", "-trimpath", "-ldflags=-s -w", "-o", str(go), "./experiments/rust-eval/go-probe"]
        rust_build = ["cargo", "build", "--release", "--locked", "--manifest-path", str(MANIFEST)]
        report["build_seconds"] = {"go_fresh_cache": build(go_build, env_go, "go-cold"),
                                   "rust_fresh_cache": build(rust_build, env_rust, "rust-cold"),
                                   "go_noop": build(go_build, env_go, "go-warm"),
                                   "rust_noop": build(rust_build, env_rust, "rust-warm")}
        shutil.copy2(Path(cache) / "rust" / "release" / "tdx-rust-probe", rust)
    report["binary_bytes"] = {"go_probe": go.stat().st_size, "rust_probe": rust.stat().st_size}
    failure = False
    fixture_dir = OUT / "fixtures"
    fixture_dir.mkdir(exist_ok=True)
    print("Checking shared fixtures before timing.", flush=True)
    for fixture in fixtures:
        path = fixture_dir / (fixture["name"] + ".md")
        path.write_bytes(fixture["source"].encode())
        expected = fixture["markers"]
        record = {"name": fixture["name"], "markers": len(expected), "patch_cases": len(expected)}
        for name, binary, operation in [("go", go, "inspect"), ("rust", rust, "inspect"), ("production_go", go, "production-inspect")]:
            actual = json.loads(invoke(binary, operation, path, 0)["stdout"])
            record[name + "_markers_match"] = actual == expected
            if actual != expected:
                record[name + "_actual"] = actual
                failure = True
        record["matching_single_byte_patches"] = True
        record["production_toggle_semantics"] = True
        record["production_byte_preservation"] = True
        for index in range(len(expected)):
            outputs = [invoke(binary, "patch", path, index)["stdout"] for binary in (go, rust)]
            before, after = fixture["source"].encode(), outputs[0].encode()
            match = outputs[0] == outputs[1] and len(before) == len(after) and sum(a != b for a, b in zip(before, after)) == 1
            record["matching_single_byte_patches"] &= match
            failure |= not match
            result = invoke(go, "production", path, index)["stdout"]
            record["production_byte_preservation"] &= result == outputs[0]
            check_path = fixture_dir / "edited.md"
            check_path.write_bytes(result.encode())
            toggled = [dict(marker) for marker in expected]
            toggled[index]["checked"] = not toggled[index]["checked"]
            semantics = json.loads(invoke(go, "inspect", check_path, 0)["stdout"]) == toggled
            record["production_toggle_semantics"] &= semantics
            failure |= not semantics
        for name, binary in [("go", go), ("rust", rust)]:
            invalid = process([binary, "patch", path, len(expected)])
            record[name + "_rejects_invalid_index"] = invalid["exit"] != 0 and not invalid["stdout"]
            failure |= not record[name + "_rejects_invalid_index"]
        report["correctness"].append(record)
    (OUT / "results.json").write_text(json.dumps(report, indent=2) + "\n")
    if failure:
        raise SystemExit("Correctness gate failed; inspect dist/rust-eval/results.json. No timings compared.")
    print("Correctness gate passed. Timing matched scans and patches, plus production Go context.", flush=True)
    for count in (100, 1000, 10000):
        path = fixture_dir / f"generated-{count}.md"
        path.write_text("# Project\n\n" + "".join(f"- [ ] Task {i} #backend !p2 @due(2026-10-01)\n" for i in range(count)))
        for operation in ("scan", "patch-bench", "production-bench"):
            engines = [("go", go), ("rust", rust)] if operation != "production-bench" else [("production_go", go)]
            pilots = [json.loads(invoke(binary, operation, path, 1)["stdout"])["ns_per_op"] for _, binary in engines]
            iterations = max(1, min(5000, int(150_000_000 / max(pilots))))
            groups = {name: [] for name, _ in engines}
            for trial in range(TRIALS):
                # Alternate implementation order to limit systematic thermal/cache bias.
                order = engines if trial % 2 == 0 else list(reversed(engines))
                for name, binary in order:
                    result = invoke(binary, operation, path, iterations)
                    payload = json.loads(result.pop("stdout"))
                    result.pop("stderr")
                    result.update(payload)
                    groups[name].append(result)
            for name, trials in groups.items():
                report["benchmarks"].append({"engine": name, "operation": operation, "tasks": count,
                                              "iterations": iterations, "summary": summarize(trials), "samples": trials})
        print(f"Measured {count} tasks.", flush=True)
    empty = fixture_dir / "empty.md"
    report["startup"] = {name: summarize([invoke(binary, "inspect", empty, 0) for _ in range(9)]) for name, binary in [("go", go), ("rust", rust)]}
    profile = ["go", "test", "./internal/editor", "-run=^$", "-bench=BenchmarkDocumentToggle/1000$", "-benchtime=2s", "-benchmem",
               "-cpuprofile=" + str(OUT / "go.cpu"), "-memprofile=" + str(OUT / "go.heap"), "-o", str(OUT / "editor.test")]
    build(profile, os.environ.copy(), "go-profile")
    for mode, profile_path, flags in [("cpu", "go.cpu", []), ("alloc", "go.heap", ["-alloc_space"])]:
        (OUT / f"go-{mode}-top.txt").write_text(command(["go", "tool", "pprof", "-top", *flags, str(OUT / "editor.test"), str(OUT / profile_path)]) + "\n")
    report["rust_profile"] = profile_rust(report)
    report["limitations"] = ["Parser libraries/architectures differ (Go AST, Rust pull events); ratios are not isolated language effects.",
        "Rust only scans task markers and patches one checkbox; it has no TUI, metadata extraction, undo, history, safe-save, or conflict resolution.",
        "Production Go parses metadata and serializes the entire document; compare its timing as context, not feature-equivalent to a Rust rewrite.",
        "RSS includes runtime/process memory; Go heap profiles measure different quantities. Rust stack sampling is optional on macOS; no Rust heap allocation profile is collected.",
        "Fresh-cache builds exclude downloads; Go rebuilds its standard library whereas Rust distributes a precompiled standard library.",
        "Build times are single observations; runtime summaries use seven trials. Startup uses nine empty-file processes.",
        "One host and OS per run; this does not validate Rust behavior on other platforms or predict interactive latency."]
    (OUT / "results.json").write_text(json.dumps(report, indent=2) + "\n")
    print("Results and profiles: " + str(OUT), flush=True)


if __name__ == "__main__":
    main()
