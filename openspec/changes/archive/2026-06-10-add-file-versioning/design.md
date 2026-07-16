## Context

tdx saves markdown todo files on every user action (toggle, add, edit, move). The save path runs through
`markdown.WriteFileUnchecked`, which does an atomic write (temp file + rename). Version captures must be fast enough not
to block the TUI event loop — a SQLite append in WAL mode typically completes in
< 1 ms on SSD.

The versioning store must also be safe to use from Go test goroutines (each test gets an isolated db path via a dir
override helper).

## Goals / Non-Goals

- Goals:
    - Snapshot every disk write automatically.
    - Detect externally-made edits on file open and create a version for them.
    - Keep writes non-blocking enough for interactive use.
    - Use a pure-Go SQLite driver so cross-compilation stays simple.
- Non-Goals:
    - A UI to browse or restore versions (future work).
    - Pruning or retention policies (all versions kept indefinitely for now).
    - Remote/cloud sync of the version database.

## Decisions

- **Driver**: `modernc.org/sqlite` — pure Go, no CGO, matches cross-compilation requirements stated in the feature spec.
- **WAL mode**: `PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL;` applied once on DB open. Sequential appends to the
  WAL file require far fewer IOPS than modifying the main db file each transaction.
- **Deduplication via content hash**: SHA-256 of file content. `version_hash` has a `UNIQUE` constraint so re-saving
  identical content is a no-op (`INSERT OR IGNORE`). This avoids duplicate rows when tdx auto-saves without any change.
- **Integration pattern — package-level hooks**: Rather than having `internal/markdown` import
  `internal/versioning` (which would force every consumer to carry SQLite), two optional function vars are added to the
  markdown package:
  ```go
  var WriteHook func(filePath, content string)
  var ReadHook  func(filePath, content string)
  ```
  Both default to `nil` (no-op). `cmd/tdx/main.go` sets them at startup. Tests that do not need versioning are
  unaffected.
- **DB path**: `filepath.Join(GetConfigDir(), filepath.Base(filePath) + ".sqlite")`. Using only the basename keeps the
  db path short and consistent regardless of where the markdown file is located. Two different files with the same name
  but different directories would share a db — acceptable for the initial scope; can be addressed by hashing the full
  path in a follow-up.
- **Store lifecycle**: A single `*Store` is opened per markdown file path. `cmd/tdx/main.go`
  maintains a package-level store map, keyed by file path, and closes all stores on exit.

## Alternatives Considered

| Option                       | Rationale for rejection                                                                  |
|------------------------------|------------------------------------------------------------------------------------------|
| Git-based versioning         | Requires git binary; adds process-fork overhead per save; not suitable for < 1 ms writes |
| Plain files / shadow copies  | No deduplication; O(n) disk growth even without changes                                  |
| `mattn/go-sqlite3` (CGO)     | Breaks cross-compilation; contradicts project goal of single-binary portability          |
| Versioning in TUI layer only | Would miss saves triggered by non-TUI code paths (CLI commands)                          |

## Risks / Trade-offs

- **DB file proliferation**: Each markdown file gets a `.sqlite` sibling in the config dir. Users with many files will
  accumulate databases. Mitigation: documented; pruning can be added later.
- **Concurrent access**: Current implementation is single-process; WAL mode handles concurrent readers safely. No
  multi-process concern for now.
- **Test isolation**: `SetVersionStoreDirForTesting` must be called before any hook fires in tests; documented in the
  test helpers.

## Migration Plan

No migration needed. The version database is created on first use. Users who do not want versioning can delete the
`.sqlite` files; they will be recreated on next save.

## Open Questions

- None blocking implementation. Retention / pruning policies can be specified in a follow-up change.
