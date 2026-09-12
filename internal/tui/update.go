package tui

import (
	"errors"
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/niklas-heer/tdx/internal/config"
	"github.com/niklas-heer/tdx/internal/editor"
	"github.com/niklas-heer/tdx/internal/markdown"
	"github.com/niklas-heer/tdx/internal/util"
)

// FileChangedMsg is sent when the file changes on disk
type FileChangedMsg struct{}

// Update handles all TUI updates
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		resized := m.TermWidth > 0 && m.TermHeight > 0 && (m.TermWidth != msg.Width || m.TermHeight != msg.Height)
		m.TermWidth = msg.Width
		m.TermHeight = msg.Height
		if m.ConflictDiffMode {
			m.clampConflictDiffScroll(len(m.conflictDiffLines()))
		}
		if resized {
			return m, tea.ClearScreen
		}
		return m, nil
	case clipboardCopiedMsg:
		m.CopyFeedback = msg.err == nil
		m.Err = msg.err
		if msg.err != nil {
			return m, nil
		}
		return m, tea.Tick(1500*time.Millisecond, func(time.Time) tea.Msg { return ClearCopyFeedbackMsg{} })
	case clipboardPastedMsg:
		if msg.file != m.FilePath || msg.mode != m.clipboardInputMode() || msg.buffer != m.InputBuffer || msg.cursor != m.CursorPos {
			return m, nil
		}
		if msg.err != nil {
			m.Err = msg.err
			return m, nil
		}
		return m.handlePaste(msg.text)
	case ClearCopyFeedbackMsg:
		m.CopyFeedback = false
		return m, nil
	case FileChangedMsg:
		// File changed on disk - try to auto-reload
		return m, m.checkAndReloadFile()
	case reloadedMsg:
		// Successfully reloaded from disk
		m = msg.model
		m.clearSections()
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
	case tea.PasteMsg:
		return m.handlePaste(msg.Content)
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+d" && !m.VersionsMode {
			return m, tea.Quit
		}
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if m.ConflictDiffMode {
		return m.handleConflictDiffKey(key)
	}

	// Handle error display - any key dismisses
	if m.Err != nil {
		m.Err = nil
		return m, nil
	}

	// Version browser mode takes priority over all other modes.
	if m.VersionsMode {
		return m.handleVersionsKey(msg)
	}

	if m.ViewMode != "" {
		return m.handleViewKey(msg)
	}

	if m.HeadingInput != "" {
		return m.handleHeadingInput(msg)
	}
	if m.SectionsMode {
		return m.handleSectionsKey(msg)
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

	// Handle filter mode (tags)
	if m.FilterMode {
		return m.handleFilterKey(msg)
	}

	// Handle priority filter mode
	if m.PriorityFilterMode {
		return m.handlePriorityFilterKey(msg)
	}

	// Handle due date filter mode
	if m.DueFilterMode {
		return m.handleDueFilterKey(msg)
	}

	// Handle theme picker mode
	if m.ThemeMode {
		return m.handleThemeKey(msg)
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

	// Handle recent files mode
	if m.RecentFilesMode {
		return m.handleRecentFilesKey(msg)
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
	case "s":
		m.openSections()
		return m, nil
	case "S":
		m.clearSections()
		return m, nil
	case "space", "enter", "e", "d", "c", "m", "tab", "shift+tab":
		if !m.isTodoVisible(m.SelectedIndex) {
			return m, nil
		}
	}
	switch key {
	case "esc", "ctrl+c":
		return m, tea.Quit

	case "j", "down":
		if m.hasActiveFilters() || m.ShowHeadings {
			// Use document tree for filtered navigation
			tree := m.GetDocumentTree()
			tree.NavigateDown(count)
			if selectedNode := tree.GetSelectedNode(); selectedNode != nil && selectedNode.Type == DocNodeTodo {
				m.SelectedIndex = selectedNode.TodoIndex
			}
		} else {
			m.SelectedIndex = util.Min(m.SelectedIndex+count, len(m.FileModel.Todos)-1)
			if m.SelectedIndex < 0 {
				m.SelectedIndex = 0
			}
		}

	case "k", "up":
		if m.hasActiveFilters() || m.ShowHeadings {
			// Use document tree for filtered navigation
			tree := m.GetDocumentTree()
			tree.NavigateUp(count)
			if selectedNode := tree.GetSelectedNode(); selectedNode != nil && selectedNode.Type == DocNodeTodo {
				m.SelectedIndex = selectedNode.TodoIndex
			}
		} else {
			m.SelectedIndex = util.Max(m.SelectedIndex-count, 0)
		}

	case "space", "enter":
		if len(m.FileModel.Todos) > 0 {
			m.saveHistory()
			m.Err = m.applyAction(editor.Action{Kind: editor.Toggle, Index: m.SelectedIndex})
			m.writeIfPersist()
			// Adjust selection if item is now hidden by any filter
			if !m.isTodoVisible(m.SelectedIndex) {
				m.SelectedIndex = m.findBestVisibleSelection(m.SelectedIndex)
				m.InvalidateDocumentTree()
			}
		}

	case "n":
		// Insert new todo after cursor position (like vim's 'o')
		m.history.Begin(&m.FileModel)
		m.InputMode = true
		m.InsertAfterCursor = true
		m.InputBuffer = ""
		m.CursorPos = 0

	case "N":
		// Append new todo at end of file (like vim's 'O' but at end)
		m.history.Begin(&m.FileModel)
		m.InputMode = true
		m.InsertAfterCursor = false
		m.InputBuffer = ""
		m.CursorPos = 0

	case "e":
		if len(m.FileModel.Todos) > 0 {
			m.history.Begin(&m.FileModel)
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
			clipboard := m.clipboard()
			text := m.FileModel.Todos[m.SelectedIndex].Text
			m.CopyFeedback = false
			return m, func() tea.Msg { return clipboardCopiedMsg{clipboard.Copy(text)} }
		}

	case "m":
		if len(m.FileModel.Todos) > 0 {
			m.history.Begin(&m.FileModel)
			m.SavedCursorIndex = m.SelectedIndex // Save cursor position for cancel
			m.MoveMode = true
		}

	case "u":
		if m.history.Undo(&m.FileModel) {
			m.RefreshAvailableTags()
			m.clearSections()
			m.InvalidateHeadingsCache()
			m.InvalidateDocumentTree()
			m.writeIfPersist()
			if m.SelectedIndex >= len(m.FileModel.Todos) {
				m.SelectedIndex = util.Max(0, len(m.FileModel.Todos)-1)
			}
			// If filters are active, ensure cursor moves to a visible task
			if m.hasActiveFilters() {
				tree := m.GetDocumentTree()
				if selectedNode := tree.GetSelectedNode(); selectedNode != nil && selectedNode.Type == DocNodeTodo {
					m.SelectedIndex = selectedNode.TodoIndex
				}
			}
		}

	case "?":
		m.HelpMode = true

	case "r":
		// Load and display recent files
		if recentFiles, err := m.Config().Recent.Load(); err == nil {
			recentFiles.SortByScore()
			m.RecentFiles = recentFiles.Files
			m.RecentFilesCursor = 0
			m.RecentFilesSearch = ""
			m.RecentFilesMode = true
		}

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

	case "t":
		// Always allow entering tag filter mode - show helpful message if no tags
		// Refresh available tags to pick up any new tags added during this session
		m.RefreshAvailableTags()
		m.FilterMode = true
		m.TagFilterCursor = 0

	case "p":
		// Always allow entering priority filter mode - show helpful message if no priorities
		m.PriorityFilterMode = true
		m.PriorityFilterCursor = 0

	case "D":
		// Enter due date filter mode (capital D to not conflict with delete)
		m.DueFilterMode = true
		m.DueFilterCursor = 0

	case "G":
		// Go to bottom (vim-style)
		if m.hasActiveFilters() || m.ShowHeadings {
			tree := m.GetDocumentTree()
			tree.NavigateToBottom()
			if selectedNode := tree.GetSelectedNode(); selectedNode != nil && selectedNode.Type == DocNodeTodo {
				m.SelectedIndex = selectedNode.TodoIndex
			}
		} else if len(m.FileModel.Todos) > 0 {
			m.SelectedIndex = len(m.FileModel.Todos) - 1
		}
		m.gPressed = false

	case "g":
		// First g press - wait for second g
		if m.gPressed {
			// gg - go to top
			if m.hasActiveFilters() || m.ShowHeadings {
				tree := m.GetDocumentTree()
				tree.NavigateToTop()
				if selectedNode := tree.GetSelectedNode(); selectedNode != nil && selectedNode.Type == DocNodeTodo {
					m.SelectedIndex = selectedNode.TodoIndex
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

	case "tab":
		// Indent: make current todo a child of its previous sibling
		if len(m.FileModel.Todos) > 0 && !m.ReadOnly {
			m.saveHistory()
			if err := m.applyAction(editor.Action{Kind: editor.Indent, Index: m.SelectedIndex}); err == nil {
				m.InvalidateDocumentTree()
				m.writeIfPersist()
			}
			// Silently ignore errors (e.g., can't indent first item)
		}

	case "shift+tab":
		// Outdent: move current todo up one level in hierarchy
		if len(m.FileModel.Todos) > 0 && !m.ReadOnly {
			m.saveHistory()
			if err := m.applyAction(editor.Action{Kind: editor.Outdent, Index: m.SelectedIndex}); err == nil {
				m.InvalidateDocumentTree()
				m.writeIfPersist()
			}
			// Silently ignore errors (e.g., can't outdent top-level item)
		}
	}

	return m, nil
}

// prevRuneLen returns the byte length of the rune immediately before
// byte offset pos in s. CursorPos is a byte offset, not a rune count,
// so callers must step by this width (not by 1) to stay on a valid
// UTF-8 boundary when the buffer contains multi-byte characters.
func prevRuneLen(s string, pos int) int {
	if pos <= 0 || pos > len(s) {
		return 0
	}
	_, size := utf8.DecodeLastRuneInString(s[:pos])
	return size
}

// nextRuneLen returns the byte length of the rune starting at byte
// offset pos in s. See prevRuneLen for why this matters.
func nextRuneLen(s string, pos int) int {
	if pos < 0 || pos >= len(s) {
		return 0
	}
	_, size := utf8.DecodeRuneInString(s[pos:])
	return size
}

func (m Model) handleInputKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.Text != "" {
		m.insertInputText(msg.Text)
		return m, nil
	}
	key := msg.String()

	switch key {
	case "enter", "ctrl+m":
		if m.InputBuffer == "" {
			m.history.Cancel(&m.FileModel)
		} else {
			m.history.Commit()
		}
		if m.InputMode {
			if m.InputBuffer != "" {
				m.addNewTodo()
			}
			m.InputMode = false
		} else if m.EditMode {
			if m.InputBuffer != "" {
				m.Err = m.applyAction(editor.Action{Kind: editor.Edit, Index: m.SelectedIndex, Text: m.InputBuffer})
				m.InvalidateDocumentTree() // Text change affects document tree
				m.RefreshAvailableTags()   // Edit may add or remove tags
				m.writeIfPersist()
			}
			m.EditMode = false
		}

	case "esc":
		m.InputMode = false
		m.EditMode = false
		if m.history.Cancel(&m.FileModel) {
			m.InvalidateHeadingsCache()
		}

	case "backspace", "ctrl+h":
		if n := prevRuneLen(m.InputBuffer, m.CursorPos); n > 0 {
			m.InputBuffer = m.InputBuffer[:m.CursorPos-n] + m.InputBuffer[m.CursorPos:]
			m.CursorPos -= n
		}

	case "delete":
		if n := nextRuneLen(m.InputBuffer, m.CursorPos); n > 0 {
			m.InputBuffer = m.InputBuffer[:m.CursorPos] + m.InputBuffer[m.CursorPos+n:]
		}

	case "left":
		m.CursorPos -= prevRuneLen(m.InputBuffer, m.CursorPos)

	case "right":
		m.CursorPos += nextRuneLen(m.InputBuffer, m.CursorPos)

	case "home", "ctrl+a":
		m.CursorPos = 0

	case "end", "ctrl+e":
		m.CursorPos = len(m.InputBuffer)

	case "ctrl+v", "ctrl+shift+v", "ctrl+y":
		clipboard := m.clipboard()
		result := clipboardPastedMsg{file: m.FilePath, mode: m.clipboardInputMode(), buffer: m.InputBuffer, cursor: m.CursorPos}
		return m, func() tea.Msg { result.text, result.err = clipboard.Paste(); return result }

	}

	return m, nil
}

func (m Model) handleMaxVisibleInputKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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

func (m Model) handleMoveKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "j", "down", "k", "up":
		down := key == "j" || key == "down"
		action := editor.Action{Kind: editor.Move, Index: m.SelectedIndex}
		if m.hasActiveFilters() || m.ShowHeadings {
			tree := m.GetDocumentTree()
			selected := tree.GetSelectedNode()
			if selected == nil || selected.Type != DocNodeTodo {
				break
			}
			var from, target int
			var after bool
			if down {
				from, target, after = tree.MoveDown()
			} else {
				from, target, after = tree.MoveUp()
			}
			if from < 0 || target < 0 {
				break
			}
			action = editor.Action{Kind: editor.MoveToPosition, Index: from, Target: target, InsertAfter: after}
		} else {
			if action.Index < 0 || action.Index >= len(m.FileModel.Todos) {
				break
			}
			depth := m.FileModel.Todos[action.Index].Depth
			target := action.Index - 1
			if down {
				target = action.Index + 1
				for target < len(m.FileModel.Todos) && m.FileModel.Todos[target].Depth > depth {
					target++
				}
			} else {
				for target >= 0 && m.FileModel.Todos[target].Depth > depth {
					target--
				}
			}
			if target < 0 || target >= len(m.FileModel.Todos) {
				break
			}
			action.Target = target
		}
		index, err := editor.Apply(&m.FileModel, action)
		if err != nil {
			m.Err = err
			break
		}
		m.SelectedIndex = index
		m.InvalidateHeadingsCache()
		m.InvalidateDocumentTree()

	case "enter":
		m.history.Commit()
		m.writeIfPersist()
		m.MoveMode = false

	case "esc":
		if m.history.Cancel(&m.FileModel) {
			m.InvalidateHeadingsCache()
			m.InvalidateDocumentTree()
			m.InvalidateHeadingsCache()
		}
		// Restore cursor to position before entering move mode
		m.SelectedIndex = m.SavedCursorIndex
		m.MoveMode = false
	}

	return m, nil
}

func (m Model) handleSearchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.Text != "" {
		m.insertInputText(msg.Text)
		m.searchPending = true
		return m, searchDebounceCmd()
	}
	key := msg.String()

	// Selection must use the current query even when its debounce timer is pending.
	if m.searchPending {
		switch key {
		case "enter", "down", "up", "ctrl+n", "ctrl+j", "ctrl+p", "ctrl+k":
			m.updateSearchResults()
			m.searchPending = false
		}
	}

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
		if n := prevRuneLen(m.InputBuffer, m.CursorPos); n > 0 {
			m.InputBuffer = m.InputBuffer[:m.CursorPos-n] + m.InputBuffer[m.CursorPos:]
			m.CursorPos -= n
			// Debounce search update
			m.searchPending = true
			return m, searchDebounceCmd()
		}

	default:
		// Insert character
		if msg.Text != "" {
			key := msg.Text
			m.InputBuffer = m.InputBuffer[:m.CursorPos] + key + m.InputBuffer[m.CursorPos:]
			m.CursorPos += len(key)
			// Debounce search update
			m.searchPending = true
			return m, searchDebounceCmd()
		}
	}

	return m, nil
}

