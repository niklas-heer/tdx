## MODIFIED Requirements

### Requirement: Automatic version capture on file write

The system SHALL save a version automatically for every markdown revision successfully committed by the shared safe-save boundary, regardless of whether the write was triggered by the TUI, a CLI command, force-save, or version restore.

- A per-instance `markdown.Store.OnWrite` callback SHALL receive the canonical file path and exact committed content after successful replacement.
- `cmd/tdx/main.go` SHALL inject the version store's save function into its Markdown store before accessing a markdown file.
- A version-capture failure after replacement SHALL be surfaced as a post-commit failure and SHALL NOT be described as an uncommitted markdown save.
- An explicit force-save SHALL capture the current disk content before replacement as well as the committed content after replacement.

#### Scenario: TUI action triggers version capture

- **WHEN** the user changes a todo item in the TUI
- **AND** the safe-save operation commits the replacement
- **THEN** the post-commit hook SHALL receive the exact committed content
- **AND** a new row SHALL appear in the version database unless that content hash already exists for the file

#### Scenario: Version capture fails after commit

- **WHEN** the markdown replacement commits successfully
- **AND** the version store returns an error
- **THEN** the caller SHALL be told that the markdown content was committed but version capture failed
- **AND** the in-memory file revision SHALL reflect the committed markdown content

#### Scenario: Hook is nil

- **WHEN** the store has no post-commit callback
- **AND** the safe-save operation commits a replacement
- **THEN** no versioning code SHALL execute
- **AND** the markdown save SHALL complete normally

### Requirement: External-edit detection on file open

The system SHALL detect when a markdown file was modified outside of tdx and create a version for the
externally-modified content when the file is next opened.

- `markdown.Store.ReadFile` SHALL call its instance callback
  `OnRead func(filePath, content string) error` after successfully reading the file content.
- The versioning implementation of `OnRead` SHALL compute the SHA-256 of the content and insert a row
  only if the hash is not already present in the store for that file.

#### Scenario: External edit captured on open

- **WHEN** a user edits the markdown file with an external editor
- **AND** then opens the file with `tdx`
- **THEN** `OnRead` SHALL be called with the file's current content
- **AND** a version SHALL be inserted if the content hash is new

#### Scenario: No duplicate on re-open without external change

- **WHEN** the user closes and immediately reopens the same file in tdx without any external edits
- **THEN** `OnRead` is called
- **AND** no new row is inserted because the hash already exists in the store

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
  1. After each `SaveVersion` call in the `OnWrite` and `OnRead` callbacks.
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
