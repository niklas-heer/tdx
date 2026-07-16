# file-versioning Specification

## Purpose
Provide automatic, compressed snapshots of markdown files so users can recover earlier content.
## Requirements
### Requirement: SQLite-backed version store

The system SHALL maintain a continuous version history for all markdown files opened or modified by
tdx in a **single shared** SQLite database located in the tdx config directory (`GetConfigDir()`).

- The database SHALL be a single file named `versions.sqlite` stored in `GetConfigDir()`.
- All markdown files share the same database; file identity is tracked via a `files` lookup table.
- The database SHALL use the following two-table schema:

  ```sql
  CREATE TABLE IF NOT EXISTS files (
      id        INTEGER PRIMARY KEY AUTOINCREMENT,
      file_path TEXT NOT NULL UNIQUE
  );

  CREATE TABLE IF NOT EXISTS file_versions (
      id             INTEGER PRIMARY KEY AUTOINCREMENT,
      file_id        INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
      version_hash   TEXT NOT NULL,
      content        BLOB NOT NULL,
      created_at     DATETIME DEFAULT CURRENT_TIMESTAMP,
      commit_message TEXT,
      UNIQUE(file_id, version_hash)
  );

  CREATE INDEX IF NOT EXISTS idx_file_versions_file_id ON file_versions(file_id);
  ```

  Rationale: storing `file_path` as TEXT on every version row wastes space and produces a larger
  B-tree index. The integer FK (`file_id`) is 4 bytes per row, compact, and enables
  `ON DELETE CASCADE` for future "forget this file" operations.

- On first open the store SHALL execute:

  ```sql
  PRAGMA journal_mode=WAL;
  PRAGMA synchronous=NORMAL;
  ```

- The `version_hash` column SHALL hold the hex-encoded SHA-256 of the **uncompressed** file content.
- Inserting a version with a hash that already exists for the same file SHALL be a no-op
  (`INSERT OR IGNORE` on the `UNIQUE(file_id, version_hash)` constraint).
- `Store.SaveVersion(filePath, content string) error` SHALL:
  1. UPSERT `filePath` into `files` and retrieve its integer `file_id`.
  2. INSERT OR IGNORE into `file_versions` using the resolved `file_id`.
- The `Store` SHALL cache the `filePath → file_id` mapping in a `map[string]int64` to avoid
  repeated DB lookups within the same process session.
- `versioning.DBPath() (string, error)` SHALL return `filepath.Join(getStoreDir(), "versions.sqlite")`.
- For commands that access a markdown file, `cmd/tdx/main.go` SHALL maintain a **single**
  `*versioning.Store` (not a per-file map), opened once before file access with
  `defer store.Close()`.

#### Scenario: New version saved on first write

- **WHEN** `WriteFileUnchecked` is called for a file
- **AND** the resulting content hash is not present in the store for that file path
- **THEN** a new row SHALL be inserted in `file_versions` with the correct `file_id`, content blob,
  hash, and current timestamp

#### Scenario: Duplicate write is silently ignored

- **WHEN** `WriteFileUnchecked` is called twice with identical content for the same file
- **THEN** only one row SHALL exist in `file_versions` for that `(file_id, version_hash)` combination

#### Scenario: Multiple files share one database

- **WHEN** two different markdown files are opened and modified in the same tdx session
- **THEN** both files' versions SHALL be stored in the same `versions.sqlite` database
- **AND** each file SHALL have its own row in `files` with a distinct `file_id`
- **AND** versions from different files SHALL not interfere with each other

#### Scenario: Store directory is created automatically

- **WHEN** the tdx config directory does not yet exist
- **AND** a version save is triggered
- **THEN** the config directory SHALL be created and `versions.sqlite` SHALL be initialised without error

#### Scenario: Informational commands do not open the store

- **WHEN** the user runs an informational command such as `--version` or `help`
- **THEN** tdx SHALL NOT create or open `versions.sqlite`
- **AND** the command SHALL work when the config directory is not writable

