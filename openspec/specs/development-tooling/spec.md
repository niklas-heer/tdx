# development-tooling Specification

## Purpose
Define reproducible mise tasks, pinned development tools, compatible application and CI toolchains, and command-layer regression coverage.

## Requirements

### Requirement: Markdown task interface
The project SHALL expose maintained developer tasks through mise.toml with pinned tools and descriptions, replacing Mask.
- Tasks SHALL cover setup, build, install, development, todo commands, formatting, lint, tests, CI, and release artifacts.
- Local installation SHALL default to a user-writable binary directory.
#### Scenario: Developer starts from a fresh clone
- **WHEN** a developer trusts the project and runs mise install followed by mise run check
- **THEN** the documented tools SHALL be available and validation SHALL execute with pinned versions
#### Scenario: Developer discovers and runs a task
- **WHEN** a developer runs mise tasks
- **THEN** task names and descriptions SHALL be visible
- **AND** a documented task SHALL execute the corresponding operation

### Requirement: Synchronized Go toolchain
The application module and portable build container SHALL use the same current Go release. The isolated Dagger SDK module SHALL target the newest Go release supported by its generator; compatibility exceptions SHALL be documented.
#### Scenario: Go version is updated
- **WHEN** the application Go release is updated
- **THEN** mise and the portable build container SHALL be updated together
- **AND** the Dagger SDK module SHALL remain compatible with its generator
- **AND** portable and native test suites SHALL pass

### Requirement: Command-layer regression coverage

Automated tests SHALL exercise the successful file-mutating behavior of the non-interactive todo command layer.

#### Scenario: Todo command behavior regresses

- **WHEN** list, add, toggle, edit, delete, or valid command dispatch stops producing the expected file or output state
- **THEN** the command package test suite SHALL fail

### Requirement: Safe developer feedback loop
Mise SHALL provide an isolated demo with disposable todo/config/history data, forward arguments to focused Go tests, and generate text and HTML coverage reports in an ignored output directory. Demo cleanup SHALL occur on exit without changing repository examples or personal configuration.

#### Scenario: Contributor experiments with the TUI
- **WHEN** a contributor runs `mise run demo`
- **THEN** tdx SHALL open a temporary copy of the project example with isolated configuration and history
- **AND** changes SHALL be discarded when the demo exits

#### Scenario: Contributor runs a focused regression
- **WHEN** a contributor runs `mise run test -- ./internal/cmd -run TestList`
- **THEN** Go SHALL receive those exact arguments
- **AND** invoking the task without arguments SHALL run all application packages

### Requirement: Evidence-based Rust evaluation
The project SHALL provide an isolated Rust prototype and Go comparison probes, shared correctness fixtures, reproducible benchmarks, Go CPU/allocation profiles, Rust stack samples where the host profiler is supported, and a report recording toolchains, host, methodology, results, and feature gaps. The existing portable Dagger CI SHALL check Rust formatting, lint, and the shared correctness corpus without running performance timing gates. The application SHALL remain Go and pre-1.0. Generated binaries and raw profiles SHALL be ignored build outputs; the report MAY include a checked-in measurement snapshot; runs SHALL never edit user todo files.
#### Scenario: Maintainer evaluates a rewrite
- **WHEN** a maintainer runs the documented evaluation task
- **THEN** correctness checks SHALL precede timing comparisons
- **AND** results SHALL distinguish comparable parser work from unequal application feature scope
- **AND** the report SHALL identify untested platforms and missing Rust persistence/TUI/history guarantees

### Requirement: Reviewed release documentation
The release-note generator SHALL prefer the checked-in `docs/releases/<version>.md` for a matching semantic version tag. This path SHALL work without external text-generation credentials and SHALL reject empty reviewed notes. Existing generation remains available when reviewed notes are absent.

#### Scenario: Publish a reviewed release
- **WHEN** release notes exist for the current version tag
- **THEN** the published notes are exactly the reviewed content without an external API request

#### Scenario: Invalid or missing reviewed notes
- **WHEN** a tag is not a supported semantic version
- **THEN** it cannot select a file outside the release-notes directory
- **AND** missing notes use the existing generation path while empty notes fail clearly

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

### Requirement: Hardened Rust development workflow
The complete Rust rewrite SHALL use a dated nightly toolchain with reproducible formatter, Clippy and editor support, while retaining a tested stable compatibility floor. Local tasks and native CI SHALL check all application targets and features, enforce selected strict production safety lints, run isolated tests and validate documentation. Developer tools SHALL remain separate from application dependencies. Exceptions SHALL be narrow and explained; guidance without a concrete application benefit SHALL be recorded rather than installed indiscriminately.

#### Scenario: Safety finding in the complete rewrite
- **WHEN** a lint, interpreted test or compatibility contract detects an unsafe assumption or incorrect boundary behavior
- **THEN** the underlying defect SHALL be corrected with a focused regression
- **AND** all documented Go feature and persistence contracts SHALL remain satisfied

#### Scenario: Reproduce the development environment
- **WHEN** a developer follows the documented mise workflow
- **THEN** the chosen nightly, stable check, nextest and optional continuous feedback SHALL use explicit versions
- **AND** unsupported interpreted native operations SHALL remain covered by native tests
- **AND** the implementation SHALL not introduce Nix/devenv or unrelated runtime dependencies

### Requirement: Deterministic Rust save simulation
The Rust candidate SHALL execute the same save protocol in native operation and in a seeded simulation with controlled time and I/O outcomes. A fixed seed and simulator version SHALL produce an identical ordered trace. Reports SHALL identify tested source, fault coverage, safety and recovery results, and modeled limitations. Native persistence checks SHALL remain required.

#### Scenario: Fault and recovery replay
- **WHEN** simulated writers encounter delayed I/O, contention, failed writes, unavailable history or process/power failures
- **THEN** the simulator SHALL check that uncommitted failures preserve target contents, committed outcomes are reported accurately, stale saves are rejected and successful durable saves survive modeled power loss
- **AND** writers SHALL make progress after faults stop and revisions are reloaded
- **AND** injected implementation defects SHALL be detected by independent checks

### Requirement: Rust full-document Markdown editing
The Rust candidate SHALL provide full-document Unicode source editing as an additional tool for quick document changes using the existing save/history engine. Existing Go-compatible commands SHALL remain unchanged.

#### Scenario: Edit the complete source
- **WHEN** the user opens the document editor
- **THEN** the source draft SHALL remain isolated until explicit save, with the full raw Markdown available in a full-width editor on wide and narrow terminals
- **AND** Markdown mode SHALL show headings, tasks, frontmatter, prose and other source content without a rendered preview pane
- **AND** the normal checklist SHALL retain its existing headings-and-tasks presentation
- **AND** saving SHALL preserve the exact draft bytes, update derived tasks and metadata, and participate in undo/history

#### Scenario: External change during source editing
- **WHEN** the file changes externally before a draft is saved
- **THEN** the save SHALL reject the stale revision without overwriting the file or replacing the editor's accepted document
- **AND** the draft SHALL remain available until the user explicitly discards it

#### Scenario: Edit source with preview
- **WHEN** the user invokes the former preview entry point `:markdown`
- **THEN** it SHALL open the complete source editor directly, without a preview
- **AND** `:edit-markdown` SHALL open the same source editor
