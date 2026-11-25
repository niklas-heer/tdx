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
  - Pressing `q` or `Esc` SHALL terminate the TUI and exit the process, unless the document has changes that are not on disk.
  - In the interactive TUI, with unsaved changes, the first quit key (`Esc`, `Ctrl+C` or `Ctrl+D`) SHALL keep the TUI open and show a one-line warning that the changes would be lost, in place of any other view or overlay so it is never hidden; pressing a quit key again after the warning has been shown SHALL exit without saving, while a quit key arriving together with the first (within a short delay) or reported as a key repeat SHALL NOT confirm. Any other key SHALL cancel the pending quit.
  - Piped (non-interactive) input has no one to confirm a quit; it SHALL keep stopping at `q` or `Esc`, and no quit key SHALL show a warning or consume the next scripted key.
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

#### Scenario: Quit with unsaved changes
- **WHEN** the user has unsaved changes, for example in manual-save mode
- **AND** presses `Esc`
- **THEN** the TUI SHALL stay open and warn that the changes are unsaved
- **AND** pressing `Esc` again SHALL exit without writing the file
