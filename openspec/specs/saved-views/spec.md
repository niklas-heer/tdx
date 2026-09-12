# saved-views Specification

## Purpose
Let users save and restore named task views per file without changing their Markdown documents.
## Requirements
### Requirement: Named per-file views
The TUI SHALL save, load, list and delete named views per canonical file. Views SHALL capture section focus/folds and tag, priority, due-date and completion filters, with explicit active-view feedback. Restoration on opening SHALL be opt-in.

#### Scenario: A user saves and reopens a project view
- **WHEN** a user saves and reopens a project view
- **THEN** the same filters and valid section context are applied without changing Markdown

### Requirement: Compatible manual-save mode
The TUI SHALL describe edits that require explicit saving as manual-save mode and retain existing read-only flags/configuration as compatible aliases. CLI read-only mutation rejection SHALL remain intact.

#### Scenario: An existing read-only checklist is opened
- **WHEN** an existing read-only checklist is opened
- **THEN** temporary edits remain possible and only an explicit save writes them
