## ADDED Requirements
### Requirement: Complete Ratatui presentation parity
The standalone Rust application SHALL use Ratatui for a complete terminal interface with the documented Go commands, modes, contextual information, themes and terminal hyperlinks. Layout may improve while feature behavior and on-disk compatibility remain intact. Long Unicode input SHALL keep its cursor visible; narrow terminal layouts SHALL retain navigation and recovery actions. Help and diff views SHALL expose their complete content through bounded scrolling.

#### Scenario: Inspect the full application
- **WHEN** a user opens tasks, filters, sections, commands, recent files, themes or history
- **THEN** the Rust UI SHALL preserve required information and actions with readable contextual styling
- **AND** rendered-content assertions and native terminal contracts SHALL validate those behaviors beyond merely accepting input without panic

#### Scenario: Review the actual UI
- **WHEN** the presentation changes
- **THEN** reproducible previews SHALL be generated from actual Ratatui buffers
- **AND** historical benchmarks SHALL remain identified by their measured source revision
