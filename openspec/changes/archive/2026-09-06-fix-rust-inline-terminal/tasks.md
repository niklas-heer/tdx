## Implementation
- [x] Restore compact normal-buffer rendering and safe terminal cleanup/resizing.
- [x] Place task creation/editing in the list with a visible cursor.
- [x] Verify context preservation, inline placement, cancellation/save, Unicode, resize and overlays.
- [x] Run local checks and native platform gates, rebuild the trial executable, and document results.

## Validation evidence
- Local: `mise run rust-rewrite:check` passed (63 nextest tests, strict Clippy, all-target compilation, documentation and Python harness checks).
- Native CI [34026117920](https://github.com/niklas-heer/tdx/actions/runs/34026117920) passed on revision `415cfb4` (Rust application code from `2104211`). macOS, Linux and Windows each passed 1,154 action cases / 2,993 steps, 58 application workflows, native terminal/Markdown/context checks and the deterministic simulation campaign. Linux additionally passed stable compatibility and Miri.
- The first Windows context-test launcher exited on `os.execv`; retaining its ConPTY root with a waiting subprocess fixed the test harness. The Rust application was unchanged by that correction.
- Local additional terminal checks retained 27 shell lines through screen growth/shrink, preserved a partial prompt line, and kept long Unicode input's caret visible through wrapping, resize, Home/End and cancellation.
- Rebuilt `dist/rust-rewrite/tdx-rust` for Apple Silicon macOS; `--version` reports `tdx v0.14.0`.
