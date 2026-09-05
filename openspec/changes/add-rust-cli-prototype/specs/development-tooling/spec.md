## ADDED Requirements

### Requirement: Runnable Rust rewrite evaluation
An isolated experimental Rust CLI SHALL support list, JSON task queries, add, edit, delete and toggle for its documented document subset. It SHALL reject unsupported edits explicitly, preserve unrelated bytes for checkbox changes, and guard file replacement against stale reads. It SHALL NOT replace the production Go application.

#### Scenario: Executable compatibility
- **WHEN** the existing CLI usage harness runs an identical trace against Go and the Rust candidate
- **THEN** both implementations SHALL be checked against the same independent task-state oracle, including metadata and persisted task lines
- **AND** unsupported application contracts SHALL be recorded as feature gaps rather than passing checks

### Requirement: Reproducible application comparison
The evaluation SHALL gate timings on correctness, alternate repeated implementation order, use identical input documents and release builds, and record raw timings, toolchains, host, source fingerprint, binary sizes and process memory. Results SHALL distinguish matched workloads from differences in history, configuration or persistence guarantees.

#### Scenario: Rewrite recommendation
- **WHEN** the experiment completes
- **THEN** its report SHALL explain observed latency and resource differences, implementation effort and remaining compatibility work
- **AND** the recommendation SHALL account for current optimized Go behavior rather than relying on the historical parser-only ratio

### Requirement: Basic Rust terminal prototype
The candidate SHALL provide an interactive terminal editor with navigation, toggle, add, edit, delete, bounded undo, cancellation, reload and read-only handling. Failed saves SHALL retain the local candidate and report conflicts without overwriting observed external content.

#### Scenario: Interactive feasibility
- **WHEN** real PTY tests exercise Unicode editing, cancellation, undo, resize and an external edit during pending input
- **THEN** the candidate SHALL preserve the expected document and restore terminal mode on normal exit
- **AND** missing advanced TUI and history features SHALL remain explicit rewrite gaps
