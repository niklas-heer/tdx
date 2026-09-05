## Why
The existing Rust probe compares parsing only. Go now has optimized loaded checkbox edits and executable usage contracts, so a rewrite decision needs a usable candidate and matched application workloads.

## What Changes
- Add an isolated experimental Rust CLI supporting list/JSON queries, add, edit, delete and toggle, with task metadata and explicit errors for unsupported operations.
- Preserve source bytes for checkbox edits; conservatively reject structural edits the prototype cannot safely represent.
- Implement guarded atomic saves, permission checks and conflict detection, with tests and documented platform limits.
- Exercise the candidate with the existing language-neutral CLI replay harness and additional Markdown/error cases.
- Measure alternating repeated Go/Rust CLI workloads on identical disposable inputs, reporting latency, process RSS, release binary size, build time and feature gaps.
- Add a basic Ratatui terminal interface with navigation, toggle, add/edit/delete, bounded undo, reload, read-only handling and conflict reporting; verify it through real PTYs.
- Record a reproducible recommendation in the existing experiment documentation. Keep production Go and existing parser baselines intact.

## Impact
- Affected specs: development-tooling.
- Affected code: experiments/rust-rewrite/, experiments/rust-eval/README.md, mise.toml.
- Prototype scope is CLI/editor and persistence feasibility; full TUI parity beyond the basic editor, configuration/theme parity, SQLite history, Windows save guarantees and full Markdown structural parity remain follow-up gates, not claimed successes.
- Benchmark results must separate read-only matched queries from writes where Go provides extra history/configuration behavior.
