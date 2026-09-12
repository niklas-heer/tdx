## ADDED Requirements

### Requirement: Safe structural Markdown editing
Task and heading edits SHALL retain unrelated Markdown content, including reference definitions, HTML, tables, multiline bodies and line-ending boundaries. Unsupported operations SHALL fail without changing document content.

#### Scenario: A task is renamed near a reference link
- **WHEN** a task is renamed near a reference link
- **THEN** the reference definition and unrelated content remain intact through saving and undo

## MODIFIED Requirements
### Requirement: Source-preserving checkbox editing
Toggling a task in a freshly parsed document SHALL change only the AST-identified checkbox byte. Repeated toggles, snapshots, saves and undo SHALL retain surrounding Markdown, frontmatter and line endings while settings are unchanged. Checkbox changes SHALL update cached checked state without re-extracting task text and metadata.

#### Scenario: Rich Markdown
- **WHEN** a checkbox is toggled in a file with HTML, multiline paragraphs, tables, fenced examples or unknown frontmatter
- **THEN** unrelated bytes remain identical through save and undo

#### Scenario: Structural source patches
- **WHEN** a task is added, removed, moved or renamed
- **THEN** validated patches SHALL update only affected source ranges and reparse the candidate document before committing
- **AND** unsupported source boundaries SHALL produce an error without mutation
- **AND** subsequent checkbox edits SHALL use the reparsed source locations
