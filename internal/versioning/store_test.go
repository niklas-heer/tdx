package versioning

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// --- helpers ---

func rowCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM file_versions").Scan(&n); err != nil {
		t.Fatalf("row count query error: %v", err)
	}
	return n
}

func rowCountForFile(t *testing.T, db *sql.DB, fileID int64) int {
	t.Helper()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM file_versions WHERE file_id = ?", fileID).Scan(&n); err != nil {
		t.Fatalf("row count query error: %v", err)
	}
	return n
}

func openStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(0)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// --- DBPath ---

func TestDBPath(t *testing.T) {
	dir := t.TempDir()
	SetStoreDirForTesting(dir)
	defer ResetStoreDirForTesting()

	got, err := DBPath()
	if err != nil {
		t.Fatalf("DBPath() error: %v", err)
	}
	want := filepath.Join(dir, "versions.sqlite")
	if got != want {
		t.Errorf("DBPath() = %q, want %q", got, want)
	}
}

// --- Open ---

func TestOpen_CreatesDBAndBothTables(t *testing.T) {
	dir := t.TempDir()
	SetStoreDirForTesting(dir)
	defer ResetStoreDirForTesting()

	s := openStore(t)

	// db file must exist
	dbPath := filepath.Join(dir, "versions.sqlite")
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("versions.sqlite not created: %v", err)
	}

	// both tables must be queryable
	for _, table := range []string{"files", "file_versions"} {
		var n int
		if err := s.db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
			t.Fatalf("table %s not accessible: %v", table, err)
		}
	}
}

// --- SaveVersion ---

func TestSaveVersion_InsertsRow(t *testing.T) {
	dir := t.TempDir()
	SetStoreDirForTesting(dir)
	defer ResetStoreDirForTesting()

	s := openStore(t)
	if err := s.SaveVersion("/a/todo.md", "# Todos\n\n- [ ] task one\n"); err != nil {
		t.Fatalf("SaveVersion() error: %v", err)
	}

	if got := rowCount(t, s.db); got != 1 {
		t.Errorf("expected 1 row, got %d", got)
	}
}

func TestSaveVersion_CreatesFilesRow(t *testing.T) {
	dir := t.TempDir()
	SetStoreDirForTesting(dir)
	defer ResetStoreDirForTesting()

	s := openStore(t)
	if err := s.SaveVersion("/a/todo.md", "content"); err != nil {
		t.Fatalf("SaveVersion() error: %v", err)
	}

	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM files WHERE file_path = ?", "/a/todo.md").Scan(&count); err != nil {
		t.Fatalf("files query error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 row in files, got %d", count)
	}
}

func TestSaveVersion_DuplicateIsIgnored(t *testing.T) {
	dir := t.TempDir()
	SetStoreDirForTesting(dir)
	defer ResetStoreDirForTesting()

	s := openStore(t)
	content := "# Todos\n\n- [ ] same content\n"
	for i := 0; i < 2; i++ {
		if err := s.SaveVersion("/a/todo.md", content); err != nil {
			t.Fatalf("SaveVersion() [%d] error: %v", i, err)
		}
	}

	if got := rowCount(t, s.db); got != 1 {
		t.Errorf("expected 1 row after duplicate save, got %d", got)
	}
}

func TestSaveVersion_DifferentContentAddsRow(t *testing.T) {
	dir := t.TempDir()
	SetStoreDirForTesting(dir)
	defer ResetStoreDirForTesting()

	s := openStore(t)
	_ = s.SaveVersion("/a/todo.md", "# Todos\n\n- [ ] first\n")
	_ = s.SaveVersion("/a/todo.md", "# Todos\n\n- [x] first\n")

	if got := rowCount(t, s.db); got != 2 {
		t.Errorf("expected 2 rows, got %d", got)
	}
}

func TestSaveVersion_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	SetStoreDirForTesting(dir)
	defer ResetStoreDirForTesting()

	s := openStore(t)
	_ = s.SaveVersion("/a/work.md", "# Work\n\n- [ ] task\n")
	_ = s.SaveVersion("/b/home.md", "# Home\n\n- [ ] task\n") // same content, different file

	// Two rows in files
	var fileCount int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM files").Scan(&fileCount); err != nil {
		t.Fatalf("files count error: %v", err)
	}
	if fileCount != 2 {
		t.Errorf("expected 2 rows in files, got %d", fileCount)
	}

	// Two rows in file_versions (same content but different file_id → distinct UNIQUE key)
	if got := rowCount(t, s.db); got != 2 {
		t.Errorf("expected 2 version rows (one per file), got %d", got)
	}
}

