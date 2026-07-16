## Why

The versioning store now captures a full history of every markdown file but there is no way for the
user to see, browse, or restore a previous version from within tdx. This change adds an interactive
version browser accessible via the existing colon-command palette, letting users inspect diffs and
restore any prior state without leaving the TUI.

## What Changes

- A `versions` command is added to the colon-command palette (`:`). Selecting it opens a full-screen
  version browser modal (≈90% of the terminal area).
- The modal has a two-column layout:
  - **Left pane** (≈25%): scrollable version list — each row shows the version date and a short ID.
  - **Right pane** (≈75%): live diff of the selected version against the current file content,
    rendered with ANSI colours (green = added lines, red = removed lines, default = unchanged).
- Diff computation uses `github.com/sergi/go-diff/diffmatchpatch`. The raw diff operations are
  rendered via a custom lipgloss-based ANSI colouriser (not the HTML `DiffPrettyText` method).
- Pressing `Enter` on a version shows an inline confirmation prompt. Confirming writes the version
  content to disk via `markdown.WriteFileUnchecked`, reloads the `FileModel`, and closes the modal.
- The right pane supports vertical scrolling (`PgUp`/`PgDn`) for long diffs.
- The TUI is decoupled from the `versioning` package through two injected functions on `ConfigType`:
  `ListVersionsFunc` and `ReadVersionFunc`. `cmd/tdx/main.go` wires them to the open `*Store`.
- A `VersionInfo` struct `{ID int64, CreatedAt time.Time}` is added to the `tui` package (adapter
  type; avoids importing `versioning` from `tui`).
- `versioning.Store` gains `ListVersions(filePath string) ([]versioning.VersionInfo, error)` and a
  package-local `versioning.VersionInfo` struct.
- If no versioning store is configured (`ListVersionsFunc == nil`) the command is hidden from the
  palette.

## Impact

- Affected specs: `file-versioning` (ADDED: listing API), new `version-browser-tui` capability
- Affected code:
  - `internal/versioning/store.go` — add `VersionInfo` type and `ListVersions` method
  - `internal/versioning/store_test.go` — add `TestListVersions_*`
  - `internal/tui/model.go` — add `VersionInfo` type; add `ListVersionsFunc`/`ReadVersionFunc`
    to `ConfigType`; add `VersionsMode`, `VersionsCursor`, `VersionsList`, `VersionsDiffScroll`,
    `VersionsConfirmMode` fields to `Model`
  - `internal/tui/commands.go` — add `versions` command (conditional on `ListVersionsFunc != nil`)
  - `internal/tui/update.go` — add `handleVersionsKey()` handler and mode-dispatch in `handleKey()`
  - `internal/tui/view.go` — add `renderVersionsBrowser()` full-screen renderer; dispatch in `View()`
  - `cmd/tdx/main.go` — wire `ListVersionsFunc` and `ReadVersionFunc` into `tui.Config`
  - `go.mod` / `go.sum` — add `github.com/sergi/go-diff`
