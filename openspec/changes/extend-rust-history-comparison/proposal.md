## Why
The runnable prototype showed lower resource usage but omitted Go's SQLite version history. The user asked to continue the Rust implementation and evaluate whether benefits remain. History/recovery and representative editing parity were explicitly identified as the next milestone in the reviewed report.

## What Changes
- Add default Rust version capture using Go-compatible SQLite tables, SHA-256 hashes, zstd snapshots, deduplication, per-file retention, contention bounds and shutdown checkpoints.
- Add snapshot listing/read/restore commands and a terminal browser with preview and confirmed restore. Preserve conflict and post-commit error semantics; capture overwritten disk content before explicit force-save.
- Poll external changes while idle, defer during input/history/local conflicts, and support renaming single-line parent tasks while preserving children.
- Verify bidirectional Go/Rust snapshot interoperability, retention, corruption/failure handling and real terminal recovery; attempt native Linux validation where the local container runtime is available.
- Extend benchmarks with history verification and distinct-version editing sessions as well as toggles. Preserve the original prototype baseline and report remaining service/platform differences.

## Impact
- Affected specs: development-tooling (experimental capability only).
- Affected code: experiments/rust-rewrite/, experiments/rust-eval/README.md, scripts/usage-pty.py, mise.toml.
- Production Go remains unchanged. General structural Markdown, full search/move/section UI, themes, character-level history diffs and Windows parity remain future milestones.
- Scope authorization: the user's instruction to continue the Rust implementation and evaluate benefits, following the report's proposed history/recovery milestone.
