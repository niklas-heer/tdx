package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/niklas-heer/tdx/internal/markdown"
)

func TestHandleKey_Navigation(t *testing.T) {
	m := testModel([]string{"Task 1", "Task 2", "Task 3"})

	// Move down with j
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	result, _ := m.handleKey(msg)
	m = result.(Model)

	if m.SelectedIndex != 1 {
		t.Errorf("After 'j', SelectedIndex = %d, want 1", m.SelectedIndex)
	}

	// Move down again
	result, _ = m.handleKey(msg)
	m = result.(Model)

	if m.SelectedIndex != 2 {
		t.Errorf("After second 'j', SelectedIndex = %d, want 2", m.SelectedIndex)
	}

	// Move up with k
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	result, _ = m.handleKey(msg)
	m = result.(Model)

	if m.SelectedIndex != 1 {
		t.Errorf("After 'k', SelectedIndex = %d, want 1", m.SelectedIndex)
	}
}

func TestHandleKey_NavigationBounds(t *testing.T) {
	m := testModel([]string{"Task 1", "Task 2"})

	// Try to move up from first item
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	result, _ := m.handleKey(msg)
	m = result.(Model)

	if m.SelectedIndex != 0 {
		t.Errorf("Should not go below 0, SelectedIndex = %d", m.SelectedIndex)
	}

	// Move to last item
	m.SelectedIndex = 1

	// Try to move down from last item
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	result, _ = m.handleKey(msg)
	m = result.(Model)

	if m.SelectedIndex != 1 {
		t.Errorf("Should not go past last item, SelectedIndex = %d", m.SelectedIndex)
	}
}

func TestHandleKey_NavigationWithCount(t *testing.T) {
	m := testModel([]string{"Task 1", "Task 2", "Task 3", "Task 4", "Task 5"})

	// Type '3' then 'j' to move down 3
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}}
	result, _ := m.handleKey(msg)
	m = result.(Model)

	if m.NumberBuffer != "3" {
		t.Errorf("NumberBuffer = %q, want %q", m.NumberBuffer, "3")
	}

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	result, _ = m.handleKey(msg)
	m = result.(Model)

	if m.SelectedIndex != 3 {
		t.Errorf("After '3j', SelectedIndex = %d, want 3", m.SelectedIndex)
	}

	if m.NumberBuffer != "" {
		t.Errorf("NumberBuffer should be cleared, got %q", m.NumberBuffer)
	}
}

func TestHandleKey_EnterInputMode(t *testing.T) {
	m := testModel([]string{"Task 1"})

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	result, _ := m.handleKey(msg)
	m = result.(Model)

	if !m.InputMode {
		t.Error("Should enter InputMode on 'n'")
	}
	if m.InputBuffer != "" {
		t.Errorf("InputBuffer should be empty, got %q", m.InputBuffer)
	}
}

func TestHandleKey_EnterEditMode(t *testing.T) {
	m := testModel([]string{"Task 1"})

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
	result, _ := m.handleKey(msg)
	m = result.(Model)

	if !m.EditMode {
		t.Error("Should enter EditMode on 'e'")
	}
	if m.InputBuffer != "Task 1" {
		t.Errorf("InputBuffer = %q, want %q", m.InputBuffer, "Task 1")
	}
	if m.CursorPos != len("Task 1") {
		t.Errorf("CursorPos = %d, want %d", m.CursorPos, len("Task 1"))
	}
}

func TestHandleKey_EnterMoveMode(t *testing.T) {
	m := testModel([]string{"Task 1", "Task 2"})

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}}
	result, _ := m.handleKey(msg)
	m = result.(Model)

	if !m.MoveMode {
		t.Error("Should enter MoveMode on 'm'")
	}
}

func TestHandleKey_EnterSearchMode(t *testing.T) {
	m := testModel([]string{"Task 1", "Task 2"})

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	result, _ := m.handleKey(msg)
	m = result.(Model)

	if !m.SearchMode {
		t.Error("Should enter SearchMode on '/'")
	}
	if len(m.SearchResults) != 2 {
		t.Errorf("SearchResults should have all items, got %d", len(m.SearchResults))
	}
}

func TestHandleKey_EnterCommandMode(t *testing.T) {
	m := testModel([]string{"Task 1"})

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}}
	result, _ := m.handleKey(msg)
	m = result.(Model)

	if !m.CommandMode {
		t.Error("Should enter CommandMode on ':'")
	}
	if len(m.FilteredCmds) == 0 {
		t.Error("FilteredCmds should have all commands")
	}
}

