## Context

`FileModel` currently records only the source file's modification time. `CheckFileModified` accepts timestamps within one second, and callers commonly perform that check separately from an unchecked temp-file rename. This creates both false negatives and a time-of-check/time-of-use race. The temp name is process-wide rather than operation-unique, replacement can change permissions or replace a symlink, and force-save bypasses the atomic writer entirely.

Multiple tdx processes can cooperate through a shared lock. Arbitrary external applications generally do not use the same lock, so no portable implementation can prevent an external writer from replacing the file after tdx's final comparison. The design therefore provides a strong compare-and-swap guarantee among tdx processes, detects external changes observed before commit, makes each replacement atomic, and keeps before/after versions recoverable. It does not claim mandatory locking of other applications.

Context7 is unavailable in this environment. The proposed lock dependency was checked against its current upstream source and cross-platform implementation instead.

## Goals / Non-Goals

### Goals

- Never expose partially written markdown through a tdx save.
- Never silently overwrite a revision that changed after tdx loaded it.
- Serialize competing saves from tdx instances and return a bounded busy/conflict error rather than hanging.
- Preserve the target's symlink, permission bits, and exact restored bytes.
- Distinguish failures before replacement from failures in post-commit durability or version capture.
- Let users inspect the retained local candidate against authoritative external disk content before resolving a conflict.
- Make race and failure behavior deterministic enough to test in CI.

### Non-Goals

- Mandatory locking of editors or tools that do not cooperate with tdx.
- Automatic semantic merging of arbitrary Markdown edits.
- Portable preservation of every filesystem-specific attribute, ACL, owner, or extended attribute in this change.
- Distributed coordination across network filesystems whose rename or lock implementation does not provide local-filesystem semantics.

## Decisions

### Exact content revisions

`ReadFile` will record whether the target existed and a SHA-256 revision of the exact bytes read. Normal saves will compare that expected revision with the current target bytes. Modification time remains informational only and is not used for correctness.

An absent file is a distinct revision. If another process creates the path before tdx commits its initial file, the save conflicts instead of replacing the newly created file.

### One compare-and-swap save boundary

All callers will delegate to one internal save operation. A normal save supplies an expected revision. An explicit force save omits the precondition but retains every other safety guarantee.

The operation will:

1. Resolve an existing symlink to its target and derive a canonical lock identity.
2. Create an operation-unique temporary file in the target directory.
3. Apply the existing target's permission bits when present, write all bytes, sync, and close the temporary file.
4. Acquire the canonical target's process-shared lock with a bounded timeout.
5. Read and hash the current target while holding the lock and reject a stale expected revision.
6. Atomically replace the target where the platform and filesystem support it.
7. Sync the containing directory where supported, update the in-memory revision, invoke post-commit version capture, release the lock, and remove any leftover temporary file.

Preparing and syncing the temporary file before taking the lock keeps the serialized section short. Revalidation happens after the lock is acquired and immediately before replacement.

### Stable process-shared locks

Use `github.com/gofrs/flock` with lock files stored under the tdx configuration directory and named by a SHA-256 hash of the canonical target path. Lock files are stable and are not deleted after each write, avoiding the inode-replacement race caused by deleting and recreating a sidecar lock.

The lock is advisory. It serializes tdx instances for the same canonical target but cannot prove that another application has the file open and cannot block an application that ignores the lock. Lock acquisition uses a bounded context and returns a distinct busy error when the timeout expires.

### Conservative conflict behavior

The current text-keyed smart merge is not safe for duplicate todo text and cannot reliably distinguish edits, deletion, and reordering. Normal persistence will stop on a revision conflict and retain the in-memory edit. Reload remains available to accept disk content. Automatic semantic merging is removed from the save-critical path until it can be implemented as a real three-way merge against a stored baseline.

On conflict, the external version remains authoritative on disk. The TUI retains both the serialized local candidate and the observed disk bytes in memory and offers a lightweight read-only diff overlay. This reuses the existing `sergi/go-diff` renderer. Reload discards the local candidate and adopts disk; force-save explicitly replaces disk after version capture. The diff itself performs no writes and is not a merge editor.

Version restore uses the revision active when the restore is confirmed. If the file changed while the version browser was open, restore fails without overwriting the newer bytes. Force-save is the only unconditional overwrite operation and must use the same atomic pipeline.

### Commit-aware errors and version capture

Errors before replacement guarantee that the original path was not changed by tdx. Errors after replacement will use a distinct error type/status so callers can say that the markdown was saved but durability confirmation or version capture failed.

Before an explicit force overwrite, the current disk bytes will be sent to version capture, followed by the committed replacement bytes. Deduplication makes repeated capture harmless. Version-store access will use a busy timeout and synchronized in-process state so simultaneous tdx instances do not turn a successful markdown commit into a misleading generic save failure.

## Testing Strategy

### Deterministic unit tests

- Replace time-based conflict tests with same-timestamp and same-size content changes that must conflict immediately.
- Inject failures at create, chmod, write, sync, close, revision read, lock, replace, directory sync, and version-hook boundaries.
- Assert that pre-commit failures leave the original bytes intact, temporary files are cleaned up, and post-commit failures report committed state.
- Verify new-file races, deletion/recreation, permission preservation, symlink preservation, exact restore bytes, and duplicate todo text.
- Verify the conflict overlay compares the retained local candidate with the observed authoritative disk bytes and that closing it changes neither side.

### Multi-process integration tests

- Use helper subprocesses and barriers so two writers load the same baseline and attempt to commit together; exactly one normal save may succeed and the final file must be one complete candidate.
- Hold the shared lock in one subprocess and verify another times out with a busy error, then succeeds after release.
- Simulate a non-cooperating editor with direct and atomic-replace writes before tdx's revision check; tdx must report a conflict and retain the external bytes.
- Kill a helper after temp-file sync and around replacement; the target must contain either the complete old revision or the complete new revision, never a partial document.

### CI and stress coverage

- Run the save suite with `-race`, repeated concurrent iterations, and existing full tests.
- Execute filesystem integration tests on Linux, macOS, and Windows runners, skipping only assertions unsupported by a documented platform/filesystem contract.
- Cross-build all supported targets and run `go vet`, formatting, module tidiness, and strict OpenSpec validation.

## Risks / Trade-offs

- Advisory locks cannot control other applications. Mitigation: exact precondition checks, a minimal check-to-replace window, atomic replacement, post-commit verification where useful, and recoverable versions.
- Atomic replacement can affect attributes beyond portable permission bits. Mitigation: preserve supported mode bits now and document filesystem-specific attributes as follow-up work.
- Locking in the config directory adds a dependency on that directory. This matches the existing version-store requirement; lock setup failures are surfaced before replacement.
- Removing heuristic auto-merge is more conservative. It prefers an explicit conflict over silent loss and keeps local state available for a user decision.

## Migration Plan

No file-format or database migration is required. Existing markdown and version history remain valid. The implementation will first add the new save primitive and tests, then migrate normal writes, force-save, and restore before removing the timestamp-based path.

## Follow-up Findings

- `recent.json` and theme/config writes still use direct truncate-and-write persistence and can lose updates or become partial if multiple tdx processes write them concurrently. They are secondary metadata rather than the primary markdown store and should move to a separate generic atomic-config change.
- The legacy `tdx-cli` specification still describes the retired Bun/TypeScript implementation. Correcting that baseline is specification maintenance outside this save-safety change.

## Resolved Questions

- External content remains authoritative on disk after a conflict; tdx does not auto-merge or overwrite it.
- tdx retains the local candidate and provides a lightweight internal diff, with `:reload` and `:force-save` as explicit resolutions.
