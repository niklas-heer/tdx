"""Cross-language history contracts using Go's production versioning package."""
import hashlib
import json
import os
from pathlib import Path
import sqlite3
import subprocess
import tempfile


def snapshots(config, path):
    with sqlite3.connect(config / 'tdx/versions.sqlite') as db:
        return db.execute('SELECT v.id, v.version_hash FROM file_versions v JOIN files f ON f.id=v.file_id WHERE f.file_path=? ORDER BY v.id DESC', (str(path.resolve()),)).fetchall()


def verify_snapshots(config, path, contents):
    rows = snapshots(config, path)
    assert {row[1] for row in rows} == {hashlib.sha256(c).hexdigest() for c in contents}, rows
    assert len(rows) == len(set(contents)), rows
    wal = config / 'tdx/versions.sqlite-wal'
    assert not wal.exists() or wal.stat().st_size == 0, 'shutdown did not truncate history WAL'
    return len(rows)


def history_contracts(binaries, adapter):
    from contracts import invoke
    results = []
    with tempfile.TemporaryDirectory(prefix='tdx-history-contract-') as tmp:
        base = Path(tmp); path = base / 'tasks.md'; config = base / 'shared'
        original = '# café 🦀\r\n\r\n- [ ] One\r\n'.encode()
        path.write_bytes(original)
        for binary in binaries.values():
            for args in [('list', '--json'), ('--help',), ('--version',)]:
                invoke(binary, path, config, *args)
                assert not (config / 'tdx/versions.sqlite').exists()
        env = {**os.environ, 'XDG_CONFIG_HOME': str(config)}
        def go(*args):
            return subprocess.check_output([str(adapter), *map(str, args)], env=env, timeout=15)
        invoke(binaries['rust'], path, config, 'toggle', '1')
        checked = path.read_bytes()
        for version, digest in snapshots(config, path):
            content = go('read', path, version)
            assert hashlib.sha256(content).hexdigest() == digest
            assert content in (original, checked)
        external = '- [ ] Go snapshot 🦀\r\n'.encode(); path.write_bytes(external)
        go('capture', path)
        versions = json.loads(invoke(binaries['rust'], path, config, 'versions', '--json').stdout)
        assert len(versions) == 3
        assert invoke(binaries['rust'], path, config, 'show-version', str(versions[0]['id'])).stdout == external
        alias = base / 'alias.md'; alias.symlink_to(path)
        invoke(binaries['rust'], alias, config, 'restore', str(versions[-1]['id']))
        assert path.read_bytes() == original and alias.is_symlink()
        verify_snapshots(config, path, [original, checked, external])
        with sqlite3.connect(config / 'tdx/versions.sqlite') as db:
            assert db.execute('select count(*) from files').fetchone()[0] == 1
        results.append({'cross_language_read': 'Go reads Rust zstd; Rust reads Go zstd; hashes and exact CRLF/Unicode bytes match', 'canonical_alias_and_restore': 'passed'})
        for name, binary in binaries.items():
            for limit in (3, 0):
                config = base / f'{name}-{limit}'
                (config / 'tdx').mkdir(parents=True)
                (config / 'tdx/config.toml').write_text(f'[versioning]\nmax_versions={limit}\n')
                path.write_bytes(original); contents = [original]
                for i in range(6):
                    invoke(binary, path, config, 'edit', '1', f'Unique {i}')
                    contents.append(path.read_bytes())
                invoke(binary, path, config, 'edit', '1', 'Unique 5')
                verify_snapshots(config, path, contents[-limit:] if limit else contents)
                other = base / 'other.md'; other.write_bytes(original)
                invoke(binary, other, config, 'toggle', '1')
                verify_snapshots(config, path, contents[-limit:] if limit else contents)
                verify_snapshots(config, other, [original, original.replace(b'[ ]', b'[x]')])
                if name == 'rust':
                    foreign = snapshots(config, other)[0][0]
                    invoke(binary, path, config, 'restore', str(foreign), success=False)
                    assert path.read_bytes() == contents[-1]
                results.append({'engine': name, 'retention': limit, 'dedup_and_file_isolation': 'passed'})
    return results


