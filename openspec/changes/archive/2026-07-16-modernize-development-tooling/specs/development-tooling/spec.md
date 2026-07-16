## ADDED Requirements

### Requirement: Markdown task interface

The project SHALL expose maintained developer tasks through a root `maskfile.md` that is both human-readable documentation and executable by Mask.

- The task interface SHALL cover building, installing, development execution, todo commands, code quality, Dagger CI, release artifacts, maintenance, and releases.
- Task chaining SHALL propagate failures and remain valid when Mask is called with an explicit maskfile path.
- Maintained developer documentation SHALL use Mask commands rather than Just commands.

#### Scenario: Developer discovers and runs a task

- **WHEN** a developer opens `maskfile.md` or runs `mask --help`
- **THEN** the available task names and descriptions SHALL be visible
- **AND** running a documented command SHALL execute the corresponding project operation

### Requirement: Synchronized Go toolchain

The application module, Dagger module, and pinned Dagger Go container SHALL use the same supported Go release.

#### Scenario: Go version is updated

- **WHEN** the project updates its supported Go release
- **THEN** both module directives and the portable build container SHALL be updated together
- **AND** the complete portable and native-focused test suites SHALL pass with the updated toolchain

### Requirement: Command-layer regression coverage

Automated tests SHALL exercise the successful file-mutating behavior of the non-interactive todo command layer.

#### Scenario: Todo command behavior regresses

- **WHEN** list, add, toggle, edit, delete, or valid command dispatch stops producing the expected file or output state
- **THEN** the command package test suite SHALL fail
