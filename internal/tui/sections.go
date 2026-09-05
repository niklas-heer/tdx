package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/niklas-heer/tdx/internal/markdown"
)

// sectionBounds includes a heading's descendants, up to the next peer or ancestor.
func sectionBounds(headings []markdown.Heading, index, total int) (int, int) {
	if index < 0 || index >= len(headings) {
		return 0, 0
	}
	end := total
	for _, h := range headings[index+1:] {
		if h.Level <= headings[index].Level {
			end = h.BeforeTodoIndex
			break
		}
	}
	return headings[index].BeforeTodoIndex, end
}

func (m *Model) sectionAllowsTodo(index int) bool {
	headings := m.GetHeadings()
	if m.SectionFocus > 0 {
		start, end := sectionBounds(headings, m.SectionFocus-1, len(m.FileModel.Todos))
		if index < start || index >= end {
			return false
		}
	}
	for section, folded := range m.FoldedSections {
		if !folded {
			continue
		}
		start, end := sectionBounds(headings, section, len(m.FileModel.Todos))
		if index >= start && index < end {
			return false
		}
	}
	return true
}

func (m *Model) openSections() {
	m.SectionsMode = true
	m.SectionCursor = 0
	for i, h := range m.GetHeadings() {
		if h.BeforeTodoIndex <= m.SelectedIndex {
			m.SectionCursor = i
		}
	}
	if m.SectionFocus > 0 {
		m.SectionCursor = m.SectionFocus - 1
	}
}

func (m *Model) clearSections() {
	m.SectionFocus = 0
	m.FoldedSections = nil
	m.InvalidateDocumentTree()
	m.adjustSelectionForFilter()
}

func (m *Model) focusSection(index int) {
	if index < 0 || index >= len(m.GetHeadings()) {
		return
	}
	m.SectionFocus = index + 1
	m.FoldedSections = nil
	m.ShowHeadings = true
	m.SectionsMode = false
	m.InvalidateDocumentTree()
	m.adjustSelectionForFilter()
}

func (m *Model) startHeadingInput(mode string) {
	if m.ReadOnly {
		m.Err = fmt.Errorf("read-only file: section editing is disabled")
		return
	}
	headings := m.GetHeadings()
	if mode == "rename" && len(headings) == 0 {
		return
	}
	if mode == "child" && len(headings) > 0 && headings[m.SectionCursor].Level == 6 {
		m.Err = fmt.Errorf("level 6 headings cannot have a subsection")
		return
	}
	m.HeadingInput = mode
	m.InputBuffer = ""
	if mode == "rename" {
		m.InputBuffer = headings[m.SectionCursor].Text
	}
	m.CursorPos = len(m.InputBuffer)
}

func (m Model) handleSectionsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	headings := m.GetHeadings()
	switch msg.String() {
	case "esc", "s":
		m.SectionsMode = false
	case "down", "j":
		m.SectionCursor = min(m.SectionCursor+1, max(0, len(headings)-1))
	case "up", "k":
		m.SectionCursor = max(0, m.SectionCursor-1)
	case "home", "g":
		m.SectionCursor = 0
	case "end", "G":
		m.SectionCursor = max(0, len(headings)-1)
	case "enter":
		m.focusSection(m.SectionCursor)
	case "a":
		m.clearSections()
		m.SectionsMode = false
	case "space":
		if len(headings) > 0 {
			if m.FoldedSections == nil {
				m.FoldedSections = make(map[int]bool)
			}
			m.FoldedSections[m.SectionCursor] = !m.FoldedSections[m.SectionCursor]
			m.InvalidateDocumentTree()
			m.adjustSelectionForFilter()
		}
	case "e":
		m.startHeadingInput("rename")
	case "n":
		m.startHeadingInput("sibling")
	case "N":
		m.startHeadingInput("child")
	}
	return m, nil
}

func (m Model) handleHeadingInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.HeadingInput = ""
		m.InputBuffer = ""
		m.CursorPos = 0
	case "enter":
		if strings.TrimSpace(m.InputBuffer) == "" {
			return m, nil
		}
		if m.ReadOnly {
			m.Err = fmt.Errorf("read-only file: section editing is disabled")
			return m, nil
		}
		m.saveHistory()
		headings := m.GetHeadings()
		index := m.SectionCursor
		if m.HeadingInput == "rename" {
			m.Err = m.FileModel.RenameHeading(index, m.InputBuffer)
		} else {
			level := 1
			if len(headings) > 0 {
				level = headings[index].Level
				if m.HeadingInput == "child" {
					level++
				}
			} else {
				index = -1
			}
			m.SectionCursor, m.Err = m.FileModel.CreateHeading(index, level, m.InputBuffer)
		}
		if m.Err != nil {
			return m, nil
		}
		m.HeadingInput = ""
		m.InputBuffer = ""
		m.CursorPos = 0
		m.InvalidateHeadingsCache()
		m.clearSections()
		m.writeIfPersist()
	default:
		return m.handleInputKey(msg)
	}
	return m, nil
}

func (m Model) renderSections() string {
	styles := m.Styles()
	var b strings.Builder
	b.WriteString(styles.Cyan("Sections") + "\n\n")
	headings := m.GetHeadings()
	if len(headings) == 0 {
		b.WriteString("No sections yet. Press n to create one.\n")
	}
	height := m.TermHeight
	if height <= 0 {
		height = 20
	}
	rows := max(1, height-9)
	start := max(0, m.SectionCursor-rows/2)
	if start+rows > len(headings) {
		start = max(0, len(headings)-rows)
	}
	if start > 0 {
		b.WriteString(styles.Dim("↑ more sections") + "\n")
	}
	for i := start; i < len(headings) && i < start+rows; i++ {
		h := headings[i]
		first, last := sectionBounds(headings, i, len(m.FileModel.Todos))
		done := 0
		for _, todo := range m.FileModel.Todos[first:last] {
			if todo.Checked {
				done++
			}
		}
		marker := "  "
		if i == m.SectionCursor {
			marker = "➜ "
		}
		fold := "▾"
		if m.FoldedSections[i] {
			fold = "▸"
		}
		title := fmt.Sprintf("%s%s%s %s  %d/%d done", marker, strings.Repeat("  ", h.Level-1), fold, h.Text, done, last-first)
		if m.TermWidth > 4 {
			title = ansi.Truncate(title, m.TermWidth-4, "…")
		}
		if i == m.SectionCursor {
			title = styles.Cyan(title)
		}
		b.WriteString(title + "\n")
	}
	if start+rows < len(headings) {
		b.WriteString(styles.Dim("↓ more sections") + "\n")
	}
	if m.HeadingInput != "" {
		label := "New section"
		switch m.HeadingInput {
		case "rename":
			label = "Rename section"
		case "child":
			label = "New subsection"
		}
		b.WriteString("\n" + label + ": " + m.InputBuffer[:m.CursorPos] + lipgloss.NewStyle().Reverse(true).Render(" ") + m.InputBuffer[m.CursorPos:] + "\n")
		b.WriteString(styles.Dim("enter save · esc cancel"))
	} else {
		b.WriteString("\n" + styles.Dim("↑/↓ select · enter focus · space fold · a all tasks · esc close"))
		if !m.ReadOnly {
			b.WriteString("\n" + styles.Dim("e rename · n new section · N new subsection"))
		}
	}
	if m.Err != nil {
		b.WriteString("\n" + m.Err.Error())
	}
	return b.String() + "\n"
}
