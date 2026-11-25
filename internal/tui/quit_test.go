package tui

import (
	"os"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func quits(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

// seen backdates the armed quit past the confirmation delay, as if the user had
// read the warning before pressing the next key.
func seen(m Model) Model {
	m.quitArmedAt = m.quitArmedAt.Add(-time.Second)
	return m
}

func pressForQuit(t *testing.T, m Model, msg tea.KeyPressMsg) (Model, bool) {
	t.Helper()
	result, cmd := m.Update(msg)
	return result.(Model), quits(cmd)
}

var (
	escKey   = tea.KeyPressMsg{Code: tea.KeyEscape}
	ctrlCKey = tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	ctrlDKey = tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl}
)

func TestQuitWithoutChangesIsImmediate(t *testing.T) {
	m, _ := manualSaveModel(t, "# Todos\n\n- [ ] a\n")
	for _, key := range []tea.KeyPressMsg{escKey, ctrlCKey, ctrlDKey} {
		if _, quit := pressForQuit(t, m, key); !quit {
			t.Fatalf("%s did not quit a document without changes", key)
		}
	}
}

func TestQuitWithUnsavedChangesWarnsFirst(t *testing.T) {
	for _, key := range []tea.KeyPressMsg{escKey, ctrlCKey, ctrlDKey} {
		t.Run(key.String(), func(t *testing.T) {
			m, path := manualSaveModel(t, "# Todos\n\n- [ ] a\n")
			m, _ = pressForQuit(t, m, tea.KeyPressMsg{Code: tea.KeySpace})
			if !m.FileModel.HasUnsavedChanges() {
				t.Fatal("toggle in manual-save mode left no unsaved change")
			}

			m, quit := pressForQuit(t, m, key)
			if quit || m.Err == nil || !strings.Contains(m.Err.Error(), "unsaved changes") {
				t.Fatalf("first %s: quit=%v err=%v, want a warning", key, quit, m.Err)
			}
			// Any other key stays and dismisses the warning.
			m, quit = pressForQuit(t, m, tea.KeyPressMsg{Text: "j", Code: 'j'})
			if quit || m.Err != nil {
				t.Fatalf("j after the warning: quit=%v err=%v", quit, m.Err)
			}
			m, quit = pressForQuit(t, m, key)
			if quit {
				t.Fatal("quit without a fresh warning after staying")
			}
			if _, quit = pressForQuit(t, seen(m), key); !quit {
				t.Fatalf("second %s did not quit", key)
			}
			if got, _ := os.ReadFile(path); string(got) != "# Todos\n\n- [ ] a\n" {
				t.Fatalf("quitting wrote the file: %q", got)
			}
		})
	}
}

// Regression: Ctrl+D armed the quit from Help, whose full-screen view hid the
// warning, so the next Esc quit without the user having seen it.
func TestQuitWarningIsVisibleOverFullScreenViews(t *testing.T) {
	m, _ := manualSaveModel(t, "# Todos\n\n- [ ] a\n")
	m.TermWidth, m.TermHeight = 80, 12
	m, _ = pressForQuit(t, m, tea.KeyPressMsg{Code: tea.KeySpace})
	m, _ = pressForQuit(t, m, tea.KeyPressMsg{Text: "?", Code: '?'})
	if !m.HelpMode {
		t.Fatal("help did not open")
	}
	m, quit := pressForQuit(t, m, ctrlDKey)
	if quit || !strings.Contains(m.View().Content, "unsaved changes") {
		t.Fatalf("Ctrl+D in help: quit=%v, warning shown=%v", quit, strings.Contains(m.View().Content, "unsaved changes"))
	}
	if _, quit = pressForQuit(t, seen(m), escKey); !quit {
		t.Fatal("Esc after the visible warning did not quit")
	}
}

// Regression: the armed quit outlived its warning when another message cleared
// the error, so a later Esc quit without a visible warning.
func TestQuitDisarmsWhenTheWarningIsReplaced(t *testing.T) {
	m, _ := manualSaveModel(t, "# Todos\n\n- [ ] a\n")
	m.TermWidth, m.TermHeight = 80, 12
	m, _ = pressForQuit(t, m, tea.KeyPressMsg{Code: tea.KeySpace})
	m, _ = pressForQuit(t, m, escKey)
	if !strings.Contains(m.View().Content, "esc again quits without saving") {
		t.Fatalf("warning not fully visible at 80 columns:\n%s", m.View().Content)
	}
	result, _ := m.Update(clipboardCopiedMsg{})
	m = result.(Model)
	if m.Err != nil {
		t.Fatalf("clipboard result left error %v", m.Err)
	}
	if m, quit := pressForQuit(t, m, escKey); quit || m.Err == nil {
		t.Fatalf("Esc after the warning was cleared: quit=%v err=%v, want a fresh warning", quit, m.Err)
	}
}

// Regression: the command palette and short terminals hid the warning, so the
// next Esc quit without the user having seen it.
func TestQuitWarningIsVisibleOverOverlaysAndInShortTerminals(t *testing.T) {
	for _, tc := range []struct {
		name          string
		width, height int
		want          string
		open          []tea.KeyPressMsg
	}{
		{"command palette", 40, 12, "esc again quits", []tea.KeyPressMsg{{Text: ":", Code: ':'}}},
		{"six-row terminal", 40, 6, "esc again quits", nil},
		{"two-row narrow terminal", 20, 2, "unsaved", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, _ := manualSaveModel(t, "# Todos\n\n- [ ] a\n- [ ] b\n- [ ] c\n- [ ] d\n- [ ] e\n")
			m.TermWidth, m.TermHeight = tc.width, tc.height
			m, _ = pressForQuit(t, m, tea.KeyPressMsg{Code: tea.KeySpace})
			for _, key := range tc.open {
				m, _ = pressForQuit(t, m, key)
			}
			m, quit := pressForQuit(t, m, ctrlDKey)
			view := m.View().Content
			if quit || !strings.Contains(view, tc.want) {
				t.Fatalf("Ctrl+D: quit=%v, view:\n%s", quit, view)
			}
			if _, quit = pressForQuit(t, seen(m), escKey); !quit {
				t.Fatal("Esc after the visible warning did not quit")
			}
		})
	}
}