---

### Requirement: Automatic version capture on file write

The system SHALL save a version automatically every time a markdown file is written to disk via
`markdown.WriteFileUnchecked`, regardless of whether the write was triggered by the TUI or a CLI
command.

- A package-level hook `markdown.WriteHook func(filePath, content string) error` SHALL be called at
  the end of `WriteFileUnchecked` after a successful rename.
- When `WriteHook` is non-nil, it SHALL receive the file path and the serialised content string.
- `cmd/tdx/main.go` SHALL register the versioning store's save function into `markdown.WriteHook`
  before accessing a markdown file.

#### Scenario: TUI action triggers version capture

- **WHEN** the user toggles a todo item in the TUI
- **AND** `WriteFileUnchecked` succeeds
- **THEN** `WriteHook` SHALL be called with the updated file content
- **AND** a new row SHALL appear in the version database (unless content is identical to the last version)

#### Scenario: Hook is nil — no versioning side effect

- **WHEN** `markdown.WriteHook` is nil (e.g., in unit tests that do not register it)
- **AND** `WriteFileUnchecked` is called
- **THEN** no versioning code SHALL be executed and the write SHALL complete normally

---

### Requirement: External-edit detection on file open

The system SHALL detect when a markdown file was modified outside of tdx and create a version for the
externally-modified content when the file is next opened.

- `markdown.ReadFile` SHALL call a package-level hook
  `markdown.ReadHook func(filePath, content string) error` after successfully reading the file content.
- The versioning implementation of `ReadHook` SHALL compute the SHA-256 of the content and insert a row
  only if the hash is not already present in the store for that file.

#### Scenario: External edit captured on open

- **WHEN** a user edits the markdown file with an external editor
- **AND** then opens the file with `tdx`
- **THEN** `ReadHook` SHALL be called with the file's current content
- **AND** a version SHALL be inserted if the content hash is new

#### Scenario: No duplicate on re-open without external change

- **WHEN** the user closes and immediately reopens the same file in tdx without any external edits
- **THEN** `ReadHook` is called
- **AND** no new row is inserted because the hash already exists in the store

---

### Requirement: Test isolation for versioning

The system SHALL provide a mechanism for tests to override the version store directory so that test
runs do not write to the real tdx config directory.

- A helper `versioning.SetStoreDirForTesting(dir string)` SHALL override the directory used to derive
  the `versions.sqlite` path.
- A corresponding `versioning.ResetStoreDirForTesting()` SHALL restore the default.
- These helpers SHALL follow the same pattern as `config.SetConfigDirForTesting`.
- `versioning.DBPath() (string, error)` SHALL derive the shared database path:
  `filepath.Join(getStoreDir(), "versions.sqlite")`.

#### Scenario: Test uses isolated temp directory

- **WHEN** a test calls `versioning.SetStoreDirForTesting(t.TempDir())`
- **AND** triggers a file write
- **THEN** `versions.sqlite` SHALL be created inside the temp directory, not in `~/.config/tdx/`
- **AND** calling `versioning.ResetStoreDirForTesting()` SHALL restore normal behaviour

---

### Requirement: zstd content compression

The system SHALL compress the file content with zstd before writing it to the `content` BLOB column,
using `github.com/klauspost/compress/zstd` (pure Go, no CGO, cross-compilation-safe) at speed level 1
(`zstd.SpeedFastest`).

- The `version_hash` column SHALL be computed from the **uncompressed** content string so that
  deduplication continues to operate on the original text.
- The `content` BLOB SHALL hold the **zstd-compressed** bytes of the file content.
- `Store.SaveVersion(filePath, content string) error` SHALL compress the content before executing the
  INSERT.
- `Store.ReadVersion(filePath string, id int64) (string, error)` SHALL decompress the BLOB and return
  the original string (required for future restore/browse features).

#### Scenario: Content is stored compressed

- **WHEN** `Store.SaveVersion` is called with a markdown string
- **THEN** the bytes stored in the `content` BLOB SHALL be smaller than the original UTF-8 bytes
  (for any non-trivial markdown content)
