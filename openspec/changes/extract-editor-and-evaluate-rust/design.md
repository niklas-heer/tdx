## Decisions
Keep the existing Markdown AST and guarded save boundary. An editor action describes the change independently of terminal keybindings; presentation layers translate selection/filter state into action arguments. Undo stores bounded document snapshots and retains current disk revisions on restoration. Input and move previews retain their existing grouping/cancel behavior.

An explicit Markdown Store carries read/write hooks. Stateless package functions remain hook-free conveniences. CLI services own writers/styles/stores; TUI configuration owns store and theme dependencies. Production startup passes instances instead of modifying package state.

The Rust spike is an isolated experiment, not a replacement executable. Compare equivalent parser/task-marker scans separately from production Go edit/serialize behavior and the Rust source-patch prototype. Use identical fixtures, optimized builds, repeated trials, per-process CPU/RSS, and recorded machine/toolchain information. Report correctness failures and missing persistence/TUI/history features before interpreting speed ratios. Do not compare the small prototype binary to the full application as if feature-equivalent.
