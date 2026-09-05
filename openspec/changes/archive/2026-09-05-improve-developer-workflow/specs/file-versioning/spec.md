## MODIFIED Requirements
### Requirement: SQLite-backed version store

The system SHALL maintain a continuous version history for all markdown files opened interactively or modified by
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
- For interactive and file-mutating commands, `cmd/tdx/main.go` SHALL maintain a **single**
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

#### Scenario: Script lists without history

- **WHEN** a script invokes `tdx list`, including JSON and filtered queries
- **THEN** the application SHALL not open the version store or register versioning hooks
- **AND** the query SHALL work without a writable configuration directory

