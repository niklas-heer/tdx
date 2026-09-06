"""Native platform gate, including PTY/ConPTY and shared Go/Rust file locking."""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import platform
import select
import subprocess
import sys
import tempfile
import time

from evidence import fingerprint
from application_contracts import compare as application
from parity_contracts import compare as actions
ROOT = Path(__file__).resolve().parents[2]


class WindowsTerminal:
    def __init__(self, binary, path, config, readonly=False, context="", screen=None):
        from winpty import PtyProcess
        self.screen = screen
        if screen is not None:
            import pyte
            self.stream = pyte.ByteStream(screen)
        self.output = bytearray()
        args = [str(binary), '--file', str(path)] + (['--read-only'] if readonly else [])
        if context:
            # Keep ConPTY's root process alive until the application exits.
            # Windows os.execv terminates that root instead of replacing it.
            args = [sys.executable, '-c', 'import subprocess, sys; print(sys.argv[1], end="", flush=True); raise SystemExit(subprocess.call(sys.argv[2:]))', context, *args]
        self.proc = PtyProcess.spawn(args, env={**os.environ, 'XDG_CONFIG_HOME': str(config), 'TERM': 'xterm-256color', 'PYWINPTY_BACKEND': '0'}, dimensions=(32, 100), backend=0)
    def send(self, data): self.proc.write(data.decode('utf-8'))
    def resize(self, width, height):
        if self.screen is not None: self.screen.resize(lines=height, columns=width)
        self.proc.setwinsize(height, width)
    def pump(self, seconds=0.05):
        end = time.monotonic() + seconds
        while time.monotonic() < end:
            if select.select([self.proc.fileobj], [], [], min(.05, max(0, end-time.monotonic())))[0]:
                try:
                    data = self.proc.read(65536).encode()
                    self.output.extend(data)
                    if self.screen is not None: self.stream.feed(data)
                except EOFError: break
    def until(self, predicate, label):
        end = time.monotonic() + 15
        read_error = None
        while time.monotonic() < end:
            self.pump()
            try:
                if predicate(): return
            except PermissionError as error:
                # A polling read can overlap Windows atomic replacement. Retry
                # within the same deadline; persistent access errors still fail.
                read_error = error
        raise AssertionError(f'{label}: last read error={read_error!r}; {self.output[-800:]!r}')
    def close(self):
        self.send(b'\x04')
        self.until(lambda: not self.proc.isalive(), 'clean terminal shutdown')
        assert self.proc.exitstatus == 0
    def cleanup(self, transcript):
        transcript.write_bytes(self.output)
        self.proc.close(force=True)


def terminal_type():
    if os.name == 'nt': return WindowsTerminal
    spec = importlib.util.spec_from_file_location('usage_pty', ROOT / 'scripts/usage-pty.py')
    module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module)
    return module.Terminal