func (m Model) handleFilterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "enter", "space":
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

			// Filter change affects document tree
			m.InvalidateDocumentTree()

			// Close filter mode after selection
			m.FilterMode = false
		}

	case "esc":
		m.FilterMode = false

	case "c":
		// Clear all filters
		m.FilteredTags = []string{}
		m.InvalidateDocumentTree()

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

func (m Model) handlePriorityFilterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "enter", "space":
		// Toggle priority filter
		if len(m.AvailablePriorities) > 0 && m.PriorityFilterCursor < len(m.AvailablePriorities) {
			selectedPriority := m.AvailablePriorities[m.PriorityFilterCursor]

			// Check if priority is already in filter
			found := false
			for i, p := range m.FilteredPriorities {
				if p == selectedPriority {
					// Remove priority from filter
					m.FilteredPriorities = append(m.FilteredPriorities[:i], m.FilteredPriorities[i+1:]...)
					found = true
					break
				}
			}

			if !found {
				// Add priority to filter
				m.FilteredPriorities = append(m.FilteredPriorities, selectedPriority)
			}

			// Filter change affects document tree
			m.InvalidateDocumentTree()

			// Close filter mode after selection
			m.PriorityFilterMode = false
		}

	case "esc":
		m.PriorityFilterMode = false

	case "c":
		// Clear all priority filters
		m.FilteredPriorities = []int{}
		m.InvalidateDocumentTree()

	case "down", "ctrl+n", "ctrl+j", "j":
		// Move down in priority list
		if len(m.AvailablePriorities) > 0 && m.PriorityFilterCursor < len(m.AvailablePriorities)-1 {
			m.PriorityFilterCursor++
		}

	case "up", "ctrl+p", "ctrl+k", "k":
		// Move up in priority list
		if m.PriorityFilterCursor > 0 {
			m.PriorityFilterCursor--
		}
	}

	return m, nil
}

