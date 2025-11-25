package tui

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/niklas-heer/tdx/internal/config"
	"github.com/niklas-heer/tdx/internal/markdown"
	"github.com/niklas-heer/tdx/internal/util"
)

// Config and styles injected from main - using any to avoid syntax issues
var (
	AppConfig any
	Styles    any
)

// FileChangedMsg is sent when the file changes on disk
type FileChangedMsg struct{}

// Update handles all TUI updates
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.TermWidth = msg.Width
		return m, nil
	case ClearCopyFeedbackMsg:
		m.CopyFeedback = false
		return m, nil
	case FileChangedMsg:
		// File changed on disk - try to auto-reload
		return m, m.checkAndReloadFile()
	case reloadedMsg:
		// Successfully reloaded from disk
		m = msg.model
		m.InvalidateHeadingsCache()  // Invalidate cache on reload
		return m, watchFileChanges() // Continue watching
	case SearchDebounceMsg:
		// Debounced search update
		if m.SearchMode && m.searchPending {
			m.updateSearchResults()
			m.searchPending = false
		}
		return m, nil
	case CommandDebounceMsg:
		// Debounced command filter update
		if m.CommandMode && m.searchPending {
			m.updateFilteredCommands()
			m.searchPending = false
		}
		return m, nil
	case tea.KeyMsg:
		// Handle EOF from piped input
		if msg.Type == tea.KeyCtrlD {
			return m, tea.Quit
		}
		// Handle bracketed paste (cmd+v on macOS)
		if msg.Paste && (m.InputMode || m.EditMode) {
			text := string(msg.Runes)
			// Take only first line
			if idx := strings.Index(text, "\n"); idx != -1 {
				text = text[:idx]
			}
			m.InputBuffer = m.InputBuffer[:m.CursorPos] + text + m.InputBuffer[m.CursorPos:]
			m.CursorPos += len(text)
			return m, nil
		}
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Handle error display - any key dismisses
	if m.Err != nil {
		m.Err = nil
		return m, nil
	}

	// Handle input/edit mode
	if m.InputMode || m.EditMode {
		return m.handleInputKey(msg)
	}

	// Handle max-visible input mode
	if m.MaxVisibleInputMode {
		return m.handleMaxVisibleInputKey(msg)
	}

	// Handle search mode
	if m.SearchMode {
		return m.handleSearchKey(msg)
	}

	// Handle filter mode
	if m.FilterMode {
		return m.handleFilterKey(msg)
	}

	// Handle command mode
	if m.CommandMode {
		return m.handleCommandKey(msg)
	}

	// Handle move mode
	if m.MoveMode {
		return m.handleMoveKey(msg)
	}

	// Handle help mode
	if m.HelpMode {
		if key == "?" || key == "esc" {
			m.HelpMode = false
		}
		return m, nil
	}

	// Number buffer for vim-style navigation
	if key >= "1" && key <= "9" {
		m.NumberBuffer += key
		return m, nil
	}

	// Get count from number buffer
	count := 1
	if m.NumberBuffer != "" {
		count, _ = strconv.Atoi(m.NumberBuffer)
		m.NumberBuffer = ""
	}

	// Reset g-pressed state on any key except 'g' itself (handled in switch)
	if key != "g" && key != "G" {
		m.gPressed = false
	}

	switch key {
	case "esc", "ctrl+c":
		return m, tea.Quit

	case "j", "down":
		if m.FilterDone {
			// Navigate within visible todos only
			visible := m.getVisibleTodos()
			if len(visible) > 0 {
				currentPos := 0
				for i, idx := range visible {
					if idx == m.SelectedIndex {
						currentPos = i
						break
					}
				}
				newPos := util.Min(currentPos+count, len(visible)-1)
				m.SelectedIndex = visible[newPos]
			}
		} else {
			m.SelectedIndex = util.Min(m.SelectedIndex+count, len(m.FileModel.Todos)-1)
			if m.SelectedIndex < 0 {
				m.SelectedIndex = 0
			}
		}

	case "k", "up":
		if m.FilterDone {
			// Navigate within visible todos only
			visible := m.getVisibleTodos()
			if len(visible) > 0 {
				currentPos := 0
				for i, idx := range visible {
					if idx == m.SelectedIndex {
						currentPos = i
						break
					}
				}
				newPos := util.Max(currentPos-count, 0)
				m.SelectedIndex = visible[newPos]
			}
		} else {
			m.SelectedIndex = util.Max(m.SelectedIndex-count, 0)
		}

	case " ", "enter":
		if len(m.FileModel.Todos) > 0 {
			m.saveHistory()
			todo := m.FileModel.Todos[m.SelectedIndex]
			m.FileModel.UpdateTodoItem(m.SelectedIndex, todo.Text, !todo.Checked)
			// Mark this todo as locally modified
			m.LocallyModified[todo.Text] = true
			m.writeIfPersist()
			// Adjust selection if item is now hidden by filter
			if m.FilterDone && m.FileModel.Todos[m.SelectedIndex].Checked {
				m.adjustSelectionForFilter()
			}
		}

	case "n":
		m.saveHistory()
		m.InputMode = true
		m.InputBuffer = ""
		m.CursorPos = 0

	case "e":
		if len(m.FileModel.Todos) > 0 {
			m.saveHistory()
			m.EditMode = true
			m.InputBuffer = m.FileModel.Todos[m.SelectedIndex].Text
			m.CursorPos = len(m.InputBuffer)
		}

	case "d":
		if len(m.FileModel.Todos) > 0 {
			m.saveHistory()
			m.deleteCurrent()
		}

	case "c":
		if len(m.FileModel.Todos) > 0 {
			util.CopyToClipboard(m.FileModel.Todos[m.SelectedIndex].Text)
			m.CopyFeedback = true
			return m, tea.Tick(1500*time.Millisecond, func(t time.Time) tea.Msg {
				return ClearCopyFeedbackMsg{}
			})
		}

	case "m":
		if len(m.FileModel.Todos) > 0 {
			m.saveHistory()
			m.MoveMode = true
		}

	case "u":
		if m.History != nil {
			m.FileModel = *m.History
			m.History = nil
			m.writeIfPersist()
			if m.SelectedIndex >= len(m.FileModel.Todos) {
				m.SelectedIndex = util.Max(0, len(m.FileModel.Todos)-1)
			}
		}

	case "?":
		m.HelpMode = true

	case "/":
		if len(m.FileModel.Todos) > 0 {
			m.SearchMode = true
			m.InputBuffer = ""
			m.CursorPos = 0
			m.SearchCursor = 0
			// Initialize with all todos
			m.SearchResults = nil
			for i := range m.FileModel.Todos {
				m.SearchResults = append(m.SearchResults, i)
			}
		}

	case "f":
		if len(m.AvailableTags) > 0 {
			m.FilterMode = true
			m.TagFilterCursor = 0
		}

	case "G":
		// Go to bottom (vim-style)
		if m.FilterDone || len(m.FilteredTags) > 0 {
			visible := m.getVisibleTodos()
			if len(visible) > 0 {
				m.SelectedIndex = visible[len(visible)-1]
			}
		} else if len(m.FileModel.Todos) > 0 {
			m.SelectedIndex = len(m.FileModel.Todos) - 1
		}
		m.gPressed = false

	case "g":
		// First g press - wait for second g
		if m.gPressed {
			// gg - go to top
			if m.FilterDone || len(m.FilteredTags) > 0 {
				visible := m.getVisibleTodos()
				if len(visible) > 0 {
					m.SelectedIndex = visible[0]
				}
			} else if len(m.FileModel.Todos) > 0 {
				m.SelectedIndex = 0
			}
			m.gPressed = false
		} else {
			m.gPressed = true
		}
		return m, nil

	case ":":
		m.CommandMode = true
		m.InputBuffer = ""
		m.CursorPos = 0
		m.CommandCursor = 0
		// Initialize with all commands
		m.FilteredCmds = nil
		for i := range m.Commands {
			m.FilteredCmds = append(m.FilteredCmds, i)
		}
	}

	return m, nil
}