// --- ReadVersion ---

func TestReadVersion_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	SetStoreDirForTesting(dir)
	defer ResetStoreDirForTesting()

	s := openStore(t)
	const filePath = "/a/todo.md"
	const content = "# Todos\n\n- [ ] round-trip test\n"

	if err := s.SaveVersion(filePath, content); err != nil {
		t.Fatalf("SaveVersion() error: %v", err)
	}

	// Find the inserted row id
	var id int64
	fileID, _ := s.resolveFileID(filePath)
	if err := s.db.QueryRow("SELECT id FROM file_versions WHERE file_id = ?", fileID).Scan(&id); err != nil {
		t.Fatalf("SELECT id error: %v", err)
	}

	got, err := s.ReadVersion(filePath, id)
	if err != nil {
		t.Fatalf("ReadVersion() error: %v", err)
	}
	if got != content {
		t.Errorf("ReadVersion() = %q, want %q", got, content)
	}
}

// --- Prune ---

func TestPrune_KeepsNewestRows(t *testing.T) {
	dir := t.TempDir()
	SetStoreDirForTesting(dir)
	defer ResetStoreDirForTesting()

	s := openStore(t)
	const filePath = "/a/todo.md"
	for i := 0; i < 5; i++ {
		content := fmt.Sprintf("# Todos\n\n- [ ] item %d\n", i)
		if err := s.SaveVersion(filePath, content); err != nil {
			t.Fatalf("SaveVersion(%d) error: %v", i, err)
		}
	}

	if err := s.Prune(filePath, 3); err != nil {
		t.Fatalf("Prune() error: %v", err)
	}

	fileID, _ := s.resolveFileID(filePath)
	if got := rowCountForFile(t, s.db, fileID); got != 3 {
		t.Errorf("expected 3 rows after Prune(3), got %d", got)
	}
}

func TestPrune_ZeroIsNoOp(t *testing.T) {
	dir := t.TempDir()
	SetStoreDirForTesting(dir)
	defer ResetStoreDirForTesting()

	s := openStore(t)
	const filePath = "/a/todo.md"
	for i := 0; i < 5; i++ {
		_ = s.SaveVersion(filePath, fmt.Sprintf("content %d", i))
	}

	if err := s.Prune(filePath, 0); err != nil {
		t.Fatalf("Prune(0) error: %v", err)
	}

	if got := rowCount(t, s.db); got != 5 {
		t.Errorf("expected 5 rows (Prune(0) is no-op), got %d", got)
	}
}

func TestPrune_DoesNotAffectOtherFiles(t *testing.T) {
	dir := t.TempDir()
	SetStoreDirForTesting(dir)
	defer ResetStoreDirForTesting()

	s := openStore(t)
	for i := 0; i < 5; i++ {
		_ = s.SaveVersion("/a/a.md", fmt.Sprintf("a content %d", i))
	}
	for i := 0; i < 2; i++ {
		_ = s.SaveVersion("/b/b.md", fmt.Sprintf("b content %d", i))
	}

	// Prune file A down to 3, file B should be untouched.
	if err := s.Prune("/a/a.md", 3); err != nil {
		t.Fatalf("Prune() error: %v", err)
	}

	fileIDB, _ := s.resolveFileID("/b/b.md")
	if got := rowCountForFile(t, s.db, fileIDB); got != 2 {
		t.Errorf("expected 2 rows for file B after pruning file A, got %d", got)
	}
}

// --- Compression round-trip ---

func TestReadVersion_RoundTrip_WithCompression(t *testing.T) {
	dir := t.TempDir()
	SetStoreDirForTesting(dir)
	defer ResetStoreDirForTesting()

	s := openStore(t)
	const filePath = "/a/todo.md"
	// Longer content to confirm compression actually runs.
	content := "# Todos\n\n"
	for i := 0; i < 50; i++ {
		content += fmt.Sprintf("- [ ] task number %d with some extra text to make it compressible\n", i)
	}

	if err := s.SaveVersion(filePath, content); err != nil {
		t.Fatalf("SaveVersion() error: %v", err)
	}

	fileID, _ := s.resolveFileID(filePath)
	var id int64
	if err := s.db.QueryRow("SELECT id FROM file_versions WHERE file_id = ?", fileID).Scan(&id); err != nil {
		t.Fatalf("SELECT id error: %v", err)
	}

	got, err := s.ReadVersion(filePath, id)
	if err != nil {
		t.Fatalf("ReadVersion() error: %v", err)
	}
	if got != content {
		t.Errorf("ReadVersion() did not round-trip: len(got)=%d, len(want)=%d", len(got), len(content))
	}

	// Verify the stored bytes are actually smaller (i.e. compression ran).
	var compressed []byte
	if err := s.db.QueryRow("SELECT content FROM file_versions WHERE id = ?", id).Scan(&compressed); err != nil {
		t.Fatalf("SELECT content error: %v", err)
	}
	if len(compressed) >= len(content) {
		t.Errorf("compressed (%d bytes) should be smaller than original (%d bytes)", len(compressed), len(content))
	}
}