// Due date filter options
var dueFilterOptions = []string{"overdue", "today", "week", "all"}

func (m Model) handleDueFilterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "enter", "space":
		// Select due date filter
		if m.DueFilterCursor < len(dueFilterOptions) {
			selectedFilter := dueFilterOptions[m.DueFilterCursor]

			// Toggle filter - if same filter is already active, clear it
			if m.FilteredDueDate == selectedFilter {
				m.FilteredDueDate = ""
			} else {
				m.FilteredDueDate = selectedFilter
			}

			// Filter change affects document tree
			m.InvalidateDocumentTree()

			// Close filter mode after selection
			m.DueFilterMode = false
		}

	case "esc":
		m.DueFilterMode = false

	case "c":
		// Clear due date filter
		m.FilteredDueDate = ""
		m.InvalidateDocumentTree()

	case "down", "ctrl+n", "ctrl+j", "j":
		// Move down in filter list
		if m.DueFilterCursor < len(dueFilterOptions)-1 {
			m.DueFilterCursor++
		}

	case "up", "ctrl+p", "ctrl+k", "k":
		// Move up in filter list
		if m.DueFilterCursor > 0 {
			m.DueFilterCursor--
		}
	}

	return m, nil
}

func (m Model) handleThemeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "enter":
		// Confirm theme selection and save to config
		if len(m.AvailableThemes) > 0 && m.ThemeCursor < len(m.AvailableThemes) {
			selectedTheme := m.AvailableThemes[m.ThemeCursor]
			m.CurrentThemeName = selectedTheme
			// Save theme to config
			if m.ThemeSaveFunc != nil {
				_ = m.ThemeSaveFunc(selectedTheme)
			}
		}
		m.ThemeMode = false
		m.OriginalStyles = nil // Clear saved styles

	case "esc":
		// Cancel and restore original theme
		if m.OriginalStyles != nil {
			m.styles = m.OriginalStyles
		}
		m.ThemeMode = false
		m.OriginalStyles = nil

	case "down", "ctrl+n", "ctrl+j", "j":
		// Move down in theme list and apply preview
		if len(m.AvailableThemes) > 0 && m.ThemeCursor < len(m.AvailableThemes)-1 {
			m.ThemeCursor++
			// Apply theme preview
			if m.ThemeApplyFunc != nil {
				m.styles = m.ThemeApplyFunc(m.AvailableThemes[m.ThemeCursor])
			}
		}

	case "up", "ctrl+p", "ctrl+k", "k":
		// Move up in theme list and apply preview
		if m.ThemeCursor > 0 {
			m.ThemeCursor--
			// Apply theme preview
			if m.ThemeApplyFunc != nil {
				m.styles = m.ThemeApplyFunc(m.AvailableThemes[m.ThemeCursor])
			}
		}
	}

	return m, nil
}

