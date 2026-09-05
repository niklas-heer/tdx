"""Portable executable parity: real CLI and complete scripted TUI workflows."""
import argparse
import json
import os
import random
from pathlib import Path
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[2]
SOURCE = '# Work\n\n- [ ] Alpha #work !p2 @due(2028-01-01)\n  - [ ] Child #home !p1\n- [x] Beta #home !p3\n- [ ] Gamma #work !p1\n\n## Next\n\n- [ ] Delta #other\n'
SCENARIOS = {
    'navigation-counts': '2j k G k gg j q',
    'nested-insert': 'nInserted café 🦀\rq',
    'append': 'NAppended\rq',
    'edit-cursor': 'e\x01new \x05\x7f!\rq',
    'unicode-paste': 'e\x1b[200~ café 🦀\x1b[201~\rq',
    'cancel-input': 'eignored\x1bq',
    'delete-parent': 'dq',
    'indent-outdent': 'G\t\x1b[Zq',
    'move-subtree': 'mj\rq',
    'move-cancel': 'mjj\x1bq',
    'move-undo': 'mj\ruq',
    'search-select': '/Gamma\r q',
    'search-unicode': '/Alpha\x1bq',
    'tag-filter': 't  q',
    'priority-filter': 'p  q',
    'due-filter': 'Djjj  q',
    'filter-done': ':filter-done\rj q',
    'sort-done': ':sort-done\rq',
    'sort-priority': ':sort-priority\rq',
    'sort-due': ':sort-due\rq',
    'bulk-check': ':check-all\rq',
    'bulk-uncheck': ':uncheck-all\rq',
    'clear-done': ':clear-done\rq',
    'readonly-explicit-save': ':read-only\r :save\rq',
    'readonly-ephemeral': ':read-only\r q',
    'sections-focus': 'sj\r q',
    'sections-rename': 'se\x01New \r\x1bq',
    'sections-create': 'snSibling\r\x1bq',
    'sections-subsection': 'sNSubsection\r\x1bq',
    'sections-fold-reset': 's \x1bS q',
    'section-add': 'sj\rNSection task\rq',
    'help': '?\x1b[B\x1b[6~\x1bq',
    'display-options': ':wrap\r:line-numbers\r:show-headings\r:set-max-visible\r5\rq',
    'theme-cancel': ':theme\r\x1b[B\x1bq',
    'theme-save': ':theme\r\x1b[A\rq',
    'recent-close': 'r\x1bq',
    'history-close': ':versions\r\x1bq',
    'reload': ':reload\rq',
}

for seed in range(12):
    rng = random.Random(seed)
    operations = ['j', 'k', 'gg', 'G', ' ', 'u', 'd', 'mj\r', 'mk\r', 'mj\x1b', 'nInserted\r', 'NAppended\r', 'e changed\r', '\t', '\x1b[Z']
    SCENARIOS[f'trace-{seed}'] = ''.join(rng.choice(operations) for _ in range(35)) + 'q'


def invoke(binary, path, env, args=(), keys=None):
    result = subprocess.run([str(binary), '--file', str(path), *args], env=env, input=keys, text=True, capture_output=True, timeout=20)
    assert result.returncode == 0, (binary, args, result.stderr)
    return result.stdout


def compare(binaries, output):
    failures, passed = [], []
    with tempfile.TemporaryDirectory(prefix='tdx-application-parity-') as tmp:
        base = Path(tmp)
        for name, keys in SCENARIOS.items():
            results = {}
            for engine, binary in binaries.items():
                directory = base / name / engine
                config = directory / 'config/tdx'
                config.mkdir(parents=True)
                (config / 'config.toml').write_text('[versioning]\nmax_versions=7\n[custom]\nkeep=true\n')
                path = directory / 'tasks.md'; path.write_text(SOURCE)
                env = {**os.environ, 'XDG_CONFIG_HOME': str(config.parent), 'HOME': str(directory), 'USERPROFILE': str(directory)}
                screen = invoke(binary, path, env, keys=keys)
                tasks = json.loads(invoke(binary, path, env, ['list', '--json']))
                if name.startswith('theme-'):
                    import tomllib
                    saved = tomllib.loads((config / 'config.toml').read_text())
                    assert saved['versioning']['max_versions'] == 7 and saved['custom']['keep'] is True
                    if name == 'theme-save': assert saved['theme']['name'] != 'tokyo-night'
                    else: assert 'theme' not in saved

                # Headings and unrelated blocks are verified separately by action contracts.
                results[engine] = {'tasks': tasks, 'source': path.read_text(), 'screen': screen, 'config': (config / 'config.toml').read_text()}
            if results['go']['tasks'] != results['rust']['tasks']:
                failures.append({'case': name, **results})
            else:
                passed.append(name)
        for engine, binary in binaries.items():
            directory = base / 'precedence' / engine
            config = directory / 'config/tdx'; config.mkdir(parents=True)
            (config / 'config.toml').write_text('[defaults]\nread_only=true\n[versioning]\nmax_versions=7\n')
            path = directory / 'tasks.md'
            env = {**os.environ, 'XDG_CONFIG_HOME': str(config.parent), 'HOME': str(directory), 'USERPROFILE': str(directory)}
            source = '---\nread-only: false\nunknown: retained\n---\n# Tasks\n\n- [ ] Allowed\n'
            path.write_text(source)
            invoke(binary, path, env, ['toggle', '1'])
            assert '[x]' in path.read_text(), engine
            path.write_text(source.replace('read-only: false', 'read-only: true'))
            rejected = subprocess.run([str(binary), '--file', str(path), 'toggle', '1'], env=env, capture_output=True)
            assert rejected.returncode != 0 and '[ ]' in path.read_text(), engine
            invoke(binary, path, env, ['recent'])
            invoke(binary, path, env, ['recent', 'clear'])
            assert not json.loads((config / 'recent.json').read_text()).get('files'), engine
            passed.append(engine + '/config-precedence-readonly-recent')
            (config / 'config.toml').write_text('[defaults]\nread_only=false\n')
            first = directory / 'first.md'
            second = directory / 'second.md'
            first.write_text('---\nread-only: true\nfilter-done: true\n---\n# First\n\n- [ ] First\n')
            second.write_text('# Second\n\n- [ ] Second\n')
            invoke(binary, second, env, keys='q')
            invoke(binary, first, env, keys='rsecond\r q')
            assert '[x] Second' in second.read_text(), (engine, 'file-switch settings leaked')
            assert '[ ] First' in first.read_text(), engine
            recent = json.loads((config / 'recent.json').read_text())
            assert any(f['path'] == str(second) and f['access_count'] >= 2 for f in recent['files']), (engine, recent)
            passed.append(engine + '/recent-switch-metadata-cursor')

    report = {'passed': passed, 'failures': failures}
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(report, indent=2) + '\n')
    print(f'{len(passed)} application workflows passed; {len(failures)} failed')
    for fail in failures: print(fail['case'])
    return report


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('--go', type=Path, default=ROOT / 'dist/rust-rewrite/tdx-go')
    p.add_argument('--rust', type=Path, default=ROOT / 'dist/rust-rewrite/target/debug/tdx-rust')
    p.add_argument('--output', type=Path, default=ROOT / 'dist/rust-rewrite/application.json')
    args = p.parse_args()
    raise SystemExit(bool(compare({'go': args.go.resolve(), 'rust': args.rust.resolve()}, args.output)['failures']))
