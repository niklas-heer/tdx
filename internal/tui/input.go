package tui

import (
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
)

// singleLineText keeps pasted input within one field and excludes terminal controls.
func singleLineText(text string) string {
	text, _, _ = strings.Cut(text, "\n")
	text, _, _ = strings.Cut(text, "\r")
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, text)
}

func (m Model) handlePaste(text string) (tea.Model, tea.Cmd) {
	text = singleLineText(text)
	if text == "" {
		return m, nil
	}

	switch {
	case m.HeadingInput != "" || m.InputMode || m.EditMode || m.SearchMode || m.CommandMode:
		m.insertInputText(text)
		if m.SearchMode {
			m.searchPending = true
			return m, searchDebounceCmd()
		}
		if m.CommandMode {
			m.searchPending = true
			return m, commandDebounceCmd()
		}
	case m.RecentFilesMode:
		m.RecentFilesSearch += text
		m.RecentFilesCursor = 0
	}
	return m, nil
}

func (m Model) handleRecentFilesKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.Text != "" {
		text := singleLineText(msg.Text)
		if m.RecentFilesSearch == "" {
			text = strings.TrimLeft(text, " ")
		}
		m.RecentFilesSearch += text
		m.RecentFilesCursor = 0
		return m, nil
	}
	return m.handleRecentFilesInput(msg.String())
}

func (m *Model) insertInputText(text string) {
	text = singleLineText(text)
	m.InputBuffer = m.InputBuffer[:m.CursorPos] + text + m.InputBuffer[m.CursorPos:]
	m.CursorPos += len(text)
}
