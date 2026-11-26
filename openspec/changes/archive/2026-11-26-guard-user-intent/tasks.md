## Implementation
- [x] 1. CLI mutations honour the file's `read-only` setting with flag > frontmatter > config precedence, before history opens and again on the edited snapshot.
- [x] 2. Warn before quitting with unsaved changes.
- [x] 3. Refuse to replace write-protected files.
- [x] 4. Move a revisited version to the newest position.
- [x] 5. Keep section focus and folds across undo when headings are unchanged.
- [x] 6. Run `mise run check`, the terminal and usage suites, and `mise run ci`.

Validation: every commit passes gofmt, `go vet` and `go test`; `mise run check`, `mise run test:terminal`, `mise run test:usage` (100 simulated hours) and `mise run ci` passed. Council review by Claude Fable 5.2 and GPT Sol 6.2 ended satisfied.
