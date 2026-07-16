## Why

Markdown files are tdx's primary data store, but the current save path can miss rapid external edits, allows two tdx processes to overwrite the same baseline, and does not fully protect file metadata or durability. The new version history reduces the impact of data loss, but it does not replace a fail-safe write boundary.

## What Changes

- Replace modification-time conflict detection with an exact content revision captured when the file is read.
- Serialize writes from multiple tdx processes with a bounded, cross-platform advisory lock and revalidate the expected revision while holding that lock.
- Replace every markdown write path with one atomic save pipeline that uses unique same-directory temporary files, preserves permissions and symlinks, syncs data, and cleans up after failures.
- Make normal saves and version restores refuse stale writes; retain a clearly explicit force-save path that still uses atomic replacement and captures the overwritten content in version history.
- Retain the uncommitted local candidate after a conflict and provide a lightweight local-versus-disk diff before the user chooses reload or force-save.
- Add deterministic fault-injection, multi-process, external-editor, symlink, permission, and crash-boundary tests.
- Harden the shared version store for concurrent tdx processes and surface post-commit versioning failures without reporting that the markdown replacement itself failed.

## Impact

- Affected specs: new `file-saving`; modified `file-versioning` and `version-browser-tui`
- Affected code: `internal/markdown`, `internal/tui`, `internal/versioning`, `cmd/tdx`, platform CI, and their tests
- New dependency: `github.com/gofrs/flock` for process-shared advisory locks on supported Unix and Windows platforms
- User-visible behavior: stale normal saves and stale restores fail safely; the external version remains authoritative on disk and can be compared with the retained local candidate before reload or explicit, recoverable force-save