def terminal_gate(binaries, output):
    Terminal = terminal_type()
    results = []
    for name, binary in binaries.items():
        with tempfile.TemporaryDirectory(prefix='tdx-native-terminal-') as tmp:
            base = Path(tmp); path = base / 'tasks.md'; config = base / 'config'
            (config / 'tdx').mkdir(parents=True)
            (config / 'tdx/config.toml').write_text('[versioning]\nmax_versions=100\n')
            original = b'# Work\n\n- [ ] Alpha\n  - [ ] Child\n- [x] Beta\n- [ ] [Guide](https://ratatui.rs)\n'
            path.write_bytes(original)
            term = Terminal(binary, path, config)
            try:
                term.until(lambda: b'Alpha' in term.output, 'initial render')
                term.until(lambda: b'\x1b]8;;https://ratatui.rs' in term.output, 'clickable terminal hyperlink')
                for width in (24, 80, 120):
                    term.resize(width, 24); term.send(b' ')
                    term.until(lambda: path.read_bytes() == original.replace(b'[ ] Alpha', b'[x] Alpha'), 'resize toggle')
                    term.send(b'u'); term.until(lambda: path.read_bytes() == original, 'undo')
                term.send('e café 🦀\r'.encode())
                term.until(lambda: 'Alpha café 🦀' in path.read_text(), 'Unicode keyboard edit')
                term.send(b'u'); term.until(lambda: path.read_bytes() == original, 'Unicode undo')
                term.send('e \x1b[200~paste 🦀\x1b[201~\r'.encode())
                term.until(lambda: 'Alpha paste 🦀' in path.read_text(), 'bracketed Unicode paste')
                term.send(b'u'); term.until(lambda: path.read_bytes() == original, 'paste undo')
                term.send(b'mj\r')
                term.until(lambda: path.read_text().index('Beta') < path.read_text().index('Alpha'), 'nested move')
                term.send(b'u'); term.until(lambda: path.read_bytes() == original, 'nested undo')
                start = len(term.output); term.send(b'gge')
                term.until(lambda: b'EDIT' in term.output[start:], 'open input')
                external = original.replace(b'Alpha', b'External'); path.write_bytes(external)
                term.pump(.7); term.send(b' local\r')
                term.until(lambda: b':force-save' in term.output[start:], 'conflict visible')
                assert path.read_bytes() == external
                term.send(b'\x1b'); term.pump(.65); term.send(b':reload\r'); term.pump(.7)
                term.send(b' ')
                term.until(lambda: path.read_bytes() == external.replace(b'[ ] External', b'[x] External'), 'reload revision')
                term.close()
                results.append({'engine': name, 'resize_unicode_paste_move_undo_conflict_reload_shutdown': 'passed'})
            finally: term.cleanup(output / f'{name}-native.ansi')
    return results


def lock_gate(binaries, rust_adapter):
    with tempfile.TemporaryDirectory(prefix='tdx-native-lock-') as tmp:
        base = Path(tmp); path = (base / 'tasks.md').resolve(); path.write_text('- [ ] Locked\n')
        config = base / 'config'; locks = config / 'tdx/locks'; locks.mkdir(parents=True)
        digest = hashlib.sha256(str(path).encode()).hexdigest()
        child = subprocess.Popen([str(rust_adapter), '--hold-lock', str(locks / (digest + '.lock'))], stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True)
        try:
            assert child.stdout.readline().strip() == 'locked'
            for binary in binaries.values():
                result = subprocess.run([str(binary), '--file', str(path), 'toggle', '1'], env={**os.environ, 'XDG_CONFIG_HOME': str(config)}, capture_output=True, timeout=15)
                assert result.returncode != 0 and b'busy' in result.stderr, result.stderr
                assert path.read_text() == '- [ ] Locked\n'
        finally:
            child.communicate('\n', timeout=5)
    return 'both executables reject the same advisory lock without changing disk'


def main():
    p = argparse.ArgumentParser()
    for name in ['go', 'rust', 'go-adapter', 'rust-adapter']: p.add_argument('--'+name, type=Path, required=True)
    p.add_argument('--output', type=Path, required=True)
    args = p.parse_args(); args.output.mkdir(parents=True, exist_ok=True)
    binaries = {'go': args.go.resolve(), 'rust': args.rust.resolve()}
    a = actions(args.go_adapter.resolve(), args.rust_adapter.resolve(), args.output / 'actions.json')
    assert not a['failures']
    b = application(binaries, args.output / 'application.json')
    assert not b['failures']
    report = {'source': fingerprint(), 'platform': platform.platform(), 'python': platform.python_version(), 'action_cases': a['cases'], 'action_steps': a['steps'], 'application_workflows': len(b['passed']), 'shared_lock': lock_gate(binaries, args.rust_adapter.resolve()), 'terminal': terminal_gate(binaries, args.output), 'binary_sha256': {k: hashlib.sha256(v.read_bytes()).hexdigest() for k,v in binaries.items()}}
    from markdown_check import check as markdown_check
    report['rust_markdown'] = markdown_check(binaries['rust'], args.output)
    from inline_check import check as inline_check
    report['rust_inline'] = inline_check(binaries['rust'], args.output)
    (args.output / 'report.json').write_text(json.dumps(report, indent=2)+'\n')
    print(json.dumps(report, indent=2))

if __name__ == '__main__': main()
