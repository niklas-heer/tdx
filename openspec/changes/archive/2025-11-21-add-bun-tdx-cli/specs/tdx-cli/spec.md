## ADDED Requirements

### Requirement: Bun-based TypeScript CLI entry point
The system SHALL provide a Bun-based TypeScript command-line tool named `tdx` as the primary interface for managing markdown-based todos.

- The CLI implementation SHALL target the Bun runtime (not Node).
- The primary CLI entrypoint SHALL be `src/cli.ts`.
- The project SHALL be runnable in development via:
  - `bun run src/cli.ts`
- The project SHALL be buildable into a single binary via:
  - `bun build --compile --minify src/cli.ts --outfile tdx`
- The compiled `tdx` binary SHALL behave equivalently to `bun run src/cli.ts` for all supported commands.

#### Scenario: Build and run tdx CLI
- **WHEN** a developer runs `bun run src/cli.ts` from the project root  
- **THEN** the CLI SHALL execute and respond to supported commands, including at least:
  - `tdx` (interactive TUI mode)
  - `tdx list`
  - `tdx add "Text"`
  - `tdx toggle 3`
  - `tdx edit 2 "New text"`  

- **WHEN** a developer runs `bun build --compile --minify src/cli.ts --outfile tdx`  
- **THEN** a single binary named `tdx` SHALL be produced in the current directory  
- **AND** running `./tdx` with the same arguments as `bun run src/cli.ts` SHALL produce equivalent behavior and output.

---

### Requirement: Markdown todo storage in todo.md
The system SHALL store and manage todos in a markdown file named `todo.md` located in the current working directory.

- The primary todo storage file SHALL be `./todo.md` relative to the process working directory.
- If `todo.md` does not exist when any todo-related command is invoked, the system SHALL create it automatically with a simple markdown header and an empty task list (e.g., `# Todos` followed by a blank line).
- A todo item SHALL be represented only by lines that start with one of the following exact prefixes:
  - `- [ ] ` (unchecked)
  - `- [x] ` (checked, using lowercase `x`)
- Todo text SHALL be defined as all content after the checkbox prefix on the same line.
- The system SHALL preserve:
  - The exact content of all non-todo lines (including headers, paragraphs, comments, and blank lines).
  - The original ordering of all lines (todo and non-todo).
- On write, the system SHALL:
  - Rewrite only the lines that correspond to todo items when their state or text changes.
  - Leave all non-todo lines byte-for-byte unchanged.

- All modifications to `todo.md` SHALL be performed as safe atomic writes:
  - Write to a temporary file in the same directory.
  - Flush and close the temporary file.
  - Replace the original file with the temporary file using an atomic rename, where supported by the platform.

#### Scenario: Create missing todo.md
- **WHEN** the user runs any `tdx` command that reads or writes todos  
- **AND** `todo.md` does not exist in the current working directory  
- **THEN** the system SHALL create `todo.md` with a simple markdown header (e.g., `# Todos`) and no tasks  
- **AND** the command SHALL continue using this new file without failing.

#### Scenario: Preserve non-todo markdown content
- **WHEN** `todo.md` contains a mix of markdown headings, paragraphs, blank lines, comments, and todo lines  
- **AND WHEN** the user toggles or edits one todo through any CLI or TUI operation  
- **THEN** only the corresponding todo line in `todo.md` SHALL be modified  
- **AND** all non-todo lines SHALL remain unchanged in content, spacing, and ordering.

#### Scenario: Atomic write on modification
- **WHEN** any command or TUI interaction changes todo state or text  
- **THEN** the system SHALL write the new file contents to a temporary file in the same directory as `todo.md`  
- **AND** then replace `todo.md` with the temporary file in a single rename operation  
- **AND** at no point SHALL a partially written `todo.md` be visible on disk.

---

### Requirement: Minimal custom markdown parser and writer
The system SHALL implement a minimal, custom markdown parser and writer for managing todos, without relying on heavy markdown libraries.

- The system SHALL NOT depend on large, general-purpose markdown parsers (e.g., Remark, markdown-it, or similar libraries) for todo handling.
- Todo parsing SHALL be implemented using simple string and/or regular expression matching.
- The parser SHALL:
  - Read `todo.md` line-by-line.
  - Identify todo lines strictly by the prefixes `- [ ] ` and `- [x] ` at the start of the line (after any optional leading BOM handling, but before other characters).
  - Extract a structured representation for each todo, including at least:
    - The 1-based line number or position.
    - The checked/unchecked state.
    - The todo text after the checkbox.
  - Preserve the original content of every line, whether or not it is a todo.
- The writer SHALL:
  - Reconstruct the full file by iterating over the original lines.
  - Replace only the lines that correspond to todo items whose state or text has changed.
  - Emit updated todo lines in the same format (`- [ ] ` or `- [x] ` followed by text).
  - Keep all other lines unchanged.

