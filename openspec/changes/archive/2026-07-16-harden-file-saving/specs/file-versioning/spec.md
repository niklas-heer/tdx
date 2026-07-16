## MODIFIED Requirements

### Requirement: Automatic version capture on file write

The system SHALL save a version automatically for every markdown revision successfully committed by the shared safe-save boundary, regardless of whether the write was triggered by the TUI, a CLI command, force-save, or version restore.

- A package-level post-commit hook SHALL receive the canonical file path and exact committed content after successful replacement.
- `cmd/tdx/main.go` SHALL register the version store's save function before accessing a markdown file.
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

- **WHEN** no post-commit hook is registered
- **AND** the safe-save operation commits a replacement
- **THEN** no versioning code SHALL execute
- **AND** the markdown save SHALL complete normally

## ADDED Requirements

### Requirement: Concurrent version store access

The shared version store SHALL tolerate simultaneous access from multiple tdx processes and goroutines without corrupting state or failing immediately on ordinary SQLite writer contention.

- SQLite connections SHALL use a bounded busy timeout appropriate for short version inserts and pruning operations.
- Mutable in-process caches SHALL be synchronized or confined so concurrent access is race-free.
- Version insert deduplication SHALL remain correct under concurrent attempts to save identical content.

#### Scenario: Two processes capture versions concurrently

- **WHEN** two tdx processes save versions into the shared database at the same time
- **THEN** both operations SHALL complete within the configured contention bound or return a clear bounded busy error
- **AND** the database SHALL remain valid
- **AND** duplicate hashes for one file SHALL still occupy at most one row

