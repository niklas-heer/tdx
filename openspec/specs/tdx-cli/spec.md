# tdx-cli Specification

## Purpose
Provide a standalone Go CLI and Bubble Tea TUI for managing Markdown tasks, with consistent storage, styling, and guarded atomic saves.
## Requirements
### Requirement: Go CLI entry point
The system SHALL provide a Go command-line tool named tdx with entry point cmd/tdx/main.go, built with the toolchain pinned in mise.toml and metadata from tdx.toml.
#### Scenario: Build and run tdx CLI
- **WHEN** a developer runs mise run build
- **THEN** a standalone tdx binary SHALL support interactive mode and the documented non-interactive commands
- **AND** its version SHALL match tdx.toml

---

### Requirement: Markdown todo storage in todo.md
The system SHALL use the configured file, defaulting to ./todo.md, as the task source of truth. Missing files SHALL be treated as empty documents without being created by listing or revision queries; a successful mutation SHALL create the file. Actual Goldmark task-list nodes, including supported unordered, ordered and nested containers, SHALL determine task indexes. Mutations SHALL preserve unrelated document bytes and use guarded atomic replacement. Unsupported operations SHALL fail without modifying the document.

#### Scenario: Query a missing file
- **WHEN** a user lists tasks or requests a revision for a nonexistent file
- **THEN** no file SHALL be created and the result SHALL represent an empty/missing document

#### Scenario: First successful edit
- **WHEN** a user adds a task to a nonexistent file
- **THEN** the file SHALL be created through the guarded save with the new task

#### Scenario: Preserve non-task Markdown
- **WHEN** a supported edit changes a task near reference definitions, HTML or tables
- **THEN** unrelated source bytes SHALL remain intact

### Requirement: AST-based Markdown parser and writer
The system SHALL use Goldmark to identify tasks and headings and validate source patches before committing edits. It SHALL preserve unrelated Markdown source and reject unsupported operations without mutation. The legacy serializer SHALL NOT silently rewrite unsupported source during normal editing.

#### Scenario: Round-trip consistency with no changes
- **WHEN** a document is parsed and serialized without edits
- **THEN** source content SHALL remain identical

#### Scenario: Correct parsing of examples
- **WHEN** a document includes fenced examples that resemble tasks
- **THEN** only actual task-list nodes SHALL be editable

### Requirement: TUI layout and styling
The system SHALL present an interactive terminal UI (TUI) using Bubble Tea that renders todos with a specific layout and styling.

- The TUI implementation SHALL use Bubble Tea v2.
- Styling and overlay composition SHALL use Lip Gloss v2.
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
The TUI SHALL support keyboard navigation, deletion, and toggling of todos, immediately persisting changes back to `todo.md`.

- The TUI SHALL respond to the following keys:
  - `j` or Down arrow: move the selection down by one todo.
  - `k` or Up arrow: move the selection up by one todo.
  - Enter: toggle the checked state of the currently selected todo.
  - `d`: delete the currently selected todo immediately (no confirmation prompt).
  - `q` or `Esc`: exit the TUI.
- Navigation behavior:
  - Moving up or down SHALL move the selected indicator accordingly.
  - If the selection attempts to move past the first or last item, the implementation MAY either wrap around or clamp to the edge, but SHALL do so consistently.
- Toggling behavior:
  - Pressing Enter SHALL flip the checkbox of the selected todo between `- [ ]` and `- [x]`.
  - After toggling, the visual state SHALL update immediately in the TUI.
- Deletion behavior:
  - Pressing `d` SHALL remove the selected todo from the list without a confirmation prompt.
  - After deletion, the selection SHALL move to the next available todo, or the previous one if the deleted todo was last; if no todos remain, the empty-state message SHALL render.
  - Each deletion SHALL be persisted using the same atomic write guarantees as other modifications.
- Persistence:
  - After each toggle or deletion, the updated todo state SHALL be written back to `todo.md` using the atomic write behavior defined in previous requirements.
  - The written file SHALL reflect all toggles and deletions performed so far.
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

#### Scenario: Delete selected todo
- **WHEN** the user runs `tdx`, selects a todo, and presses `d`  
- **THEN** the todo SHALL be removed from the UI immediately  
- **AND** `todo.md` SHALL be rewritten without that todo using atomic write semantics  
- **AND** the selection SHALL move to the next available todo (or the previous one if the deleted todo was last, or render the empty state if none remain).

---

### Requirement: Non-TUI CLI commands
The CLI SHALL provide non-interactive commands for listing, adding, toggling, editing, and deleting todos without launching the TUI.

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

  - `tdx delete <index>`  
    - SHALL remove the todo at the given 1-based index from `todo.md`.
    - If `<index>` is out of range or non-numeric, the command SHALL fail with a clear error and SHALL NOT modify `todo.md`.
    - Successful deletion SHALL confirm the removed todo text and exit with status code `0`.

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

#### Scenario: Delete a todo by index
- **WHEN** the user runs `tdx delete 4` in a directory where at least four todos exist  
- **THEN** the fourth todo SHALL be removed from `todo.md` using atomic write semantics  
- **AND** rerunning `tdx list` SHALL show one fewer todo with subsequent indices shifted down by one  
- **AND** invalid indices SHALL result in a non-zero exit code without modifying `todo.md`.

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

### Requirement: Scriptable task queries
The CLI SHALL support `list --json`, `--status all|open|done`, and exact `--tag` filters. JSON SHALL be an array with documented lowercase fields, original one-based task indexes, text, completion, depth, parent index, tags, priority, and optional due date. Filters SHALL compose without renumbering tasks. Listing SHALL not require a writable configuration/history directory or create missing files.

#### Scenario: Script reads a filtered project
- **WHEN** a developer runs `tdx --file tasks.md list --json --status open --tag backend`
- **THEN** stdout SHALL contain only a JSON array of matching tasks with original indexes
- **AND** an empty result SHALL be `[]`
- **AND** no history or recent-file data SHALL be written

### Requirement: Predictable command arguments and errors
The CLI SHALL support `--file`/`-f`, retain positional Markdown paths, honor `--` to delimit literal arguments, reject unknown options and unsupported list options before side effects, send errors to stderr with nonzero exit status, and reject CLI mutations in read-only mode. Command functions SHALL return errors instead of terminating the process.

#### Scenario: Task text resembles a flag
- **WHEN** a developer runs `tdx add -- --read-only`
- **THEN** the literal text `--read-only` SHALL be added

#### Scenario: Read-only script tries to mutate a file
- **WHEN** a developer runs a mutating command with read-only mode enabled
- **THEN** the command SHALL fail on stderr without changing the task file or opening history storage

### Requirement: Extended automation contracts
The CLI SHALL support composable due-date, priority and section queries, explicit idempotent done/undone commands, and optional revision-bearing queries and guarded mutations. Existing JSON array output SHALL remain compatible and unfiltered indexes SHALL be preserved.

#### Scenario: A script attempts a mutation using a stale revision
- **WHEN** a script attempts a mutation using a stale revision
- **THEN** the mutation fails without changing the document

### Requirement: Shell completion and schema policy
The CLI SHALL generate documented shell completions without reading task files or opening history. The JSON compatibility policy SHALL distinguish additive changes from breaking schema changes.

#### Scenario: A user requests shell completion
- **WHEN** a user requests shell completion
- **THEN** completion output is produced without task-file side effects