func (m Model) handleCommandKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.Text != "" {
		m.insertInputText(msg.Text)
		m.searchPending = true
		return m, commandDebounceCmd()
	}
	key := msg.String()

	// Selection must use the current query even when its debounce timer is pending.
	if m.searchPending {
		switch key {
		case "enter", "tab", "down", "up", "ctrl+n", "ctrl+j", "ctrl+p", "ctrl+k":
			m.updateFilteredCommands()
			m.searchPending = false
		}
	}

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
		if n := prevRuneLen(m.InputBuffer, m.CursorPos); n > 0 {
			m.InputBuffer = m.InputBuffer[:m.CursorPos-n] + m.InputBuffer[m.CursorPos:]
			m.CursorPos -= n
			// Debounce command filter update
			m.searchPending = true
			return m, commandDebounceCmd()
		}

	default:
		// Insert character
		if msg.Text != "" {
			key := msg.Text
			m.InputBuffer = m.InputBuffer[:m.CursorPos] + key + m.InputBuffer[m.CursorPos:]
			m.CursorPos += len(key)
			// Debounce command filter update
			m.searchPending = true
			return m, commandDebounceCmd()
		}
	}

	return m, nil
}

// Helper functions

func (m *Model) saveHistory() { m.history.Push(&m.FileModel) }

func (m *Model) applyAction(action editor.Action) error {
	_, err := editor.Apply(&m.FileModel, action)
	return err
}

func (m *Model) addNewTodo() {
	action := editor.Action{Kind: editor.Add, Text: m.InputBuffer}
	if m.SectionFocus > 0 && (!m.InsertAfterCursor || !m.isTodoVisible(m.SelectedIndex)) {
		action.Kind, action.Index = editor.AddInSection, m.SectionFocus-1
	} else if m.InsertAfterCursor && len(m.FileModel.Todos) > 0 {
		action.Kind, action.Index = editor.Insert, m.SelectedIndex
	}
	index, err := editor.Apply(&m.FileModel, action)
	if err != nil {
		m.Err = err
		return
	}
	m.SelectedIndex = index
	m.InvalidateHeadingsCache() // New todo may affect heading positions
	m.InvalidateDocumentTree()  // New todo affects document tree
	m.RefreshAvailableTags()    // New todo may introduce new tags
	m.writeIfPersist()
}

