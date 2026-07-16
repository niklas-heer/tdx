package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/niklas-heer/tdx/internal/markdown"
)

func conflictModel(t *testing.T) (Model, string, string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path := filepath.Join(t.TempDir(), "todo.md")
	initial := "# Todos\n\n- [ ] local task\n"
	external := "# Todos\n\n- [ ] external task\n"
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}
	fm, err := markdown.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m := New(path, fm, false, false, -1, testConfig(), testStyles(), "test")
	_ = m.FileModel.UpdateTodoItem(0, "local task", true)
	local := markdown.SerializeMarkdown(&m.FileModel)
	if err := os.WriteFile(path, []byte(external), 0o644); err != nil {
		t.Fatal(err)
	}
	return m, local, external
}

func TestSaveConflictRetainsLocalAndAuthoritativeDiskContent(t *testing.T) {
	m, local, external := conflictModel(t)
	m.writeIfPersist()

	if !errors.Is(m.Err, markdown.ErrFileChanged) {
		t.Fatalf("save error = %v, want ErrFileChanged", m.Err)
	}
	if !m.ConflictDiffMode {
		t.Fatal("conflict diff did not open")
	}
	if m.ConflictLocalContent != local || m.ConflictDiskContent != external {
		t.Fatalf("retained local=%q disk=%q", m.ConflictLocalContent, m.ConflictDiskContent)
	}
	got, _ := os.ReadFile(m.FilePath)
	if string(got) != external {
		t.Fatalf("disk content = %q, want authoritative external content", got)
	}

	rendered := m.View()
	if !strings.Contains(rendered, "FILE CONFLICT") || !strings.Contains(rendered, "external task") {
		t.Fatalf("conflict view missing context:\n%s", rendered)
	}
	result, _ := m.handleConflictDiffKey("esc")
	m = result.(Model)
	if m.ConflictDiffMode || m.ConflictLocalContent != local || m.ConflictDiskContent != external {
		t.Fatal("closing diff did not retain both candidates")
	}
}

func TestConflictReloadAcceptsDisk(t *testing.T) {
	m, _, external := conflictModel(t)
	m.writeIfPersist()
	m.ConflictDiffMode = false
	executeCommand(&m, "reload")

	if m.ConflictLocalContent != "" || m.ConflictDiskContent != "" {
		t.Fatal("reload did not clear conflict")
	}
	if got := markdown.SerializeMarkdown(&m.FileModel); got != external {
		t.Fatalf("reloaded model = %q, want external content", got)
	}
}

func TestConflictForceSaveCommitsRetainedLocalCandidate(t *testing.T) {
	m, local, _ := conflictModel(t)
	m.writeIfPersist()
	m.ConflictDiffMode = false
	executeCommand(&m, "force-save")

	if m.Err != nil {
		t.Fatalf("force-save error = %v", m.Err)
	}
	if m.ConflictLocalContent != "" || m.ConflictDiskContent != "" {
		t.Fatal("force-save did not clear conflict")
	}
	got, err := os.ReadFile(m.FilePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != local {
		t.Fatalf("force-saved content = %q, want retained local candidate", got)
	}
}

func TestVersionRestoreRefusesExternalChange(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path := filepath.Join(t.TempDir(), "todo.md")
	initial := "# Current\n\n- [ ] current\n"
	external := "# External\n\n- [ ] external\n"
	snapshot := "# Historic\n\n- [ ] historic\n"
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}
	fm, err := markdown.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg := testConfig()
	cfg.ReadVersionFunc = func(string, int64) (string, error) { return snapshot, nil }
	m := New(path, fm, false, false, -1, cfg, testStyles(), "test")
	m.VersionsMode = true
	m.VersionsConfirmMode = true
	m.VersionsList = []VersionInfo{{ID: 1}}
	if err := os.WriteFile(path, []byte(external), 0o644); err != nil {
		t.Fatal(err)
	}

	result, _ := m.restoreSelectedVersion()
	restored := result.(Model)
	if !errors.Is(restored.Err, markdown.ErrFileChanged) || !restored.ConflictDiffMode {
		t.Fatalf("restore error=%v diff=%v, want conflict", restored.Err, restored.ConflictDiffMode)
	}
	got, _ := os.ReadFile(path)
	if string(got) != external {
		t.Fatalf("restore overwrote external content: %q", got)
	}
	if restored.ConflictLocalContent != snapshot {
		t.Fatalf("retained restore candidate = %q, want snapshot", restored.ConflictLocalContent)
	}
}
