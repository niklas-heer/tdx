package markdown

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteHook_CalledAfterSuccessfulWrite(t *testing.T) {
	// Reset hook after test.
	original := WriteHook
	defer func() { WriteHook = original }()

	var gotPath, gotContent string
	WriteHook = func(filePath, content string) error {
		gotPath = filePath
		gotContent = content
		return nil
	}

	dir := t.TempDir()
	mdPath := filepath.Join(dir, "todo.md")

	fm := ParseMarkdown("# Todos\n\n- [ ] hook test\n")
	fm.FilePath = mdPath

	if err := WriteFileUnchecked(mdPath, fm); err != nil {
		t.Fatalf("WriteFileUnchecked() error: %v", err)
	}

	if gotPath != mdPath {
		t.Errorf("WriteHook filePath = %q, want %q", gotPath, mdPath)
	}
	if gotContent == "" {
		t.Error("WriteHook content was empty")
	}
}

func TestWriteHook_NilHookIsNoOp(t *testing.T) {
	original := WriteHook
	defer func() { WriteHook = original }()
	WriteHook = nil

	dir := t.TempDir()
	mdPath := filepath.Join(dir, "todo.md")
	fm := ParseMarkdown("# Todos\n\n- [ ] no hook\n")
	fm.FilePath = mdPath

	// Should not panic when hook is nil.
	if err := WriteFileUnchecked(mdPath, fm); err != nil {
		t.Fatalf("WriteFileUnchecked() error: %v", err)
	}
}

func TestWriteContentUnchecked_PreservesExactContent(t *testing.T) {
	original := WriteHook
	defer func() { WriteHook = original }()

	var hookedContent string
	WriteHook = func(_ string, content string) error {
		hookedContent = content
		return nil
	}

	mdPath := filepath.Join(t.TempDir(), "todo.md")
	content := "---\nfilter-done: true\n---\n\nplain text\n\n- [ ] task\n"
	if err := WriteContentUnchecked(mdPath, content); err != nil {
		t.Fatalf("WriteContentUnchecked() error: %v", err)
	}

	written, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("os.ReadFile() error: %v", err)
	}
	if string(written) != content {
		t.Fatalf("written content = %q, want %q", written, content)
	}
	if hookedContent != content {
		t.Fatalf("WriteHook content = %q, want %q", hookedContent, content)
	}
}

func TestReadHook_CalledAfterSuccessfulRead(t *testing.T) {
	original := ReadHook
	defer func() { ReadHook = original }()

	var gotPath, gotContent string
	ReadHook = func(filePath, content string) error {
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

	if _, err := ReadFile(mdPath); err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	if gotPath != mdPath {
		t.Errorf("ReadHook filePath = %q, want %q", gotPath, mdPath)
	}
	if gotContent == "" {
		t.Error("ReadHook content was empty")
	}
}

func TestReadHook_NotCalledForNewFile(t *testing.T) {
	original := ReadHook
	defer func() { ReadHook = original }()

	called := false
	ReadHook = func(filePath, content string) error {
		called = true
		return nil
	}

	dir := t.TempDir()
	// File does not exist — ReadFile returns a placeholder FileModel.
	nonExistentPath := filepath.Join(dir, "nonexistent.md")

	if _, err := ReadFile(nonExistentPath); err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	if called {
		t.Error("ReadHook should not be called for a new (non-existent) file")
	}
}
