## MODIFIED Requirements

### Requirement: Locally reproducible portable CI

The project SHALL provide a version-pinned Dagger pipeline written in Go that runs the same portable formatting, vet, lint, race-test, coverage, workflow-lint, and build logic locally and in hosted CI.

- The Dagger module SHALL be isolated from the application Go module.
- The Dagger engine, Go toolchain, linter, and workflow linter versions SHALL be pinned.
- The pipeline SHALL not depend on Dagger Cloud for correctness.
- A documented `mask ci` command SHALL execute the complete portable pipeline.

#### Scenario: Developer runs CI locally

- **WHEN** a developer with Mask, Dagger, and a compatible container runtime runs `mask ci`
- **THEN** Dagger SHALL execute all portable checks used by GitHub CI
- **AND** the command SHALL return a non-zero status if any check fails

#### Scenario: GitHub runs portable CI

- **WHEN** GitHub CI runs for a push or pull request
- **THEN** it SHALL invoke the same pinned Dagger pipeline used locally
- **AND** workflow YAML SHALL not duplicate the underlying Go commands
