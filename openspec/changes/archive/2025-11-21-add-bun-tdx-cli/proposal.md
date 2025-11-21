## Why

The project needs a fast, self-contained command-line tool for managing todos stored in a local `todo.md` file, using a lightweight, modern toolchain. Existing tools like `mdt` provide powerful markdown-based task management, but they typically depend on external shells, multiple utilities, or Node.js runtimes.

We want a single binary that:

- Runs on the Bun runtime using TypeScript
- Provides a pleasant interactive TUI for daily use
- Uses a simple, custom markdown model that preserves the rest of the file
- Offers non-interactive commands for scripting and automation

This will make it easy for developers to keep their todos close to their code with minimal dependencies and friction.

## What Changes

- Add a new Bun-based TypeScript CLI tool named `tdx` that manages todos stored in `todo.md`.
- Implement `tdx` as a single entrypoint (`src/cli.ts`) that can be:
  - Run in development mode with `bun run src/cli.ts`
  - Compiled into a single binary with `bun build --compile --minify src/cli.ts --outfile tdx`
- Implement a minimal markdown parser and writer for `todo.md` that:
  - Recognizes todos as lines starting with `- [ ] ` or `- [x] `
  - Extracts todo text after the checkbox
  - Preserves all non-todo content exactly and maintains line order
  - Supports safe atomic writes (temp file → replace original) and only rewrites todo lines
- Implement an Ink-based TUI that:
  - Renders todos with the exact required layout:
    - Non-selected: `"  [✓] Text"` / `"  [ ] Text"`
    - Selected: `"➜ [✓] Text"` / `"➜ [ ] Text"`
  - Uses the specified colors:
    - Arrow `➜`: cyan
    - Checked `[✓]`: magenta/purple
    - Unchecked `[ ]`: dim white
    - Selected text: bold or bright
  - Supports keyboard controls:
    - `j`/Down: move selection down
    - `k`/Up: move selection up
    - Enter: toggle checkbox
    - `q` or `Esc`: quit
  - Immediately writes changes back to `todo.md` whenever a todo is toggled
  - Automatically creates `todo.md` with a simple header if it does not exist
- Implement non-TUI CLI commands that operate on `todo.md`:
  - `tdx list` – list todos with minimal, clean, colorized output
  - `tdx add "Text"` – append a new unchecked todo
  - `tdx toggle 3` – toggle the 3rd todo (1-based index)
  - `tdx edit 2 "New text"` – edit text of the 2nd todo
- Provide documentation and examples:
  - Installation, development, and build instructions
  - Description of TUI and non-TUI usage
  - Example `todo.md` demonstrating expected formatting and behavior

## Impact

- **Affected specs:**
  - New capability: `tdx-cli` (added under `openspec/specs/tdx-cli/spec.md`)

- **Affected code:**
  - New project structure for the `tdx` CLI, including:
    - `src/cli.ts` – main entrypoint and argument routing
    - `src/tui/App.tsx` (or similar) – Ink TUI implementation
    - `src/todos/parser.ts` – minimal markdown parser for todo lines
    - `src/todos/writer.ts` – markdown writer with atomic file writes
    - `src/todos/model.ts` – shared todo model types and helpers
    - `src/fs/fileStore.ts` – small file I/O utility for `todo.md`
  - New configuration and metadata:
    - `tsconfig.json` – TypeScript configuration targeting Bun
    - `package.json` / `bunfig.toml` – dependencies (Ink, chalk) and scripts
    - `README.md` – usage, development, and build instructions
    - Example `todo.md` file

- **Behavioral impact:**
  - Introduces a new end-user-visible command-line tool `tdx` that can be installed and used as a standalone binary.
  - Establishes conventions for how todos are represented in `todo.md` and how they are preserved and updated.
  - Adds a new Ink-based TUI workflow inspired by `mdt`, but focused on a single `todo.md` inbox-style file with minimal flags and options.

No existing specs or capabilities are modified or removed; this change is additive.