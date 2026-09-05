## ADDED Requirements
### Requirement: Deterministic Rust save simulation
The Rust candidate SHALL execute the same save protocol in native operation and in a seeded simulation with controlled time and I/O outcomes. A fixed seed and simulator version SHALL produce an identical ordered trace. Reports SHALL identify tested source, fault coverage, safety and recovery results, and modeled limitations. Native persistence checks SHALL remain required.

#### Scenario: Fault and recovery replay
- **WHEN** simulated writers encounter delayed I/O, contention, failed writes, unavailable history or process/power failures
- **THEN** the simulator SHALL check that uncommitted failures preserve target contents, committed outcomes are reported accurately, stale saves are rejected and successful durable saves survive modeled power loss
- **AND** writers SHALL make progress after faults stop and revisions are reloaded
- **AND** injected implementation defects SHALL be detected by independent checks

### Requirement: Rust full-document Markdown editing
The Rust candidate SHALL provide full-document Unicode source editing and rendered Markdown preview using the existing save/history engine. Existing Go-compatible commands SHALL remain unchanged.

#### Scenario: Edit source with preview
- **WHEN** the user opens the document editor
- **THEN** the source draft SHALL remain isolated until explicit save, with a rendered preview available on wide and narrow terminals
- **AND** saving SHALL preserve the exact draft bytes, update derived tasks and metadata, and participate in undo/history

#### Scenario: External change during source editing
- **WHEN** the file changes externally before a draft is saved
- **THEN** the save SHALL reject the stale revision without overwriting the file or replacing the editor's accepted document
- **AND** the draft SHALL remain available until the user explicitly discards it
