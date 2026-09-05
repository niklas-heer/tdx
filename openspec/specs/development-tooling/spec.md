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
