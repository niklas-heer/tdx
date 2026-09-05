"""Differential action contracts against both production document implementations.

Formatting normalization is permitted for structural edits. Task metadata,
hierarchy, headings, rejection state and unrelated document content are compared.
Neither application uses this driver at runtime.
"""
import argparse
import json
from pathlib import Path
import random
import subprocess

ROOT = Path(__file__).resolve().parents[2]


def corpus():
    cases = []
    fixtures = json.loads((ROOT / 'experiments/rust-eval/fixtures.json').read_text())
    fixtures += [
        {'name': 'rich-inline', 'source': '# Work\n\n- [ ] **bold** *italic* ~~gone~~ `a & b` [link](https://example.com "title") ![alt](img) <a@b.com> &amp; \\*literal\\*\n  continuation\n'},
        {'name': 'nested-blocks', 'source': '# Work\n\n- [ ] parent !p3\n  - [x] child !p1\n    - [ ] grandchild\n\n  paragraph retained\n\n  1. [ ] ordered child\n- [ ] sibling !p1\n\n## Next\n\n- [x] end @due(2028-01-01)\n\n| a | b |\n| - | - |\n| c | d |\n\n<div>retained</div>\n'},
        {'name': 'ordinary-parent', 'source': '# Work\n\n- ordinary\n  - [ ] child\n- [ ] sibling\n'},
        {'name': 'empty', 'source': ''},
        {'name': 'unclosed-fence', 'source': '# Work\n\n```txt\nkeep literal\n'},
        {'name': 'quoted', 'source': '# Work\n\n> - [ ] quote\n>   - [ ] nested\n\n- [x] root\n'},
    ]
    for fixture in fixtures:
        source, name = fixture['source'], fixture['name']
        cases.append({'name': f'{name}/query', 'source': source, 'actions': []})
        for kind in ('add', 'set-all-checked', 'clear-done', 'sort-done', 'sort-priority', 'sort-due'):
            cases.append({'name': f'{name}/{kind}', 'source': source, 'actions': [{'kind': kind, 'text': 'new café #work !p2', 'checked': True}]})
        for kind in ('edit', 'delete', 'insert', 'toggle', 'indent', 'outdent'):
            for index in (0, 1, 2, -1, 99):
                cases.append({'name': f'{name}/{kind}/{index}', 'source': source, 'actions': [{'kind': kind, 'index': index, 'text': 'replacement 🦀 #work !p2'}]})
        for kind in ('move', 'move-to-position'):
            for index, target in ((0, 1), (1, 0), (0, 2), (2, 0), (0, 0), (0, 99)):
                cases.append({'name': f'{name}/{kind}/{index}-{target}', 'source': source, 'actions': [{'kind': kind, 'index': index, 'target': target, 'insert_after': True}]})
    sections = '# Root\n\n- [ ] one\n\n## Child\n\n- [ ] two\n\n# Other\n\nText\n'
    for kind in ('rename-heading', 'create-heading', 'add-in-section'):
        for index in (-1, 0, 1, 2, 3):
            cases.append({'name': f'sections/{kind}/{index}', 'source': sections, 'actions': [{'kind': kind, 'index': index, 'level': 2, 'text': 'New section'}]})
    # Fixed seeds, independent actions and long traces exercise index rebuilding.
    for seed in range(12):
        rng = random.Random(seed)
        actions = []
        for _ in range(60):
            kind = rng.choice(['add', 'insert', 'edit', 'toggle', 'delete', 'move', 'indent', 'outdent', 'sort-done', 'sort-priority', 'sort-due', 'set-all-checked'])
            actions.append({'kind': kind, 'index': rng.randrange(8), 'target': rng.randrange(8), 'text': f'task {rng.randrange(5)} #tag !p{rng.randrange(1, 4)}', 'checked': bool(rng.randrange(2))})
        cases.append({'name': f'trace/{seed}', 'source': sections, 'actions': actions})
    return cases


def run(binary, requests):
    result = subprocess.run([str(binary)], input=''.join(json.dumps(r) + '\n' for r in requests), text=True, capture_output=True, timeout=120)
    assert result.returncode == 0, (binary, result.stderr)
    return [json.loads(line) for line in result.stdout.splitlines()]


def compare(go, rust, output):
    cases = corpus()
    left, right = run(go, cases), run(rust, cases)
    assert len(left) == len(right) == len(cases)
    failures = []
    steps = 0
    for case, a, b in zip(cases, left, right):
        assert len(a) == len(b)
        for i, (x, y) in enumerate(zip(a, b)):
            steps += 1
            differing = [key for key in ('tasks', 'headings') if x[key] != y[key]]
            if bool(x.get('error')) != bool(y.get('error')):
                differing.append('rejection')
            if differing:
                failures.append({'case': case['name'], 'step': i, 'differences': differing, 'action': case['actions'][i-1] if i else None, 'go': x, 'rust': y})
                break
    report = {'cases': len(cases), 'steps': steps, 'failures': failures}
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(report, indent=2) + '\n')
    print(f'{len(cases)} cases, {steps} steps, {len(failures)} failed; {output}')
    for fail in failures[:30]:
        print(fail['case'], fail['step'], ', '.join(fail['differences']))
    return report


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--go', type=Path, default=ROOT / 'dist/rust-rewrite/go-parity')
    parser.add_argument('--rust', type=Path, default=ROOT / 'dist/rust-rewrite/target/debug/parity')
    parser.add_argument('--output', type=Path, default=ROOT / 'dist/rust-rewrite/parity.json')
    args = parser.parse_args()
    report = compare(args.go.resolve(), args.rust.resolve(), args.output)
    raise SystemExit(bool(report['failures']))
