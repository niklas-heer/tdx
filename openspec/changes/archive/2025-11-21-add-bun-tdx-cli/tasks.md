## 1. Project Setup

- [x] 1.1 Initialize Bun + TypeScript project layout
  - [x] Create `src/` directory and baseline `tsconfig.json`
  - [x] Configure TypeScript target and module settings compatible with Bun
- [x] 1.2 Add runtime and UI dependencies
  - [x] Add `ink` for TUI rendering
  - [x] Add `chalk` for colored output (if not using Ink colors exclusively)
- [x] 1.3 Add development dependencies as needed
  - [x] Type definitions and minimal tooling (only if required by Bun/TS setup)

## 2. Markdown Todo Model

- [x] 2.1 Define in-memory todo model
  - [x] Represent todos as `{ index, checked, text, lineNo }`
  - [x] Keep raw lines array for full-file reconstruction
- [x] 2.2 Implement minimal markdown parser for `todo.md`
  - [x] Scan file line-by-line
  - [x] Identify todo lines strictly starting with `- [ ] ` or `- [x] `
  - [x] Extract checkbox state and text
  - [x] Preserve all non-todo lines exactly
- [x] 2.3 Implement markdown writer with atomic save
  - [x] Rebuild file by replacing only todo lines based on the model
  - [x] Preserve all non-todo lines byte-for-byte
  - [x] Write to a temporary file and atomically replace `todo.md`
- [x] 2.4 Implement creation of missing `todo.md`
  - [x] On first use, create `todo.md` with a simple markdown header and blank line

## 3. Core CLI Behavior

- [x] 3.1 Implement CLI entrypoint in `src/cli.ts`
  - [x] Parse process arguments (subcommands and options)
  - [x] Route to non-TUI commands or TUI mode
- [x] 3.2 Implement `tdx list`
  - [x] Load and parse `todo.md`
  - [x] Render todos in a clean, readable, colored format consistent with TUI style
- [x] 3.3 Implement `tdx add "Text"`
  - [x] Append a new unchecked todo to `todo.md` using the writer
  - [x] Confirm success with minimal output
- [x] 3.4 Implement `tdx toggle <index>`
  - [x] Interpret index as 1-based
  - [x] Validate index bounds and report clear errors on invalid input
  - [x] Toggle checkbox state and persist via writer
- [x] 3.5 Implement `tdx edit <index> "New text"`
  - [x] Validate index bounds
  - [x] Update todo text while preserving checked state
  - [x] Persist changes and keep non-todo content unchanged
- [x] 3.6 Implement consistent error handling and exit codes
  - [x] Non-zero exit codes on errors
  - [x] Minimal, clearly colored error messages

## 4. Ink TUI Implementation

- [x] 4.1 Implement main Ink TUI component
  - [x] Load todos from `todo.md` at startup
  - [x] Render one line per todo
- [x] 4.2 Match required TUI layout and styling
  - [x] Non-selected: `  [✓] Text` or `  [ ] Text`
  - [x] Selected: `➜ [✓] Text` or `➜ [ ] Text`
  - [x] Arrow `➜` in cyan
  - [x] Checked `[✓]` in magenta/purple
  - [x] Unchecked `[ ]` in dim white
  - [x] Selected text bold or bright
- [x] 4.3 Implement keyboard interaction
  - [x] `j` / Down: move selection down
  - [x] `k` / Up: move selection up
  - [x] Enter: toggle selected todo
  - [x] `q` / `Esc`: quit TUI
- [x] 4.4 Integrate persistence
  - [x] On each toggle, immediately persist to `todo.md` using the writer
  - [x] Ensure TUI exits with correct status codes on success/error

## 5. Build and Tooling

- [x] 5.1 Ensure `bun run src/cli.ts` works
  - [x] Test all non-TUI commands
  - [x] Test TUI launch with `tdx` (no subcommand)
- [x] 5.2 Ensure single-binary build works
  - [x] Build via `bun build --compile --minify src/cli.ts --outfile tdx`
  - [x] Verify resulting `./tdx` binary supports all commands and TUI behavior
- [x] 5.3 Add basic documentation
  - [x] Document installation and development steps
  - [x] Document CLI usage and examples
  - [x] Provide an example `todo.md` with expected formatting

## 6. Validation

- [x] 6.1 Verify markdown round-trip behavior
  - [x] Read `todo.md` and immediately write it back with no semantic changes
  - [x] Confirm file is identical byte-for-byte
- [x] 6.2 Verify behavior against requirements
  - [x] Check all TUI keyboard shortcuts
  - [x] Confirm atomic writes and preservation of non-todo content
  - [x] Validate error handling for invalid indices and missing files