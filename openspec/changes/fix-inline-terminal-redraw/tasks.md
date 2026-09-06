## Tasks
- [x] Reproduce the screenshot with a real PTY and terminal screen emulator.
- [x] Pin compatible rendering dependencies and add screen-state regressions.
- [ ] Pass local checks and Linux/macOS terminal CI plus Windows native checks.
- [ ] Rebuild executable and document validation.

## Validation
- `mise run check`: passed formatting, vet, lint, all Go tests, pipeline tests, release-note tests and workflow validation.
- `mise run test:terminal`: passed existing terminal contracts and 21 final-screen assertions, including repeated palette/Rezero transitions, Unicode editing, undo and resizing.
- Negative control built with the previous dependencies: the new screen test fails at `review-0: stale command text`, reproducing the reported screenshot.
