# version-browser-tui Specification

## Purpose
Provide an interactive TUI for browsing, comparing, and restoring captured file versions.
## Requirements
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
- `cmd/tdx/main.go` SHALL assign these fields after opening `versionStore`, wrapping
  `versionStore.ListVersions` and `versionStore.ReadVersion` with the `tui.VersionInfo` adapter.

#### Scenario: Functions are nil when versioning is unavailable

- **WHEN** `tui.Config.ListVersionsFunc` is `nil`
- **THEN** the `versions` command SHALL NOT appear in the command palette
- **AND** `VersionsMode` SHALL NOT be settable via normal key handling

#### Scenario: Functions are wired at startup when store is open

- **WHEN** `versionStore` is successfully opened in `cmd/tdx/main.go`
- **THEN** `tui.Config.ListVersionsFunc` and `tui.Config.ReadVersionFunc` SHALL be non-nil
- **AND** the `versions` command SHALL appear in the command palette

---

### Requirement: versions command in the colon-command palette

The `versions` command SHALL be registered in the TUI's colon-command palette and, when selected,
SHALL open the version browser modal for the currently open file.

- The command name SHALL be `versions` and the description SHALL be
  `"Browse and restore file version history"`.
- The command SHALL only be included in `InitCommands()` when
  `tui.Config.ListVersionsFunc != nil`.
- The command handler SHALL:
  1. Call `tui.Config.ListVersionsFunc(m.FilePath)` to load the version list.
  2. Store the result in `m.VersionsList`.
  3. Set `m.VersionsCursor = 0`, `m.VersionsDiffScroll = 0`, `m.VersionsConfirmMode = false`.
  4. Set `m.VersionsMode = true`.

#### Scenario: Command opens version browser

- **WHEN** the user types `:versions` and presses `Enter` in the command palette
- **THEN** `VersionsMode` SHALL be `true`
- **AND** `VersionsList` SHALL contain the versions for the current file

#### Scenario: Command absent when no store configured

- **WHEN** `tui.Config.ListVersionsFunc` is `nil`
- **AND** the user opens the command palette
- **THEN** `versions` SHALL NOT appear in the command list

---

### Requirement: Full-screen version browser modal

The version browser SHALL render as a full-screen overlay replacing the normal TUI view while
`VersionsMode` is active, occupying ≈90% of the terminal height.

- `View()` SHALL return `m.renderVersionsBrowser()` directly when `m.VersionsMode` is `true`,
  bypassing the normal `overlay.Composite` path.
- The browser SHALL display a title bar: `FILE VERSION HISTORY - [<basename of FilePath>]`.
- The outer border SHALL use the same box-drawing characters as existing TUI overlays
  (`┌─┬─┐│ │└─┴─┘`) with the accent border colour (`#7aa2f7`).
- A footer bar SHALL display the key bindings:
  `[↑/↓] Navigate  •  [PgUp/PgDn] Scroll Diff  •  [Enter] Restore  •  [Esc] Close`.

#### Scenario: Browser fills terminal

- **WHEN** `VersionsMode` is `true`
- **AND** `View()` is called
- **THEN** the output SHALL span ≈90% of `m.TermHeight` rows
- **AND** the normal todo list SHALL NOT be visible

#### Scenario: Browser closes on Esc

- **WHEN** `VersionsMode` is `true`
- **AND** `VersionsConfirmMode` is `false`
- **AND** the user presses `Esc`
- **THEN** `VersionsMode` SHALL be set to `false`
- **AND** the normal TUI view SHALL be restored

---

### Requirement: Two-column layout — version list and diff pane

The version browser SHALL be divided into two vertical columns.

- **Left pane** (≈25% of browser width): version list.
  - Each row SHALL show the version date formatted as `YYYY-MM-DD HH:MM` followed by ` - #NNN`
    where `NNN` is the `VersionInfo.ID` zero-padded to three digits.
  - The currently selected row SHALL be highlighted (reverse/accent style).
  - Navigation: `↑`/`k` and `↓`/`j` move the cursor; `VersionsCursor` is clamped to
    `[0, len(VersionsList)-1]`.
  - Changing `VersionsCursor` SHALL reset `VersionsDiffScroll` to `0`.
