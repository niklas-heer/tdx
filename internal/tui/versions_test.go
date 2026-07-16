package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/niklas-heer/tdx/internal/markdown"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// stubVersions returns a fixed list of VersionInfo for testing.
func stubVersions() []VersionInfo {
	return []VersionInfo{
		{ID: 3, CreatedAt: time.Date(2026, 1, 3, 10, 0, 0, 0, time.UTC)},
		{ID: 2, CreatedAt: time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)},
		{ID: 1, CreatedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)},
	}
}

// testVersionModel creates a model with the versions browser open and a stub ListVersionsFunc.
func testVersionModel() Model {
	cfg := testConfig()
	cfg.ListVersionsFunc = func(filePath string) ([]VersionInfo, error) {
		return stubVersions(), nil
	}
	cfg.ReadVersionFunc = func(filePath string, id int64) (string, error) {
		return "# Version content\n\n- [ ] task\n", nil
	}

	fm := &markdown.FileModel{
		Todos: []markdown.Todo{{Text: "existing task", Checked: false}},
	}
	m := New("/tmp/test.md", fm, false, false, -1, cfg, testStyles(), "test")
	m.VersionsList = stubVersions()
	m.VersionsMode = true
	m.VersionsCursor = 0
	m.VersionsDiffScroll = 0
	m.VersionsConfirmMode = false
	m.TermHeight = 40
	m.TermWidth = 120
	return m
}

func sendVersionsKey(m Model, key string) Model {
	result, _ := m.handleVersionsKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	if newModel, ok := result.(Model); ok {
		return newModel
	}
	return m
}

func sendVersionsKeyType(m Model, keyType tea.KeyType) Model {
	result, _ := m.handleVersionsKey(tea.KeyMsg{Type: keyType})
	if newModel, ok := result.(Model); ok {
		return newModel
	}
	return m
}

// TestHandleVersionsKey_DownIncrementsCursorAndResetsScroll verifies j/↓ behaviour.
func TestHandleVersionsKey_DownIncrementsCursorAndResetsScroll(t *testing.T) {
	m := testVersionModel()
	m.VersionsDiffScroll = 5 // pre-set scroll to verify reset

	m = sendVersionsKey(m, "j")

	if m.VersionsCursor != 1 {
		t.Errorf("expected VersionsCursor = 1, got %d", m.VersionsCursor)
	}
	if m.VersionsDiffScroll != 0 {
		t.Errorf("expected VersionsDiffScroll reset to 0, got %d", m.VersionsDiffScroll)
	}
}

// TestHandleVersionsKey_EscClosesVersionsBrowser verifies Esc sets VersionsMode = false.
func TestHandleVersionsKey_EscClosesVersionsBrowser(t *testing.T) {
	m := testVersionModel()

	m = sendVersionsKeyType(m, tea.KeyEsc)

	if m.VersionsMode {
		t.Error("expected VersionsMode = false after Esc")
	}
}

// TestHandleVersionsKey_EnterSetsConfirmMode verifies Enter enables VersionsConfirmMode.
func TestHandleVersionsKey_EnterSetsConfirmMode(t *testing.T) {
	m := testVersionModel()

	m = sendVersionsKeyType(m, tea.KeyEnter)

	if !m.VersionsConfirmMode {
		t.Error("expected VersionsConfirmMode = true after Enter")
	}
	if !m.VersionsMode {
		t.Error("expected VersionsMode = true after Enter")
	}
}

// TestHandleVersionsKey_NInConfirmModeCancels verifies 'n' cancels confirm but keeps browser open.
func TestHandleVersionsKey_NInConfirmModeCancels(t *testing.T) {
	m := testVersionModel()
	m.VersionsConfirmMode = true

	m = sendVersionsKey(m, "n")

	if m.VersionsConfirmMode {
		t.Error("expected VersionsConfirmMode = false after 'n'")
	}
	if !m.VersionsMode {
		t.Error("expected VersionsMode = true after cancelling confirm")
	}
}

