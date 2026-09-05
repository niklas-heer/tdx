## Why
CLI and TUI duplicate editing decisions while process-wide configuration, style, and history hooks couple otherwise independent instances. A Rust rewrite should be evaluated against measured costs and compatibility rather than presumed language advantages.

## What Changes
- Route document edits through shared, explicit actions and a UI-independent bounded undo history.
- Replace production global CLI/TUI configuration, styling, and history wiring with per-instance dependencies and an explicit Markdown store.
- Flush pending command/search filters before selection; fast typing must not execute a stale action.
- Fix ordered-task numbering and multiline quote prefixes exposed by the comparison corpus.
- Add regression tests for independent instances, action validation/undo, and safe persistence.
- Add a disposable Rust parser/checkbox-edit prototype, shared correctness fixtures, reproducible timing/RSS/build-size comparisons, and Go CPU/allocation profiles.
- Publish the measured findings and prototype limitations in the experiment directory; retain Go and version 0.14.0 unless evidence supports a later decision.

## Impact
- Affected specs: editor-core, development-tooling, file-versioning.
- Affected code: internal/editor, internal/markdown, internal/cmd, internal/tui, cmd/tdx, experiments/rust-eval, mise tasks.
- Authorized by the user's request to implement the proposed architecture and test the Rust trade-offs.
