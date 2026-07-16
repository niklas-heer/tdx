## MODIFIED Requirements

### Requirement: Version restore with confirmation

The system SHALL allow the user to restore the current file to any historic version after confirming the action, without silently overwriting a file revision that changed while the browser was open.

- Pressing `Enter` when `VersionsMode` is `true` and `VersionsConfirmMode` is `false` SHALL set `VersionsConfirmMode = true` and display an inline confirmation prompt: `"Restore version #NNN (YYYY-MM-DD HH:MM)? [y/N]"`.
- In confirmation mode, pressing `y` or `Y` SHALL:
  1. Call `tui.Config.ReadVersionFunc(m.FilePath, selectedVersion.ID)` to get the historic content.
  2. Conditionally write it byte-for-byte through the safe-save boundary using the active file revision.
  3. Reload `m.FileModel` from the committed file and apply restored frontmatter settings.
  4. Close confirmation and version browser modes after success.
- Pressing `n`, `N`, or `Esc` SHALL return to the version list without writing.
- Read, conflict, busy, or save failures SHALL set `m.Err`, close confirmation, and leave the disk file and active in-memory model unchanged.

#### Scenario: Restore replaces unchanged file content

- **WHEN** the user selects a version and confirms restore
- **AND** the active file still matches the revision loaded by tdx
- **THEN** the file on disk SHALL contain the historic version's exact content
- **AND** `m.FileModel` and active TUI settings SHALL reflect the restored content
- **AND** version browser and confirmation modes SHALL close

#### Scenario: Restore refuses a stale revision

- **WHEN** the active file changes after tdx loaded it
- **AND** the user confirms a version restore
- **THEN** restore SHALL report a conflict
- **AND** SHALL not overwrite the changed disk content
- **AND** SHALL retain the pre-restore in-memory model

#### Scenario: Cancelling confirmation returns to list

- **WHEN** the user presses `Enter` to start confirmation
- **AND** then presses `Esc` or `n`
- **THEN** `VersionsConfirmMode` SHALL be `false`
- **AND** `VersionsMode` SHALL remain `true`