- **AND** `Store.ReadVersion` SHALL return a string byte-for-byte identical to the original input

#### Scenario: Deduplication still works after adding compression

- **WHEN** `Store.SaveVersion` is called twice with identical content for the same file path
- **THEN** only one row SHALL exist in `file_versions` for that `(file_id, version_hash)` combination
- **AND** the hash SHALL be the SHA-256 of the uncompressed string, not of the compressed bytes

---

### Requirement: Version retention policy

The system SHALL enforce a configurable maximum number of stored versions **per file path**, pruning
the oldest rows whenever the limit is exceeded.

- A `[versioning]` section SHALL be added to `config.toml` with a `max_versions` integer key
  (default `100`).
- The corresponding `VersioningConfig` struct SHALL be added to `UserConfig` in `cmd/tdx/userconfig.go`:

  ```toml
  [versioning]
  max_versions = 100   # 0 = unlimited
  ```

- A `Store.Prune(filePath string, maxVersions int) error` method SHALL delete the oldest rows for the
  given `filePath`, keeping only the `maxVersions` most recent by `id DESC`:

  ```sql
  DELETE FROM file_versions
  WHERE file_id = ?
    AND id NOT IN (
      SELECT id FROM file_versions WHERE file_id = ? ORDER BY id DESC LIMIT ?
    );
  ```

- When `maxVersions` ≤ 0 the method SHALL be a no-op (unlimited retention).
- `versioning.Open(maxVersions int) (*Store, error)` SHALL store `maxVersions` on the `Store` struct.
- Pruning SHALL be performed in `cmd/tdx/main.go`:
  1. After each `SaveVersion` call in the `WriteHook` and `ReadHook` closures.
  2. Before `store.Close()` in the deferred shutdown (iterate `store.fileIDCache` for all seen paths).
- The `maxVersions` value SHALL be read from `appConfig.Versioning.MaxVersions` and passed to
  `versioning.Open`.

#### Scenario: Oldest versions are pruned per file when limit is exceeded

- **WHEN** a store is configured with `max_versions = 3`
- **AND** 5 distinct versions have been saved for file A
- **AND** 2 distinct versions have been saved for file B
- **AND** pruning is triggered for file A
- **THEN** only the 3 most recent rows for file A SHALL remain in `file_versions`
- **AND** both rows for file B SHALL be unaffected

#### Scenario: Zero max_versions disables pruning

- **WHEN** `max_versions = 0` in the config
- **AND** any number of versions are saved
- **THEN** `Prune(filePath, 0)` SHALL be a no-op and all rows SHALL be retained

#### Scenario: Default max_versions is 100

- **WHEN** no `[versioning]` section is present in `config.toml`
- **THEN** `appConfig.Versioning.MaxVersions` SHALL equal `100`

### Requirement: Version listing API

The `versioning.Store` SHALL expose a method to retrieve the version history of a specific file,
enabling consumers (such as the TUI) to display and browse stored snapshots.

- A `versioning.VersionInfo` struct SHALL be defined in `internal/versioning/store.go`:
  ```go
  type VersionInfo struct {
      ID        int64
      CreatedAt time.Time
  }
  ```
- `Store.ListVersions(filePath string) ([]VersionInfo, error)` SHALL return rows from
  `file_versions` for the given file path, ordered by `id DESC` (most recent first).
- If the file path has no versions (or has never been seen), the method SHALL return an empty slice
  without error.

#### Scenario: List returns versions most-recent-first

- **WHEN** three distinct versions have been saved for a file
- **AND** `Store.ListVersions(filePath)` is called
- **THEN** it SHALL return exactly three `VersionInfo` entries
- **AND** the first entry SHALL have the highest `ID`

#### Scenario: List returns empty slice for unknown file

- **WHEN** `Store.ListVersions` is called for a file path that has no saved versions
- **THEN** it SHALL return an empty slice and a nil error
