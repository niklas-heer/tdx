"""Source identity shared by native parity checks and performance reports."""
import hashlib
from pathlib import Path
ROOT = Path(__file__).resolve().parents[2]


def fingerprint():
    experiment = ROOT / 'experiments/rust-rewrite'
    paths = sorted(set(list((ROOT / 'cmd').rglob('*.go')) + list((ROOT / 'cmd/tdx/themes').glob('*.toml'))
                       + list((ROOT / 'internal').rglob('*.go')) + list(experiment.glob('*.py'))
                       + list((experiment / 'src').rglob('*.rs')) + list((experiment / 'go-history').glob('*.go'))
                       + list((experiment / 'go-parity').glob('*.go'))
                       + [experiment / 'Cargo.toml', experiment / 'Cargo.lock', ROOT / 'go.mod', ROOT / 'go.sum',
                          ROOT / 'tdx.toml', ROOT / 'scripts/usage-pty.py', ROOT / 'experiments/rust-eval/fixtures.json']))
    # Checkout line-ending conversion must not give identical source different identities.
    return {'sha256': hashlib.sha256(b''.join(p.relative_to(ROOT).as_posix().encode() + b'\0' + p.read_bytes().replace(b'\r\n', b'\n') for p in paths)).hexdigest(),
            'files': [p.relative_to(ROOT).as_posix() for p in paths]}
