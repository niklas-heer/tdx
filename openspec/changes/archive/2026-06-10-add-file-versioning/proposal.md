## Why

Users who edit markdown todo files with tdx risk losing work if a save is interrupted or if they want to review or
recover a prior state of the file. There is currently no history mechanism. This change adds continuous, automatic
versioning: every time tdx writes the markdown file to disk, a snapshot is appended to a lightweight SQLite database
stored in the tdx config directory.

## What Changes

- New package `internal/versioning` implements a SQLite-backed version store using `modernc.org/sqlite`
  (pure-Go, no CGO, cross-compilation-safe).
- The database file is named `<basename>.sqlite` and lives in `GetConfigDir()`, e.g.
  `~/.config/tdx/test.md.sqlite` for a file called `test.md`.
- A version is saved on every write (`markdown.WriteFileUnchecked`).
- On `markdown.ReadFile`, if the loaded content hash does not match any existing version, a new version is inserted
  automatically (captures externally-made edits).
- SQLite is configured in WAL mode (`PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL;`) to minimise IOPS given
  high-frequency saves.
- A package-level write-hook (`markdown.WriteHook`) and read-hook (`markdown.ReadHook`) wire the versioning store into
  the markdown package without introducing a circular import.
- The hooks are registered at startup in `cmd/tdx/main.go`.
- A dedicated `SetVersionStoreDirForTesting` helper mirrors `SetConfigDirForTesting` so unit tests remain isolated.

- Content BLOBs are compressed with **zstd at speed level 1** (`github.com/klauspost/compress/zstd`,
  pure Go) before storage. The `version_hash` is computed from the uncompressed content so
  deduplication is unaffected. A `Store.ReadVersion` method decompresses on demand.
- A new `[versioning]` config section exposes `max_versions` (default `100`; `0` = unlimited).
  `Store.Prune(filePath, maxVersions)` enforces the limit per file; pruning runs after each save
  and on shutdown.
- **REVISED**: The initial per-file database design (`<basename>.sqlite` per markdown file) is
  replaced by a single **`versions.sqlite`** shared across all files. A two-table schema
  (`files` + `file_versions` with integer FK) avoids path duplication in every version row and
  reduces index size. The WAL overhead (main + -shm + -wal) is paid once instead of once per file.

## Impact

- Affected specs: new `file-versioning` capability (ADDED + MODIFIED requirements)
- Affected code:
    - `internal/versioning/` — new package (store.go + store_test.go); refactored to single shared DB
    - `internal/markdown/parser.go` — add `WriteHook` and `ReadHook` vars; call them in
      `WriteFileUnchecked` and `ReadFile`
    - `cmd/tdx/main.go` — single `*versioning.Store`; register hooks at startup; pass `MaxVersions`
      to `versioning.Open`; prune after each save and on shutdown
    - `cmd/tdx/userconfig.go` — add `VersioningConfig` struct and `Versioning` field to `UserConfig`
    - `go.mod` / `go.sum` — add `modernc.org/sqlite`, `github.com/klauspost/compress`