// findBestVisibleSelection finds the best visible todo to select when the item at
// hiddenIdx becomes hidden (e.g., toggled to done with filter-done active).
// It considers subtask relationships:
// 1. Next sibling at same depth with same parent
// 2. Previous sibling if no next sibling
// 3. Parent if no siblings remain
// 4. Fall back to next/previous visible item
// Returns the index to select (no adjustment needed since nothing is removed)
func (m *Model) findBestVisibleSelection(hiddenIdx int) int {
	if len(m.FileModel.Todos) == 0 {
		return 0
	}

	todos := m.FileModel.Todos
	hiddenTodo := todos[hiddenIdx]
	hiddenDepth := hiddenTodo.Depth
	hiddenParent := hiddenTodo.ParentIndex

	// Look for next sibling (same parent, same depth, comes after hiddenIdx)
	for i := hiddenIdx + 1; i < len(todos); i++ {
		todo := todos[i]
		// If we hit a shallower depth, we've left the subtree
		if todo.Depth < hiddenDepth {
			break
		}
		// Found a sibling at same depth with same parent
		if todo.Depth == hiddenDepth && todo.ParentIndex == hiddenParent {
			if m.isTodoVisible(i) {
				return i
			}
		}
	}

	// Look for previous sibling (same parent, same depth, comes before hiddenIdx)
	for i := hiddenIdx - 1; i >= 0; i-- {
		todo := todos[i]
		// If we hit a shallower depth, we've left the subtree
		if todo.Depth < hiddenDepth {
			break
		}
		// Found a sibling at same depth with same parent
		if todo.Depth == hiddenDepth && todo.ParentIndex == hiddenParent {
			if m.isTodoVisible(i) {
				return i
			}
		}
	}

	// No siblings found - select parent if this was a subtask
	if hiddenParent >= 0 && hiddenParent < len(todos) {
		if m.isTodoVisible(hiddenParent) {
			return hiddenParent
		}
	}

	// Fall back to default behavior: next visible item or previous
	// Try next item first
	for i := hiddenIdx + 1; i < len(todos); i++ {
		if m.isTodoVisible(i) {
			return i
		}
	}
	// Try previous item
	for i := hiddenIdx - 1; i >= 0; i-- {
		if m.isTodoVisible(i) {
			return i
		}
	}

	// No visible items - return current index (will show empty state)
	return hiddenIdx
}

// findBestSelectionAfterDelete calculates the best todo index to select after deleting
// the todo at deletedIdx. Uses findBestVisibleSelection logic but adjusts indices
// for the deletion.
func (m *Model) findBestSelectionAfterDelete(deletedIdx int) int {
	if len(m.FileModel.Todos) <= 1 {
		return 0 // Will be empty or just one item
	}

	todos := m.FileModel.Todos
	deletedTodo := todos[deletedIdx]
	deletedDepth := deletedTodo.Depth
	deletedParent := deletedTodo.ParentIndex

	// Look for next sibling (same parent, same depth, comes after deletedIdx)
	for i := deletedIdx + 1; i < len(todos); i++ {
		todo := todos[i]
		// If we hit a shallower depth, we've left the subtree
		if todo.Depth < deletedDepth {
			break
		}
		// Found a sibling at same depth with same parent
		if todo.Depth == deletedDepth && todo.ParentIndex == deletedParent {
			if m.isTodoVisible(i) {
				// Adjust for deletion: this index will shift down by 1
				return i - 1
			}
		}
	}

	// Look for previous sibling (same parent, same depth, comes before deletedIdx)
	for i := deletedIdx - 1; i >= 0; i-- {
		todo := todos[i]
		// If we hit a shallower depth, we've left the subtree
		if todo.Depth < deletedDepth {
			break
		}
		// Found a sibling at same depth with same parent
		if todo.Depth == deletedDepth && todo.ParentIndex == deletedParent {
			if m.isTodoVisible(i) {
				// No adjustment needed: this index comes before deletion
				return i
			}
		}
	}

	// No siblings found - select parent if this was a subtask
	if deletedParent >= 0 && deletedParent < len(todos) {
		if m.isTodoVisible(deletedParent) {
			// Parent comes before deletion, no adjustment needed
			return deletedParent
		}
	}

	// Fall back to default behavior: next visible item or previous
	// Try next item first
	for i := deletedIdx + 1; i < len(todos); i++ {
		if m.isTodoVisible(i) {
			return i - 1 // Adjust for deletion
		}
	}
	// Try previous item
	for i := deletedIdx - 1; i >= 0; i-- {
		if m.isTodoVisible(i) {
			return i
		}
	}

	return 0
}

func (m *Model) deleteCurrent() {
	if len(m.FileModel.Todos) == 0 {
		return
	}

	deletedIdx := m.SelectedIndex

	// Calculate the best selection BEFORE deletion while we still have the full todo list
	newSelection := m.findBestSelectionAfterDelete(deletedIdx)

	// Use document tree path for filters/headings to handle visibility
	if m.hasActiveFilters() || m.ShowHeadings {
		tree := m.GetDocumentTree()
		actualDeletedIdx := tree.DeleteSelected()
		if actualDeletedIdx >= 0 {
			deletedIdx = actualDeletedIdx
			// Recalculate with actual deleted index if different
			if actualDeletedIdx != m.SelectedIndex {
				newSelection = m.findBestSelectionAfterDelete(actualDeletedIdx)
			}
		}
	}

	// Perform the deletion
	if err := m.applyAction(editor.Action{Kind: editor.Delete, Index: deletedIdx}); err != nil {
		m.Err = err
		return
	}
	m.InvalidateHeadingsCache()
	m.InvalidateDocumentTree()
	m.RefreshAvailableTags() // Delete may remove tags

	// Set the new selection
	if len(m.FileModel.Todos) == 0 {
		m.SelectedIndex = 0
	} else if newSelection >= len(m.FileModel.Todos) {
		m.SelectedIndex = len(m.FileModel.Todos) - 1
	} else if newSelection < 0 {
		m.SelectedIndex = 0
	} else {
		m.SelectedIndex = newSelection
	}

	m.writeIfPersist()
}

