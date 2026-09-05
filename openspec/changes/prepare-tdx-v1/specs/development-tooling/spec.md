## MODIFIED Requirements
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