// Regression: in piped input the quit warning swallowed the next scripted key.
func TestPipedInputHasNoQuitConfirmation(t *testing.T) {
	m, _ := manualSaveModel(t, "# Todos\n\n- [ ] a\n")
	m.ProcessPipedInput([]byte{' ', 4, ' '})
	if m.Err != nil || m.FileModel.Todos[0].Checked {
		t.Fatalf("err=%v checked=%v; want both toggles applied and no warning", m.Err, m.FileModel.Todos[0].Checked)
	}
}

// Regression: two quit keys in one terminal read, from a double press or key
// repeat, quit before the warning could be read.
func TestQuitKeyArrivingWithTheFirstDoesNotConfirm(t *testing.T) {
	m, _ := manualSaveModel(t, "# Todos\n\n- [ ] a\n")
	m, _ = pressForQuit(t, m, tea.KeyPressMsg{Code: tea.KeySpace})
	m, _ = pressForQuit(t, m, ctrlDKey)
	m, quit := pressForQuit(t, m, ctrlDKey)
	if quit || !m.quitPending() {
		t.Fatalf("immediate second Ctrl+D: quit=%v pending=%v; want the warning kept", quit, m.quitPending())
	}
	if _, quit = pressForQuit(t, seen(m), ctrlDKey); !quit {
		t.Fatal("Ctrl+D after the warning was shown did not quit")
	}
}

// Regression: holding Ctrl+D confirmed the quit with the first key repeat.
func TestQuitKeyRepeatDoesNotConfirm(t *testing.T) {
	m, _ := manualSaveModel(t, "# Todos\n\n- [ ] a\n")
	m, _ = pressForQuit(t, m, tea.KeyPressMsg{Code: tea.KeySpace})
	m, _ = pressForQuit(t, m, ctrlDKey)
	repeat := ctrlDKey
	repeat.IsRepeat = true
	if m, quit := pressForQuit(t, seen(m), repeat); quit || !m.quitPending() {
		t.Fatalf("held Ctrl+D: quit=%v pending=%v; want the warning kept", quit, m.quitPending())
	}
}
