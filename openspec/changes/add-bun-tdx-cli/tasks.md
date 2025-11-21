## 1. Project Setup

- [ ] 1.1 Initialize Bun + TypeScript project layout
  - [ ] Create `src/` directory and baseline `tsconfig.json`
  - [ ] Configure TypeScript target and module settings compatible with Bun
- [ ] 1.2 Add runtime and UI dependencies
  - [ ] Add `ink` for TUI rendering
  - [ ] Add `chalk` for colored output (if not using Ink colors exclusively)
- [ ] 1.3 Add development dependencies as needed
  - [ ] Type definitions and minimal tooling (only if required by Bun/TS setup)

## 2. Markdown Todo Model

- [ ] 2.1 Define in-memory todo model
  - [ ] Represent todos as `{ index, checked, text, lineNo }`
  - [ ] Keep raw lines array for full-file reconstruction
- [ ] 2.2 Implement minimal markdown parser for `todo.md`
  - [ ] Scan file line-by-line
  - [ ] Identify todo lines strictly starting with `- [ ] ` or `- [x] `
  - [ ] Extract checkbox state and text
  - [ ] Preserve all non-todo lines exactly
- [ ] 2.3 Implement markdown writer with atomic save
  - [ ] Rebuild file by replacing only todo lines based on the model
  - [ ] Preserve all non-todo lines byte-for-byte
  - [ ] Write to a temporary file and atomically replace `todo.md`
- [ ] 2.4 Implement creation of missing `todo.md`
  - [ ] On first use, create `todo.md` with a simple markdown header and blank line

## 3. Core CLI Behavior

- [ ] 3.1 Implement CLI entrypoint in `src/cli.ts`
  - [ ] Parse process arguments (subcommands and options)
  - [ ] Route to non-TUI commands or TUI mode
- [ ] 3.2 Implement `tdx list`
  - [ ] Load and parse `todo.md`
  - [ ] Render todos in a clean, readable, colored format consistent with TUI style
- [ ] 3.3 Implement `tdx add "Text"`
  - [ ] Append a new unchecked todo to `todo.md` using the writer
  - [ ] Confirm success with minimal output
- [ ] 3.4 Implement `tdx toggle <index>`
  - [ ] Interpret index as 1-based
  - [ ] Validate index bounds and report clear errors on invalid input
  - [ ] Toggle checkbox state and persist via writer
- [ ] 3.5 Implement `tdx edit <index> "New text"`
  - [ ] Validate index bounds
  - [ ] Update todo text while preserving checked state
  - [ ] Persist changes and keep non-todo content unchanged
- [ ] 3.6 Implement consistent error handling and exit codes
  - [ ] Non-zero exit codes on errors
  - [ ] Minimal, clearly colored error messages

## 4. Ink TUI Implementation

- [ ] 4.1 Implement main Ink TUI component
  - [ ] Load todos from `todo.md` at startup
  - [ ] Render one line per todo
- [ ] 4.2 Match required TUI layout and styling
  - [ ] Non-selected: `  [✓] Text` or `  [ ] Text`
  - [ ] Selected: `➜ [✓] Text` or `➜ [ ] Text`
  - [ ] Arrow `➜` in cyan
  - [ ] Checked `[✓]` in magenta/purple
  - [ ] Unchecked `[ ]` in dim white
  - [ ] Selected text bold or bright
- [ ] 4.3 Implement keyboard interaction
  - [ ] `j` / Down: move selection down
  - [ ] `k` / Up: move selection up
  - [ ] Enter: toggle selected todo
  - [ ] `q` / `Esc`: quit TUI
- [ ] 4.4 Integrate persistence
  - [ ] On each toggle, immediately persist to `todo.md` using the writer
  - [ ] Ensure TUI exits with correct status codes on success/error

## 5. Build and Tooling

- [ ] 5.1 Ensure `bun run src/cli.ts` works
  - [ ] Test all non-TUI commands
  - [ ] Test TUI launch with `tdx` (no subcommand)
- [ ] 5.2 Ensure single-binary build works
  - [ ] Build via `bun build --compile --minify src/cli.ts --outfile tdx`
  - [ ] Verify resulting `./tdx` binary supports all commands and TUI behavior
- [ ] 5.3 Add basic documentation
  - [ ] Document installation and development steps
  - [ ] Document CLI usage and examples
  - [ ] Provide an example `todo.md` with expected formatting

## 6. Validation

- [ ] 6.1 Verify markdown round-trip behavior
  - [ ] Read `todo.md` and immediately write it back with no semantic changes
  - [ ] Confirm file is identical byte-for-byte
- [ ] 6.2 Verify behavior against requirements
  - [ ] Check all TUI keyboard shortcuts
  - [ ] Confirm atomic writes and preservation of non-todo content
  - [ ] Validate error handling for invalid indices and missing files