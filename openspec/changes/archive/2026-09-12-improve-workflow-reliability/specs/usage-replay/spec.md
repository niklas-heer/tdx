## ADDED Requirements

### Requirement: Structural regression campaigns
Regression checks SHALL cover nested/ordered tasks, sections, multiline bodies, unrelated Markdown preservation and exact undo. Failing generated campaigns SHALL retain replayable reduced cases or clearly report why reduction is unavailable. Scheduled campaigns SHALL record rotating seeds and retain failure artifacts.

#### Scenario: A generated edit sequence loses unrelated content
- **WHEN** a generated edit sequence loses unrelated content
- **THEN** the campaign fails and retains a reproducible regression artifact

### Requirement: Native executable coverage
CI SHALL exercise CLI contracts on supported native macOS and Windows runners and terminal contracts on macOS and Linux, with Windows terminal checks where supported by ConPTY.

#### Scenario: Native verification runs
- **WHEN** native verification runs
- **THEN** argument handling, paths, Unicode, configuration isolation and terminal lifecycle are checked on their native platforms

