## ADDED Requirements
### Requirement: Write-protected targets

The save pipeline SHALL NOT replace an existing file that the current user cannot open for writing, including files whose permission bits, read-only attribute or access-control rules forbid writing. The check SHALL happen under the save lock, after the conflict check and immediately before replacement, so a file that also changed on disk reports the conflict first; the error SHALL say the file is write-protected and name it. Force-save SHALL NOT bypass this check.

#### Scenario: Save a write-protected file

- **WHEN** an existing markdown file has mode `0444` in a writable directory
- **AND** a CLI command or the TUI tries to save a change to it
- **THEN** the save SHALL fail with a write-protection error
- **AND** the file content and permission bits SHALL be unchanged
- **AND** no temporary replacement SHALL remain beside the file

#### Scenario: The file changes after a refused autosave

- **WHEN** an autosave was refused, leaving an edit only in memory
- **AND** the file then changes on disk
- **THEN** the TUI SHALL keep the edit instead of reloading the file
- **AND** the next save SHALL report the external change as a conflict, even if the file is still write-protected
