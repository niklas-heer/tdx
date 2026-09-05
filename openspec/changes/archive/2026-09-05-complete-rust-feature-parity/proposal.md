## Why
The user explicitly requests a complete, full-featured Rust rewrite with Go feature parity so application comparisons have equivalent services. The earlier prototype and history milestones deliberately omitted capabilities and cannot answer that question.

## What Changes
- Implement the documented Go CLI, document actions, complete TUI interactions, configuration/themes, recent-file state, clipboard, history/diffs, and supported-platform persistence in standalone Rust.
- Record a concrete parity matrix and executable Go/Rust contracts for every feature group. Compare externally observable behavior and document semantics; terminal rendering may use different libraries while preserving controls, information, styles and accessibility.
- Fix demonstrated Go defects discovered by differential tests where copying them would violate documented behavior or lose data. Retain regressions and explain corrected shared semantics.
- Remove prototype-only feature rejections and no-history/no-configuration comparisons; retain historical results. Publish fresh correctness-gated measurements only after all parity groups pass.
- Add reproducible native platform/build checks and comparison packaging without replacing or publishing a production release.

## Impact
- Affected spec: development-tooling, plus specific application specs if a demonstrated inconsistency needs correction.
- Affected code: experiments/rust-rewrite, focused Go bug fixes, shared tests, mise and CI/build tasks.
- Scope authorization: the user's explicit instruction to implement everything and make the rewrite full-featured and parity-compliant. No additional scope permission is needed.