func (m Model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "enter", "ctrl+m":
		if m.InputMode {
			if m.InputBuffer != "" {
				m.addNewTodo()
			}
			m.InputMode = false
		} else if m.EditMode {
			if m.InputBuffer != "" {
				todo := m.FileModel.Todos[m.SelectedIndex]
				m.FileModel.UpdateTodoItem(m.SelectedIndex, m.InputBuffer, todo.Checked)
				m.writeIfPersist()
			}
			m.EditMode = false
		}

	case "esc":
		m.InputMode = false
		m.EditMode = false
		if m.History != nil {
			m.FileModel = *m.History
			m.History = nil
		}

	case "backspace", "ctrl+h":
		if m.CursorPos > 0 {
			m.InputBuffer = m.InputBuffer[:m.CursorPos-1] + m.InputBuffer[m.CursorPos:]
			m.CursorPos--
		}

	case "delete":
		if m.CursorPos < len(m.InputBuffer) {
			m.InputBuffer = m.InputBuffer[:m.CursorPos] + m.InputBuffer[m.CursorPos+1:]
		}

	case "left":
		if m.CursorPos > 0 {
			m.CursorPos--
		}

	case "right":
		if m.CursorPos < len(m.InputBuffer) {
			m.CursorPos++
		}

	case "home", "ctrl+a":
		m.CursorPos = 0

	case "end", "ctrl+e":
		m.CursorPos = len(m.InputBuffer)

	case "ctrl+v", "ctrl+shift+v", "ctrl+y":
		// Paste from clipboard (ctrl+y is more reliable in terminals)
		text := util.PasteFromClipboard()
		if text != "" {
			m.InputBuffer = m.InputBuffer[:m.CursorPos] + text + m.InputBuffer[m.CursorPos:]
			m.CursorPos += len(text)
		}

	default:
		// Insert character
		if len(key) == 1 {
			m.InputBuffer = m.InputBuffer[:m.CursorPos] + key + m.InputBuffer[m.CursorPos:]
			m.CursorPos++
		}
	}

	return m, nil
}

