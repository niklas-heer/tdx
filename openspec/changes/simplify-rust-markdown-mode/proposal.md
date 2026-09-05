## Why
Markdown mode is an occasional tool for editing headings and text omitted from the normal checklist. A second rendered view duplicates the checklist and takes space from source editing.

## What Changes
- Open the entire raw document in a full-width source editor through either existing Markdown command.
- Remove rendered preview and preview controls; keep normal checklist rendering, guarded saves, undo and draft retention.
- Update existing UI/native checks and the current usage guide.

## Approval
The user explicitly requested this adjustment to the implemented Markdown mode. This proposal records that authorized scope.

## Impact
- Affected spec: development-tooling (Rust full-document Markdown editing).
- Affected code: Rust TUI, obsolete preview renderer, native Markdown checks, Miri groups and usage guide.
