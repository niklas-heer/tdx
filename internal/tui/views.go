package tui

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/niklas-heer/tdx/internal/config"
)

func (m *Model) sectionRefs() []config.SectionRef {
	var refs []config.SectionRef
	var levels []int
	var path []string
	counts := map[string]int{}
	for _, h := range m.GetHeadings() {
		for len(levels) > 0 && levels[len(levels)-1] >= h.Level {
			levels = levels[:len(levels)-1]
			path = path[:len(path)-1]
		}
		levels = append(levels, h.Level)
		path = append(path, h.Text)
		key := fmt.Sprintf("%q", path)
		counts[key]++
		refs = append(refs, config.SectionRef{Path: slices.Clone(path), Occurrence: counts[key]})
	}
	return refs
}
func (m *Model) captureView() config.SavedView {
	v := config.SavedView{Tags: slices.Clone(m.FilteredTags), Priorities: slices.Clone(m.FilteredPriorities), Due: m.FilteredDueDate, FilterDone: m.FilterDone, ShowHeadings: m.ShowHeadings}
	slices.Sort(v.Tags)
	slices.Sort(v.Priorities)
	for i, ref := range m.sectionRefs() {
		if m.SectionFocus == i+1 {
			r := ref
			v.Focus = &r
		}
		if m.FoldedSections[i] {
			v.Folded = append(v.Folded, ref)
		}
	}
	return v
}
func (m *Model) applyView(name string, v config.SavedView) {
	m.FilteredTags = slices.Clone(v.Tags)
	m.FilteredPriorities = slices.Clone(v.Priorities)
	m.FilteredDueDate = v.Due
	m.FilterDone = v.FilterDone
	m.ShowHeadings = v.ShowHeadings
	m.SectionFocus = 0
	m.FoldedSections = nil
	for i, ref := range m.sectionRefs() {
		if v.Focus != nil && reflect.DeepEqual(ref, *v.Focus) {
			m.SectionFocus = i + 1
		}
		for _, fold := range v.Folded {
			if reflect.DeepEqual(ref, fold) {
				if m.FoldedSections == nil {
					m.FoldedSections = map[int]bool{}
				}
				m.FoldedSections[i] = true
			}
		}
	}
	m.ActiveView = name
	state := m.captureView()
	m.activeViewState = &state
	m.InvalidateDocumentTree()
	m.adjustSelectionForFilter()
}
func (m *Model) activeViewLabel() string {
	label := "View: " + m.ActiveView
	if m.activeViewState != nil && !reflect.DeepEqual(*m.activeViewState, m.captureView()) {
		label += " (modified)"
	}
	return label
}
func (m *Model) restoreSavedView() {
	store := m.Config().Views
	if store == nil || !m.Config().ViewsRestore {
		return
	}
	state, err := store.Load(m.FilePath)
	if err != nil {
		m.Err = err
		return
	}
	if view, ok := state.Views[state.Active]; ok {
		m.applyView(state.Active, view)
	}
}
func (m *Model) openViews(mode string) {
	if m.Config().Views == nil {
		m.Err = fmt.Errorf("saved views are unavailable in this session")
		return
	}
	state, err := m.Config().Views.Load(m.FilePath)
	if err != nil {
		m.Err = err
		return
	}
	m.ViewNames = nil
	for name := range state.Views {
		m.ViewNames = append(m.ViewNames, name)
	}
	slices.Sort(m.ViewNames)
	m.ViewMode = mode
	m.ViewCursor = 0
	m.ViewConfirm = false
	m.CommandMode = false
	m.InputBuffer = ""
	m.CursorPos = 0
}
func (m *Model) closeViews() {
	m.ViewMode = ""
	m.ViewConfirm = false
	m.InputBuffer = ""
	m.CursorPos = 0
}
func (m *Model) clearSavedView() {
	// Persist clearing the last-used view before changing the visible state.
	if store := m.Config().Views; store != nil {
		state, err := store.Load(m.FilePath)
		if err != nil {
			m.Err = err
			return
		}
		state.Active = ""
		if err = store.Save(m.FilePath, state); err != nil {
			m.Err = err
			return
		}
	}
	m.FilteredTags = nil
	m.FilteredPriorities = nil
	m.FilteredDueDate = ""
	m.FilterDone = false
	m.ActiveView = ""
	m.activeViewState = nil
	m.clearSections()
}
func (m *Model) commitView() {
	store := m.Config().Views
	state, err := store.Load(m.FilePath)
	if err != nil {
		m.Err = err
		return
	}
	name := strings.TrimSpace(m.InputBuffer)
	if m.ViewMode != "save" {
		if len(m.ViewNames) == 0 {
			return
		}
		name = m.ViewNames[m.ViewCursor]
	}
	switch m.ViewMode {
	case "save":
		if name == "" {
			return
		}
		if _, ok := state.Views[name]; ok && !m.ViewConfirm {
			m.ViewConfirm = true
			return
		}
		state.Views[name] = m.captureView()
		state.Active = name
	case "delete":
		if !m.ViewConfirm {
			m.ViewConfirm = true
			return
		}
		delete(state.Views, name)
		if state.Active == name {
			state.Active = ""
		}
	case "load":
		if _, ok := state.Views[name]; !ok {
			m.Err = fmt.Errorf("saved view %q no longer exists", name)
			return
		}
		state.Active = name
	}
	if err = store.Save(m.FilePath, state); err != nil {
		m.Err = err
		return
	}
	if m.ViewMode == "delete" {
		if m.ActiveView == name {
			m.ActiveView = ""
			m.activeViewState = nil
		}
	} else {
		m.applyView(name, state.Views[name])
	}
	m.closeViews()
}
func (m Model) handleViewKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if m.ViewConfirm {
		switch key {
		case "y", "Y":
			m.commitView()
		case "esc", "n", "N":
			m.ViewConfirm = false
		}
		return m, nil
	}
	switch key {
	case "esc":
		m.closeViews()
	case "enter":
		m.commitView()
	default:
		if m.ViewMode == "save" {
			return m.handleInputKey(msg)
		}
		switch key {
		case "down", "j":
			m.ViewCursor = min(m.ViewCursor+1, max(0, len(m.ViewNames)-1))
		case "up", "k":
			m.ViewCursor = max(0, m.ViewCursor-1)
		}
	}
	return m, nil
}
func (m Model) renderViews() string {
	var b strings.Builder
	title := "Saved views"
	if m.ViewMode == "delete" {
		title = "Delete saved view"
	}
	if m.ViewMode == "save" {
		title = "Save current view"
	}
	b.WriteString(m.Styles().Cyan(title) + "\n\n")
	if m.ViewMode == "save" {
		before := m.InputBuffer[:m.CursorPos]
		input := before + lipgloss.NewStyle().Reverse(true).Render(" ") + m.InputBuffer[m.CursorPos:]
		if m.TermWidth > 8 {
			width := m.TermWidth - 7
			left := max(0, lipgloss.Width(before)-width+1)
			input = ansi.Cut(input, left, left+width)
		}
		b.WriteString("Name: " + input + "\n")
		b.WriteString("Includes tags, priorities, due date, completion and sections.\n")
	} else {
		if len(m.ViewNames) == 0 {
			b.WriteString("No saved views. Use :save-view to create one.\n")
		}
		rows := max(1, m.TermHeight-8)
		start := max(0, m.ViewCursor-rows/2)
		for i := start; i < len(m.ViewNames) && i < start+rows; i++ {
			marker := "  "
			if i == m.ViewCursor {
				marker = "➜ "
			}
			label := marker + m.ViewNames[i]
			if m.TermWidth > 4 {
				label = ansi.Truncate(label, m.TermWidth-4, "…")
			}
			if i == m.ViewCursor {
				label = m.Styles().Cyan(label)
			}
			b.WriteString(label + "\n")
		}
	}
	if m.ViewConfirm {
		action := "Replace existing view"
		if m.ViewMode == "delete" {
			action = "Delete selected view"
		}
		b.WriteString("\n" + action + "? y confirm · n/esc cancel")
	} else {
		b.WriteString("\nenter confirm · esc cancel")
		if m.ViewMode != "save" {
			b.WriteString(" · ↑/↓ select")
		}
	}
	if m.Err != nil {
		b.WriteString("\n" + m.Err.Error())
	}
	lines := strings.Split(b.String(), "\n")
	if m.TermHeight > 0 && len(lines) > m.TermHeight {
		lines = lines[:m.TermHeight]
	}
	if m.TermWidth > 0 {
		for i, line := range lines {
			lines[i] = ansi.Truncate(line, m.TermWidth, "…")
		}
	}
	return strings.Join(lines, "\n") + "\n"
}