- **Right pane** (≈75% of browser width): diff content.
  - Content is computed from the currently selected version using `renderDiff()`.
  - The pane is clipped to the available height; `VersionsDiffScroll` offsets the visible window.
  - `PgUp` / `PgDn` adjust `VersionsDiffScroll` by half the pane height (clamped to valid range).
- The two panes SHALL be assembled with `lipgloss.JoinHorizontal`.

#### Scenario: Selecting a version updates the diff pane

- **WHEN** the user presses `↓` in the version browser
- **THEN** `VersionsCursor` SHALL increment by 1 (if not at the last entry)
- **AND** the right pane SHALL display the diff for the newly selected version
- **AND** `VersionsDiffScroll` SHALL be reset to 0

#### Scenario: Diff pane scrolls independently

- **WHEN** the user presses `PgDn` in the version browser
- **THEN** `VersionsDiffScroll` SHALL increase by half the pane height
- **AND** `VersionsCursor` SHALL remain unchanged

---

### Requirement: ANSI diff rendering

The diff pane SHALL display a character-level diff between the selected historic version content and
the current on-disk file content, rendered with ANSI colours suitable for a terminal.

- The diff SHALL be computed using `github.com/sergi/go-diff/diffmatchpatch`:
  ```go
  dmp := diffmatchpatch.New()
  diffs := dmp.DiffMain(currentContent, versionContent, false)
  dmp.DiffCleanupSemantic(diffs)
  ```
- A `renderDiff(diffs []diffmatchpatch.Diff, styles *StyleFuncsType) string` function SHALL
  iterate the diff operations and apply lipgloss styles:
  - `diffmatchpatch.Insert` → `StyleFuncs.Green`
  - `diffmatchpatch.Delete` → `StyleFuncs.Magenta`
  - `diffmatchpatch.Equal` → `StyleFuncs.Dim`
- The current file content for the diff SHALL be `markdown.SerializeMarkdown(&m.FileModel)` — the
  in-memory (possibly unsaved) state.
- If `VersionsList` is empty, the right pane SHALL display a `StyleFuncs.Dim` message:
  `"No versions available"`.

#### Scenario: Added lines appear green

- **WHEN** the selected historic version has a line absent from the current file
- **THEN** that text SHALL be rendered with the green style in the diff pane

#### Scenario: Removed lines appear in magenta

- **WHEN** the current file has a line absent from the selected historic version
- **THEN** that text SHALL be rendered with the magenta style in the diff pane

#### Scenario: Empty versions list shows placeholder

- **WHEN** `VersionsList` is empty
- **THEN** the right pane SHALL display `"No versions available"` in dim style

---

### Requirement: Version restore with confirmation

The system SHALL allow the user to restore the current file to any historic version after
confirming the action.

- Pressing `Enter` when `VersionsMode` is `true` and `VersionsConfirmMode` is `false` SHALL set
  `VersionsConfirmMode = true` and display an inline confirmation prompt:
  `"Restore version #NNN (YYYY-MM-DD HH:MM)? [y/N]"`.
- In confirmation mode:
  - Pressing `y` (or `Y`) SHALL:
    1. Call `tui.Config.ReadVersionFunc(m.FilePath, selectedVersion.ID)` to get the historic content.
    2. Write it byte-for-byte via `markdown.WriteContentUnchecked(m.FilePath, content)` so
       frontmatter and formatting are preserved exactly.
    3. Reload `m.FileModel` from the written file via `markdown.ReadFile(m.FilePath)`.
    4. Apply restored frontmatter settings to the active TUI model.
    5. Set `m.VersionsMode = false`, `m.VersionsConfirmMode = false`.
  - Pressing `n`, `N`, or `Esc` SHALL set `VersionsConfirmMode = false` (return to list).
- If `ReadVersionFunc` returns an error, `m.Err` SHALL be set and the modal SHALL close.

#### Scenario: Restore replaces file content

- **WHEN** the user selects a version and presses `Enter`, then `y`
- **THEN** the file on disk SHALL contain the historic version's content
- **AND** `m.FileModel` SHALL reflect the restored content
- **AND** active TUI settings SHALL reflect the restored frontmatter
- **AND** `VersionsMode` SHALL be `false`

#### Scenario: Cancelling confirmation returns to list

- **WHEN** the user presses `Enter` to start confirmation
- **AND** then presses `Esc` or `n`
- **THEN** `VersionsConfirmMode` SHALL be `false`
- **AND** `VersionsMode` SHALL remain `true`