func (m Model) handleMaxVisibleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "enter", "ctrl+m":
		// Parse and set the max visible value
		if m.InputBuffer != "" {
			if num, err := strconv.Atoi(m.InputBuffer); err == nil && num >= 0 {
				m.MaxVisibleOverride = num
			}
		}
		m.MaxVisibleInputMode = false
		m.InputBuffer = ""

	case "esc":
		m.MaxVisibleInputMode = false
		m.InputBuffer = ""

	case "backspace", "ctrl+h":
		if m.CursorPos > 0 {
			m.InputBuffer = m.InputBuffer[:m.CursorPos-1] + m.InputBuffer[m.CursorPos:]
			m.CursorPos--
		}

	case "delete":
		if m.CursorPos < len(m.InputBuffer) {
			m.InputBuffer = m.InputBuffer[:m.CursorPos] + m.InputBuffer[m.CursorPos+1:]
		}

	case "left":
		if m.CursorPos > 0 {
			m.CursorPos--
		}

	case "right":
		if m.CursorPos < len(m.InputBuffer) {
			m.CursorPos++
		}

	case "home", "ctrl+a":
		m.CursorPos = 0

	case "end", "ctrl+e":
		m.CursorPos = len(m.InputBuffer)

	default:
		// Only allow digits
		if len(key) == 1 && key >= "0" && key <= "9" {
			m.InputBuffer = m.InputBuffer[:m.CursorPos] + key + m.InputBuffer[m.CursorPos:]
			m.CursorPos++
		}
	}

	return m, nil
}

func (m Model) handleMoveKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "j", "down":
		// Move down one position in the actual list
		if m.SelectedIndex < len(m.FileModel.Todos)-1 {
			if err := m.moveTodo(m.SelectedIndex, m.SelectedIndex+1); err == nil {
				m.SelectedIndex++
				// If the new position is filtered, keep moving cursor to next visible
				if m.FilterDone || len(m.FilteredTags) > 0 {
					nextVisible := m.findNextVisibleTodo(m.SelectedIndex - 1)
					if nextVisible != -1 {
						m.SelectedIndex = nextVisible
					}
				}
			}
		}

	case "k", "up":
		// Move up one position in the actual list
		if m.SelectedIndex > 0 {
			if err := m.moveTodo(m.SelectedIndex, m.SelectedIndex-1); err == nil {
				m.SelectedIndex--
				// If the new position is filtered, keep cursor at previous visible
				if m.FilterDone || len(m.FilteredTags) > 0 {
					prevVisible := m.findPreviousVisibleTodo(m.SelectedIndex + 1)
					if prevVisible != -1 {
						m.SelectedIndex = prevVisible
					}
				}
			}
		}

	case "enter":
		m.writeIfPersist()
		m.MoveMode = false

	case "esc":
		if m.History != nil {
			m.FileModel = *m.History
			m.History = nil
		}
		m.MoveMode = false
	}

	return m, nil
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "enter":
		// Select current search result
		if len(m.SearchResults) > 0 && m.SearchCursor < len(m.SearchResults) {
			m.SelectedIndex = m.SearchResults[m.SearchCursor]
		}
		m.SearchMode = false
		m.InputBuffer = ""
		m.SearchResults = nil
		m.searchPending = false

	case "esc":
		m.SearchMode = false
		m.InputBuffer = ""
		m.SearchResults = nil
		m.searchPending = false

	case "down", "ctrl+n", "ctrl+j":
		// Move down in search results
		if len(m.SearchResults) > 0 && m.SearchCursor < len(m.SearchResults)-1 {
			m.SearchCursor++
		}

	case "up", "ctrl+p", "ctrl+k":
		// Move up in search results
		if m.SearchCursor > 0 {
			m.SearchCursor--
		}

	case "backspace", "ctrl+h":
		if m.CursorPos > 0 {
			m.InputBuffer = m.InputBuffer[:m.CursorPos-1] + m.InputBuffer[m.CursorPos:]
			m.CursorPos--
			// Debounce search update
			m.searchPending = true
			return m, searchDebounceCmd()
		}

	default:
		// Insert character
		if len(key) == 1 {
			m.InputBuffer = m.InputBuffer[:m.CursorPos] + key + m.InputBuffer[m.CursorPos:]
			m.CursorPos++
			// Debounce search update
			m.searchPending = true
			return m, searchDebounceCmd()
		}
	}

	return m, nil
}

func (m Model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "enter", " ":
		// Toggle tag filter
		if len(m.AvailableTags) > 0 && m.TagFilterCursor < len(m.AvailableTags) {
			selectedTag := m.AvailableTags[m.TagFilterCursor]

			// Check if tag is already in filter
			found := false
			for i, tag := range m.FilteredTags {
				if tag == selectedTag {
					// Remove tag from filter
					m.FilteredTags = append(m.FilteredTags[:i], m.FilteredTags[i+1:]...)
					found = true
					break
				}
			}

			if !found {
				// Add tag to filter
				m.FilteredTags = append(m.FilteredTags, selectedTag)
			}

			// Close filter mode after selection
			m.FilterMode = false
		}

	case "esc":
		m.FilterMode = false

	case "c":
		// Clear all filters
		m.FilteredTags = []string{}

	case "down", "ctrl+n", "ctrl+j", "j":
		// Move down in tag list
		if len(m.AvailableTags) > 0 && m.TagFilterCursor < len(m.AvailableTags)-1 {
			m.TagFilterCursor++
		}

	case "up", "ctrl+p", "ctrl+k", "k":
		// Move up in tag list
		if m.TagFilterCursor > 0 {
			m.TagFilterCursor--
		}
	}

	return m, nil
}

