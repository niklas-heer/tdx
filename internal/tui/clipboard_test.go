package tui

import (
	tea "charm.land/bubbletea/v2"
	"errors"
	"testing"
)

type testClipboard struct {
	err  error
	text string
}

func (c testClipboard) Copy(string) error      { return c.err }
func (c testClipboard) Paste() (string, error) { return c.text, c.err }
func TestClipboardTruthfulFeedback(t *testing.T) {
	for _, failure := range []bool{false, true} {
		m := viewModel(t)
		var err error
		if failure {
			err = errors.New("clipboard unavailable")
		}
		m.Config().Clipboard = testClipboard{err: err}
		result, cmd := m.Update(tea.KeyPressMsg{Code: 'c'})
		m = result.(Model)
		if m.CopyFeedback {
			t.Fatal("success shown before copying")
		}
		result, _ = m.Update(cmd())
		m = result.(Model)
		if m.CopyFeedback == failure || (m.Err != nil) != failure {
			t.Fatalf("untruthful feedback: %v %v", m.CopyFeedback, m.Err)
		}
	}
}
func TestClipboardPasteFailureAndStaleResult(t *testing.T) {
	m := viewModel(t)
	m.InputMode = true
	m.InputBuffer = "keep"
	m.CursorPos = 4
	m.Config().Clipboard = testClipboard{err: errors.New("unavailable")}
	result, cmd := m.Update(tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl})
	m = result.(Model)
	result, _ = m.Update(cmd())
	m = result.(Model)
	if m.InputBuffer != "keep" || m.Err == nil {
		t.Fatal("failed paste changed input or hid error")
	}
	m.Err = nil
	m.Config().Clipboard = testClipboard{text: "pasted"}
	result, cmd = m.Update(tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl})
	m = result.(Model)
	m.InputBuffer = "changed"
	m.CursorPos = 7
	result, _ = m.Update(cmd())
	if result.(Model).InputBuffer != "changed" {
		t.Fatal("stale paste changed new input")
	}
}
