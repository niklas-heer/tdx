package tui

import (
	"errors"
	"github.com/charmbracelet/x/ansi"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/niklas-heer/tdx/internal/config"
	"github.com/niklas-heer/tdx/internal/editor"
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
	if !m.ConflictPending {
		t.Fatal("conflict was not marked pending")
	}
	if m.ConflictLocalContent != local || m.ConflictDiskContent != external {
		t.Fatalf("retained local=%q disk=%q", m.ConflictLocalContent, m.ConflictDiskContent)
	}
	got, _ := os.ReadFile(m.FilePath)
	if string(got) != external {
		t.Fatalf("disk content = %q, want authoritative external content", got)
	}

	rendered := ansi.Strip(m.View().Content)
	if !strings.Contains(rendered, "FILE CONFLICT") || !strings.Contains(rendered, "external task") {
		t.Fatalf("conflict view missing context:\n%s", rendered)
	}
	result, _ := m.handleConflictDiffKey("esc")
	m = result.(Model)
	if m.ConflictDiffMode || m.ConflictLocalContent != local || m.ConflictDiskContent != external {
		t.Fatal("closing diff did not retain both candidates")
	}
}

func TestEmptyConflictCandidateCanBeInspectedAndForceSaved(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path := filepath.Join(t.TempDir(), "todo.md")
	external := "# external\n"
	if err := os.WriteFile(path, []byte(external), 0o644); err != nil {
		t.Fatal(err)
	}
	fm, err := markdown.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m := New(path, fm, false, false, -1, testConfig(), testStyles(), "test")
	m.recordSaveError(&markdown.ConflictError{Path: path, DiskContent: external}, "")

	if !m.ConflictPending || !m.ConflictDiffMode {
		t.Fatal("empty local candidate was not retained as a pending conflict")
	}
	m.ConflictDiffMode = false
	executeCommand(&m, "diff")
	if !errors.Is(m.Err, markdown.ErrFileChanged) || !m.ConflictDiffMode {
		t.Fatalf("diff rejected empty candidate: mode=%v err=%v", m.ConflictDiffMode, m.Err)
	}
	m.ConflictDiffMode = false
	executeCommand(&m, "force-save")
	if m.Err != nil || m.ConflictPending {
		t.Fatalf("force-save did not resolve empty candidate: pending=%v err=%v", m.ConflictPending, m.Err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("force-saved content = %q, want empty", got)
	}
}

func TestExternalReloadRetainsReadHookErrorAndDiskModel(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path := filepath.Join(t.TempDir(), "todo.md")
	initial := "# Todos\n\n- [ ] initial\n"
	external := "# Todos\n\n- [ ] external\n"
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}
	fm, err := markdown.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m := New(path, fm, false, false, -1, testConfig(), testStyles(), "test")
	if err := os.WriteFile(path, []byte(external), 0o644); err != nil {
		t.Fatal(err)
	}

	originalHook := m.Config().Store.OnRead
	hookErr := errors.New("version capture failed")
	m.Config().Store.OnRead = func(string, string) error { return hookErr }
	t.Cleanup(func() { m.Config().Store.OnRead = originalHook })

	m.reloadIfChanged()
	if !errors.Is(m.Err, hookErr) {
		t.Fatalf("reload error = %v, want hook error", m.Err)
	}
	if got := markdown.SerializeMarkdown(&m.FileModel); got != external {
		t.Fatalf("reloaded model = %q, want authoritative disk content", got)
	}
}

