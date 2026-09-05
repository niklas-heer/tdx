## MODIFIED Requirements
### Requirement: Rust full-document Markdown editing
The Rust candidate SHALL provide full-document Unicode source editing as an additional tool for quick document changes using the existing save/history engine. Existing Go-compatible commands SHALL remain unchanged.

#### Scenario: Edit the complete source
- **WHEN** the user opens the document editor
- **THEN** the source draft SHALL remain isolated until explicit save, with the full raw Markdown available in a full-width editor on wide and narrow terminals
- **AND** Markdown mode SHALL show headings, tasks, frontmatter, prose and other source content without a rendered preview pane
- **AND** the normal checklist SHALL retain its existing headings-and-tasks presentation
- **AND** saving SHALL preserve the exact draft bytes, update derived tasks and metadata, and participate in undo/history

#### Scenario: External change during source editing
- **WHEN** the file changes externally before a draft is saved
- **THEN** the save SHALL reject the stale revision without overwriting the file or replacing the editor's accepted document
- **AND** the draft SHALL remain available until the user explicitly discards it

#### Scenario: Edit source with preview
- **WHEN** the user invokes the former preview entry point `:markdown`
- **THEN** it SHALL open the complete source editor directly, without a preview
- **AND** `:edit-markdown` SHALL open the same source editor
