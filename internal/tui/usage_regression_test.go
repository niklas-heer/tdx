package tui

import (
	tea "charm.land/bubbletea/v2"
	"fmt"
	"github.com/niklas-heer/tdx/internal/markdown"
	"os"
	"path/filepath"
	"testing"
)

func TestCancelKeepsFullUndoHistory(t *testing.T) {
	for _, mode := range []rune{'e', 'n', 'm'} {
		t.Run(string(mode), func(t *testing.T) {
			m := New("", markdown.ParseMarkdown("# Work\n\n- [ ] original\n"), true, false, -1, testConfig(), testStyles(), "")
			key := func(code rune) { next, _ := m.Update(tea.KeyPressMsg{Code: code}); m = next.(Model) }
			for i := 0; i < 100; i++ {
				m.saveHistory()
				if err := m.FileModel.UpdateTodoItem(0, fmt.Sprint(i), false); err != nil {
					t.Fatal(err)
				}
			}
			key(mode)
			key(tea.KeyEscape)
			if m.history.Len() != 100 {
				t.Fatalf("cancel reduced history to %d", m.history.Len())
			}
			for i := 0; i < 100; i++ {
				key('u')
			}
			if m.FileModel.Todos[0].Text != "original" {
				t.Fatal("oldest undo entry was lost")
			}
		})
	}
}

func TestUndoRefreshesPickers(t *testing.T) {
	m := New("", markdown.ParseMarkdown("# Work\n\n- [ ] first #old !p2\n"), true, false, -1, testConfig(), testStyles(), "")
	send := func(msg tea.Msg) { next, _ := m.Update(msg); m = next.(Model) }
	send(tea.KeyPressMsg{Code: 'N'})
	send(tea.PasteMsg{Content: "new #new !p1"})
	send(tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(m.AvailablePriorities) != 2 {
		t.Fatal("new priority missing from picker")
	}
	send(tea.KeyPressMsg{Code: 'u'})
	if len(m.AvailableTags) != 1 || m.AvailableTags[0] != "old" || len(m.AvailablePriorities) != 1 || m.AvailablePriorities[0] != 2 {
		t.Fatal("undo left stale picker values")
	}
}

func TestExternalChangeDuringInputRetainsRevision(t *testing.T) {
	for _, mode := range []rune{'e', 'n', 'm'} {
		t.Run(string(mode), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "tasks.md")
			original := "# Work\n\n- [ ] original\n"
			external := "# Work\n\n- [ ] external\n"
			if err := os.WriteFile(path, []byte(original), 0600); err != nil {
				t.Fatal(err)
			}
			fm, err := markdown.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			m := New(path, fm, false, false, -1, testConfig(), testStyles(), "")
			send := func(msg tea.Msg) { next, _ := m.Update(msg); m = next.(Model) }
			send(tea.KeyPressMsg{Code: mode})
			if err = os.WriteFile(path, []byte(external), 0600); err != nil {
				t.Fatal(err)
			}
			next, watch := m.Update(FileChangedMsg{})
			m = next.(Model)
			if watch != nil {
				send(watch())
			} // Execute one event, never recursively drain timers.
			if m.FileModel.Todos[0].Text != "original" {
				t.Fatal("watcher replaced document during pending edit")
			}
			if mode != 'm' {
				send(tea.PasteMsg{Content: " locally edited"})
			}
			send(tea.KeyPressMsg{Code: tea.KeyEnter})
			if !m.ConflictPending {
				t.Fatal("pending edit used a refreshed revision and bypassed conflict detection")
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != external {
				t.Fatal("external content overwritten", err)
			}
		})
	}
}
