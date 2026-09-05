# editor-core Specification

## Purpose
Keep document actions, undo, and application dependencies independent of terminal input while preserving consistent editing behavior.

## Requirements

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

### Requirement: Preserve task container structure
Serializing task edits SHALL preserve ordered-list numbering and delimiters, nested-list indentation, and blockquote prefixes on every rendered line.
#### Scenario: Toggle a nested quoted or numbered task
- **WHEN** a task inside a blockquote or ordered list is toggled and saved
- **THEN** reparsing SHALL preserve task count, nesting depths, and untoggled checkbox states

### Requirement: Current query action selection
Command and search selection SHALL apply any pending query update before executing, completing, or navigating the selection.
#### Scenario: Enter arrives before the debounce timer
- **WHEN** a user types a command and immediately presses Enter
- **THEN** only a command matching the current query SHALL execute, regardless of pending debounce messages
