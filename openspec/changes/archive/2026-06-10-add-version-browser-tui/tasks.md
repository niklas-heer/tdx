## 1. Dependency

- [x] 1.1 Add `github.com/sergi/go-diff` via `go get github.com/sergi/go-diff` and `go mod tidy`

## 2. Versioning store — listing API

- [x] 2.1 Add to `internal/versioning/store.go`:
  - `VersionInfo struct { ID int64; CreatedAt time.Time }` type
  - `Store.ListVersions(filePath string) ([]VersionInfo, error)` — SELECT id, created_at from
    file_versions WHERE file_id = ? ORDER BY id DESC; return empty slice (not error) if no rows
- [x] 2.2 Add to `internal/versioning/store_test.go`:
  - `TestListVersions_ReturnsMostRecentFirst` — save 3 versions, verify order and count
  - `TestListVersions_EmptyForUnknownFile` — call on path with no saves, expect empty + nil

## 3. TUI plumbing — types and ConfigType

- [x] 3.1 Add to `internal/tui/model.go`:
  - `VersionInfo struct { ID int64; CreatedAt time.Time }` type (tui-local adapter)
  - `ListVersionsFunc func(filePath string) ([]VersionInfo, error)` field on `ConfigType`
  - `ReadVersionFunc  func(filePath string, id int64) (string, error)` field on `ConfigType`
  - `VersionsMode`, `VersionsConfirmMode bool` fields on `Model`
  - `VersionsCursor`, `VersionsDiffScroll int` fields on `Model`
  - `VersionsList []VersionInfo` field on `Model`

## 4. versions command

- [x] 4.1 Add to `internal/tui/commands.go` in `InitCommands()`:
  - Guard: only append `versions` command when `Config.ListVersionsFunc != nil`
  - Handler: call `ListVersionsFunc(m.FilePath)`, populate `m.VersionsList`,
    reset cursor/scroll/confirm, set `m.VersionsMode = true`

## 5. Key handler

- [x] 5.1 Add `handleVersionsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd)` to `internal/tui/update.go`:
  - `VersionsConfirmMode == true` branch: `y`/`Y` → restore; `n`/`N`/`esc` → cancel confirm
  - Normal branch: `↑`/`k` → cursor up + reset scroll; `↓`/`j` → cursor down + reset scroll;
    `pgup`/`ctrl+u` → scroll diff up; `pgdown`/`ctrl+d` → scroll diff down;
    `enter` → set VersionsConfirmMode = true; `esc` → close browser
  - Restore path: `ReadVersionFunc` → `ParseMarkdown` → set FilePath/ModTime →
    `WriteFileUnchecked` → `ReadFile` → update `m.FileModel` → close modal
- [x] 5.2 Dispatch `handleVersionsKey` in `handleKey()` when `m.VersionsMode` is true
    (add before existing mode checks so it takes priority)

## 6. Renderer

- [x] 6.1 Add `renderDiff(diffs []diffmatchpatch.Diff, styles *StyleFuncsType) string` function
    in `internal/tui/view.go` (or a new `internal/tui/versions.go` file):
  - Iterate diffs: Insert → Green, Delete → Magenta, Equal → Dim
- [x] 6.2 Add `renderVersionsBrowser() string` method on `Model`:
  - Compute browser height: `floor(m.TermHeight * 0.9)`
  - Left pane: render `VersionsList` rows; highlight `VersionsCursor` row; format date + ID
  - Right pane: compute diff via `dmp.DiffMain` + `DiffCleanupSemantic`; call `renderDiff`;
    split into lines; apply `VersionsDiffScroll` offset; clip to pane height
  - Assemble with `lipgloss.JoinHorizontal`; wrap in outer border box with title + footer
  - If `VersionsConfirmMode`: render confirmation prompt row above footer
- [x] 6.3 Dispatch `renderVersionsBrowser()` at the top of `View()` when `m.VersionsMode == true`

## 7. Wiring in main

- [x] 7.1 In `cmd/tdx/main.go`, after `openVersionStore(...)`:
  - Set `tui.Config.ListVersionsFunc` wrapping `versionStore.ListVersions` with
    `tui.VersionInfo` conversion
  - Set `tui.Config.ReadVersionFunc` directly wrapping `versionStore.ReadVersion`

## 8. Tests

- [x] 8.1 Add unit tests in `internal/tui/` for `handleVersionsKey`:
  - Navigating down increments `VersionsCursor` and resets `VersionsDiffScroll`
  - Esc closes browser (`VersionsMode = false`)
  - Enter sets `VersionsConfirmMode = true`
  - `n` in confirm mode sets `VersionsConfirmMode = false`, keeps `VersionsMode = true`
  - Use a stub `ListVersionsFunc` returning fixed `[]VersionInfo`

## 9. Validation

- [x] 9.1 `go build ./...` — no compilation errors
- [x] 9.2 `go vet ./...` — no warnings
- [x] 9.3 `go test ./...` — all tests green
