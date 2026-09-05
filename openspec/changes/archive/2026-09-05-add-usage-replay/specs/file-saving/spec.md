## ADDED Requirements
### Requirement: Pending input retains its disk revision
The file watcher SHALL defer document reload while interactive add, edit, move or heading input is pending. Saving that input SHALL compare against the revision captured before input began.

#### Scenario: External writer during input
- **WHEN** another process changes the file while a user is editing a task
- **AND** a file-watch tick occurs before Enter
- **THEN** the user's save produces a conflict and preserves the external bytes
- **AND** explicit reload resolves the conflict using the current disk content
