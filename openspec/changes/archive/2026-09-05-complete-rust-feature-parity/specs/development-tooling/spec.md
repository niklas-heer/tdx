## ADDED Requirements

### Requirement: Full-featured Rust comparison candidate
The standalone Rust application SHALL implement the documented Go CLI, Markdown document operations, all TUI commands and controls, configuration/theme precedence, recent-file state, clipboard, history/recovery and platform persistence. The runtime SHALL NOT invoke Go to implement features. A checked parity matrix SHALL link every feature group to executable evidence.

#### Scenario: Equivalent application services
- **WHEN** either executable runs the same document actions and terminal workflow
- **THEN** task/heading semantics, configuration effects, persisted state and recoverability SHALL match
- **AND** checkbox edits SHALL preserve unrelated source bytes
- **AND** terminal controls and available information SHALL be equivalent even when renderer internals differ

### Requirement: Differential bug handling
The comparison SHALL distinguish genuine compatibility failures from demonstrated Go bugs. Data loss or violation of documented behavior SHALL receive a shared intended-behavior regression and a focused correction instead of a silent Rust exception.

#### Scenario: Reference implementation loses settings
- **WHEN** a differential case demonstrates unrelated settings or document content being lost
- **THEN** the defect SHALL be fixed and tested in the affected implementation
- **AND** the comparison SHALL record the corrected behavior

### Requirement: Parity-gated performance results
New full-featured comparison results SHALL run after document, CLI, TUI, persistence and supported-platform parity checks. They SHALL use equivalent services, preserve historical baselines, record source/build identities and distinguish measured results from unverified behavior. An incomplete group SHALL prevent a claim of full parity.

#### Scenario: Required contract fails
- **WHEN** any required parity group fails or lacks native evidence
- **THEN** the runner SHALL report that gap and SHALL NOT label the candidate fully parity-compliant
