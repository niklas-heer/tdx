package tui

import (
	"fmt"
	"os"
	"sort"
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

// todoSection represents a group of todos under a heading (or before any heading)
type todoSection struct {
	startIndex int // Index of first todo in this section
	endIndex   int // Index after last todo in this section (exclusive)
}

// getTodoSections divides todos into sections based on headings
// Each section contains todos that belong under a particular heading
func getTodoSections(todos []markdown.Todo, headings []markdown.Heading) []todoSection {
	if len(todos) == 0 {
		return nil
	}

	// Sort headings by BeforeTodoIndex to process in order
	sortedHeadings := make([]markdown.Heading, len(headings))
	copy(sortedHeadings, headings)
	sort.Slice(sortedHeadings, func(i, j int) bool {
		return sortedHeadings[i].BeforeTodoIndex < sortedHeadings[j].BeforeTodoIndex
	})

	var sections []todoSection

	// Find section boundaries from headings
	prevBoundary := 0
	for _, h := range sortedHeadings {
		// BeforeTodoIndex tells us which todo this heading appears before
		if h.BeforeTodoIndex > prevBoundary && h.BeforeTodoIndex <= len(todos) {
			// There are todos before this heading that form a section
			sections = append(sections, todoSection{
				startIndex: prevBoundary,
				endIndex:   h.BeforeTodoIndex,
			})
			prevBoundary = h.BeforeTodoIndex
		}
	}

	// Add final section for remaining todos
	if prevBoundary < len(todos) {
		sections = append(sections, todoSection{
			startIndex: prevBoundary,
			endIndex:   len(todos),
		})
	}

	// If no sections were created (no headings), treat all todos as one section
	if len(sections) == 0 {
		sections = append(sections, todoSection{
			startIndex: 0,
			endIndex:   len(todos),
		})
	}

	return sections
}

// sortTodosInSections sorts todos within each section using the provided sort function
// sortFn should sort the slice in place
func sortTodosInSections(todos []markdown.Todo, headings []markdown.Heading, sortFn func([]markdown.Todo)) {
	sections := getTodoSections(todos, headings)

	for _, section := range sections {
		if section.endIndex > section.startIndex {
			sectionTodos := todos[section.startIndex:section.endIndex]
			sortFn(sectionTodos)
		}
	}
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
			Name:        "sort-done",
			Description: "Sort todos by completion (incomplete first)",
			Handler: func(m *Model) {
				m.saveHistory()
				// Get headings to sort within sections
				headings := m.FileModel.GetHeadings()

				// Sort function: incomplete first, then complete (stable)
				sortByDone := func(todos []markdown.Todo) {
					sort.SliceStable(todos, func(i, j int) bool {
						// Incomplete (false) comes before complete (true)
						return !todos[i].Checked && todos[j].Checked
					})
				}

				// Sort within each heading section
				sortTodosInSections(m.FileModel.Todos, headings, sortByDone)

				// Update indices and rebuild line structure
				markdown.RebuildFileStructure(&m.FileModel)
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
				// Get headings to sort within sections
				headings := m.FileModel.GetHeadings()

				// Sort function: by due date (earliest first), no due date at end (stable)
				sortByDueDate := func(todos []markdown.Todo) {
					sort.SliceStable(todos, func(i, j int) bool {
						di, dj := todos[i].DueDate, todos[j].DueDate
						// Both have no due date - maintain order
						if di == nil && dj == nil {
							return false
						}
						// No due date goes after those with due date
						if di == nil {
							return false
						}
						if dj == nil {
							return true
						}
						// Both have due dates - earlier date first
						return di.Before(*dj)
					})
				}

				// Sort within each heading section
				sortTodosInSections(m.FileModel.Todos, headings, sortByDueDate)

				// Update indices and rebuild line structure
				markdown.RebuildFileStructure(&m.FileModel)
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
				// Get headings to sort within sections
				headings := m.FileModel.GetHeadings()

				// Sort function: by priority (p1 first), unprioritized at end (stable)
				sortByPriority := func(todos []markdown.Todo) {
					sort.SliceStable(todos, func(i, j int) bool {
						pi, pj := todos[i].Priority, todos[j].Priority
						// Both unprioritized - maintain order
						if pi == 0 && pj == 0 {
							return false
						}
						// Unprioritized goes after prioritized
						if pi == 0 {
							return false
						}
						if pj == 0 {
							return true
						}
						// Both prioritized - lower number = higher priority
						return pi < pj
					})
				}

				// Sort within each heading section
				sortTodosInSections(m.FileModel.Todos, headings, sortByPriority)

				// Update indices and rebuild line structure
				markdown.RebuildFileStructure(&m.FileModel)
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