def recovery_terminal(binary, output):
    from contracts import Terminal, invoke
    output.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='tdx-recovery-pty-') as tmp:
        base = Path(tmp); path = base / 'tasks.md'; config = base / 'config'
        original = b'- [ ] Original\n'; changed = b'- [x] Original\n'
        path.write_bytes(original)
        terminal = Terminal(binary, path, config)
        try:
            terminal.until(lambda: b'Original' in terminal.output, 'recovery initial render')
            terminal.send(b' '); terminal.until(lambda: path.read_bytes() == changed, 'recovery toggle')
            start = len(terminal.output); terminal.send(b':versions\r')
            terminal.until(lambda: b'[Enter]' in terminal.output[start:], 'recovery browser')
            start = len(terminal.output); terminal.send(b'j\r')
            terminal.until(lambda: b'[y/N' in terminal.output[start:], 'restore confirmation')
            terminal.send(b'n'); terminal.pump(0.1); assert path.read_bytes() == changed
            terminal.send(b'\ry'); terminal.until(lambda: path.read_bytes() == original, 'confirmed exact restore')
            # Hold the browser open across an external change: restore must fail without replacing disk.
            start = len(terminal.output); terminal.send(b'v')
            terminal.until(lambda: b'[Enter]' in terminal.output[start:], 'conflict browser')
            external = b'- [ ] Browser external\n'; path.write_bytes(external)
            terminal.pump(0.7); start = len(terminal.output); terminal.send(b'\ry')
            terminal.until(lambda: b'externally' in terminal.output[start:], 'restore conflict')
            assert path.read_bytes() == external
            terminal.send(b'\x1b'); start = len(terminal.output)
            terminal.until(lambda: hashlib.sha256(external).hexdigest() in {h for _, h in snapshots(config, path)}, 'idle external reload capture')
            start = len(terminal.output); terminal.send(b'e')
            terminal.until(lambda: b'EDIT' in terminal.output[start:], 'pending recovery edit')
            overwritten = b'- [ ] Overwritten but recoverable\n'; path.write_bytes(overwritten)
            terminal.pump(0.7); start = len(terminal.output); terminal.send(b' local\r')
            terminal.until(lambda: b':force-save' in terminal.output[start:], 'pending edit conflict')
            assert path.read_bytes() == overwritten
            terminal.send(b'\x1b'); terminal.pump(0.1)
            terminal.send(b':force-save\r')
            candidate = b'# Todos\n\n- [ ] Browser external local\n'
            terminal.until(lambda: path.read_bytes() == candidate, 'force save')
            terminal.close()
            verify_snapshots(config, path, [original, changed, external, overwritten, candidate])
            versions = json.loads(invoke(binary, path, config, 'versions', '--json').stdout)
            contents = [invoke(binary, path, config, 'show-version', str(v['id'])).stdout for v in versions]
            assert overwritten in contents
        finally:
            terminal.cleanup(output / 'recovery.ansi')
        terminal = Terminal(binary, path, config, readonly=True)
        try:
            terminal.until(lambda: b'READ' in terminal.output, 'read-only recovery')
            terminal.send(b'v'); terminal.until(lambda: b'[Enter]' in terminal.output, 'read-only browser')
            start = len(terminal.output); terminal.send(b'j\ry')
            terminal.until(lambda: b'disabled' in terminal.output[start:], 'read-only restore denied')
            assert path.read_bytes() == candidate
            terminal.send(b'\x1b'); terminal.pump(0.1)
            terminal.close()
        finally:
            terminal.cleanup(output / 'readonly-recovery.ansi')
    return {'confirmed_restore_cancel_conflict': 'passed', 'idle_reload_and_deferred_input': 'passed', 'force_save_captures_overwritten_disk': 'passed', 'read_only_restore': 'passed'}
