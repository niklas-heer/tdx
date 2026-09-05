package markdown

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteHook_CalledAfterSuccessfulWrite(t *testing.T) {
	store := Store{}
	// Reset hook after test.
	original := store.OnWrite
	defer func() { store.OnWrite = original }()

	var gotPath, gotContent string
	store.OnWrite = func(filePath, content string) error {
		gotPath = filePath
		gotContent = content
		return nil
	}

	dir := t.TempDir()
	mdPath := filepath.Join(dir, "todo.md")

	fm := ParseMarkdown("# Todos\n\n- [ ] hook test\n")
	fm.FilePath = mdPath

	if err := store.WriteFileUnchecked(mdPath, fm); err != nil {
		t.Fatalf("store.WriteFileUnchecked() error: %v", err)
	}

	wantPath, err := resolveTarget(mdPath)
	if err != nil {
		t.Fatalf("resolveTarget() error: %v", err)
	}
	if gotPath != wantPath {
		t.Errorf("store.OnWrite filePath = %q, want %q", gotPath, wantPath)
	}
	if gotContent == "" {
		t.Error("store.OnWrite content was empty")
	}
}

func TestWriteHook_NilHookIsNoOp(t *testing.T) {
	store := Store{}
	original := store.OnWrite
	defer func() { store.OnWrite = original }()
	store.OnWrite = nil

	dir := t.TempDir()
	mdPath := filepath.Join(dir, "todo.md")
	fm := ParseMarkdown("# Todos\n\n- [ ] no hook\n")
	fm.FilePath = mdPath

	// Should not panic when hook is nil.
	if err := store.WriteFileUnchecked(mdPath, fm); err != nil {
		t.Fatalf("store.WriteFileUnchecked() error: %v", err)
	}
}

func TestWriteContentUnchecked_PreservesExactContent(t *testing.T) {
	store := Store{}
	original := store.OnWrite
	defer func() { store.OnWrite = original }()

	var hookedContent string
	store.OnWrite = func(_ string, content string) error {
		hookedContent = content
		return nil
	}

	mdPath := filepath.Join(t.TempDir(), "todo.md")
	content := "---\nfilter-done: true\n---\n\nplain text\n\n- [ ] task\n"
	if err := store.WriteContentUnchecked(mdPath, content); err != nil {
		t.Fatalf("store.WriteContentUnchecked() error: %v", err)
	}

	written, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("os.ReadFile() error: %v", err)
	}
	if string(written) != content {
		t.Fatalf("written content = %q, want %q", written, content)
	}
	if hookedContent != content {
		t.Fatalf("store.OnWrite content = %q, want %q", hookedContent, content)
	}
}

func TestReadHook_CalledAfterSuccessfulRead(t *testing.T) {
	store := Store{}
	original := store.OnRead
	defer func() { store.OnRead = original }()

	var gotPath, gotContent string
	store.OnRead = func(filePath, content string) error {
		gotPath = filePath
		gotContent = content
		return nil
	}

	dir := t.TempDir()
	mdPath := filepath.Join(dir, "todo.md")

	rawContent := "# Todos\n\n- [ ] read hook test\n"
	if err := os.WriteFile(mdPath, []byte(rawContent), 0644); err != nil {
		t.Fatalf("os.WriteFile() error: %v", err)
	}

	if _, err := store.ReadFile(mdPath); err != nil {
		t.Fatalf("store.ReadFile() error: %v", err)
	}

	wantPath, err := resolveTarget(mdPath)
	if err != nil {
		t.Fatalf("resolveTarget() error: %v", err)
	}
	if gotPath != wantPath {
		t.Errorf("store.OnRead filePath = %q, want %q", gotPath, wantPath)
	}
	if gotContent == "" {
		t.Error("store.OnRead content was empty")
	}
}

func TestReadHook_NotCalledForNewFile(t *testing.T) {
	store := Store{}
	original := store.OnRead
	defer func() { store.OnRead = original }()

	called := false
	store.OnRead = func(filePath, content string) error {
		called = true
		return nil
	}

	dir := t.TempDir()
	// File does not exist — ReadFile returns a placeholder FileModel.
	nonExistentPath := filepath.Join(dir, "nonexistent.md")

	if _, err := store.ReadFile(nonExistentPath); err != nil {
		t.Fatalf("store.ReadFile() error: %v", err)
	}

	if called {
		t.Error("store.OnRead should not be called for a new (non-existent) file")
	}
}