func TestHandleKey_ToggleHelp(t *testing.T) {
	m := testModel([]string{"Task 1"})

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	result, _ := m.handleKey(msg)
	m = result.(Model)

	if !m.HelpMode {
		t.Error("Should enter HelpMode on '?'")
	}

	// Press ? again to exit
	result, _ = m.handleKey(msg)
	m = result.(Model)

	if m.HelpMode {
		t.Error("Should exit HelpMode on second '?'")
	}
}

func TestHandleKey_VimGG(t *testing.T) {
	m := testModel([]string{"Task 1", "Task 2", "Task 3", "Task 4", "Task 5"})
	m.SelectedIndex = 2

	// Press 'g' once
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
	result, _ := m.handleKey(msg)
	m = result.(Model)

	if !m.gPressed {
		t.Error("gPressed should be true after first 'g'")
	}
	if m.SelectedIndex != 2 {
		t.Errorf("SelectedIndex should not change on first 'g', got %d", m.SelectedIndex)
	}

	// Press 'g' again for gg
	result, _ = m.handleKey(msg)
	m = result.(Model)

	if m.gPressed {
		t.Error("gPressed should be false after 'gg'")
	}
	if m.SelectedIndex != 0 {
		t.Errorf("After 'gg', SelectedIndex = %d, want 0", m.SelectedIndex)
	}
}

func TestHandleKey_VimG(t *testing.T) {
	m := testModel([]string{"Task 1", "Task 2", "Task 3", "Task 4", "Task 5"})
	m.SelectedIndex = 2

	// Press 'G' to go to end
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
	result, _ := m.handleKey(msg)
	m = result.(Model)

	if m.SelectedIndex != 4 {
		t.Errorf("After 'G', SelectedIndex = %d, want 4", m.SelectedIndex)
	}
}

func TestHandleKey_ErrorDismissal(t *testing.T) {
	m := testModel([]string{"Task 1"})
	m.Err = errors.New("test error")

	// Any key should dismiss error
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	result, _ := m.handleKey(msg)
	m = result.(Model)

	if m.Err != nil {
		t.Error("Error should be dismissed after any key")
	}
}

func TestHandleInputKey_InsertCharacter(t *testing.T) {
	m := testModel([]string{})
	m.InputMode = true
	m.InputBuffer = ""
	m.CursorPos = 0

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	result, _ := m.handleInputKey(msg)
	m = result.(Model)

	if m.InputBuffer != "a" {
		t.Errorf("InputBuffer = %q, want %q", m.InputBuffer, "a")
	}
	if m.CursorPos != 1 {
		t.Errorf("CursorPos = %d, want 1", m.CursorPos)
	}
}

func TestHandleInputKey_Backspace(t *testing.T) {
	m := testModel([]string{})
	m.InputMode = true
	m.InputBuffer = "test"
	m.CursorPos = 4

	msg := tea.KeyMsg{Type: tea.KeyBackspace}
	result, _ := m.handleInputKey(msg)
	m = result.(Model)

	if m.InputBuffer != "tes" {
		t.Errorf("InputBuffer = %q, want %q", m.InputBuffer, "tes")
	}
	if m.CursorPos != 3 {
		t.Errorf("CursorPos = %d, want 3", m.CursorPos)
	}
}

func TestHandleInputKey_CursorMovement(t *testing.T) {
	m := testModel([]string{})
	m.InputMode = true
	m.InputBuffer = "test"
	m.CursorPos = 4

	// Move left
	msg := tea.KeyMsg{Type: tea.KeyLeft}
	result, _ := m.handleInputKey(msg)
	m = result.(Model)

	if m.CursorPos != 3 {
		t.Errorf("After left, CursorPos = %d, want 3", m.CursorPos)
	}

	// Move right
	msg = tea.KeyMsg{Type: tea.KeyRight}
	result, _ = m.handleInputKey(msg)
	m = result.(Model)

	if m.CursorPos != 4 {
		t.Errorf("After right, CursorPos = %d, want 4", m.CursorPos)
	}

	// Home
	msg = tea.KeyMsg{Type: tea.KeyHome}
	result, _ = m.handleInputKey(msg)
	m = result.(Model)

	if m.CursorPos != 0 {
		t.Errorf("After home, CursorPos = %d, want 0", m.CursorPos)
	}

	// End
	msg = tea.KeyMsg{Type: tea.KeyEnd}
	result, _ = m.handleInputKey(msg)
	m = result.(Model)

	if m.CursorPos != 4 {
		t.Errorf("After end, CursorPos = %d, want 4", m.CursorPos)
	}
}

