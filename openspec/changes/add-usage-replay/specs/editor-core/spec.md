## ADDED Requirements
### Requirement: Source-preserving checkbox editing
Toggling a task in a freshly parsed document SHALL change only the AST-identified checkbox byte. Repeated toggles, snapshots, saves and undo SHALL retain surrounding Markdown, frontmatter and line endings while settings are unchanged. Checkbox changes SHALL update cached checked state without re-extracting task text and metadata.

#### Scenario: Rich Markdown
- **WHEN** a checkbox is toggled in a file with HTML, multiline paragraphs, tables, fenced examples or unknown frontmatter
- **THEN** unrelated bytes remain identical through save and undo

#### Scenario: Structural edit fallback
- **WHEN** a task is added, removed, moved or renamed
- **THEN** the AST serializer remains the structural editing path and subsequent checkbox edits reflect the current tree rather than stale source offsets