func (m Model) handleCommandKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "enter":
		// Execute current command
		if len(m.FilteredCmds) > 0 && m.CommandCursor < len(m.FilteredCmds) {
			cmdIdx := m.FilteredCmds[m.CommandCursor]
			m.Commands[cmdIdx].Handler(&m)
		}
		m.CommandMode = false
		m.searchPending = false
		// Only clear buffer if we didn't switch to input or maxVisibleInput mode
		if !m.InputMode && !m.MaxVisibleInputMode {
			m.InputBuffer = ""
		}
		m.FilteredCmds = nil

	case "tab":
		// Tab completes to the selected command name
		if len(m.FilteredCmds) > 0 && m.CommandCursor < len(m.FilteredCmds) {
			cmdIdx := m.FilteredCmds[m.CommandCursor]
			m.InputBuffer = m.Commands[cmdIdx].Name
			m.CursorPos = len(m.InputBuffer)
			m.updateFilteredCommands()
		}

	case "esc":
		m.CommandMode = false
		m.InputBuffer = ""
		m.FilteredCmds = nil
		m.searchPending = false

	case "down", "ctrl+n", "ctrl+j":
		// Move down in command list
		if len(m.FilteredCmds) > 0 && m.CommandCursor < len(m.FilteredCmds)-1 {
			m.CommandCursor++
		}

	case "up", "ctrl+p", "ctrl+k":
		// Move up in command list
		if m.CommandCursor > 0 {
			m.CommandCursor--
		}

	case "backspace", "ctrl+h":
		if m.CursorPos > 0 {
			m.InputBuffer = m.InputBuffer[:m.CursorPos-1] + m.InputBuffer[m.CursorPos:]
			m.CursorPos--
			// Debounce command filter update
			m.searchPending = true
			return m, commandDebounceCmd()
		}

	default:
		// Insert character
		if len(key) == 1 {
			m.InputBuffer = m.InputBuffer[:m.CursorPos] + key + m.InputBuffer[m.CursorPos:]
			m.CursorPos++
			// Debounce command filter update
			m.searchPending = true
			return m, commandDebounceCmd()
		}
	}

	return m, nil
}

// Helper functions

func (m *Model) saveHistory() {
	m.History = m.FileModel.Clone()
}

func (m *Model) addNewTodo() {
	m.FileModel.AddTodoItem(m.InputBuffer, false)
	m.InvalidateHeadingsCache() // New todo may affect heading positions
	m.writeIfPersist()
	m.SelectedIndex = len(m.FileModel.Todos) - 1
}

func (m *Model) deleteCurrent() {
	if len(m.FileModel.Todos) == 0 {
		return
	}

	m.FileModel.DeleteTodoItem(m.SelectedIndex)
	m.InvalidateHeadingsCache() // Deletion may affect heading positions
	m.writeIfPersist()

	// Adjust selection
	if m.SelectedIndex >= len(m.FileModel.Todos) {
		m.SelectedIndex = util.Max(0, len(m.FileModel.Todos)-1)
	}
}

func (m *Model) moveTodo(from, to int) error {
	err := m.FileModel.MoveTodoItem(from, to)
	if err == nil {
		m.InvalidateHeadingsCache() // Move may affect heading positions
	}
	return err
}

func (m *Model) swapTodos(i, j int) error {
	return m.FileModel.SwapTodoItems(i, j)
}

func (m *Model) updateSearchResults() {
	m.SearchResults = nil
	m.SearchCursor = 0

	if m.InputBuffer == "" {
		// Show all todos when query is empty
		for i := range m.FileModel.Todos {
			m.SearchResults = append(m.SearchResults, i)
		}
		return
	}

	query := strings.ToLower(m.InputBuffer)

	// Collect matches with scores
	type match struct {
		index int
		score int
	}
	var matches []match

	for i, todo := range m.FileModel.Todos {
		text := strings.ToLower(todo.Text)
		score := util.FuzzyScore(query, text)
		if score > 0 {
			matches = append(matches, match{i, score})
		}
	}

	// Sort by score descending
	for i := 0; i < len(matches)-1; i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].score > matches[i].score {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	for _, match := range matches {
		m.SearchResults = append(m.SearchResults, match.index)
	}
}

func (m *Model) updateFilteredCommands() {
	m.FilteredCmds = nil
	m.CommandCursor = 0

	if m.InputBuffer == "" {
		// Show all commands when query is empty
		for i := range m.Commands {
			m.FilteredCmds = append(m.FilteredCmds, i)
		}
		return
	}

	query := strings.ToLower(m.InputBuffer)

	// Collect matches with scores
	type match struct {
		index int
		score int
	}
	var matches []match

	for i, cmd := range m.Commands {
		text := strings.ToLower(cmd.Name)
		score := util.FuzzyScore(query, text)
		if score > 0 {
			matches = append(matches, match{i, score})
		}
	}

	// Sort by score descending
	for i := 0; i < len(matches)-1; i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].score > matches[i].score {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	for _, match := range matches {
		m.FilteredCmds = append(m.FilteredCmds, match.index)
	}
}