func TestHandleInputKey_EscapeCancels(t *testing.T) {
	m := testModel([]string{"Task 1"})
	m.InputMode = true
	m.InputBuffer = "new task"

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	result, _ := m.handleInputKey(msg)
	m = result.(Model)

	if m.InputMode {
		t.Error("InputMode should be false after Escape")
	}
}

func TestHandleSearchKey_FilterResults(t *testing.T) {
	m := testModel([]string{"apple", "banana", "apricot"})
	m.SearchMode = true
	m.InputBuffer = ""
	m.SearchResults = []int{0, 1, 2}

	// Type 'a' to filter
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	result, _ := m.handleSearchKey(msg)
	m = result.(Model)

	if m.InputBuffer != "a" {
		t.Errorf("InputBuffer = %q, want %q", m.InputBuffer, "a")
	}
	if !m.searchPending {
		t.Error("searchPending should be true after typing")
	}
}

func TestHandleSearchKey_SelectResult(t *testing.T) {
	m := testModel([]string{"apple", "banana", "apricot"})
	m.SearchMode = true
	m.SearchResults = []int{2, 0} // apricot, apple
	m.SearchCursor = 1            // pointing to apple (index 0)

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	result, _ := m.handleSearchKey(msg)
	m = result.(Model)

	if m.SearchMode {
		t.Error("SearchMode should be false after Enter")
	}
	if m.SelectedIndex != 0 {
		t.Errorf("SelectedIndex = %d, want 0 (apple)", m.SelectedIndex)
	}
}

func TestHandleMoveKey_MoveDown(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task 1", Checked: false},
			{Text: "Task 2", Checked: false},
			{Text: "Task 3", Checked: false},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.MoveMode = true
	m.SelectedIndex = 0

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	result, _ := m.handleMoveKey(msg)
	m = result.(Model)

	if m.SelectedIndex != 1 {
		t.Errorf("SelectedIndex = %d, want 1", m.SelectedIndex)
	}
	if m.FileModel.Todos[1].Text != "Task 1" {
		t.Errorf("Task 1 should have moved to position 1")
	}
}

func TestHandleMoveKey_ExitOnEnter(t *testing.T) {
	m := testModel([]string{"Task 1", "Task 2"})
	m.MoveMode = true
	m.ReadOnly = true // Prevent actual write

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	result, _ := m.handleMoveKey(msg)
	m = result.(Model)

	if m.MoveMode {
		t.Error("MoveMode should be false after Enter")
	}
}

func TestHandleCommandKey_FilterCommands(t *testing.T) {
	m := testModel([]string{"Task 1"})
	m.CommandMode = true
	m.FilteredCmds = []int{0, 1, 2, 3}

	// Type 'c' to filter commands
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	result, _ := m.handleCommandKey(msg)
	m = result.(Model)

	if m.InputBuffer != "c" {
		t.Errorf("InputBuffer = %q, want %q", m.InputBuffer, "c")
	}
	if !m.searchPending {
		t.Error("searchPending should be true after typing")
	}
}

func TestUpdateSearchResults_EmptyQuery(t *testing.T) {
	m := testModel([]string{"apple", "banana", "cherry"})
	m.InputBuffer = ""

	m.updateSearchResults()

	if len(m.SearchResults) != 3 {
		t.Errorf("Empty query should return all items, got %d", len(m.SearchResults))
	}
}

func TestUpdateSearchResults_WithQuery(t *testing.T) {
	m := testModel([]string{"apple", "banana", "apricot"})
	m.InputBuffer = "ap"

	m.updateSearchResults()

	// Should find apple and apricot
	if len(m.SearchResults) != 2 {
		t.Errorf("Query 'ap' should find 2 items, got %d", len(m.SearchResults))
	}
}

func TestUpdateFilteredCommands_EmptyQuery(t *testing.T) {
	m := testModel([]string{"Task 1"})
	m.InputBuffer = ""

	m.updateFilteredCommands()

	if len(m.FilteredCmds) != len(m.Commands) {
		t.Errorf("Empty query should return all commands, got %d", len(m.FilteredCmds))
	}
}