func TestSaveVersion_DeduplicationWithCompression(t *testing.T) {
	dir := t.TempDir()
	SetStoreDirForTesting(dir)
	defer ResetStoreDirForTesting()

	s := openStore(t)
	const filePath = "/a/todo.md"
	content := "# Todos\n\n- [ ] dedup test\n"

	for i := 0; i < 3; i++ {
		if err := s.SaveVersion(filePath, content); err != nil {
			t.Fatalf("SaveVersion() [%d] error: %v", i, err)
		}
	}

	if got := rowCount(t, s.db); got != 1 {
		t.Errorf("expected 1 row after 3 identical saves, got %d", got)
	}
}

// --- ListVersions ---

func TestListVersions_ReturnsMostRecentFirst(t *testing.T) {
	dir := t.TempDir()
	SetStoreDirForTesting(dir)
	defer ResetStoreDirForTesting()

	s := openStore(t)
	const filePath = "/a/todo.md"

	_ = s.SaveVersion(filePath, "version one content")
	_ = s.SaveVersion(filePath, "version two content")
	_ = s.SaveVersion(filePath, "version three content")

	versions, err := s.ListVersions(filePath)
	if err != nil {
		t.Fatalf("ListVersions() error: %v", err)
	}
	if len(versions) != 3 {
		t.Fatalf("expected 3 versions, got %d", len(versions))
	}
	// Most recent first: IDs should be descending.
	if versions[0].ID <= versions[1].ID || versions[1].ID <= versions[2].ID {
		t.Errorf("expected descending IDs, got %v, %v, %v",
			versions[0].ID, versions[1].ID, versions[2].ID)
	}
}

func TestListVersions_EmptyForUnknownFile(t *testing.T) {
	dir := t.TempDir()
	SetStoreDirForTesting(dir)
	defer ResetStoreDirForTesting()

	s := openStore(t)

	versions, err := s.ListVersions("/no/such/file.md")
	if err != nil {
		t.Fatalf("ListVersions() unexpected error: %v", err)
	}
	if len(versions) != 0 {
		t.Errorf("expected empty slice, got %d versions", len(versions))
	}
}

func TestListVersions_CreatedAtIsNonZero(t *testing.T) {
	dir := t.TempDir()
	SetStoreDirForTesting(dir)
	defer ResetStoreDirForTesting()

	s := openStore(t)
	if err := s.SaveVersion("/a/todo.md", "content"); err != nil {
		t.Fatalf("SaveVersion: %v", err)
	}

	versions, err := s.ListVersions("/a/todo.md")
	if err != nil {
		t.Fatalf("ListVersions() error: %v", err)
	}
	if len(versions) == 0 {
		t.Fatal("expected at least one version")
	}
	if versions[0].CreatedAt.IsZero() {
		t.Error("CreatedAt is zero; datetime parse failed — check the format returned by the SQLite driver")
	}
}

// --- Test isolation ---

func TestSetStoreDirForTesting_Isolation(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()

	SetStoreDirForTesting(dirA)
	sA, err := Open(0)
	if err != nil {
		t.Fatalf("Open() in dirA: %v", err)
	}
	t.Cleanup(func() {
		if err := sA.Close(); err != nil {
			t.Errorf("close store in dirA: %v", err)
		}
	})
	if want := filepath.Join(dirA, "versions.sqlite"); sA.dbPath != want {
		t.Errorf("sA.dbPath = %q, want %q", sA.dbPath, want)
	}

	SetStoreDirForTesting(dirB)
	sB, err := Open(0)
	if err != nil {
		t.Fatalf("Open() in dirB: %v", err)
	}
	t.Cleanup(func() {
		if err := sB.Close(); err != nil {
			t.Errorf("close store in dirB: %v", err)
		}
	})
	if want := filepath.Join(dirB, "versions.sqlite"); sB.dbPath != want {
		t.Errorf("sB.dbPath = %q, want %q", sB.dbPath, want)
	}

	ResetStoreDirForTesting()
}
