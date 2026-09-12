## ADDED Requirements
### Requirement: Bounded inline terminal frames
The interactive TUI SHALL keep each rendered frame within the measured terminal width and height, reserving a row for the inline cursor. Headings, wrapped task content, section banners and status rows SHALL count toward the height budget. The selected task or text cursor SHALL remain in the visible task viewport. Before terminal dimensions arrive, the interactive application SHALL defer document rendering.

#### Scenario: Focus after an overflowing document
- **WHEN** a user navigates a document with many headings and wrapped tasks and focuses a section
- **THEN** prior frames SHALL NOT leave repeated headings in terminal scrollback
- **AND** the focused section and selected task SHALL remain visible

#### Scenario: Resize with an active editor or overlay
- **WHEN** the terminal becomes narrower or shorter during editing or an overlay
- **THEN** the frame SHALL fit the new dimensions without terminal soft wrapping or excess rows
