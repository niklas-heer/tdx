## ADDED Requirements
### Requirement: Reviewed release documentation
The release-note generator SHALL prefer the checked-in `docs/releases/<version>.md` for a matching semantic version tag. This path SHALL work without external text-generation credentials and SHALL reject empty reviewed notes. Existing generation remains available when reviewed notes are absent.

#### Scenario: Publish a reviewed release
- **WHEN** release notes exist for the current version tag
- **THEN** the published notes are exactly the reviewed content without an external API request

#### Scenario: Invalid or missing reviewed notes
- **WHEN** a tag is not a supported semantic version
- **THEN** it cannot select a file outside the release-notes directory
- **AND** missing notes use the existing generation path while empty notes fail clearly
