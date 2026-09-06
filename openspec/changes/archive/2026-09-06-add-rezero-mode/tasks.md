## Implementation
- [x] Add source-preserving append/continuation actions and fidelity regression tests.
- [x] Add complete review/work cycles, dots, inline input, read-only handling and compact status.
- [x] Integrate undo, guarded saves, reload and conflict recovery without stale selections.
- [x] Add interaction and real terminal tests; run project checks, race tests and engine simulation.
- [x] Update user documentation/task checklist and push the implementation branch.

## Local validation
- Full mise check, race suite, actual terminal contracts and 1,000-seed deterministic engine campaign passed.
- A 10-second source-preservation fuzz campaign executed 34,702 cases without a failure.
- Terminal checks cover normal editing and read-only behavior plus Rezero Unicode continuation, undo, resizing, completed-task context and absence of alternate-screen entry.

## Native CI evidence
- CI run [34034974245](https://github.com/niklas-heer/tdx/actions/runs/34034974245) passed for implementation revision dd617df.
- Portable Dagger CI, native macOS/Windows builds, and save/history/editor/TUI race checks on Linux/macOS/Windows passed.
- Linux and macOS terminal artifacts each report all four scenarios passed: editing, readonly, rezero and rezero-readonly.
- Each native engine job passed the 1,000-seed deterministic campaign and negative controls.