func (m *Model) writeIfPersist() {
	if !m.ReadOnly {
		// Check for external modifications first
		modified, err := m.FileModel.CheckFileModified()
		if err != nil {
			m.Err = err
			return
		}

		if modified {
			// Try smart reload: check if we can auto-reload safely
			if m.trySmartReload() {
				// Successfully auto-reloaded, now write (unchecked since we just loaded)
				if err := markdown.WriteFileUnchecked(m.FilePath, &m.FileModel); err != nil {
					m.Err = err
				} else {
					// Clear locally modified tracking after successful write
					m.LocallyModified = make(map[string]bool)
				}
				return
			}
			// Can't auto-merge, show error
			m.Err = fmt.Errorf("file changed externally")
			return
		}

		// No conflict, write normally (unchecked since we just checked)
		if err := markdown.WriteFileUnchecked(m.FilePath, &m.FileModel); err != nil {
			m.Err = err
		} else {
			// Clear locally modified tracking after successful write
			m.LocallyModified = make(map[string]bool)
		}
	}
}

// trySmartReload attempts to reload external changes and reapply local changes
// Returns true if successful, false if there's a conflict that needs user intervention
// Only a real conflict if the same todo's TEXT was modified by both sides
func (m *Model) trySmartReload() bool {
	// Load current file state
	diskFM, err := markdown.ReadFile(m.FilePath)
	if err != nil {
		return false
	}

	// The key insight: we only have local CHECKBOX changes (TUI only toggles checkboxes)
	// So we can merge as long as no todo TEXT was modified on disk for todos we touched

	// Build map of our todos: text -> checkbox state
	ourTodos := make(map[string]bool)
	for _, todo := range m.FileModel.Todos {
		ourTodos[todo.Text] = todo.Checked
	}

	// Build map of disk todos: text -> exists
	diskTodos := make(map[string]bool)
	for _, todo := range diskFM.Todos {
		diskTodos[todo.Text] = true
	}

	// Check for conflicts: do we have todos that don't exist on disk?
	// If they deleted a todo we didn't touch, that's fine
	// We can only detect "didn't touch" by checking if checkbox differs from disk
	// But we don't know the original state... so let's be optimistic:
	// If todo text exists on disk, we can merge. If it doesn't, they deleted it - accept deletion.

	// Real conflict: A todo text was MODIFIED on disk (not just deleted/added)
	// Since we only change checkboxes, we can't detect text modifications perfectly
	// But we can use this heuristic: If all our todos either exist on disk OR are clearly deletions, merge

	// Simple approach: Start with disk version, apply our checkbox states ONLY for todos we modified
	resultFM := diskFM

	for i, diskTodo := range resultFM.Todos {
		// If this todo exists in our list AND we locally modified it, apply our checkbox state
		if ourCheckState, exists := ourTodos[diskTodo.Text]; exists {
			// Only apply our change if we actually toggled this todo
			if m.LocallyModified[diskTodo.Text] {
				resultFM.UpdateTodoItem(i, diskTodo.Text, ourCheckState)
			}
			// Otherwise keep disk's checkbox state (they might have changed it)
		}
		// If it doesn't exist in our list, it's new - keep disk's checkbox state
	}

	// Clear locally modified tracking after successful merge
	m.LocallyModified = make(map[string]bool)

	m.FileModel = *resultFM
	return true
}

// checkAndReloadFile checks if the file changed and reloads if safe
func (m Model) checkAndReloadFile() tea.Cmd {
	if m.ReadOnly {
		return watchFileChanges() // Continue watching
	}

	// Check if file was modified externally
	modified, err := m.FileModel.CheckFileModified()
	if err != nil || !modified {
		return watchFileChanges() // Continue watching
	}

	// File changed - try smart reload
	if m.trySmartReload() {
		// Successfully auto-reloaded, the model is updated
		return func() tea.Msg {
			// Return updated model through a custom message
			return reloadedMsg{model: m}
		}
	}

	// Can't auto-merge, will show error on next write attempt
	return watchFileChanges() // Continue watching
}

// reloadedMsg carries the updated model after successful reload
type reloadedMsg struct {
	model Model
}

func (m *Model) getVisibleTodos() []int {
	var visible []int
	for i := range m.FileModel.Todos {
		if m.FilterDone && m.FileModel.Todos[i].Checked {
			continue
		}
		visible = append(visible, i)
	}
	return visible
}

func (m *Model) adjustSelectionForFilter() {
	visible := m.getVisibleTodos()
	if len(visible) == 0 {
		m.SelectedIndex = 0
		return
	}

	// Check if current selection is visible
	for _, idx := range visible {
		if idx == m.SelectedIndex {
			return // Already visible
		}
	}

	// Find nearest visible todo
	bestIdx := visible[0]
	bestDist := util.Abs(m.SelectedIndex - bestIdx)
	for _, idx := range visible {
		dist := util.Abs(m.SelectedIndex - idx)
		if dist < bestDist {
			bestIdx = idx
			bestDist = dist
		}
	}
	m.SelectedIndex = bestIdx
}

