package tui

import (
	"os"
	"strings"

	"github.com/niklas-heer/tdx/internal/markdown"
	"github.com/niklas-heer/tdx/internal/util"
)

// Command represents a command in the command palette
type Command struct {
	Name        string
	Description string
	Handler     func(m *Model)
}

// InitCommands initializes the command palette with all available commands
func InitCommands() []Command {
	return []Command{
		{
			Name:        "check-all",
			Description: "Mark all todos as complete",
			Handler: func(m *Model) {
				m.saveHistory()
				// Use index-based loop with bounds check since UpdateTodoItem
				// can re-extract todos from AST, potentially changing slice
				for i := 0; i < len(m.FileModel.Todos); i++ {
					if i >= len(m.FileModel.Todos) {
						break // Safety check if slice shrinks
					}
					todo := m.FileModel.Todos[i]
					if !todo.Checked {
						_ = m.FileModel.UpdateTodoItem(i, todo.Text, true)
					}
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
				// Use index-based loop with bounds check since UpdateTodoItem
				// can re-extract todos from AST, potentially changing slice
				for i := 0; i < len(m.FileModel.Todos); i++ {
					if i >= len(m.FileModel.Todos) {
						break // Safety check if slice shrinks
					}
					todo := m.FileModel.Todos[i]
					if todo.Checked {
						_ = m.FileModel.UpdateTodoItem(i, todo.Text, false)
					}
				}
				m.InvalidateDocumentTree()
				m.writeIfPersist()
			},
		},
		{
			Name:        "sort",
			Description: "Sort todos (incomplete first)",
			Handler: func(m *Model) {
				m.saveHistory()
				// Stable sort: incomplete first, then complete
				var incomplete, complete []markdown.Todo
				for _, todo := range m.FileModel.Todos {
					if todo.Checked {
						complete = append(complete, todo)
					} else {
						incomplete = append(incomplete, todo)
					}
				}
				m.FileModel.Todos = append(incomplete, complete...)
				// Update indices and rebuild line structure
				markdown.RebuildFileStructure(&m.FileModel)
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
			Name:        "clear-done",
			Description: "Delete all completed todos",
			Handler: func(m *Model) {
				m.saveHistory()
				// Delete completed todos from the end backwards to preserve indices
				for i := len(m.FileModel.Todos) - 1; i >= 0; i-- {
					if m.FileModel.Todos[i].Checked {
						_ = m.FileModel.DeleteTodoItem(i)
					}
				}
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
				_ = markdown.WriteFile(m.FilePath, &m.FileModel)
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
				fm, err := markdown.ReadFile(m.FilePath)
				if err != nil {
					m.Err = err
					return
				}
				m.FileModel = *fm
				m.History = nil // Clear history
				if m.SelectedIndex >= len(m.FileModel.Todos) {
					m.SelectedIndex = util.Max(0, len(m.FileModel.Todos)-1)
				}
			},
		},
		{
			Name:        "force-save",
			Description: "Force save even if file was modified externally",
			Handler: func(m *Model) {
				// Save without checking for external modifications
				content := markdown.SerializeMarkdown(&m.FileModel)
				err := markdown.WriteFile(m.FilePath, &m.FileModel)
				if err != nil {
					// If still fails, write directly without checks
					if err := os.WriteFile(m.FilePath, []byte(content), 0644); err != nil {
						m.Err = err
					}
				}
			},
		},
	}
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
