## ADDED Requirements
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
