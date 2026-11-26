# section-management Specification

## Purpose
Let users organize Markdown projects through a section overview, heading edits, focused task lists, and folding while preserving safe saves and existing task workflows.
## Requirements
### Requirement: Interactive sections
The TUI SHALL provide a discoverable section browser showing nested and empty Markdown headings and permitting heading creation and renaming through guarded saves and undo.
#### Scenario: Edit an empty section
- **WHEN** a user selects an empty section and renames it or adds a task
- **THEN** the change SHALL persist in that section without altering other sections
- **AND** read-only mode SHALL prevent writes

### Requirement: Section focus and folding
The TUI SHALL support focusing a section including its descendants, folding sections, and returning to all tasks. Existing task filters SHALL compose with section visibility. Undo SHALL keep section focus and folds when the restored document has the same headings.
#### Scenario: Focus a project
- **WHEN** a user focuses a heading
- **THEN** only tasks in that section and its descendant sections SHALL appear
- **AND** the active focus and a way to clear it SHALL be visible

#### Scenario: Undo a task change inside a focused section
- **WHEN** a user undoes a task change while a section is focused or folded
- **AND** the undo does not change any heading
- **THEN** the section focus and folds SHALL remain

#### Scenario: Undo a heading change
- **WHEN** an undo adds, removes, renames or re-levels a heading
- **THEN** section focus and folds SHALL be cleared