func (m *Model) updateSearchResults() {
	m.SearchResults = nil
	m.SearchCursor = 0

	if m.InputBuffer == "" {
		// Show all todos when query is empty
		for i := range m.FileModel.Todos {
			if !m.isTodoVisible(i) {
				continue
			}
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
		if !m.isTodoVisible(i) {
			continue
		}
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
		local := markdown.SerializeMarkdown(&m.FileModel)
		if err := m.Config().Store.WriteFile(m.FilePath, &m.FileModel); err != nil {
			m.recordSaveError(err, local)
			return
		}
		m.clearConflict()
	}
}

// checkAndReloadFile checks if the file changed and reloads if safe
func (m Model) checkAndReloadFile() tea.Cmd {
	if m.ConflictPending || m.InputMode || m.EditMode || m.MoveMode || m.HeadingInput != "" {
		// Keep the revision captured before interactive input. Refreshing it here
		// would let a later Enter overwrite external changes without a conflict.
		return watchFileChanges() // Continue watching
	}

	// Check if file was modified externally
	modified, err := m.FileModel.CheckFileModified()
	if err != nil || !modified {
		return watchFileChanges() // Continue watching
	}

	// With no pending local candidate, external disk content is authoritative.
	diskFM, err := m.Config().Store.ReadFile(m.FilePath)
	if diskFM == nil {
		return watchFileChanges()
	}
	m.FileModel = *diskFM
	m.RefreshAvailableTags()
	m.history.Clear()
	m.Err = err
	return func() tea.Msg { return reloadedMsg{model: m} }
}

func (m *Model) recordSaveError(err error, localContent string) {
	m.Err = err
	var conflict *markdown.ConflictError
	if errors.As(err, &conflict) {
		m.ConflictPending = true
		m.ConflictLocalContent = localContent
		m.ConflictDiskContent = conflict.DiskContent
		m.ConflictDiffScroll = 0
		m.ConflictDiffMode = true
	}
}

func (m *Model) clearConflict() {
	if errors.Is(m.Err, markdown.ErrFileChanged) {
		m.Err = nil
	}
	m.ConflictDiffMode = false
	m.ConflictDiffScroll = 0
	m.ConflictPending = false
	m.ConflictLocalContent = ""
	m.ConflictDiskContent = ""
}

func (m Model) handleConflictDiffKey(key string) (tea.Model, tea.Cmd) {
	scrollStep := m.TermHeight / 3
	if scrollStep < 1 {
		scrollStep = 1
	}
	switch key {
	case "up", "k", "pgup", "ctrl+u":
		m.ConflictDiffScroll -= scrollStep
		if m.ConflictDiffScroll < 0 {
			m.ConflictDiffScroll = 0
		}
	case "down", "j", "pgdown", "ctrl+d":
		m.ConflictDiffScroll += scrollStep
	case "esc", "q":
		m.ConflictDiffMode = false
		if errors.Is(m.Err, markdown.ErrFileChanged) {
			m.Err = nil
		}
	}
	m.clampConflictDiffScroll(len(m.conflictDiffLines()))
	return m, nil
}

// reloadedMsg carries the updated model after successful reload
type reloadedMsg struct {
	model Model
}

// isTodoVisible returns true if the todo at the given index is visible given current filters
func (m *Model) isTodoVisible(idx int) bool {
	if idx < 0 || idx >= len(m.FileModel.Todos) {
		return false
	}
	if !m.sectionAllowsTodo(idx) {
		return false
	}
	todo := m.FileModel.Todos[idx]

	// Hidden by filter-done
	if m.FilterDone && todo.Checked {
		return false
	}

	// Hidden by tag filters
	if len(m.FilteredTags) > 0 && !todo.HasAnyTag(m.FilteredTags) {
		return false
	}

	// Hidden by priority filters
	if len(m.FilteredPriorities) > 0 && !todo.HasAnyPriority(m.FilteredPriorities) {
		return false
	}

	// Hidden by due date filter
	if m.FilteredDueDate != "" && !todo.HasDueDateFilter(m.FilteredDueDate) {
		return false
	}

	return true
}

// hasActiveFilters returns true if any visibility filter is active
func (m *Model) hasActiveFilters() bool {
	return m.SectionFocus > 0 || len(m.FoldedSections) > 0 || m.FilterDone || len(m.FilteredTags) > 0 || len(m.FilteredPriorities) > 0 || m.FilteredDueDate != ""
}

func (m *Model) getVisibleTodos() []int {
	var visible []int
	for i := range m.FileModel.Todos {
		if m.isTodoVisible(i) {
			visible = append(visible, i)
		}
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

// byteToKeyMsg converts a raw byte to a tea.KeyPressMsg for unified input handling
func byteToKeyMsg(b byte) tea.KeyPressMsg {
	switch b {
	case '\r', '\n':
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case 27: // Escape
		return tea.KeyPressMsg{Code: tea.KeyEsc}
	case 127, 8: // Backspace (DEL and BS)
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case '\t':
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case 4: // Ctrl+D
		return tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl}
	default:
		if b >= 32 && b < 127 { // Printable ASCII
			return tea.KeyPressMsg{Code: rune(b), Text: string(b)}
		}
		// Non-printable, return empty
		return tea.KeyPressMsg{}
	}
}

// ProcessPipedInput decodes UTF-8 and terminal editing keys for testing/scripting.
func (m *Model) ProcessPipedInput(input []byte) {
	for i := 0; i < len(input); i++ {
		b := input[i]
		if strings.HasPrefix(string(input[i:]), "\x1b[200~") {
			content, _, found := strings.Cut(string(input[i+6:]), "\x1b[201~")
			if !found {
				return
			}
			result, _ := m.Update(tea.PasteMsg{Content: content})
			*m = result.(Model)
			if m.SearchMode {
				m.updateSearchResults()
			}
			if m.CommandMode {
				m.updateFilteredCommands()
			}
			m.searchPending = false
			i += 6 + len(content) + 6 - 1
			continue
		}
		msg := byteToKeyMsg(b)
		if b >= utf8.RuneSelf {
			r, size := utf8.DecodeRune(input[i:])
			if r == utf8.RuneError && size == 1 {
				continue
			}
			msg = tea.KeyPressMsg{Code: r, Text: string(r)}
			i += size - 1
		} else if b == 27 {
			for sequence, keyType := range map[string]rune{
				"\x1b[D": tea.KeyLeft, "\x1b[C": tea.KeyRight,
				"\x1b[A": tea.KeyUp, "\x1b[B": tea.KeyDown,
				"\x1b[H": tea.KeyHome, "\x1b[F": tea.KeyEnd,
				"\x1b[3~": tea.KeyDelete,
			} {
				if strings.HasPrefix(string(input[i:]), sequence) {
					msg = tea.KeyPressMsg{Code: keyType}
					i += len(sequence) - 1
					break
				}
			}
		}

		// Skip empty messages (non-printable bytes)
		if msg.Code == 0 && len(msg.Text) == 0 {
			continue
		}

		// Check for quit in normal mode (q or esc without other modes active)
		if !m.InputMode && !m.EditMode && !m.SearchMode && !m.CommandMode &&
			!m.MoveMode && !m.FilterMode && !m.MaxVisibleInputMode && !m.HelpMode && !m.RecentFilesMode && !m.SectionsMode && m.HeadingInput == "" && m.ViewMode == "" {
			if msg.String() == "q" || msg.Code == tea.KeyEsc {
				return
			}
		}

		// Delegate to the unified Update handler
		newModel, _ := m.Update(msg)
		*m = newModel.(Model)

		// In piped mode, execute debounced updates synchronously
		// (normally these would be triggered by tea.Tick after a delay)
		if m.searchPending {
			if m.SearchMode {
				m.updateSearchResults()
			} else if m.CommandMode {
				m.updateFilteredCommands()
			}
			m.searchPending = false
		}
	}
}

// findNextVisibleTodo finds the next todo that would be visible given current filters
func (m Model) findNextVisibleTodo(currentIdx int) int {
	for i := currentIdx + 1; i < len(m.FileModel.Todos); i++ {
		if m.isTodoVisible(i) {
			return i
		}
	}
	return -1 // No visible todo found
}

// findPreviousVisibleTodo finds the previous todo that would be visible given current filters
func (m Model) findPreviousVisibleTodo(currentIdx int) int {
	for i := currentIdx - 1; i >= 0; i-- {
		if m.isTodoVisible(i) {
			return i
		}
	}
	return -1 // No visible todo found
}

// handleRecentFilesInput handles keyboard input in recent files mode
func (m Model) handleRecentFilesInput(key string) (tea.Model, tea.Cmd) {
	// Filter recent files based on search
	filteredFiles := []config.RecentFile{}
	for _, file := range m.RecentFiles {
		if m.RecentFilesSearch == "" || strings.Contains(strings.ToLower(file.Path), strings.ToLower(m.RecentFilesSearch)) {
			filteredFiles = append(filteredFiles, file)
		}
	}

	switch key {
	case "esc":
		// Exit recent files mode
		m.RecentFilesMode = false
		m.RecentFilesSearch = ""
		return m, nil

	case "down":
		if len(filteredFiles) > 0 {
			m.RecentFilesCursor = (m.RecentFilesCursor + 1) % len(filteredFiles)
		}

	case "up":
		if len(filteredFiles) > 0 {
			m.RecentFilesCursor--
			if m.RecentFilesCursor < 0 {
				m.RecentFilesCursor = len(filteredFiles) - 1
			}
		}

	case "enter":
		// Open selected file
		if m.RecentFilesCursor < len(filteredFiles) {
			selectedFile := filteredFiles[m.RecentFilesCursor]

			// Save current file's cursor position before switching
			_ = m.Config().Recent.SaveFile(m.FilePath, m.SelectedIndex)

			// Load the new file
			fm, err := m.Config().Store.ReadFile(selectedFile.Path)
			if err != nil {
				m.Err = err
				m.RecentFilesMode = false
				return m, nil
			}

			// Update model with new file
			m.FilePath = selectedFile.Path
			m.FileModel = *fm
			m.SelectedIndex = 0
			m.SectionFocus = 0
			m.FoldedSections = nil
			m.history.Clear()
			m.RecentFilesMode = false
			m.RecentFilesSearch = ""

			m.FilteredTags = nil
			m.FilteredPriorities = nil
			m.FilteredDueDate = ""
			m.FilterDone = m.Config().Defaults.FilterDone
			m.applyFileMetadata()
			m.RefreshAvailableTags()
			m.ActiveView = ""
			m.activeViewState = nil
			// Invalidate caches to refresh AST, headings, and tree
			m.InvalidateHeadingsCache()
			m.InvalidateDocumentTree()
			m.restoreSavedView()
			m.adjustSelectionForFilter()

			// Restore only after the destination's metadata and saved view have
			// established visibility; otherwise keep the first visible fallback.
			if recentFiles, err := m.Config().Recent.Load(); err == nil {
				if savedPos := recentFiles.GetCursorPosition(selectedFile.Path); savedPos >= 0 && savedPos < len(m.FileModel.Todos) && m.isTodoVisible(savedPos) {
					m.SelectedIndex = savedPos
				}
			}
			m.InvalidateDocumentTree()

			return m, nil
		}

	case "backspace":
		if s := m.RecentFilesSearch; s != "" {
			_, size := utf8.DecodeLastRuneInString(s)
			m.RecentFilesSearch = s[:len(s)-size]
			m.RecentFilesCursor = 0 // Reset cursor when search changes
		}

	default:
		// Add to search buffer (printable characters, but skip leading spaces)
		if r, size := utf8.DecodeRuneInString(key); size == len(key) && r >= 32 && r != utf8.RuneError {
			// Skip leading spaces
			if m.RecentFilesSearch != "" || key != " " {
				m.RecentFilesSearch += key
				m.RecentFilesCursor = 0 // Reset cursor when search changes
			}
		}
	}

	return m, nil
}

// RunPiped runs the TUI with piped input for testing
func (runtime Runtime) RunPiped(filePath string, input []byte, readOnly bool) string {
	fm, err := runtime.store().ReadFile(filePath)
	if err != nil {
		return fmt.Sprintf("Error: %v\n", err)
	}

	// Apply frontmatter settings for showHeadings and maxVisible
	showHeadings := false
	maxVisible := -1
	if fm.Metadata != nil {
		if fm.Metadata.ShowHeadings != nil {
			showHeadings = *fm.Metadata.ShowHeadings
		}
		if fm.Metadata.MaxVisible != nil {
			maxVisible = *fm.Metadata.MaxVisible
		}
		if fm.Metadata.ReadOnly != nil {
			readOnly = *fm.Metadata.ReadOnly
		}
	}

	m := New(filePath, fm, readOnly, showHeadings, maxVisible, runtime.Config, runtime.Styles, runtime.Version)

	// Note: FilterDone and WordWrap are now applied in New() from metadata
	// This ensures cursor positioning happens after filters are applied

	// Try to restore cursor position from recent files (if file content hasn't changed)
	if recentFiles, err := m.Config().Recent.Load(); err == nil {
		if savedPos := recentFiles.GetCursorPosition(filePath); savedPos >= 0 && savedPos < len(m.FileModel.Todos) && m.isTodoVisible(savedPos) {
			m.SelectedIndex = savedPos
			// Invalidate tree to ensure correct positioning
			m.InvalidateDocumentTree()
		}
	}

	m.ProcessPipedInput(input)
	output := m.View().Content

	// Save cursor position to recent files when exiting
	_ = m.Config().Recent.SaveFile(filePath, m.SelectedIndex)

	return ansi.Strip(output)
}

// Run starts the TUI with Bubbletea
func (runtime Runtime) Run(filePath string, readOnly bool, showHeadings bool, maxVisible int) {
	fm, err := runtime.store().ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Config defaults arrive through the runtime from main.go (loaded from config.toml)
	// Priority: CLI flags > frontmatter > config.toml defaults

	// Apply frontmatter settings (higher priority than config.toml)
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

	m := New(filePath, fm, readOnly, showHeadings, maxVisible, runtime.Config, runtime.Styles, runtime.Version)

	// Try to restore cursor position from recent files (if file content hasn't changed)
	if recentFiles, err := m.Config().Recent.Load(); err == nil {
		if savedPos := recentFiles.GetCursorPosition(filePath); savedPos >= 0 && savedPos < len(m.FileModel.Todos) && m.isTodoVisible(savedPos) {
			m.SelectedIndex = savedPos
			// Invalidate tree to ensure correct positioning
			m.InvalidateDocumentTree()
		}
	}

	// Check if we have a TTY
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// Piped input - process directly without Bubbletea event loop
		input, _ := io.ReadAll(os.Stdin)
		m.ProcessPipedInput(input)
		fmt.Print(ansi.Strip(m.View().Content))
		// Save cursor position to recent files
		_ = m.Config().Recent.SaveFile(filePath, m.SelectedIndex)
		return
	}

	// Normal TTY - use Bubbletea (no alt screen to keep context visible)
	m.waitForSize = true
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Save cursor position to recent files when exiting
	if m, ok := finalModel.(Model); ok {
		// Save with current cursor position
		_ = m.Config().Recent.SaveFile(filePath, m.SelectedIndex)
	}
}

// handleVersionsKey handles key events when the version browser modal is active.
func (m Model) handleVersionsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if m.VersionsConfirmMode {
		switch key {
		case "y", "Y":
			return m.restoreSelectedVersion()
		case "n", "N", "esc":
			m.VersionsConfirmMode = false
		}
		return m, nil
	}

	// Calculate pane height for scroll step (half pane height, min 1).
	paneHeight := int(float64(m.TermHeight)*0.9) - 4 // subtract borders/footer rows
	if paneHeight < 2 {
		paneHeight = 2
	}
	scrollStep := paneHeight / 2
	if scrollStep < 1 {
		scrollStep = 1
	}

	switch key {
	case "up", "k":
		if m.VersionsCursor > 0 {
			m.VersionsCursor--
			m.VersionsDiffScroll = 0
		}
	case "down", "j":
		if m.VersionsCursor < len(m.VersionsList)-1 {
			m.VersionsCursor++
			m.VersionsDiffScroll = 0
		}
	case "pgup", "ctrl+u":
		m.VersionsDiffScroll -= scrollStep
		if m.VersionsDiffScroll < 0 {
			m.VersionsDiffScroll = 0
		}
	case "pgdown", "ctrl+d":
		m.VersionsDiffScroll += scrollStep
	case "enter":
		if len(m.VersionsList) > 0 {
			m.VersionsConfirmMode = true
		}
	case "esc":
		m.VersionsMode = false
		m.VersionsConfirmMode = false
	}
	return m, nil
}

