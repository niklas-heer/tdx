## 1. Characterize and test current failure modes

- [x] 1.1 Replace sleep-based modification tests with exact-revision tests for rapid, same-size, timestamp-preserving, deleted, recreated, and newly created files.
- [x] 1.2 Add deterministic fault-injection coverage for every pre-commit and post-commit save stage.
- [x] 1.3 Add symlink, permission, exact-byte, temp-cleanup, and hook-result tests.
- [x] 1.4 Add helper-subprocess tests for simultaneous tdx writers, lock timeout/recovery, external writers, and crash boundaries.

## 2. Implement the safe save boundary

- [x] 2.1 Add exact content revision state to `markdown.FileModel` and populate it from the exact bytes returned by `ReadFile`.
- [x] 2.2 Add the cross-platform process-shared lock with canonical target identities and bounded acquisition.
- [x] 2.3 Implement unique same-directory temp files, mode preservation, file sync/close, atomic replacement, directory sync where supported, and cleanup.
- [x] 2.4 Add typed conflict, busy, pre-commit, and post-commit errors and update the in-memory revision after a committed replacement.
- [x] 2.5 Run the writer tests with the race detector and repeated stress iterations.

## 3. Migrate callers and version capture

- [x] 3.1 Route CLI and TUI saves through the single conditional save operation and remove separate check-then-unchecked-write sequences.
- [x] 3.2 Make `:save` surface errors and replace the direct `os.WriteFile` force-save fallback with explicit unconditional atomic saving.
- [x] 3.3 Make version restore conditional on the current revision and preserve the active model when a conflict occurs.
- [x] 3.4 Remove the unsafe text-keyed smart merge from the save-critical path and update watcher behavior for external changes.
- [x] 3.5 Add a read-only conflict diff overlay backed by retained local and authoritative disk content, with reload and force-save as explicit resolutions.
- [x] 3.6 Capture the current disk revision before force overwrite and distinguish committed markdown from post-commit version-store errors.
- [x] 3.7 Add SQLite busy handling and synchronize version-store in-memory state for concurrent access.

## 4. Review and verification

- [x] 4.1 Run `go fmt ./...`, `go vet ./...`, `go test ./...`, targeted and full `go test -race`, and repeated multi-process stress tests.
- [x] 4.2 Build the CLI for Linux, macOS, and Windows and verify platform-specific filesystem tests in CI.
- [x] 4.3 Run `go mod tidy` and `openspec validate --all --strict`.
- [x] 4.4 Perform a general correctness review of persistence, error handling, resource ownership, and concurrent state; fix in-scope findings and record separate follow-ups only when they require distinct design work.
