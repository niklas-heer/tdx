## ADDED Requirements
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