// restoreSelectedVersion writes the selected historic version back to disk and
// reloads the FileModel. Called when the user confirms restoration with 'y'.
func (m Model) restoreSelectedVersion() (tea.Model, tea.Cmd) {
	if len(m.VersionsList) == 0 {
		m.VersionsMode = false
		m.VersionsConfirmMode = false
		return m, nil
	}

	cfg := m.Config()
	if cfg == nil || cfg.ReadVersionFunc == nil {
		m.VersionsMode = false
		m.VersionsConfirmMode = false
		return m, nil
	}

	selected := m.VersionsList[m.VersionsCursor]
	content, err := cfg.ReadVersionFunc(m.FilePath, selected.ID)
	if err != nil {
		m.Err = err
		m.VersionsMode = false
		m.VersionsConfirmMode = false
		return m, nil
	}

	// Restore exact bytes only if the active revision is still current.
	saveErr := m.Config().Store.WriteContent(m.FilePath, content, &m.FileModel)
	var postCommit *markdown.PostCommitError
	if saveErr != nil && !errors.As(saveErr, &postCommit) {
		m.recordSaveError(saveErr, content)
		m.VersionsMode = false
		m.VersionsConfirmMode = false
		return m, nil
	}

	// Reload from disk.
	reloaded, readErr := m.Config().Store.ReadFile(m.FilePath)
	if reloaded != nil {
		m.FileModel = *reloaded
		m.applyFileMetadata()
		m.InvalidateHeadingsCache()
		m.InvalidateDocumentTree()
	}
	m.Err = errors.Join(saveErr, readErr)

	m.VersionsMode = false
	m.VersionsConfirmMode = false
	return m, nil
}

func (runtime Runtime) store() markdown.Store {
	if runtime.Config != nil {
		return runtime.Config.Store
	}
	return markdown.Store{}
}
