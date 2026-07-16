# file-saving Specification

## Purpose
TBD - created by archiving change harden-file-saving. Update Purpose after archive.
## Requirements
### Requirement: Exact revision conflict detection

The system SHALL base markdown save conflicts on the exact bytes read from disk, not on file timestamps.

- A loaded file revision SHALL include whether the path existed and a SHA-256 digest of the exact source bytes.
- Conditional write APIs SHALL reject models without a revision established by loading the target.
- A normal save SHALL replace the target only when its existence and digest match the loaded revision at the final validation immediately before replacement.
- An absent loaded path SHALL conflict if any file appears at that path before commit.
- A conflict SHALL leave the current disk bytes unchanged and retain the caller's in-memory edits.
- The external disk bytes SHALL be treated as authoritative unless the user explicitly force-saves.

#### Scenario: Rapid external edit is detected

- **WHEN** tdx reads a markdown file
- **AND** another application changes its content within the same timestamp precision window
- **AND** tdx attempts a normal save
- **THEN** the save SHALL fail with a conflict
- **AND** the external content SHALL remain on disk

#### Scenario: New file is created concurrently

- **WHEN** tdx loads a model for an absent path
- **AND** another process creates that path before tdx saves
- **THEN** the normal save SHALL fail with a conflict
- **AND** the concurrently created file SHALL not be overwritten

### Requirement: Serialized tdx saves

The system SHALL serialize writes by tdx processes that target the same canonical file.

- Lock identity SHALL be derived from the canonical target path so aliases and symlinks within the supported resolution model coordinate on the same lock.
- Lock acquisition SHALL have a bounded timeout and SHALL return a distinct busy error on timeout.
- The expected revision SHALL be revalidated while the lock is held and immediately before replacement.
- Lock files SHALL have stable identities and SHALL not be deleted during normal unlock.

#### Scenario: Two tdx instances save one baseline

- **WHEN** two tdx instances load the same file revision
- **AND** both attempt different normal saves concurrently
- **THEN** no more than one save SHALL commit against that revision
- **AND** the other save SHALL return a conflict or bounded busy error
- **AND** the final file SHALL equal one complete candidate, never a mixture or partial write

#### Scenario: Lock holder exits unexpectedly

- **WHEN** a tdx process exits while holding the process-shared lock
- **THEN** the operating system SHALL release the lock
- **AND** a later tdx save SHALL be able to acquire it without manual stale-lock cleanup

### Requirement: Atomic and durable replacement

Every tdx markdown write, including force-save and version restore, SHALL use the same safe replacement pipeline.

- The system SHALL use an operation-unique temporary file in the target directory.
- It SHALL write all content, sync it, and close it before replacing the target.
- It SHALL atomically replace the target where the platform and filesystem provide atomic replace semantics.
- It SHALL sync the containing directory where the platform supports directory sync.
- It SHALL remove leftover temporary files after success and handled failures.
- A failure before replacement SHALL leave the original target bytes unchanged.
- A failure after replacement SHALL be reported as committed with a post-commit warning or error classification.

#### Scenario: Write fails before replacement

- **WHEN** creating, writing, syncing, closing, locking, or validating the prepared replacement fails
- **THEN** the existing target SHALL remain byte-for-byte unchanged
- **AND** no operation temporary file SHALL remain after cleanup

#### Scenario: Process stops around replacement

- **WHEN** a tdx process stops after preparing a replacement or immediately after replacing the target on a platform and filesystem that provide atomic replace semantics
- **THEN** the target SHALL contain either the complete previous revision or the complete replacement revision
- **AND** a partially written target SHALL never be visible

### Requirement: Target path and permission preservation

The save pipeline SHALL preserve the user's logical file path and portable permission bits.

- When the opened path is a symlink to an existing regular file, tdx SHALL update the resolved target and SHALL not replace the symlink entry.
- When replacing an existing file, tdx SHALL apply its portable permission bits to the replacement.
- When creating a new file, tdx SHALL retain the temporary file's restrictive creation mode rather than widening it after creation.
- Unsupported targets such as directories SHALL fail before replacement.

#### Scenario: Save through a symlink

- **WHEN** tdx opens a symlink to a markdown file and saves a change
- **THEN** the symlink SHALL still exist and point to the same target
- **AND** the target SHALL contain the complete saved content

#### Scenario: Preserve restrictive permissions

- **WHEN** an existing markdown file has restrictive permission bits
- **AND** tdx saves it
- **THEN** those permission bits SHALL remain on the replacement file

#### Scenario: Create a file with restrictive permissions

- **WHEN** tdx saves a newly loaded absent path
- **THEN** the new file SHALL not grant group or other permissions on platforms with portable mode bits

### Requirement: Explicit overwrite policy

Normal saves and version restores SHALL be conditional; only an explicit force-save operation MAY overwrite a stale revision.

- The system SHALL not automatically merge stale Markdown with a text-keyed heuristic.
- A restore SHALL fail safely if the active file changed before confirmation.
- Force-save SHALL use locking, prepared atomic replacement, metadata preservation, and durability handling even though it omits the revision precondition.
- Force-save SHALL capture the current disk bytes in version history before replacement and the committed bytes after replacement when versioning is enabled.
- After a conflict, the TUI SHALL retain the local candidate and the observed disk content for a read-only local-versus-disk diff.
- Viewing or closing the conflict diff SHALL modify neither the disk version nor the retained local candidate.

#### Scenario: Restore conflicts with an external edit

- **WHEN** the version browser is open
- **AND** another application changes the active file
- **AND** the user confirms restore
- **THEN** restore SHALL report a conflict
- **AND** the external content SHALL remain on disk
- **AND** the selected historic content SHALL remain available for a later explicit decision

#### Scenario: User explicitly force-saves

- **WHEN** a normal save reports a conflict
- **AND** the user explicitly invokes force-save
- **THEN** tdx SHALL atomically commit the in-memory content without the stale-revision precondition
- **AND** the overwritten disk content SHALL remain recoverable from version history when versioning is enabled

#### Scenario: User inspects a conflict

- **WHEN** a normal save conflicts with external content
- **AND** the user opens the conflict diff
- **THEN** the TUI SHALL compare the retained local candidate with the authoritative disk content
- **AND** closing the diff SHALL leave both versions unchanged
- **AND** the user SHALL still be able to choose reload or force-save explicitly

### Requirement: External application safety boundary

The system SHALL accurately communicate and test the boundary of coordination with non-tdx applications.

- tdx SHALL detect external content changes observed before its atomic replacement and reject conditional saves.
- An external change that starts after final validation is outside the conditional-save guarantee because non-cooperating applications do not share the tdx lock.
- tdx SHALL NOT claim that advisory locks can detect every open handle or block applications that do not use the tdx lock.
- Regardless of external lock cooperation, a tdx replacement SHALL never intentionally expose partial content.

#### Scenario: Non-cooperating editor changes the baseline

- **WHEN** an external application that ignores tdx locks replaces the file before tdx's final revision validation
- **THEN** the conditional tdx save SHALL report a conflict
- **AND** SHALL not overwrite that observed external revision

#### Scenario: Non-cooperating editor changes after validation

- **WHEN** an external application changes the target after tdx's final revision validation
- **THEN** tdx SHALL make no compare-and-swap guarantee for that unobserved change
- **AND** tdx SHALL still use the strongest replacement and durability primitives available on the platform

