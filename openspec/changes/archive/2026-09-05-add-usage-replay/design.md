## Decisions
Use versioned JSON traces with abstract user actions, explicit elapsed simulated seconds, and independently computed expected task states. A Go driver exercises TUI Update, rendering, undo and real conditional filesystem saves. A subprocess adapter exercises the stable CLI surface; terminal scenarios separately exercise the Bubble Tea event loop. Return errors with the failing prefix and keep generated artifacts in dist/usage.

The default workload represents six deliberate actions per minute over twenty five-hour sessions (36,000 actions). It accelerates think time; it does not certify 100 wall-clock hours, idle timers, human usability, or exhaustive interleavings. Small seeded sessions run in normal tests. The full run remains an explicit mise task.

Keep source fidelity for checkbox-only documents, backed by AST checkbox locations so fenced examples cannot be changed. Structural edits continue through the AST serializer; preserve opaque block content where possible and document remaining formatting normalization. Profile repeated loaded-document editing separately from parse-plus-edit costs. Preserve unknown frontmatter when settings are unchanged.

## Risks / Trade-offs
The task oracle intentionally covers flat task editing; dedicated cases cover nested/ordered Markdown, metadata, conflict, cancellation and terminal behavior. Future Rust candidates must implement each advertised driver contract; the existing parser probe cannot claim session parity. Source location caches must be invalidated by structural mutations. Undo must not lose an older entry when input is cancelled at capacity.
