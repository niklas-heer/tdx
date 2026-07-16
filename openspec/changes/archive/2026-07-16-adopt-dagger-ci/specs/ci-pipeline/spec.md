## ADDED Requirements

### Requirement: Locally reproducible portable CI

The project SHALL provide a version-pinned Dagger pipeline written in Go that runs the same portable formatting, vet, lint, race-test, coverage, workflow-lint, and build logic locally and in hosted CI.

- The Dagger module SHALL be isolated from the application Go module.
- The Dagger engine, Go toolchain, linter, and workflow linter versions SHALL be pinned.
- The pipeline SHALL not depend on Dagger Cloud for correctness.
- A documented `just` command SHALL execute the complete portable pipeline.

#### Scenario: Developer runs CI locally

- **WHEN** a developer with Dagger and a compatible container runtime runs the local CI command
- **THEN** Dagger SHALL execute all portable checks used by GitHub CI
- **AND** the command SHALL return a non-zero status if any check fails

#### Scenario: GitHub runs portable CI

- **WHEN** GitHub CI runs for a push or pull request
- **THEN** it SHALL invoke the same pinned Dagger pipeline used locally
- **AND** workflow YAML SHALL not duplicate the underlying Go commands

### Requirement: Native filesystem verification

The CI system SHALL retain native macOS and Windows verification for behavior that Linux containers cannot reproduce accurately.

- Each native job SHALL compile the application for its host platform.
- Each native job SHALL run the markdown and versioning package tests.
- Each native job SHALL execute the built binary and verify that it starts successfully.
- Native jobs SHALL use Node.js 24-compatible first-party GitHub Actions.

#### Scenario: Platform-specific save behavior regresses

- **WHEN** a change breaks file replacement, advisory locking, or executable behavior on macOS or Windows
- **THEN** the corresponding native CI job SHALL fail
- **AND** the portable Dagger job passing SHALL not make the overall workflow pass

### Requirement: Reproducible release artifacts

The Dagger pipeline SHALL produce the complete release artifact set from the version and description declared in `tdx.toml`.

- The artifact set SHALL contain macOS amd64 and arm64, Linux amd64 and arm64, and Windows amd64 binaries.
- Artifact filenames SHALL remain compatible with the install script and Homebrew formula.
- Version and description linker variables SHALL match the tagged source revision.
- The same artifact function SHALL be callable locally and from the release workflow.

#### Scenario: Release tag is pushed

- **WHEN** a `v*` tag triggers the release workflow
- **THEN** GitHub Actions SHALL call the Dagger release-artifact function
- **AND** SHALL publish all five returned binaries to the GitHub release
- **AND** the Homebrew update job SHALL consume those published asset names without modification

#### Scenario: Developer dry-runs release builds

- **WHEN** a developer calls the local release-artifact command
- **THEN** Dagger SHALL export the same five filenames that the tag workflow would publish
- **AND** SHALL not require GitHub or Homebrew write credentials

### Requirement: Thin and current GitHub orchestration

GitHub Actions SHALL contain only event handling, native platform validation, credentialed publication, and calls into the portable Dagger pipeline.

- Maintained first-party actions SHALL use Node.js 24-compatible major versions.
- Dagger CLI and GitHub integration versions SHALL be explicit rather than floating.
- Repository write tokens SHALL be limited to jobs and steps that publish badges, releases, or Homebrew updates.
- Dagger functions SHALL not receive GitHub repository write tokens.

#### Scenario: Portable pipeline implementation changes

- **WHEN** a portable check or release build command changes
- **THEN** the implementation SHALL be updated in the Dagger Go module
- **AND** local and hosted execution SHALL observe the change without duplicating it in workflow YAML

### Requirement: Coverage badge continuity

The Dagger pipeline SHALL calculate repository test coverage and produce the existing Shields-compatible coverage badge payload.

- Pull requests SHALL validate coverage generation without writing to the repository.
- Successful pushes to `main` SHALL commit the badge only when its content changes.
- Badge update commits SHALL continue to avoid recursive CI runs.

#### Scenario: Coverage changes on main

- **WHEN** the portable test pipeline succeeds on `main`
- **AND** the calculated badge differs from the tracked badge
- **THEN** GitHub Actions SHALL commit and push the updated badge
- **AND** the badge commit SHALL not trigger an endless workflow loop
