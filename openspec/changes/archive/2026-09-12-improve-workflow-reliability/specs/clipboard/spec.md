## ADDED Requirements

### Requirement: Platform clipboard feedback
Clipboard operations SHALL support macOS, Linux and Windows through available platform or terminal mechanisms. Failed or unavailable copy/paste SHALL report an actionable error and SHALL NOT report successful copying.

#### Scenario: The clipboard backend is unavailable
- **WHEN** the clipboard backend is unavailable
- **THEN** the document remains unchanged and an error is displayed