#### Scenario: Round-trip consistency with no changes
- **WHEN** the parser reads `todo.md` into an in-memory representation  
- **AND** the writer immediately writes this representation back out without any modifications  
- **THEN** the resulting `todo.md` file on disk SHALL be byte-for-byte identical to the original.

#### Scenario: Correct parsing of todos and non-todos
- **WHEN** `todo.md` contains:
  - Multiple todo lines
  - Lines beginning with `- ` that are not checkboxes
  - Other markdown constructs  
- **THEN** the parser SHALL classify only lines starting with `- [ ] ` or `- [x] ` as todos  
- **AND** all other lines SHALL be preserved as non-todo content.

---

### Requirement: TUI layout and styling
The system SHALL present an interactive terminal UI (TUI) using Ink that renders todos with a specific layout and styling.

- The TUI implementation SHALL use Ink (React-style terminal UI library).
- Styling for colors and emphasis SHALL use Ink’s color support and/or Chalk.
- The TUI SHALL display each todo line in one of two textual formats:

  - For non-selected items:
    - `"  [✓] Text"` for checked items
    - `"  [ ] Text"` for unchecked items

  - For the currently selected item:
    - `"➜ [✓] Text"` for checked items
    - `"➜ [ ] Text"` for unchecked items

- Alignment and prefixes:
  - Non-selected items MUST start with exactly two spaces `"  "` before the bracket.
  - The selected item MUST start with exactly the arrow and a space `"➜ "` before the bracket.
- Colors and emphasis:
  - The arrow `"➜"` SHALL be rendered in cyan.
  - The checked indicator `"[✓]"` SHALL be rendered in magenta or purple.
  - The unchecked indicator `"[ ]"` SHALL be rendered in dim white.
  - The text for the selected todo line SHALL be emphasized, using bold and/or a brighter color compared to non-selected items.
- The TUI overall appearance SHALL match the following reference layout:

  - A shell-style prompt followed by the `tdx` invocation, and then the todo list:

    - `~ ❯ tdx`  
    - `  [✓] Feed the kitten`  
    - `➜ [✓] Bake cookies`  
    - `  [ ] Water the plants`  
    - `  [ ] Organize the desk`  
    - `  [ ] Take a walk outside`  
    - `  [ ] Make a grocery list and plan out the meals for the week`

#### Scenario: Render todos with single selection
- **WHEN** the user runs `tdx` (with no additional arguments) in a directory with an existing `todo.md` file containing multiple todos  
- **THEN** the TUI SHALL render all todos, one per line, using the specified text format  
- **AND** exactly one todo SHALL be visually selected, prefixed by a cyan `"➜ "` and highlighted text  
- **AND** all other todos SHALL be prefixed with two spaces and non-highlighted text.

#### Scenario: Long todo text handling
- **WHEN** a todo’s text is longer than the terminal width  
- **THEN** the TUI SHALL still render the item in the required format (`"➜ [ ] Text"` or `"  [x] Text"`)  
- **AND** any truncation or wrapping behavior SHALL not break the required prefixes and color semantics.

---

### Requirement: TUI keyboard interaction and persistence
The TUI SHALL support keyboard navigation and toggling of todos, immediately persisting changes back to `todo.md`.

- The TUI SHALL respond to the following keys:
  - `j` or Down arrow: move the selection down by one todo.
  - `k` or Up arrow: move the selection up by one todo.
  - Enter: toggle the checked state of the currently selected todo.
  - `q` or `Esc`: exit the TUI.
- Navigation behavior:
  - Moving up or down SHALL move the selected indicator accordingly.
  - If the selection attempts to move past the first or last item, the implementation MAY either wrap around or clamp to the edge, but SHALL do so consistently.
- Toggling behavior:
  - Pressing Enter SHALL flip the checkbox of the selected todo between `- [ ]` and `- [x]`.
  - After toggling, the visual state SHALL update immediately in the TUI.
- Persistence:
  - After each toggle, the updated todo state SHALL be written back to `todo.md` using the atomic write behavior defined in previous requirements.
  - The written file SHALL reflect all toggles performed so far.

- Exit behavior:
  - Pressing `q` or `Esc` SHALL terminate the TUI and exit the process.
  - On normal exit (including via `q` or `Esc`), the process exit code SHALL be `0`, assuming no I/O errors occurred.

#### Scenario: Toggle selected todo and persist
- **WHEN** the user runs `tdx` to launch the TUI  
- **AND** navigates to a todo using `j` or `k` (or arrow keys)  
- **AND** presses Enter  
- **THEN** the selected todo’s checkbox state SHALL flip (checked ↔ unchecked) in the UI  
- **AND** `todo.md` on disk SHALL be updated immediately to reflect the new state  
- **AND** relaunching `tdx` SHALL show the updated state.

#### Scenario: Quit TUI without error
- **WHEN** the user runs `tdx` to launch the TUI  
- **AND** presses `q` or `Esc`  
- **THEN** the TUI SHALL exit cleanly  
- **AND** the process exit code SHALL be `0` if no errors occurred during I/O.

