## MODIFIED Requirements
### Requirement: Compatible manual-save mode
The TUI SHALL describe edits that require explicit saving as manual-save mode and retain existing read-only flags/configuration as compatible aliases. CLI mutations SHALL be rejected when read-only mode is in effect, decided with the same precedence as the TUI: a command-line flag, then the file's `read-only` frontmatter, then the global configuration.

#### Scenario: An existing read-only checklist is opened
- **WHEN** an existing read-only checklist is opened
- **THEN** temporary edits remain possible and only an explicit save writes them

#### Scenario: A CLI mutation targets a read-only checklist
- **WHEN** a CLI mutation targets a checklist with read-only mode enabled
- **THEN** the mutation SHALL fail without modifying its Markdown

#### Scenario: A checklist marks itself read-only
- **WHEN** a file's frontmatter sets `read-only: true` and a CLI mutation targets it without a read-only flag
- **THEN** the mutation SHALL fail on stderr without modifying the file or opening history storage

#### Scenario: A file opts out of a global manual-save default
- **WHEN** the global configuration enables manual save and a file's frontmatter sets `read-only: false`
- **THEN** CLI mutations of that file SHALL be allowed unless a read-only flag is given
