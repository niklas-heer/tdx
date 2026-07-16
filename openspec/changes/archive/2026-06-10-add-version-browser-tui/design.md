## Context

The version browser is the first full-screen modal in the TUI. All existing overlays (command
palette, tag filter, theme picker, recent files) are small popups positioned at the bottom-left of
the background via `overlay.Composite`. A 90% full-screen panel requires a different rendering
strategy.

## Goals / Non-Goals

- Goals:
  - Browse the version history of the currently open file without leaving the TUI.
  - See a coloured diff of any historic version vs the current file content.
  - Restore a historic version with a confirmation step.
  - Keep the `tui` package import-free of the `versioning` package.
- Non-Goals:
  - Comparing two historic versions against each other (future work).
  - Exporting or copying a diff to clipboard (future work).
  - Paging the version list itself (≤100 versions is the configured cap; fits in any terminal).

## Decisions

### Full-screen rendering strategy

**Decision**: When `VersionsMode` is active, `View()` returns the version browser output directly
instead of compositing it over the main background. The existing `overlay.Composite` path is only
used for small popups — trying to composite a 90% panel over a scrolling background adds complexity
with no benefit.

**Alternative considered**: `overlay.Composite(browser, background, overlay.Center, overlay.Center, …)`
— rejected because it requires the background to be pre-rendered to the full terminal height, adds
invisible rendering work, and the overlay library's centering math is tricky to tune for large panels.

### Diff library and ANSI rendering

**Decision**: Use `github.com/sergi/go-diff/diffmatchpatch`.
- `dmp.DiffMain(currentContent, versionContent, false)` computes character-level diffs.
- `dmp.DiffCleanupSemantic(diffs)` improves human readability.
- A custom `renderDiff(diffs []diffmatchpatch.Diff) string` function iterates the operations and
  applies lipgloss colours: `StyleFuncs.Green` for `Insert`, `StyleFuncs.Magenta` for `Delete`,
  `StyleFuncs.Dim` for `Equal` (large equal runs are truncated to keep the pane readable).

**Why not `DiffPrettyText`**: returns HTML `<ins>`/`<del>` tags — not usable in a terminal.

**Why character-level vs line-level**: character-level shows exactly what changed within a line,
which is useful for short todo items. `DiffCleanupSemantic` consolidates spurious character ops into
word-level boundaries, giving a natural reading experience.

### Decoupling `tui` from `versioning`

**Decision**: inject two `func` fields into `ConfigType`:
```go
ListVersionsFunc func(filePath string) ([]VersionInfo, error)
ReadVersionFunc  func(filePath string, id int64) (string, error)
```
`cmd/tdx/main.go` wraps the open `*versioning.Store` calls and assigns these fields.
When `ListVersionsFunc == nil` (e.g. in tests that do not wire versioning), the `versions` command
is omitted from `InitCommands()` and `VersionsMode` can never be set.

**Why not a `VersionStore` interface on `ConfigType`**: a single-method interface per operation
is simpler to stub in tests; we don't need the full store surface in the TUI.

### `VersionInfo` adapter type

`tui.VersionInfo{ID int64, CreatedAt time.Time}` lives in `model.go`. `cmd/tdx/main.go` converts
`versioning.VersionInfo` → `tui.VersionInfo`. Keeps the `tui` package's import graph clean.

### Navigation key bindings

| Key | Action |
|---|---|
| `↑` / `k` | Move up in version list |
| `↓` / `j` | Move down in version list |
| `PgUp` | Scroll diff pane up |
| `PgDn` | Scroll diff pane down |
| `Enter` | Open confirmation prompt for selected version |
| `y` / `Enter` (in confirm) | Confirm restore |
| `n` / `Esc` (in confirm) | Cancel confirmation, return to list |
| `Esc` (outside confirm) | Close version browser, return to main TUI |

### Layout proportions

```
Terminal width W, height H
Browser outer box: W × floor(H * 0.9)
Left pane width:   floor(W * 0.25) - 3   (border + padding)
Right pane width:  remainder
```

lipgloss `JoinHorizontal` assembles the two panes inside the outer border box.

## Risks / Trade-offs

- **Diff on large files**: character-level diff on a large file can be slow. Markdown todo files are
  typically ≤ 100 lines so this is not a concern in practice.
- **Scroll state resets**: navigating to a different version resets `VersionsDiffScroll` to 0. This
  is intentional — each diff is independent.
- **No test for actual rendering**: TUI rendering tests verify output strings via piped input.
  The version browser requires a live store; tests use a stub `ListVersionsFunc`.

## Migration Plan

No migration. The feature is purely additive — the `versions` command only appears when a store is
wired, which only happens when `cmd/tdx/main.go` initialises versioning on startup.

## Open Questions

- None blocking implementation.