---

### Requirement: Non-TUI CLI commands
The CLI SHALL provide non-interactive commands for listing, adding, toggling, and editing todos without launching the TUI.

- The CLI commands SHALL follow this behavior:

  - `tdx`  
    - With no additional arguments, SHALL launch the interactive TUI.

  - `tdx list`  
    - SHALL list all todos in `todo.md` in a concise, human-readable format.
    - Output SHALL include each todo’s index (1-based), checkbox state, and text.
    - Output styling SHALL be consistent with the TUI style (checkbox symbols, tasteful colors).

  - `tdx add "Text"`  
    - SHALL append a new unchecked todo with the given text to `todo.md`.
    - The new todo line SHALL use the syntax: `- [ ] Text`.

  - `tdx toggle <index>`  
    - SHALL toggle the checked state of the todo at the given 1-based index.
    - If `<index>` is out of range or non-numeric, the command SHALL fail with a clear error and SHALL NOT modify `todo.md`.

  - `tdx edit <index> "New text"`  
    - SHALL replace the text of the todo at the given 1-based index with `"New text"`, keeping the checkbox state unchanged.
    - If `<index>` is out of range or non-numeric, the command SHALL fail with a clear error and SHALL NOT modify `todo.md`.

- These commands SHALL operate purely in non-interactive mode and SHALL NOT launch the TUI.
- All modifications triggered by these commands SHALL use the atomic write behavior defined earlier.

#### Scenario: List todos non-interactively
- **WHEN** the user runs `tdx list` in a directory with a valid `todo.md` containing multiple todos  
- **THEN** the CLI SHALL print each todo on its own line with:
  - A 1-based index
  - The checkbox state (`[ ]` or `[x]`)
  - The todo text  
- **AND** the output SHALL be minimal, readable, and styled consistently with the TUI’s use of brackets and colors.

#### Scenario: Add a new todo
- **WHEN** the user runs `tdx add "Buy milk"`  
- **THEN** the system SHALL append a new line `- [ ] Buy milk` at the appropriate place in `todo.md` (typically at the end, preserving any non-todo content structure)  
- **AND** subsequent invocations of `tdx` or `tdx list` SHALL show this new item as an unchecked todo.

#### Scenario: Toggle a todo by index
- **WHEN** the user runs `tdx toggle 3` in a directory where at least three todos exist  
- **THEN** the third todo (by 1-based index among todos) SHALL have its checkbox state toggled  
- **AND** `todo.md` SHALL be updated using atomic write semantics  
- **AND** rerunning `tdx list` SHALL show the updated state.

#### Scenario: Edit a todo by index
- **WHEN** the user runs `tdx edit 2 "New text"` in a directory where at least two todos exist  
- **THEN** the second todo (by 1-based index among todos) SHALL have its text replaced with `New text`  
- **AND** the checkbox state for that todo SHALL remain as it was (checked or unchecked)  
- **AND** `todo.md` SHALL be updated using atomic write semantics.

---

### Requirement: Error handling and output style
The CLI and TUI SHALL provide clear, minimal error handling and consistent, tasteful use of colors.

- For successful operations:
  - The CLI and TUI SHALL exit with status code `0`.
  - Output SHALL be minimal and focused on the main result (e.g., a list of todos or a simple confirmation).
- For error conditions (e.g., invalid index, unreadable/writable file, internal parsing errors):
  - The CLI SHALL:
    - Print a clear error message to standard error.
    - Use a visible color (e.g., red) and a short prefix such as `Error:` or similar.
    - Exit with a non-zero exit code.
  - The system SHALL avoid printing stack traces or overly verbose debug information by default.

- Output styling:
  - SHALL be consistent between TUI and non-TUI commands in terms of checkbox symbols and overall aesthetic.
  - SHALL avoid excessive color or formatting that reduces readability.

#### Scenario: Invalid index on toggle
- **WHEN** the user runs `tdx toggle 999` in a directory where fewer than 999 todos exist  
- **THEN** the CLI SHALL:
  - Print a clear error message indicating that the index is out of range.
  - NOT modify `todo.md`.
  - Exit with a non-zero status code.

#### Scenario: File I/O error
- **WHEN** a command attempts to read or write `todo.md` but the operation fails (e.g., due to permissions issues or a read-only filesystem)  
- **THEN** the CLI SHALL:
  - Print a clear, colored error message describing that the file could not be read or written.
  - Leave `todo.md` unmodified if the error occurs before a successful atomic replace.
  - Exit with a non-zero status code.

#### Scenario: Minimal successful command output
- **WHEN** the user runs a valid command like `tdx add "Task"` or `tdx toggle 1`  
- **THEN** the CLI SHALL exit with status code `0`  
- **AND** any printed output SHALL be concise and human-readable, without extraneous debug information.