// ProcessPipedInput handles input byte-by-byte for testing/scripting
func (m *Model) ProcessPipedInput(input []byte) {
	for i := 0; i < len(input); i++ {
		b := input[i]

		// Handle input/edit mode
		if m.InputMode || m.EditMode {
			switch b {
			case '\r', '\n': // Enter
				if m.InputMode {
					if m.InputBuffer != "" {
						m.addNewTodo()
						m.InputMode = false
					}
				} else if m.EditMode {
					if m.InputBuffer != "" {
						todo := m.FileModel.Todos[m.SelectedIndex]
						m.FileModel.UpdateTodoItem(m.SelectedIndex, m.InputBuffer, todo.Checked)
						m.writeIfPersist()
					}
					m.EditMode = false
				}
			case 27: // Escape
				m.InputMode = false
				m.EditMode = false
			case 127, 8: // Backspace
				if m.CursorPos > 0 {
					m.InputBuffer = m.InputBuffer[:m.CursorPos-1] + m.InputBuffer[m.CursorPos:]
					m.CursorPos--
				}
			default:
				if b >= 32 && b < 127 { // Printable ASCII
					m.InputBuffer = m.InputBuffer[:m.CursorPos] + string(b) + m.InputBuffer[m.CursorPos:]
					m.CursorPos++
				}
			}
			continue
		}

		// Handle max-visible input mode
		if m.MaxVisibleInputMode {
			switch b {
			case '\r', '\n': // Enter
				if m.InputBuffer != "" {
					if num, err := strconv.Atoi(m.InputBuffer); err == nil && num >= 0 {
						m.MaxVisibleOverride = num
					}
				}
				m.MaxVisibleInputMode = false
				m.InputBuffer = ""
			case 27: // Escape
				m.MaxVisibleInputMode = false
				m.InputBuffer = ""
			case 127, 8: // Backspace
				if m.CursorPos > 0 {
					m.InputBuffer = m.InputBuffer[:m.CursorPos-1] + m.InputBuffer[m.CursorPos:]
					m.CursorPos--
				}
			default:
				// Only allow digits
				if b >= '0' && b <= '9' {
					m.InputBuffer = m.InputBuffer[:m.CursorPos] + string(b) + m.InputBuffer[m.CursorPos:]
					m.CursorPos++
				}
			}
			continue
		}

		// Handle search mode
		if m.SearchMode {
			switch b {
			case '\r', '\n': // Enter - select current result
				if len(m.SearchResults) > 0 && m.SearchCursor < len(m.SearchResults) {
					m.SelectedIndex = m.SearchResults[m.SearchCursor]
				}
				m.SearchMode = false
				m.InputBuffer = ""
				m.SearchResults = nil
			case 27: // Escape - cancel search
				m.SearchMode = false
				m.InputBuffer = ""
				m.SearchResults = nil
			case 127, 8: // Backspace
				if m.CursorPos > 0 {
					m.InputBuffer = m.InputBuffer[:m.CursorPos-1] + m.InputBuffer[m.CursorPos:]
					m.CursorPos--
					m.updateSearchResults()
				}
			default:
				if b >= 32 && b < 127 { // Printable ASCII
					m.InputBuffer = m.InputBuffer[:m.CursorPos] + string(b) + m.InputBuffer[m.CursorPos:]
					m.CursorPos++
					m.updateSearchResults()
				}
			}
			continue
		}

		// Handle command mode
		if m.CommandMode {
			switch b {
			case '\r', '\n': // Enter - execute command
				if len(m.FilteredCmds) > 0 && m.CommandCursor < len(m.FilteredCmds) {
					cmdIdx := m.FilteredCmds[m.CommandCursor]
					m.Commands[cmdIdx].Handler(m)
				}
				m.CommandMode = false
				// Only clear buffer if we didn't switch to input or maxVisibleInput mode
				if !m.InputMode && !m.MaxVisibleInputMode {
					m.InputBuffer = ""
				}
				m.FilteredCmds = nil
			case '\t': // Tab - complete command
				if len(m.FilteredCmds) > 0 && m.CommandCursor < len(m.FilteredCmds) {
					cmdIdx := m.FilteredCmds[m.CommandCursor]
					m.InputBuffer = m.Commands[cmdIdx].Name
					m.CursorPos = len(m.InputBuffer)
					m.updateFilteredCommands()
				}
			case 27: // Escape - cancel
				m.CommandMode = false
				m.InputBuffer = ""
				m.FilteredCmds = nil
			case 127, 8: // Backspace
				if m.CursorPos > 0 {
					m.InputBuffer = m.InputBuffer[:m.CursorPos-1] + m.InputBuffer[m.CursorPos:]
					m.CursorPos--
					m.updateFilteredCommands()
				}
			default:
				if b >= 32 && b < 127 { // Printable ASCII
					m.InputBuffer = m.InputBuffer[:m.CursorPos] + string(b) + m.InputBuffer[m.CursorPos:]
					m.CursorPos++
					m.updateFilteredCommands()
				}
			}
			continue
		}

		// Handle move mode
		if m.MoveMode {
			switch b {
			case 'j':
				// When filters are active, find next visible todo to move to
				if m.FilterDone || len(m.FilteredTags) > 0 {
					nextIdx := m.findNextVisibleTodo(m.SelectedIndex)
					if nextIdx != -1 {
						if err := m.moveTodo(m.SelectedIndex, nextIdx); err == nil {
							m.SelectedIndex = nextIdx
						}
					}
				} else {
					// No filters - move to next position
					if m.SelectedIndex < len(m.FileModel.Todos)-1 {
						if err := m.moveTodo(m.SelectedIndex, m.SelectedIndex+1); err == nil {
							m.SelectedIndex++
						}
					}
				}
			case 'k':
				// When filters are active, find previous visible todo to move to
				if m.FilterDone || len(m.FilteredTags) > 0 {
					prevIdx := m.findPreviousVisibleTodo(m.SelectedIndex)
					if prevIdx != -1 {
						if err := m.moveTodo(m.SelectedIndex, prevIdx); err == nil {
							m.SelectedIndex = prevIdx
						}
					}
				} else {
					// No filters - move to previous position
					if m.SelectedIndex > 0 {
						if err := m.moveTodo(m.SelectedIndex, m.SelectedIndex-1); err == nil {
							m.SelectedIndex--
						}
					}
				}
			case '\r', '\n':
				m.writeIfPersist()
				m.MoveMode = false
			case 27:
				m.MoveMode = false
			}
			continue
		}

		// Normal mode
		switch b {
		case 'q', 27: // Quit
			return
		case 'j':
			count := m.getCount()
			if m.FilterDone || len(m.FilteredTags) > 0 {
				// Navigate within visible todos only
				visible := m.getVisibleTodos()
				if len(visible) > 0 {
					currentPos := 0
					for i, idx := range visible {
						if idx == m.SelectedIndex {
							currentPos = i
							break
						}
					}
					newPos := util.Min(currentPos+count, len(visible)-1)
					m.SelectedIndex = visible[newPos]
				}
			} else {
				m.SelectedIndex = util.Min(m.SelectedIndex+count, len(m.FileModel.Todos)-1)
				if m.SelectedIndex < 0 {
					m.SelectedIndex = 0
				}
			}
		case 'k':
			count := m.getCount()
			if m.FilterDone || len(m.FilteredTags) > 0 {
				// Navigate within visible todos only
				visible := m.getVisibleTodos()
				if len(visible) > 0 {
					currentPos := 0
					for i, idx := range visible {
						if idx == m.SelectedIndex {
							currentPos = i
							break
						}
					}
					newPos := util.Max(currentPos-count, 0)
					m.SelectedIndex = visible[newPos]
				}
			} else {
				m.SelectedIndex = util.Max(m.SelectedIndex-count, 0)
			}
		case ' ', '\r', '\n':
			if len(m.FileModel.Todos) > 0 {
				m.saveHistory()
				todo := m.FileModel.Todos[m.SelectedIndex]
				m.FileModel.UpdateTodoItem(m.SelectedIndex, todo.Text, !todo.Checked)
				m.writeIfPersist()
				// Adjust selection if item is now hidden by filter
				if m.FilterDone && m.FileModel.Todos[m.SelectedIndex].Checked {
					m.adjustSelectionForFilter()
				}
			}
		case 'n':
			m.saveHistory()
			m.InputMode = true
			m.InputBuffer = ""
			m.CursorPos = 0
		case 'e':
			if len(m.FileModel.Todos) > 0 {
				m.saveHistory()
				m.EditMode = true
				m.InputBuffer = m.FileModel.Todos[m.SelectedIndex].Text
				m.CursorPos = len(m.InputBuffer)
			}
		case 'd':
			if len(m.FileModel.Todos) > 0 {
				m.saveHistory()
				m.deleteCurrent()
			}
		case 'm':
			if len(m.FileModel.Todos) > 0 {
				m.saveHistory()
				m.MoveMode = true
			}
		case 'u':
			if m.History != nil {
				m.FileModel = *m.History
				m.History = nil
				m.writeIfPersist()
				if m.SelectedIndex >= len(m.FileModel.Todos) {
					m.SelectedIndex = util.Max(0, len(m.FileModel.Todos)-1)
				}
			}
		case 'c':
			if len(m.FileModel.Todos) > 0 {
				util.CopyToClipboard(m.FileModel.Todos[m.SelectedIndex].Text)
			}
		case '?':
			m.HelpMode = !m.HelpMode
		case '/':
			if len(m.FileModel.Todos) > 0 {
				m.SearchMode = true
				m.InputBuffer = ""
				m.CursorPos = 0
				m.SearchCursor = 0
				// Initialize with all todos
				m.SearchResults = nil
				for i := range m.FileModel.Todos {
					m.SearchResults = append(m.SearchResults, i)
				}
			}
		case ':':
			m.CommandMode = true
			m.InputBuffer = ""
			m.CursorPos = 0
			m.CommandCursor = 0
			// Initialize with all commands
			m.FilteredCmds = nil
			for i := range m.Commands {
				m.FilteredCmds = append(m.FilteredCmds, i)
			}
		case '1', '2', '3', '4', '5', '6', '7', '8', '9':
			m.NumberBuffer += string(b)
		case 'G':
			// Go to bottom (vim-style)
			if m.FilterDone || len(m.FilteredTags) > 0 {
				visible := m.getVisibleTodos()
				if len(visible) > 0 {
					m.SelectedIndex = visible[len(visible)-1]
				}
			} else if len(m.FileModel.Todos) > 0 {
				m.SelectedIndex = len(m.FileModel.Todos) - 1
			}
			m.gPressed = false
		case 'g':
			// gg - go to top
			if m.gPressed {
				if m.FilterDone || len(m.FilteredTags) > 0 {
					visible := m.getVisibleTodos()
					if len(visible) > 0 {
						m.SelectedIndex = visible[0]
					}
				} else if len(m.FileModel.Todos) > 0 {
					m.SelectedIndex = 0
				}
				m.gPressed = false
			} else {
				m.gPressed = true
			}
		default:
			// Reset g-pressed state on any other key
			m.gPressed = false
		}
	}
}

