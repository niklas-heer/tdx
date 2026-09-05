## ADDED Requirements

### Requirement: History-enabled Rust evaluation
The experimental Rust application SHALL capture versions for interactive and mutating use using the production Go schema, canonical paths, uncompressed SHA-256 identity, interoperable zstd content, bounded SQLite contention and configurable retention. List/help/version SHALL not create a version store.

#### Scenario: Interoperable recovery
- **WHEN** Go and Rust capture revisions of the same file in an isolated shared database
- **THEN** each implementation SHALL read the other's snapshots byte-for-byte
- **AND** duplicate content SHALL remain one version per file

#### Scenario: Commit and history failures
- **WHEN** snapshot capture fails after a successful file replacement
- **THEN** Rust SHALL report that the file committed and retain the new revision
- **AND** a force-save SHALL refuse replacement when capture of the overwritten revision fails

### Requirement: Rust terminal recovery milestone
The Rust TUI SHALL provide version listing/preview and confirmed conditional restore, idle external-change reload, and single-line parent-label editing that preserves children. Unsupported structural operations SHALL remain explicit errors.

#### Scenario: External edits during input or restore
- **WHEN** disk content changes while input or the version browser is open
- **THEN** automatic reload SHALL defer
- **AND** edit or restore SHALL conflict without overwriting external bytes
- **AND** a rejected restore SHALL preserve the active document

### Requirement: Comparable history workloads
The rewrite runner SHALL validate snapshot capture and retention before reporting save performance. It SHALL measure both repeated toggles and distinct-content edits with history enabled, preserve the earlier baseline, and report remaining service and platform differences.

#### Scenario: Benefits survive additional services
- **WHEN** the history-enabled comparison finishes
- **THEN** the report SHALL include repeated process and terminal observations, memory, build/binary/database costs and correctness outcomes
- **AND** it SHALL distinguish matched history semantics from differences in implementation libraries and remaining UI features
