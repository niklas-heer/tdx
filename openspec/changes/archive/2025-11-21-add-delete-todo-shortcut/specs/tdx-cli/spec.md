## MODIFIED Requirements

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