package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/niklas-heer/tdx/internal/editor"
	"github.com/niklas-heer/tdx/internal/markdown"
	"github.com/niklas-heer/tdx/internal/util"
)

// Command represents a command in the command palette
type Command struct {
	Name        string
	Description string
	Handler     func(m *Model)
}

// InitCommands initializes the command palette with all available commands.
// cfg is optional; if non-nil it is used to conditionally include commands
// that require injected dependencies (e.g. versioning).
func InitCommands(cfg ...*ConfigType) []Command {
	var activeCfg *ConfigType
	if len(cfg) > 0 {
		activeCfg = cfg[0]
	}
	cmds := []Command{
		{Name: "rezero", Description: "Review readiness and work a bottom-to-top batch", Handler: func(m *Model) {
			if m.rezero.Phase != "" {
				m.stopRezero()
			} else {
				m.startRezero()
			}
		}},
		{
			Name:        "check-all",
			Description: "Mark all todos as complete",
			Handler: func(m *Model) {
				m.saveHistory()
				if err := m.applyAction(editor.Action{Kind: editor.SetAllChecked, Checked: true}); err != nil {
					m.Err = err
					return
				}
				m.InvalidateDocumentTree()
				m.writeIfPersist()
			},
		},
		{
			Name:        "uncheck-all",
			Description: "Mark all todos as incomplete",
			Handler: func(m *Model) {
				m.saveHistory()
				if err := m.applyAction(editor.Action{Kind: editor.SetAllChecked, Checked: false}); err != nil {
					m.Err = err
					return
				}
				m.InvalidateDocumentTree()
				m.writeIfPersist()
			},
		},
		{
			Name:        "sort-done",
			Description: "Sort todos by completion (incomplete first)",
			Handler: func(m *Model) {
				m.saveHistory()
				if err := m.applyAction(editor.Action{Kind: editor.SortDone}); err != nil {
					m.Err = err
					return
				}
				m.InvalidateDocumentTree()
				m.writeIfPersist()
				// Adjust selection if needed
				if m.SelectedIndex >= len(m.FileModel.Todos) {
					m.SelectedIndex = util.Max(0, len(m.FileModel.Todos)-1)
				}
			},
		},
		{
			Name:        "sort-due",
			Description: "Sort todos by due date (earliest first)",
			Handler: func(m *Model) {
				m.saveHistory()
				if err := m.applyAction(editor.Action{Kind: editor.SortDue}); err != nil {
					m.Err = err
					return
				}
				m.InvalidateDocumentTree()
				m.writeIfPersist()
				// Adjust selection if needed
				if m.SelectedIndex >= len(m.FileModel.Todos) {
					m.SelectedIndex = util.Max(0, len(m.FileModel.Todos)-1)
				}
			},
		},
		{
			Name:        "sort-priority",
			Description: "Sort todos by priority (p1 first, then p2, etc.)",
			Handler: func(m *Model) {
				m.saveHistory()
				if err := m.applyAction(editor.Action{Kind: editor.SortPriority}); err != nil {
					m.Err = err
					return
				}
				m.InvalidateDocumentTree()
				m.writeIfPersist()
				// Adjust selection if needed
				if m.SelectedIndex >= len(m.FileModel.Todos) {
					m.SelectedIndex = util.Max(0, len(m.FileModel.Todos)-1)
				}
			},
		},
		{
			Name:        "filter-done",
			Description: "Toggle showing/hiding completed todos",
			Handler: func(m *Model) {
				m.FilterDone = !m.FilterDone
				// Invalidate document tree since visibility changed
				m.InvalidateDocumentTree()
				// Adjust selection if current item is now hidden
				if m.FilterDone {
					m.adjustSelectionForFilter()
				}
			},
		},
		{
			Name:        "filter-due",
			Description: "Toggle showing only todos with due dates",
			Handler: func(m *Model) {
				// Toggle between "all" (has due date) and "" (no filter)
				if m.FilteredDueDate == "all" {
					m.FilteredDueDate = ""
				} else {
					m.FilteredDueDate = "all"
				}
				// Invalidate document tree since visibility changed
				m.InvalidateDocumentTree()
				// Adjust selection if current item is now hidden
				if m.FilteredDueDate != "" {
					m.adjustSelectionForFilter()
				}
			},
		},
		{
			Name:        "filter-overdue",
			Description: "Toggle showing only overdue todos",
			Handler: func(m *Model) {
				if m.FilteredDueDate == "overdue" {
					m.FilteredDueDate = ""
				} else {
					m.FilteredDueDate = "overdue"
				}
				m.InvalidateDocumentTree()
				if m.FilteredDueDate != "" {
					m.adjustSelectionForFilter()
				}
			},
		},
		{
			Name:        "filter-today",
			Description: "Toggle showing only todos due today",
			Handler: func(m *Model) {
				if m.FilteredDueDate == "today" {
					m.FilteredDueDate = ""
				} else {
					m.FilteredDueDate = "today"
				}
				m.InvalidateDocumentTree()
				if m.FilteredDueDate != "" {
					m.adjustSelectionForFilter()
				}
			},
		},
		{
			Name:        "filter-week",
			Description: "Toggle showing only todos due this week",
			Handler: func(m *Model) {
				if m.FilteredDueDate == "week" {
					m.FilteredDueDate = ""
				} else {
					m.FilteredDueDate = "week"
				}
				m.InvalidateDocumentTree()
				if m.FilteredDueDate != "" {
					m.adjustSelectionForFilter()
				}
			},
		},
		{
			Name:        "clear-done",
			Description: "Delete all completed todos",
			Handler: func(m *Model) {
				m.saveHistory()
				if err := m.applyAction(editor.Action{Kind: editor.ClearDone}); err != nil {
					m.Err = err
					return
				}
				m.InvalidateHeadingsCache()
				m.InvalidateDocumentTree()
				m.writeIfPersist()
				// Adjust selection
				if m.SelectedIndex >= len(m.FileModel.Todos) {
					m.SelectedIndex = util.Max(0, len(m.FileModel.Todos)-1)
				}
			},
		},
		{
			Name:        "read-only",
			Description: "Toggle read-only mode (changes not saved)",
			Handler: func(m *Model) {
				m.ReadOnly = !m.ReadOnly
			},
		},
		{
			Name:        "save",
			Description: "Save current state to file",
			Handler: func(m *Model) {
				local := markdown.SerializeMarkdown(&m.FileModel)
				if err := m.Config().Store.WriteFile(m.FilePath, &m.FileModel); err != nil {
					m.recordSaveError(err, local)
					return
				}
				m.clearConflict()
			},
		},
		{
			Name:        "wrap",
			Description: "Toggle word wrap for long lines",
			Handler: func(m *Model) {
				m.WordWrap = !m.WordWrap
			},
		},
		{
			Name:        "line-numbers",
			Description: "Toggle relative line numbers",
			Handler: func(m *Model) {
				m.HideLineNumbers = !m.HideLineNumbers
			},
		},
		{
			Name:        "set-max-visible",
			Description: "Set max visible items for this session (prompt for number)",
			Handler: func(m *Model) {
				// Switch to max-visible input mode
				m.MaxVisibleInputMode = true
				m.CommandMode = false
				m.InputBuffer = ""
				m.CursorPos = 0
			},
		},
		{Name: "sections", Description: "Browse, edit, focus, and fold Markdown sections", Handler: func(m *Model) { m.openSections() }},
		{Name: "all-sections", Description: "Clear section focus and unfold all sections", Handler: func(m *Model) { m.clearSections() }},

		{
			Name:        "show-headings",
			Description: "Toggle displaying markdown headings between tasks",
			Handler: func(m *Model) {
				m.ShowHeadings = !m.ShowHeadings
			},
		},
		{
			Name:        "reload",
			Description: "Reload file from disk (discards unsaved changes)",
			Handler: func(m *Model) {
				// Reload file from disk
				fm, err := m.Config().Store.ReadFile(m.FilePath)
				if err != nil {
					m.Err = err
					return
				}
				m.FileModel = *fm
				m.stopRezero()
				m.resetFileSettings()
				m.RefreshAvailableTags()
				m.history.Clear()
				m.InvalidateHeadingsCache()
				m.clearSections()
				m.clearConflict()
				if m.SelectedIndex >= len(m.FileModel.Todos) {
					m.SelectedIndex = util.Max(0, len(m.FileModel.Todos)-1)
				}
			},
		},
		{
			Name:        "force-save",
			Description: "Force save even if file was modified externally",
			Handler: func(m *Model) {
				content := m.ConflictLocalContent
				if !m.ConflictPending {
					content = markdown.SerializeMarkdown(&m.FileModel)
				}
				saveErr := m.Config().Store.WriteContentUnchecked(m.FilePath, content)
				var postCommit *markdown.PostCommitError
				if saveErr != nil && !errors.As(saveErr, &postCommit) {
					m.Err = saveErr
					return
				}
				fm, readErr := m.Config().Store.ReadFile(m.FilePath)
				if fm != nil {
					if m.rezero.Phase != "" && m.ConflictPending {
						m.history.Push(&m.FileModel)
					}
					m.FileModel = *fm
					m.stopRezero()
					m.clearConflict()
				}
				m.Err = errors.Join(saveErr, readErr)
			},
		},
		{
			Name:        "diff",
			Description: "Compare retained local changes with external disk content",
			Handler: func(m *Model) {
				if !m.ConflictPending {
					m.Err = fmt.Errorf("no unresolved file conflict")
					return
				}
				m.ConflictDiffMode = true
				m.ConflictDiffScroll = 0
			},
		},
		{
			Name:        "theme",
			Description: "Change color theme with live preview",
			Handler: func(m *Model) {
				// Check if themes are available
				if len(m.AvailableThemes) == 0 {
					m.Err = fmt.Errorf("no themes available")
					return
				}
				// Save original styles for cancel
				m.OriginalStyles = m.styles
				// Find current theme in list and set cursor
				m.ThemeCursor = 0
				for i, name := range m.AvailableThemes {
					if name == m.CurrentThemeName {
						m.ThemeCursor = i
						break
					}
				}
				m.ThemeMode = true
			},
		},
	}

	// Conditionally add versioning command when a store is wired.
	if activeCfg != nil && activeCfg.ListVersionsFunc != nil {
		cmds = append(cmds, Command{
			Name:        "versions",
			Description: "Browse and restore file version history",
			Handler: func(m *Model) {
				cfg := m.Config()
				if cfg == nil || cfg.ListVersionsFunc == nil {
					return
				}
				versions, err := cfg.ListVersionsFunc(m.FilePath)
				if err != nil {
					m.Err = err
					return
				}
				m.VersionsList = versions
				m.VersionsCursor = 0
				m.VersionsDiffScroll = 0
				m.VersionsConfirmMode = false
				m.VersionsMode = true
			},
		})
	}

	return cmds
}

// HighlightMatches returns text with matched characters highlighted
func HighlightMatches(text, query string, greenStyle func(string) string) string {
	if query == "" {
		return text
	}

	lowerText := strings.ToLower(text)
	lowerQuery := strings.ToLower(query)

	// Find match positions
	var matchPositions []int

	// First try exact substring match
	if idx := strings.Index(lowerText, lowerQuery); idx != -1 {
		for i := idx; i < idx+len(query); i++ {
			matchPositions = append(matchPositions, i)
		}
	} else {
		// Fuzzy match positions
		queryIdx := 0
		for i := 0; i < len(lowerText) && queryIdx < len(lowerQuery); i++ {
			if lowerText[i] == lowerQuery[queryIdx] {
				matchPositions = append(matchPositions, i)
				queryIdx++
			}
		}
	}

	// Build highlighted string
	var result strings.Builder
	matchSet := make(map[int]bool)
	for _, pos := range matchPositions {
		matchSet[pos] = true
	}

	for i, char := range text {
		if matchSet[i] {
			result.WriteString(greenStyle(string(char)))
		} else {
			result.WriteString(string(char))
		}
	}

	return result.String()
}
