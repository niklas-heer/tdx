## ADDED Requirements

### Requirement: Extended automation contracts
The CLI SHALL support composable due-date, priority and section queries, explicit idempotent done/undone commands, and optional revision-bearing queries and guarded mutations. Existing JSON array output SHALL remain compatible and unfiltered indexes SHALL be preserved.

#### Scenario: A script attempts a mutation using a stale revision
- **WHEN** a script attempts a mutation using a stale revision
- **THEN** the mutation fails without changing the document

### Requirement: Shell completion and schema policy
The CLI SHALL generate documented shell completions without reading task files or opening history. The JSON compatibility policy SHALL distinguish additive changes from breaking schema changes.

#### Scenario: A user requests shell completion
- **WHEN** a user requests shell completion
- **THEN** completion output is produced without task-file side effects

## MODIFIED Requirements
### Requirement: Markdown todo storage in todo.md
The system SHALL use the configured file, defaulting to ./todo.md, as the task source of truth. Missing files SHALL be treated as empty documents without being created by listing or revision queries; a successful mutation SHALL create the file. Actual Goldmark task-list nodes, including supported unordered, ordered and nested containers, SHALL determine task indexes. Mutations SHALL preserve unrelated document bytes and use guarded atomic replacement. Unsupported operations SHALL fail without modifying the document.

#### Scenario: Query a missing file
- **WHEN** a user lists tasks or requests a revision for a nonexistent file
- **THEN** no file SHALL be created and the result SHALL represent an empty/missing document

#### Scenario: First successful edit
- **WHEN** a user adds a task to a nonexistent file
- **THEN** the file SHALL be created through the guarded save with the new task

#### Scenario: Preserve non-task Markdown
- **WHEN** a supported edit changes a task near reference definitions, HTML or tables
- **THEN** unrelated source bytes SHALL remain intact

### Requirement: AST-based Markdown parser and writer
The system SHALL use Goldmark to identify tasks and headings and validate source patches before committing edits. It SHALL preserve unrelated Markdown source and reject unsupported operations without mutation. The legacy serializer SHALL NOT silently rewrite unsupported source during normal editing.

#### Scenario: Round-trip consistency with no changes
- **WHEN** a document is parsed and serialized without edits
- **THEN** source content SHALL remain identical

#### Scenario: Correct parsing of examples
- **WHEN** a document includes fenced examples that resemble tasks
- **THEN** only actual task-list nodes SHALL be editable
