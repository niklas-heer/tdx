## Tasks
- [x] Reproduce the screenshot with a real PTY and terminal screen emulator.
- [x] Pin compatible rendering dependencies and add screen-state regressions.
- [x] Pass local checks and Linux/macOS terminal CI plus Windows native checks.
- [x] Rebuild executable and document validation.

## Validation
- `mise run check`: passed formatting, vet, lint, all Go tests, pipeline tests, release-note tests and workflow validation.
- `mise run test:terminal`: passed existing terminal contracts and 21 final-screen assertions, including repeated palette/Rezero transitions, Unicode editing, undo and resizing.
- Negative control built with the previous dependencies: the new screen test fails at `review-0: stale command text`, reproducing the reported screenshot.
- [CI run 34040398170](https://github.com/niklas-heer/tdx/actions/runs/34040398170) passed on implementation commit `4e0132f`: portable checks, Linux/macOS terminal suites, macOS/Windows native builds and checks, and race/replay checks on all three platforms. Terminal artifacts include screen snapshots and ANSI transcripts.
- The local `tdx` executable was rebuilt with the compatible rendering dependencies.
