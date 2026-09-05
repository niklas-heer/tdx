## RENAMED Requirements
- FROM: `### Requirement: Current major release validation`
- TO: `### Requirement: Configured release validation`

## MODIFIED Requirements
### Requirement: Configured release validation
The pipeline SHALL validate the version configured in tdx.toml using synchronized Go and tooling versions, including all supported release targets. The project SHALL remain on the 0.x release line until an explicit decision to stabilize 1.0.
#### Scenario: Build a release candidate
- **WHEN** release artifacts are built
- **THEN** all target binaries SHALL embed the version from tdx.toml
