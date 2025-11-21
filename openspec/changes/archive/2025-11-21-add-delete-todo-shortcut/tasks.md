## 1. Spec + Design

- [x] 1.1 Draft delta in `specs/tdx-cli/spec.md` describing delete behavior for CLI and TUI (scenarios: delete via command, delete via `d` shortcut with immediate removal)
- [x] 1.2 Validate change with `openspec validate add-delete-todo-shortcut --strict` before implementation

## 2. CLI Implementation

- [x] 2.1 Add `delete` command wiring in `src/cli.ts` (usage, help text, error paths)
- [x] 2.2 Implement `src/commands/delete.ts`
  - [x] Read + parse todos
  - [x] Validate index, remove todo from model, persist via writer
  - [x] Emit concise success/error output (colored)
- [x] 2.3 Add tests or manual verification notes covering delete command edge cases

## 3. TUI Implementation

- [x] 3.1 Update `src/tui/App.tsx`
  - [x] Capture `d` keypress for immediate deletion
- [x] 3.2 Ensure escape routes (Esc, `q`) still exit cleanly after deletions
- [x] 3.3 Verify TUI list re-renders without gaps, including empty-state messaging post-delete

## 4. Docs + Help Text

- [x] 4.1 Update README usage section with `tdx delete <index>` and `d` shortcut notes
- [x] 4.2 Refresh CLI help output to mention delete command and shortcut summary
- [x] 4.3 Update example `todo.md` if needed to reflect new workflow tips

## 5. Validation

- [x] 5.1 Run `bun run src/cli.ts delete <index>` smoke tests (valid index, invalid index, nonexistent file)
- [x] 5.2 Run `bun run src/cli.ts` TUI smoke tests (delete middle, last, only todo)
- [x] 5.3 Confirm `openspec validate add-delete-todo-shortcut --strict` passes post-implementation