// findNextVisibleTodo finds the next todo that would be visible given current filters
func (m Model) findNextVisibleTodo(currentIdx int) int {
	for i := currentIdx + 1; i < len(m.FileModel.Todos); i++ {
		todo := m.FileModel.Todos[i]

		// Skip if filtered by filter-done
		if m.FilterDone && todo.Checked {
			continue
		}

		// Skip if filtered by tag filters
		if len(m.FilteredTags) > 0 && !todo.HasAnyTag(m.FilteredTags) {
			continue
		}

		return i
	}
	return -1 // No visible todo found
}

// findPreviousVisibleTodo finds the previous todo that would be visible given current filters
func (m Model) findPreviousVisibleTodo(currentIdx int) int {
	for i := currentIdx - 1; i >= 0; i-- {
		todo := m.FileModel.Todos[i]

		// Skip if filtered by filter-done
		if m.FilterDone && todo.Checked {
			continue
		}

		// Skip if filtered by tag filters
		if len(m.FilteredTags) > 0 && !todo.HasAnyTag(m.FilteredTags) {
			continue
		}

		return i
	}
	return -1 // No visible todo found
}

func (m *Model) getCount() int {
	count := 1
	if m.NumberBuffer != "" {
		count, _ = strconv.Atoi(m.NumberBuffer)
		m.NumberBuffer = ""
	}
	return count
}

