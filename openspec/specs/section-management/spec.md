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
The TUI SHALL support focusing a section including its descendants, folding sections, and returning to all tasks. Existing task filters SHALL compose with section visibility.
#### Scenario: Focus a project
- **WHEN** a user focuses a heading
- **THEN** only tasks in that section and its descendant sections SHALL appear
- **AND** the active focus and a way to clear it SHALL be visible
