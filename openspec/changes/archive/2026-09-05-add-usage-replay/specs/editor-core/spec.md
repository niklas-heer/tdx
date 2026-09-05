## ADDED Requirements
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
