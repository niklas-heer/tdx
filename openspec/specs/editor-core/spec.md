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

### Requirement: Source-preserving checkbox editing
Toggling a task in a freshly parsed document SHALL change only the AST-identified checkbox byte. Repeated toggles, snapshots, saves and undo SHALL retain surrounding Markdown, frontmatter and line endings while settings are unchanged. Checkbox changes SHALL update cached checked state without re-extracting task text and metadata.

#### Scenario: Rich Markdown
- **WHEN** a checkbox is toggled in a file with HTML, multiline paragraphs, tables, fenced examples or unknown frontmatter
- **THEN** unrelated bytes remain identical through save and undo

#### Scenario: Structural edit fallback
- **WHEN** a task is added, removed, moved or renamed
- **THEN** the AST serializer remains the structural editing path and subsequent checkbox edits reflect the current tree rather than stale source offsets

### Requirement: Derived task metadata stays current
Available tag and priority options SHALL reflect current task content after editing, undo and reload, removing active filters for metadata no longer present.

#### Scenario: Undo a newly introduced priority
- **WHEN** a task introducing a priority or tag is added and then undone
- **THEN** the picker stops offering the removed metadata

### Requirement: Structural serialization retains opaque content
Structural task edits SHALL retain HTML blocks, valid table syntax and ordinary paragraph line breaks even when formatting is normalized.

#### Scenario: Add a task after rich content
- **WHEN** a task is added to a document containing HTML, a table and a multiline paragraph
- **THEN** those blocks remain present and the saved file can be reopened with all tasks intact
