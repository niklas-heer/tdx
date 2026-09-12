## MODIFIED Requirements
### Requirement: Structural regression campaigns
Regression checks SHALL cover nested/ordered tasks, sections, multiline bodies, unrelated Markdown preservation and exact undo. Failing generated campaigns SHALL retain replayable reduced cases or clearly report why reduction is unavailable. Scheduled campaigns SHALL record rotating seeds and retain failure artifacts.

#### Scenario: A generated edit sequence loses unrelated content
- **WHEN** a generated edit sequence loses unrelated content
- **THEN** the campaign fails and retains a reproducible regression artifact

#### Scenario: Structural content changes ownership
- **WHEN** an edit unexpectedly relocates a protected block, detaches a descendant, or attaches a task body to the wrong task
- **THEN** the structural campaign SHALL reject the resulting state even if content occurrence counts remain unchanged
