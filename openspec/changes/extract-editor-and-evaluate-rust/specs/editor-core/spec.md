## ADDED Requirements
### Requirement: Shared document actions
The CLI and TUI SHALL express task and heading mutations as actions in a UI-independent editor package, sharing index validation and document mutation behavior. Invalid actions SHALL return errors without changing document content. UI selection and filter navigation SHALL remain outside the editor package.
#### Scenario: Same edit from two interfaces
- **WHEN** the CLI and TUI apply equivalent task edits
- **THEN** both SHALL invoke the same action implementation and produce equivalent document content

### Requirement: Bounded document undo
A UI-independent history SHALL capture document snapshots, cap retained entries at 100, and restore content without restoring stale disk revisions. Cancelled input and move operations SHALL preserve earlier undo entries.
#### Scenario: Undo after successful saves
- **WHEN** an edit is undone after intervening successful saves
- **THEN** the restored content SHALL use the current disk revision for conflict detection

### Requirement: Independent application instances
CLI/TUI instances SHALL receive configuration, styles, and history services explicitly. Markdown persistence SHALL use per-store read/write callbacks, with hook-free default helpers. Initializing or using one instance SHALL not change the configuration, styles, or history destination of another instance.
#### Scenario: Concurrent independent projects
- **WHEN** two instances use different styles and history callbacks
- **THEN** each SHALL produce output and history only through its own dependencies
