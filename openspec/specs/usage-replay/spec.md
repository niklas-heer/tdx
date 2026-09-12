# usage-replay Specification

## Purpose
Provide reproducible sustained-use contracts that expose editing and persistence regressions and support comparisons with future tdx implementations.
## Requirements
### Requirement: Reproducible sustained usage
The developer harness SHALL generate versioned language-neutral traces with deterministic seeds, independent expected task states, and explicit simulated durations. It SHALL report actual elapsed time separately and fail on state, persistence or replay-contract violations.

#### Scenario: Long campaign
- **WHEN** a developer runs the default campaign
- **THEN** at least 100 simulated hours and 36,000 actions are exercised across multiple sessions
- **AND** failing runs retain a replayable prefix with the seed and action index

#### Scenario: Alternative implementation
- **WHEN** a compatible executable is supplied
- **THEN** the same CLI task-state contract is checked without importing its implementation
- **AND** unsupported terminal/editor operations are reported rather than counted as passed

### Requirement: Bounded continuous regression checks
Normal tests SHALL include bounded seeded replay and regression cases discovered by sustained usage; a separate task SHALL exercise a real terminal process with isolated configuration and history.

#### Scenario: Cancellation and conflicts
- **WHEN** input is cancelled at undo capacity or a save meets an external edit
- **THEN** earlier undo entries and external disk content remain protected

### Requirement: Structural regression campaigns
Regression checks SHALL cover nested/ordered tasks, sections, multiline bodies, unrelated Markdown preservation and exact undo. Failing generated campaigns SHALL retain replayable reduced cases or clearly report why reduction is unavailable. Scheduled campaigns SHALL record rotating seeds and retain failure artifacts.

#### Scenario: A generated edit sequence loses unrelated content
- **WHEN** a generated edit sequence loses unrelated content
- **THEN** the campaign fails and retains a reproducible regression artifact

#### Scenario: Structural content changes ownership
- **WHEN** an edit unexpectedly relocates a protected block, detaches a descendant, or attaches a task body to the wrong task
- **THEN** the structural campaign SHALL reject the resulting state even if content occurrence counts remain unchanged

### Requirement: Native executable coverage
CI SHALL exercise CLI contracts on supported native macOS and Windows runners and terminal contracts on macOS and Linux, with Windows terminal checks where supported by ConPTY.

#### Scenario: Native verification runs
- **WHEN** native verification runs
- **THEN** argument handling, paths, Unicode, configuration isolation and terminal lifecycle are checked on their native platforms