func TestByteToKeyMsg(t *testing.T) {
	tests := []struct {
		input byte
		want  tea.KeyType
	}{
		{'\r', tea.KeyEnter},
		{'\n', tea.KeyEnter},
		{27, tea.KeyEsc},
		{127, tea.KeyBackspace},
		{8, tea.KeyBackspace},
		{'\t', tea.KeyTab},
		{4, tea.KeyCtrlD},
		{'a', tea.KeyRunes},
		{'Z', tea.KeyRunes},
		{':', tea.KeyRunes},
	}

	for _, tt := range tests {
		msg := byteToKeyMsg(tt.input)
		if msg.Type != tt.want {
			t.Errorf("byteToKeyMsg(%d) = %v, want %v", tt.input, msg.Type, tt.want)
		}
	}
}

func TestByteToKeyMsg_PrintableCharacters(t *testing.T) {
	msg := byteToKeyMsg('a')
	if msg.Type != tea.KeyRunes {
		t.Errorf("Type = %v, want KeyRunes", msg.Type)
	}
	if len(msg.Runes) != 1 || msg.Runes[0] != 'a' {
		t.Errorf("Runes = %v, want ['a']", msg.Runes)
	}
}

func TestByteToKeyMsg_NonPrintable(t *testing.T) {
	// Non-printable bytes should return empty message
	msg := byteToKeyMsg(1) // Ctrl+A raw byte
	if msg.Type != 0 && len(msg.Runes) != 0 {
		t.Errorf("Non-printable should return empty message")
	}
}

func TestProcessPipedInput_BasicNavigation(t *testing.T) {
	m := testModel([]string{"Task 1", "Task 2", "Task 3"})

	m.ProcessPipedInput([]byte("jj")) // Move down twice

	if m.SelectedIndex != 2 {
		t.Errorf("After 'jj', SelectedIndex = %d, want 2", m.SelectedIndex)
	}
}

func TestProcessPipedInput_QuitEarly(t *testing.T) {
	m := testModel([]string{"Task 1", "Task 2", "Task 3"})

	m.ProcessPipedInput([]byte("jqj")) // Move down, quit, (ignored)

	if m.SelectedIndex != 1 {
		t.Errorf("After 'jqj', SelectedIndex = %d, want 1 (q should stop)", m.SelectedIndex)
	}
}

func TestProcessPipedInput_CommandExecution(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task 1", Checked: false},
			{Text: "Task 2", Checked: false},
		},
	}
	m := New("/tmp/test.md", fm, true, false, -1, testConfig(), testStyles(), "") // ReadOnly to prevent writes

	m.ProcessPipedInput([]byte(":check-all\r"))

	// All tasks should be checked
	for i, todo := range m.FileModel.Todos {
		if !todo.Checked {
			t.Errorf("Task %d should be checked after check-all", i)
		}
	}
}

func TestProcessPipedInput_SearchAndSelect(t *testing.T) {
	m := testModel([]string{"apple", "banana", "apricot"})

	m.ProcessPipedInput([]byte("/ap\r")) // Search for "ap" and select

	// Should select apple or apricot (first in results)
	if m.SelectedIndex != 0 && m.SelectedIndex != 2 {
		t.Errorf("After '/ap\\r', should select apple(0) or apricot(2), got %d", m.SelectedIndex)
	}
}

func TestGetVisibleTodos_NoFilter(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task 1", Checked: false},
			{Text: "Task 2", Checked: true},
			{Text: "Task 3", Checked: false},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	visible := m.getVisibleTodos()

	if len(visible) != 3 {
		t.Errorf("Without filter, should show all 3 todos, got %d", len(visible))
	}
}

func TestGetVisibleTodos_FilterDone(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task 1", Checked: false},
			{Text: "Task 2", Checked: true},
			{Text: "Task 3", Checked: false},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.FilterDone = true

	visible := m.getVisibleTodos()

	if len(visible) != 2 {
		t.Errorf("With FilterDone, should show 2 unchecked todos, got %d", len(visible))
	}
}

func TestFindNextVisibleTodo(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task 1", Checked: false},
			{Text: "Task 2", Checked: true},
			{Text: "Task 3", Checked: true},
			{Text: "Task 4", Checked: false},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.FilterDone = true

	next := m.findNextVisibleTodo(0)

	if next != 3 {
		t.Errorf("Next visible after 0 should be 3 (skipping checked), got %d", next)
	}
}

func TestFindPreviousVisibleTodo(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task 1", Checked: false},
			{Text: "Task 2", Checked: true},
			{Text: "Task 3", Checked: true},
			{Text: "Task 4", Checked: false},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.FilterDone = true

	prev := m.findPreviousVisibleTodo(3)

	if prev != 0 {
		t.Errorf("Previous visible before 3 should be 0 (skipping checked), got %d", prev)
	}
}