func TestConflictDiffScrollIsClampedAfterKeysAndResize(t *testing.T) {
	m := testModel(nil)
	m.TermWidth = 80
	m.TermHeight = 10
	m.ConflictDiffMode = true
	m.ConflictPending = true
	m.ConflictLocalContent = strings.Repeat("local line\n", 30)
	m.ConflictDiskContent = strings.Repeat("disk line\n", 30)

	for range 100 {
		updated, _ := m.handleConflictDiffKey("down")
		m = updated.(Model)
	}
	maxScroll := len(m.conflictDiffLines()) - m.conflictDiffHeight()
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.ConflictDiffScroll != maxScroll {
		t.Fatalf("scroll = %d, want maximum %d", m.ConflictDiffScroll, maxScroll)
	}
	previous := m.ConflictDiffScroll
	updated, _ := m.handleConflictDiffKey("up")
	m = updated.(Model)
	if previous > 0 && m.ConflictDiffScroll >= previous {
		t.Fatalf("up did not move immediately from maximum: before=%d after=%d", previous, m.ConflictDiffScroll)
	}

	m.ConflictDiffScroll = 1_000
	updated, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = updated.(Model)
	resizedMax := len(m.conflictDiffLines()) - m.conflictDiffHeight()
	if resizedMax < 0 {
		resizedMax = 0
	}
	if m.ConflictDiffScroll > resizedMax {
		t.Fatalf("resized scroll = %d, maximum = %d", m.ConflictDiffScroll, resizedMax)
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

// Regression: switching files with a conflict pending kept the conflict's
// local candidate, and :force-save then wrote it into the other file.
func TestRecentFileSwitchRefusedWhileConflictPending(t *testing.T) {
	m, local, external := conflictModel(t)
	m.writeIfPersist()
	m.ConflictDiffMode = false
	if !m.ConflictPending {
		t.Fatal("setup: expected a pending conflict")
	}
	m.Err = nil // the user dismissed the conflict message
	other := filepath.Join(t.TempDir(), "other.md")
	const otherText = "# Other\n\n- [ ] keep me\n"
	if err := os.WriteFile(other, []byte(otherText), 0o644); err != nil {
		t.Fatal(err)
	}
	source := m.FilePath
	m.RecentFiles = []config.RecentFile{{Path: other}}
	m.RecentFilesMode = true
	m = viewKey(m, tea.KeyEnter)
	if m.FilePath != source || m.Err == nil || !strings.Contains(m.Err.Error(), "conflict") {
		t.Fatalf("switch with a pending conflict: path %q, err %v", m.FilePath, m.Err)
	}
	executeCommand(&m, "force-save")
	if got, _ := os.ReadFile(other); string(got) != otherText {
		t.Fatalf("other file was overwritten: %q", got)
	}
	if got, _ := os.ReadFile(source); string(got) != local {
		t.Fatalf("force-save should still resolve the original file: got %q, external was %q", got, external)
	}
}

// reloads reports whether the watcher replaced the document.
func reloads(m *Model) bool {
	before := markdown.SerializeMarkdown(&m.FileModel)
	m.reloadIfChanged()
	return markdown.SerializeMarkdown(&m.FileModel) != before
}

func manualSaveModel(t *testing.T, initial string) (Model, string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path := filepath.Join(t.TempDir(), "todo.md")
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}
	fm, err := markdown.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return New(path, fm, true, false, -1, testConfig(), testStyles(), "test"), path
}

// Regression: in manual-save mode the watcher replaced unsaved edits with the
// disk version, and :save then wrote over the external change unannounced.
func TestManualSaveKeepsUnsavedEditsWhenFileChanges(t *testing.T) {
	const initial = "---\nread-only: true\n---\n# Todos\n\n- [ ] a\n- [ ] b\n"
	const external = "# Todos\n\n- [ ] a\n- [ ] b\n- [ ] added elsewhere\n"

	t.Run("unedited document follows the file", func(t *testing.T) {
		m, path := manualSaveModel(t, initial)
		if err := os.WriteFile(path, []byte(external), 0o644); err != nil {
			t.Fatal(err)
		}
		// The reload is applied in the update that notices the change; no later
		// message carries an older model that could undo keys handled meanwhile.
		result, _ := m.Update(FileChangedMsg{})
		m = result.(Model)
		if got := markdown.SerializeMarkdown(&m.FileModel); got != external {
			t.Fatalf("an unedited manual-save document should reload external changes, got %q", got)
		}
	})

	t.Run("unsaved edits are kept and :save reports the conflict", func(t *testing.T) {
		m, path := manualSaveModel(t, initial)
		if err := m.applyAction(editor.Action{Kind: editor.Toggle, Index: 0}); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(external), 0o644); err != nil {
			t.Fatal(err)
		}
		if reloads(&m) {
			t.Fatal("watcher discarded unsaved manual-save edits")
		}
		executeCommand(&m, "save")
		if !m.ConflictPending {
			t.Fatal(":save should report the external change as a conflict")
		}
		if got, _ := os.ReadFile(path); string(got) != external {
			t.Fatalf(":save overwrote the external change: %q", got)
		}
	})
}

// Regression: reloading a shorter file while / search was open left search
// results pointing past the new task list, and rendering panicked.
func TestOpenSearchDefersReload(t *testing.T) {
	m, path := manualSaveModel(t, "# Todos\n\n- [ ] alpha\n- [ ] alpine\n- [ ] alps\n")
	m.ReadOnly = false
	m.SearchMode, m.InputBuffer = true, "al"
	m.updateSearchResults()
	if err := os.WriteFile(path, []byte("# Todos\n\n- [ ] alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if reloads(&m) {
		t.Fatal("reload while searching would leave stale search results")
	}
	_ = m.View()
}
