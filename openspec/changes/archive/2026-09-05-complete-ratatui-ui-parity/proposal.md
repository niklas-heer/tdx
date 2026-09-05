## Why
The user reiterates a complete Rust rewrite with 100% Go feature parity and asks whether Ratatui improves the UI. The existing standalone application already uses Ratatui, but behavioral corpus success alone did not establish presentation completeness.

## What Changes
- Audit the documented Go commands, modes and rendered information against Rust; retain the full standalone CLI/TUI and shared persistence compatibility.
- Complete and polish the Ratatui presentation: contextual status, responsive pickers/history, visible long-input cursor, scrollable help/diffs, theme-aware styling and terminal hyperlinks.
- Add assertions for rendered information and styles across sizes, real terminal regressions, and reproducible previews of actual Ratatui buffers.
- Retain earlier measurements as historical and identify validation for the final UI revision.

## Impact
- Affected spec: development-tooling.
- Affected code: experiments/rust-rewrite UI, regression/preview tools and existing evaluation guide.
- Authorization: the user's explicit instruction to complete the full Rust rewrite and 100% feature parity includes closing these UI gaps. No additional approval is needed.