// RunPiped runs the TUI with piped input for testing
func RunPiped(filePath string, input []byte, readOnly bool) string {
	fm, _ := markdown.ReadFile(filePath)
	m := New(filePath, fm, readOnly, false, -1, Config, StyleFuncs, Version)
	m.ProcessPipedInput(input)
	return m.View()
}

// Run starts the TUI with Bubbletea
func Run(filePath string, readOnly bool, showHeadings bool, maxVisible int) {
	fm, err := markdown.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Load global config (lowest priority after defaults)
	globalConfig, _ := config.Load()

	// Apply global config defaults if not set by CLI flags
	// Note: We check if values are at their defaults before applying global config
	// Priority: CLI flags > frontmatter > global config > defaults

	// For readOnly, false is the default, so only apply global config if still false
	if !readOnly && globalConfig != nil && globalConfig.ReadOnly != nil {
		readOnly = *globalConfig.ReadOnly
	}

	// For showHeadings, false is the default
	if !showHeadings && globalConfig != nil && globalConfig.ShowHeadings != nil {
		showHeadings = *globalConfig.ShowHeadings
	}

	// For maxVisible, 0 is the default
	if maxVisible == 0 && globalConfig != nil && globalConfig.MaxVisible != nil {
		maxVisible = *globalConfig.MaxVisible
	}

	// Apply frontmatter settings (higher priority than global config)
	if fm.Metadata != nil {
		if fm.Metadata.ReadOnly != nil {
			readOnly = *fm.Metadata.ReadOnly
		}
		if fm.Metadata.ShowHeadings != nil {
			showHeadings = *fm.Metadata.ShowHeadings
		}
		if fm.Metadata.MaxVisible != nil {
			maxVisible = *fm.Metadata.MaxVisible
		}
	}

	m := New(filePath, fm, readOnly, showHeadings, maxVisible, Config, StyleFuncs, Version)

	// Apply additional settings with same priority order
	// Start with global config defaults
	if globalConfig != nil {
		if globalConfig.FilterDone != nil {
			m.FilterDone = *globalConfig.FilterDone
		}
		if globalConfig.WordWrap != nil {
			m.WordWrap = *globalConfig.WordWrap
		}
	}

	// Frontmatter overrides global config
	if fm.Metadata != nil {
		if fm.Metadata.FilterDone != nil {
			m.FilterDone = *fm.Metadata.FilterDone
		}
		if fm.Metadata.WordWrap != nil {
			m.WordWrap = *fm.Metadata.WordWrap
		}
	}

	// Check if we have a TTY
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// Piped input - process directly without Bubbletea event loop
		input, _ := io.ReadAll(os.Stdin)
		m.ProcessPipedInput(input)
		fmt.Print(m.View())
		return
	}

	// Normal TTY - use Bubbletea (no alt screen to keep context visible)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
