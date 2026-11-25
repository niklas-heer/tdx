package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/niklas-heer/tdx/internal/markdown"
)

const twoSections = "# A\n\n- [ ] a1\n- [ ] a2\n\n# B\n\n- [ ] b1\n- [ ] b2\n"

func sectionsModel(t *testing.T) Model {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path := filepath.Join(t.TempDir(), "todo.md")
	if err := os.WriteFile(path, []byte(twoSections), 0o644); err != nil {
		t.Fatal(err)
	}
	fm, err := markdown.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return New(path, fm, false, true, -1, testConfig(), testStyles(), "test")
}

func press(t *testing.T, m Model, msg tea.KeyPressMsg) Model {
	t.Helper()
	result, _ := m.Update(msg)
	return result.(Model)
}

func letter(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

// Regression: undo always cleared section focus and folds, so undoing a
// checkbox toggle inside a focused section jumped back to all tasks.
func TestUndoKeepsSectionsWhenHeadingsAreUnchanged(t *testing.T) {
	t.Run("focus", func(t *testing.T) {
		m := sectionsModel(t)
		m.focusSection(1)
		m = press(t, m, letter('j'))
		m = press(t, m, tea.KeyPressMsg{Code: tea.KeySpace})
		m = press(t, m, letter('u'))
		if m.Err != nil || m.FileModel.Todos[3].Checked {
			t.Fatalf("undo failed: err=%v todos=%+v", m.Err, m.FileModel.Todos)
		}
		if m.SectionFocus != 2 {
			t.Fatalf("section focus = %d after undo, want 2", m.SectionFocus)
		}
		if m.SelectedIndex != 3 {
			t.Fatalf("selection = %d after undo, want the undone task 3", m.SelectedIndex)
		}
	})
	t.Run("folds", func(t *testing.T) {
		m := sectionsModel(t)
		m.FoldedSections = map[int]bool{0: true}
		m.adjustSelectionForFilter()
		m = press(t, m, letter('d'))
		m = press(t, m, letter('u'))
		if len(m.FileModel.Todos) != 4 || !m.FoldedSections[0] {
			t.Fatalf("after undo: %d todos, folds %v", len(m.FileModel.Todos), m.FoldedSections)
		}
		if !m.isTodoVisible(m.SelectedIndex) {
			t.Fatalf("selection %d is hidden by the fold", m.SelectedIndex)
		}
	})
}

func TestUndoOfHeadingChangeClearsSections(t *testing.T) {
	m := sectionsModel(t)
	m.focusSection(1)
	m = press(t, m, letter('s'))
	m = press(t, m, letter('n'))
	m = press(t, m, letter('C'))
	m = press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.Err != nil || len(m.GetHeadings()) != 3 {
		t.Fatalf("heading not added: err=%v headings=%v", m.Err, m.GetHeadings())
	}
	m = press(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	m.focusSection(1)
	m = press(t, m, letter('u'))
	if len(m.GetHeadings()) != 2 {
		t.Fatalf("undo kept the new heading: %v", m.GetHeadings())
	}
	if m.SectionFocus != 0 || m.FoldedSections != nil {
		t.Fatalf("focus %d, folds %v after undoing a heading change; want cleared", m.SectionFocus, m.FoldedSections)
	}
}
