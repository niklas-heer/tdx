## ADDED Requirements

### Requirement: Exact-commit release validation
Tagged release publication SHALL depend on successful portable and native validation of the exact tagged commit, even when the tag was created outside the release helper.

#### Scenario: A tag points at a failing commit
- **WHEN** a tag points at a failing commit
- **THEN** publication is blocked

### Requirement: Verified release installation
Releases SHALL include a SHA-256 checksum manifest and build provenance. The installer SHALL verify the selected artifact before replacing the destination, and checksum/download failures SHALL leave existing installations unchanged.

#### Scenario: A downloaded artifact has an incorrect checksum
- **WHEN** a downloaded artifact has an incorrect checksum
- **THEN** installation fails and the previous executable remains intact

