## MODIFIED Requirements

### Requirement: Versioning function injection into ConfigType

The `tui.ConfigType` SHALL expose two optional function fields that decouple the TUI from the
`versioning` package, allowing the version browser to be used without introducing a direct import
dependency.

- A `tui.VersionInfo` struct SHALL be defined in `internal/tui/model.go`:
  ```go
  type VersionInfo struct {
      ID        int64
      CreatedAt time.Time
  }
  ```
- `ConfigType` SHALL gain two fields:
  ```go
  ListVersionsFunc func(filePath string) ([]VersionInfo, error)
  ReadVersionFunc  func(filePath string, id int64) (string, error)
  ```
- Both fields SHALL default to `nil`. When `nil`, no versioning functionality is active.
- `cmd/tdx/main.go` SHALL assign these fields after opening the local version store, wrapping
  `versions.ListVersions` and `versions.ReadVersion` with the `tui.VersionInfo` adapter.

#### Scenario: Functions are nil when versioning is unavailable

- **WHEN** `m.Config().ListVersionsFunc` is `nil`
- **THEN** the `versions` command SHALL NOT appear in the command palette
- **AND** `VersionsMode` SHALL NOT be settable via normal key handling

#### Scenario: Functions are wired at startup when store is open

- **WHEN** the local version store is successfully opened in `cmd/tdx/main.go`
- **THEN** `m.Config().ListVersionsFunc` and `m.Config().ReadVersionFunc` SHALL be non-nil
- **AND** the `versions` command SHALL appear in the command palette

### Requirement: versions command in the colon-command palette

The `versions` command SHALL be registered in the TUI's colon-command palette and, when selected,
SHALL open the version browser modal for the currently open file.

- The command name SHALL be `versions` and the description SHALL be
  `"Browse and restore file version history"`.
- The command SHALL only be included in `InitCommands()` when
  `m.Config().ListVersionsFunc != nil`.
- The command handler SHALL:
  1. Call `m.Config().ListVersionsFunc(m.FilePath)` to load the version list.
  2. Store the result in `m.VersionsList`.
  3. Set `m.VersionsCursor = 0`, `m.VersionsDiffScroll = 0`, `m.VersionsConfirmMode = false`.
  4. Set `m.VersionsMode = true`.

#### Scenario: Command opens version browser

- **WHEN** the user types `:versions` and presses `Enter` in the command palette
- **THEN** `VersionsMode` SHALL be `true`
- **AND** `VersionsList` SHALL contain the versions for the current file

#### Scenario: Command absent when no store configured

- **WHEN** `m.Config().ListVersionsFunc` is `nil`
- **AND** the user opens the command palette
- **THEN** `versions` SHALL NOT appear in the command list

### Requirement: Version restore with confirmation

The system SHALL allow the user to restore the current file to any historic version after confirming the action, without silently overwriting a file revision that changed while the browser was open.

- Pressing `Enter` when `VersionsMode` is `true` and `VersionsConfirmMode` is `false` SHALL set `VersionsConfirmMode = true` and display an inline confirmation prompt: `"Restore version #NNN (YYYY-MM-DD HH:MM)? [y/N]"`.
- In confirmation mode, pressing `y` or `Y` SHALL:
  1. Call `m.Config().ReadVersionFunc(m.FilePath, selectedVersion.ID)` to get the historic content.
  2. Conditionally write it byte-for-byte through the safe-save boundary using the active file revision.
  3. Reload `m.FileModel` from the committed file and apply restored frontmatter settings.
  4. Close confirmation and version browser modes after success.
- Pressing `n`, `N`, or `Esc` SHALL return to the version list without writing.
- Read, conflict, busy, or other pre-commit save failures SHALL set `m.Err`, close confirmation, and leave the disk file and active in-memory model unchanged.
- Post-commit failures SHALL set `m.Err`, close both version modes, and reload the already-committed disk content into the active model.

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

## ADDED Requirements

### Requirement: Canonical history lookup
The version browser SHALL resolve file aliases to the same canonical identity used by persistence callbacks before listing or reading a saved revision.

#### Scenario: Open through a symlink
- **WHEN** a file is opened through a symlink or a symlinked parent directory
- **THEN** its captured versions SHALL be visible and restorable through that path and the canonical path
