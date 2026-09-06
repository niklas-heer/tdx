## ADDED Requirements
### Requirement: Inline terminal context and task editing
The Rust interactive interface SHALL use a compact region of the normal terminal buffer, preserve preceding terminal output, and place task creation and editing within the task list.

#### Scenario: Keep preceding work visible
- **WHEN** a user opens a short checklist in a terminal containing earlier output
- **THEN** the checklist occupies its content height and leaves preceding output visible when space permits
- **AND** resizing and leaving the application do not clear unrelated terminal rows or enter an alternate screen

#### Scenario: Edit and insert beside tasks
- **WHEN** a user edits a task or inserts a new task after the selected task
- **THEN** the text editor and cursor appear at that task or adjacent insertion position
- **AND** Enter commits through the existing guarded save and undo path while Escape cancels

#### Scenario: Append and larger tools
- **WHEN** a user appends a task, opens a larger tool, or uses a long checklist
- **THEN** the active row stays visible within a terminal-sized region
- **AND** returning to a short checklist shrinks the managed region without erasing preceding output