// TestHandleVersionsKey_UpDoesNotGoNegative verifies cursor clamp at top.
func TestHandleVersionsKey_UpDoesNotGoNegative(t *testing.T) {
	m := testVersionModel()
	m.VersionsCursor = 0

	m = sendVersionsKey(m, "k")

	if m.VersionsCursor != 0 {
		t.Errorf("expected VersionsCursor = 0 at top, got %d", m.VersionsCursor)
	}
}

// TestHandleVersionsKey_DownDoesNotExceedList verifies cursor clamp at bottom.
func TestHandleVersionsKey_DownDoesNotExceedList(t *testing.T) {
	m := testVersionModel()
	m.VersionsCursor = len(m.VersionsList) - 1

	m = sendVersionsKey(m, "j")

	if m.VersionsCursor != len(m.VersionsList)-1 {
		t.Errorf("expected cursor to stay at last index, got %d", m.VersionsCursor)
	}
}

// TestRenderDiff_NoSpuriousPaddingOnMultiLineSegments ensures that Equal/Insert/Delete
// diff segments whose text contains newlines are styled line-by-line. If a multi-line
// string is passed to a lipgloss style function it pads every line to the longest
// line's width, inserting spurious spaces that corrupt diff alignment.
func TestRenderDiff_NoSpuriousPaddingOnMultiLineSegments(t *testing.T) {
	styles := testStyles() // identity funcs — no ANSI codes, exposes padding as spaces

	diffs := []diffmatchpatch.Diff{
		{Type: diffmatchpatch.DiffEqual, Text: "long line here\nshort"},
		{Type: diffmatchpatch.DiffDelete, Text: "x"},
	}

	rendered := renderDiff(diffs, styles)
	lines := strings.Split(rendered, "\n")

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	// "short" (5 chars) must NOT be padded to "long line here" width (14 chars).
	if lines[1] != "shortx" {
		t.Errorf("line[1] = %q; want %q (spurious padding detected)", lines[1], "shortx")
	}
}

func TestRestoreSelectedVersion_PreservesSnapshotBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todo.md")
	snapshot := "---\nfilter-done: true\nword-wrap: false\nshow-headings: true\nread-only: true\nmax-visible: 7\n---\n\nplain text\n\n- [ ] task\n"
	cfg := testConfig()
	cfg.ReadVersionFunc = func(string, int64) (string, error) {
		return snapshot, nil
	}

	m := New(path, markdown.ParseMarkdown("# Current\n\n- [ ] current\n"), false, false, -1, cfg, testStyles(), "test")
	m.VersionsMode = true
	m.VersionsConfirmMode = true
	m.VersionsList = []VersionInfo{{ID: 1}}

	result, _ := m.restoreSelectedVersion()
	restored := result.(Model)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error: %v", err)
	}
	if string(content) != snapshot {
		t.Fatalf("restored content = %q, want %q", content, snapshot)
	}
	if restored.FileModel.Metadata.FilterDone == nil || !*restored.FileModel.Metadata.FilterDone {
		t.Fatal("restored frontmatter was not reloaded into the model")
	}
	if !restored.FilterDone || restored.WordWrap || !restored.ShowHeadings || !restored.ReadOnly {
		t.Fatal("restored frontmatter was not applied to the TUI state")
	}
	if restored.MaxVisibleOverride != 7 {
		t.Fatalf("MaxVisibleOverride = %d, want 7", restored.MaxVisibleOverride)
	}
}

func TestRenderVersionsBrowser_KeepsCursorVisible(t *testing.T) {
	m := testVersionModel()
	m.TermHeight = 20
	m.VersionsList = make([]VersionInfo, 30)
	for i := range m.VersionsList {
		m.VersionsList[i] = VersionInfo{
			ID:        int64(i + 1),
			CreatedAt: time.Date(2026, 1, i+1, 10, 0, 0, 0, time.UTC),
		}
	}
	m.VersionsCursor = 20

	rendered := m.renderVersionsBrowser()
	if !strings.Contains(rendered, fmt.Sprintf("#%03d", m.VersionsList[m.VersionsCursor].ID)) {
		t.Fatal("selected version is outside the rendered list viewport")
	}
}
