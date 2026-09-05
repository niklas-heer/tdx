package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestStructuredTextAndPaste(t *testing.T) {
	for _, mode := range []string{"input", "search", "command", "heading", "recent"} {
		t.Run(mode, func(t *testing.T) {
			m := testModel([]string{"Task"})
			switch mode {
			case "input":
				m.InputMode = true
			case "search":
				m.SearchMode = true
			case "command":
				m.CommandMode = true
			case "heading":
				m.HeadingInput = "rename"
				m.SectionsMode = true
			case "recent":
				m.RecentFilesMode = true
			}
			result, _ := m.Update(tea.PasteMsg{Content: "enter ä😀\nignored"})
			m = result.(Model)
			got := m.InputBuffer
			if mode == "recent" {
				got = m.RecentFilesSearch
			}
			if got != "enter ä😀" {
				t.Fatalf("paste executed a shortcut or lost text: %q", got)
			}
		})
	}
	m := testModel(nil)
	m.InputMode = true
	result, _ := m.Update(tea.KeyPressMsg{Text: "left"})
	m = result.(Model)
	if m.InputBuffer != "left" {
		t.Fatalf("text event treated as shortcut: %q", m.InputBuffer)
	}
	if strings.Contains(m.View().Content, "ignored") {
		t.Fatal("unexpected second line")
	}
}
