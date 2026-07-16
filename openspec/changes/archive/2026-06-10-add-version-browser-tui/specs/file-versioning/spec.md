## ADDED Requirements

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
