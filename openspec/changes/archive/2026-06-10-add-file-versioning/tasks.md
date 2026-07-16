## 1. Dependency

- [x] 1.1 Add `modernc.org/sqlite` to `go.mod` via `go get modernc.org/sqlite` and run `go mod tidy`

## 2. Versioning package

- [x] 2.1 Create `internal/versioning/store.go`:
  - `Store` struct holding `*sql.DB` and `dbPath`
  - `getStoreDir` package-level var pointing to `config.GetConfigDir` (mirrors `getConfigDir` in `config/recent.go`)
  - `SetStoreDirForTesting(dir string)` and `ResetStoreDirForTesting()` helpers
  - `DBPathForFile(filePath string) (string, error)` — derives `<storeDir>/<basename>.sqlite`
  - `Open(filePath string) (*Store, error)` — creates config dir if missing, opens db, applies WAL pragmas, creates table
  - `SaveVersion(content string) error` — SHA-256 hash, `INSERT OR IGNORE`
  - `Close() error`
- [x] 2.2 Create `internal/versioning/store_test.go`:
  - Test `Open` creates db and table
  - Test `SaveVersion` inserts a new row
  - Test duplicate content is ignored (row count stays 1)
  - Test `SetStoreDirForTesting` isolation

## 3. Markdown package hooks

- [x] 3.1 Add to `internal/markdown/parser.go`:
  - `var WriteHook func(filePath, content string)` (nil by default)
  - `var ReadHook  func(filePath, content string)` (nil by default)
- [x] 3.2 Call `WriteHook` at the end of `WriteFileUnchecked` (after the successful `os.Rename`)
- [x] 3.3 Call `ReadHook` in `ReadFile` after the file content is successfully read (skip for new-file path)

## 4. Wiring in main

- [x] 4.1 In `cmd/tdx/main.go`, at startup:
  - Open (or lazily open) a `*versioning.Store` for the target file
  - Assign `markdown.WriteHook` to call `store.SaveVersion(content)`
  - Assign `markdown.ReadHook` to call `store.SaveVersion(content)`
  - Ensure `store.Close()` is called on exit (defer or `tea.Quit` handler)

## 5. Tests

- [x] 5.1 Add a unit test in `internal/markdown/` that verifies `WriteHook` is called with correct args when `WriteFileUnchecked` succeeds
- [x] 5.2 Add a unit test in `internal/markdown/` that verifies `ReadHook` is called when `ReadFile` loads an existing file
- [x] 5.3 Verify existing TUI tests still pass (`just test`)

## 6. Validation

- [x] 6.1 Run `go build ./...` — no compilation errors
- [x] 6.2 Run `go vet ./...` — no warnings
- [x] 6.3 Run `just test` — all tests green
- [ ] 6.4 Manual smoke-test: open a markdown file with `tdx`, make a change, verify `~/.config/tdx/versions.sqlite` exists and contains rows

## 7. Refactor to single shared database  *(supersedes tasks 2.x; do before 8 and 9)*

- [x] 7.1 Rewrite `internal/versioning/store.go` for the single `versions.sqlite` model:
  - `DBPath() (string, error)` replaces `DBPathForFile` — returns `filepath.Join(getStoreDir(), "versions.sqlite")`
  - Two-table schema: `files (id, file_path)` + `file_versions (id, file_id FK, version_hash, content, created_at, commit_message)`
    with `UNIQUE(file_id, version_hash)` and `CREATE INDEX … ON file_versions(file_id)`
  - `Open(maxVersions int) (*Store, error)` — no longer takes a filePath; opens the shared DB,
    applies WAL pragmas, creates both tables and the index
  - `Store` gains a `fileIDCache map[string]int64` and `maxVersions int` field
  - `resolveFileID(filePath string) (int64, error)` — INSERT OR IGNORE into `files`, then SELECT id;
    cache result in `fileIDCache`
  - `SaveVersion(filePath, content string) error` — call `resolveFileID`, then INSERT OR IGNORE into `file_versions`
  - `Prune(filePath string, maxVersions int) error` — file-scoped DELETE; no-op when ≤ 0
  - `ReadVersion(filePath string, id int64) (string, error)` — stub for now (returns placeholder);
    full decompression implemented in task 8
  - `Close() error` — unchanged
- [x] 7.2 Rewrite `internal/versioning/store_test.go` for the shared DB:
  - Remove `TestDBPathForFile`; add `TestDBPath`
  - Update `TestOpen_CreatesDBAndTable` — verify both `files` and `file_versions` tables exist
  - Update `TestSaveVersion_*` — pass `filePath` arg; verify `files` row created
  - Add `TestSaveVersion_MultipleFiles` — two file paths produce two rows in `files`, independent versions
  - Keep isolation tests (`SetStoreDirForTesting`)
- [x] 7.3 Update `cmd/tdx/main.go`:
  - Replace `versionStores map[string]*versioning.Store` + `versionStoresMu` with a single
    `var versionStore *versioning.Store`
  - `openVersionStore(maxVersions int)` replaces `getOrOpenVersionStore`
  - `closeVersionStore()` replaces `closeAllVersionStores`
  - `registerVersioningHooks` closures call `versionStore.SaveVersion(filePath, content)` directly
  - `registerVersioningHooks` and `closeVersionStore` wired with `defer` in `main()`
- [x] 7.4 Run `go build ./...`, `go vet ./...`, `go test ./...` — all green

## 8. zstd compression  *(depends on task 7)*

- [x] 8.1 Add `github.com/klauspost/compress` to `go.mod` via `go get github.com/klauspost/compress` and run `go mod tidy`
- [x] 8.2 Update `internal/versioning/store.go`:
  - Import `github.com/klauspost/compress/zstd`
  - Initialise package-level `*zstd.Encoder` (SpeedFastest) and `*zstd.Decoder`
  - In `SaveVersion`: compress content bytes before the INSERT
  - Complete `ReadVersion(filePath string, id int64) (string, error)`: SELECT content BLOB,
    decompress with the decoder, return the original string
- [x] 8.3 Update `internal/versioning/store_test.go`:
  - Add test: `ReadVersion` round-trips content correctly (SaveVersion then ReadVersion → identical)
  - Add test: duplicate detection still works when compression is active

## 9. Version retention  *(depends on task 7)*

- [x] 9.1 Add `VersioningConfig` struct and `Versioning VersioningConfig` field to `UserConfig` in
  `cmd/tdx/userconfig.go`; default `MaxVersions = 100`; handle missing section in `LoadConfig` /
  `DefaultConfig`
- [x] 9.2 Update `cmd/tdx/main.go`:
  - Pass `appConfig.Versioning.MaxVersions` to `versioning.Open`
  - After each `versionStore.SaveVersion(filePath, content)` in the hooks, call
    `versionStore.Prune(filePath, versionStore.MaxVersions)`
  - Before `store.Close()` in `closeVersionStore`, call `store.Prune` for each file path seen
    this session (iterate `store.fileIDCache`)
- [x] 9.3 Update `internal/versioning/store_test.go`:
  - Test `Prune` removes oldest rows for the target file only
  - Test `Prune(filePath, 0)` is a no-op
  - Test that a second file's versions are unaffected by pruning the first
- [x] 9.4 Run `go build ./...`, `go vet ./...`, `go test ./...` — all green
