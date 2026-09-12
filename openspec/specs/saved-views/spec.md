# saved-views Specification

## Purpose
Let users save and restore named task views per file without changing their Markdown documents.
## Requirements
### Requirement: Named per-file views
The TUI SHALL save, load, list and delete named views per canonical file. Views SHALL capture section focus/folds and tag, priority, due-date and completion filters, with explicit active-view feedback. Restoration on opening SHALL be opt-in.

#### Scenario: A user saves and reopens a project view
- **WHEN** a user saves a project view and either explicitly loads it or reopens the project with restoration enabled
- **THEN** the same filters and valid section context are applied without changing Markdown

#### Scenario: Default opening leaves saved views inactive
- **WHEN** a project with a saved view is opened without opting into restoration or explicitly loading the view
- **THEN** the saved view SHALL NOT be restored automatically

#### Scenario: Blank headings are valid saved context
- **WHEN** an empty heading is focused, folded or an ancestor of saved section context
- **THEN** its empty label and occurrence SHALL round-trip without rejecting the view

#### Scenario: Restore a recent file cursor
- **WHEN** a user switches to a recent file with a remembered task position
- **THEN** cursor visibility SHALL be evaluated after the destination file's metadata and optional saved view are applied
- **AND** a hidden remembered position SHALL fall back to a visible task

### Requirement: Compatible manual-save mode
The TUI SHALL describe edits that require explicit saving as manual-save mode and retain existing read-only flags/configuration as compatible aliases. CLI read-only mutation rejection SHALL remain intact.

#### Scenario: An existing read-only checklist is opened
- **WHEN** an existing read-only checklist is opened
- **THEN** temporary edits remain possible and only an explicit save writes them

#### Scenario: A CLI mutation targets a read-only checklist
- **WHEN** a CLI mutation targets a checklist with read-only mode enabled
- **THEN** the mutation SHALL fail without modifying its Markdown
