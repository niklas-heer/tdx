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

func TestSelectionFlushesPendingQuery(t *testing.T) {
	for _, key := range []rune{tea.KeyEnter, tea.KeyTab} {
		m := testModel([]string{"Alpha", "Target"})
		selected := ""
		m.CommandMode = true
		m.Commands = []Command{
			{Name: "wrong", Handler: func(*Model) { selected = "wrong" }},
			{Name: "target", Handler: func(*Model) { selected = "target" }},
		}
		m.FilteredCmds = []int{0, 1}
		result, _ := m.Update(tea.KeyPressMsg{Text: "target"})
		m = result.(Model)
		// No CommandDebounceMsg is delivered before the selection key.
		result, _ = m.Update(tea.KeyPressMsg{Code: key})
		m = result.(Model)
		if key == tea.KeyEnter && selected != "target" {
			t.Fatalf("executed stale command: %q", selected)
		}
		if key == tea.KeyTab && m.InputBuffer != "target" {
			t.Fatalf("completed stale command: %q", m.InputBuffer)
		}
	}
	m := testModel([]string{"Alpha", "Target"})
	m.SearchMode = true
	m.SearchResults = []int{0, 1}
	result, _ := m.Update(tea.KeyPressMsg{Text: "Target"})
	m = result.(Model)
	result, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = result.(Model)
	if m.SelectedIndex != 1 {
		t.Fatalf("selected stale search result: %d", m.SelectedIndex)
	}
}
