"""Replay/coverage/negative-control gate for the real Rust save protocol simulator."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import time
from evidence import fingerprint


def check(binary, output, runs=1000, steps=200):
    output.mkdir(parents=True, exist_ok=True)
    source = fingerprint()
    command = [str(binary.resolve()), '--seed', '0', '--runs', str(runs), '--steps', str(steps)]
    started = time.perf_counter()
    first = subprocess.run(command, capture_output=True, timeout=300)
    second = subprocess.run(command, capture_output=True, timeout=300)
    if first.returncode or second.returncode:
        (output/'failure.json').write_bytes(first.stdout if first.returncode else second.stdout)
        raise AssertionError(f'simulation failed; replay seed in {output}/failure.json; {first.stderr.decode()}')
    assert first.stdout == second.stdout, 'identical seeds produced different traces'
    summary = json.loads(first.stdout)
    for key in ['process_crash', 'power_loss', 'lock_busy', 'external_edit']:
        assert summary['counts'].get(key, 0) > 0, f'unexercised fault: {key}'
    for phase in ['Prepare', 'Lock', 'Validate', 'CaptureBefore', 'Replace', 'SyncDirectory', 'CaptureAfter', 'Unlock']:
        assert summary['counts'].get('fault/'+phase, 0) > 0, f'unexercised failure phase: {phase}'
    assert summary['counts']['recovery_passed'] == runs
    controls = {}
    for mutation in ['skip-validation', 'skip-sync', 'skip-replacement']:
        run = subprocess.run([str(binary.resolve()), '--seed', '0', '--runs', '32', '--steps', str(steps), '--mutation', mutation], capture_output=True, timeout=60)
        assert run.returncode == 1, f'oracle failed to detect {mutation}'
        failure = json.loads(run.stdout)
        assert failure['failures']
        (output/f'{mutation}.json').write_text(json.dumps(failure, indent=2)+'\n')
        controls[mutation] = {'seed': failure['seed'], 'detected': failure['failures'], 'trace_sha256': failure['trace_sha256']}
    trace_path = output/'seed-0.json'
    trace_run = subprocess.run([str(binary.resolve()), '--seed', '0', '--steps', str(steps), '--trace', str(trace_path)], capture_output=True, check=True, timeout=60)
    assert json.loads(trace_run.stdout)['trace_sha256'][0] == summary['trace_sha256'][0]
    assert source == fingerprint(), 'source changed during simulation'
    report = {'source': source, 'binary_sha256': hashlib.sha256(binary.read_bytes()).hexdigest(),
              'elapsed_seconds_including_replay_and_controls': time.perf_counter()-started,
              'identical_replay': True, 'negative_controls': controls, 'simulation': summary,
              'scope': 'Production save protocol with simulated I/O outcomes and virtual time. Native filesystem/SQLite internals and arbitrary non-cooperating post-validation edits are outside the model; no network service exists.'}
    (output/'report.json').write_text(json.dumps(report, indent=2)+'\n')
    print(f'{runs} seeds x {steps} scheduled steps, exact replay, recovery and all negative controls passed')
    return report


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--runs', type=int, default=1000)
    parser.add_argument('--steps', type=int, default=200)
    args = parser.parse_args()
    check(args.binary, args.output, args.runs, args.steps)
