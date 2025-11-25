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

func TestIsTodoVisible_NoFilters(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task 1", Checked: false},
			{Text: "Task 2", Checked: true},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	if !m.isTodoVisible(0) {
		t.Error("Unchecked todo should be visible without filters")
	}
	if !m.isTodoVisible(1) {
		t.Error("Checked todo should be visible without filters")
	}
}

func TestIsTodoVisible_FilterDone(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task 1", Checked: false},
			{Text: "Task 2", Checked: true},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.FilterDone = true

	if !m.isTodoVisible(0) {
		t.Error("Unchecked todo should be visible with FilterDone")
	}
	if m.isTodoVisible(1) {
		t.Error("Checked todo should NOT be visible with FilterDone")
	}
}

func TestIsTodoVisible_TagFilters(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task 1 #work", Checked: false, Tags: []string{"work"}},
			{Text: "Task 2 #home", Checked: false, Tags: []string{"home"}},
			{Text: "Task 3", Checked: false, Tags: []string{}},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.FilteredTags = []string{"work"}

	if !m.isTodoVisible(0) {
		t.Error("Task with #work tag should be visible when filtering for work")
	}
	if m.isTodoVisible(1) {
		t.Error("Task with #home tag should NOT be visible when filtering for work")
	}
	if m.isTodoVisible(2) {
		t.Error("Task without tags should NOT be visible when filtering for work")
	}
}

func TestIsTodoVisible_BothFilters(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task 1 #work", Checked: false, Tags: []string{"work"}},
			{Text: "Task 2 #work", Checked: true, Tags: []string{"work"}},
			{Text: "Task 3 #home", Checked: false, Tags: []string{"home"}},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.FilterDone = true
	m.FilteredTags = []string{"work"}

	if !m.isTodoVisible(0) {
		t.Error("Unchecked task with #work should be visible")
	}
	if m.isTodoVisible(1) {
		t.Error("Checked task with #work should NOT be visible (FilterDone)")
	}
	if m.isTodoVisible(2) {
		t.Error("Task with #home should NOT be visible (wrong tag)")
	}
}

func TestHasActiveFilters(t *testing.T) {
	fm := &markdown.FileModel{Todos: []markdown.Todo{}}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	if m.hasActiveFilters() {
		t.Error("Should return false with no filters")
	}

	m.FilterDone = true
	if !m.hasActiveFilters() {
		t.Error("Should return true with FilterDone")
	}

	m.FilterDone = false
	m.FilteredTags = []string{"work"}
	if !m.hasActiveFilters() {
		t.Error("Should return true with FilteredTags")
	}

	m.FilterDone = true
	if !m.hasActiveFilters() {
		t.Error("Should return true with both filters")
	}
}

func TestGetVisibleTodos_TagFilters(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task 1 #work", Checked: false, Tags: []string{"work"}},
			{Text: "Task 2 #home", Checked: false, Tags: []string{"home"}},
			{Text: "Task 3 #work", Checked: false, Tags: []string{"work"}},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.FilteredTags = []string{"work"}

	visible := m.getVisibleTodos()

	if len(visible) != 2 {
		t.Errorf("With tag filter, should show 2 todos with #work, got %d", len(visible))
	}
	if visible[0] != 0 || visible[1] != 2 {
		t.Errorf("Expected indices [0, 2], got %v", visible)
	}
}

func TestDeleteCurrent_WithFilterDone_MovesToNearestVisible(t *testing.T) {
	content := `- [ ] Task 1
- [ ] Task 2
- [x] Task 3
- [x] Task 4
- [ ] Task 5`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.FilterDone = true
	m.SelectedIndex = 1 // "Task 2"

	// Delete Task 2 - cursor should move to a visible task
	// because Task 3 and Task 4 are hidden
	m.deleteCurrent()

	// After deletion: Task 1 (0), Task 3 (1, hidden), Task 4 (2, hidden), Task 5 (3)
	// Visible: 0, 3. Cursor was at 1, nearest visible is 0 or 3
	if !m.isTodoVisible(m.SelectedIndex) {
		t.Errorf("After delete, cursor at %d is not visible", m.SelectedIndex)
	}
}

func TestDeleteCurrent_WithTagFilter_MovesToNearestVisible(t *testing.T) {
	content := `- [ ] Task 1 #work
- [ ] Task 2 #work
- [ ] Task 3 #home
- [ ] Task 4 #work`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.FilteredTags = []string{"work"}
	m.SelectedIndex = 1 // "Task 2 #work"

	m.deleteCurrent()

	// After deletion: Task 1 (0), Task 3 (1, hidden), Task 4 (2)
	// Cursor should be on a visible task
	if !m.isTodoVisible(m.SelectedIndex) {
		t.Errorf("After delete, cursor at %d is not visible", m.SelectedIndex)
	}
}

func TestDeleteCurrent_LastVisibleTask_MovesToPreviousVisible(t *testing.T) {
	content := `- [ ] Task 1
- [x] Task 2
- [ ] Task 3`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.FilterDone = true
	m.SelectedIndex = 2 // "Task 3" - last visible

	m.deleteCurrent()

	// After deletion: Task 1 (0), Task 2 (1, hidden)
	// Cursor should move to Task 1 (0)
	if m.SelectedIndex != 0 {
		t.Errorf("After deleting last visible, cursor should be 0, got %d", m.SelectedIndex)
	}
	if !m.isTodoVisible(m.SelectedIndex) {
		t.Errorf("Cursor at %d is not visible", m.SelectedIndex)
	}
}

// ==================== Move Tests with Filters ====================

func TestHandleMoveKey_WithFilterDone_MoveDown(t *testing.T) {
	content := `- [ ] Task A
- [x] Task B
- [x] Task C
- [ ] Task D`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.MoveMode = true
	m.FilterDone = true
	m.SelectedIndex = 0 // Task A (visible)

	// Visible: [0, 3] (Task A, Task D)
	// With granular movement, one 'j' moves one position (swaps with Task B)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	result, _ := m.handleMoveKey(msg)
	m = result.(Model)

	// Verify cursor moved to index 1 (granular movement)
	if m.SelectedIndex != 1 {
		t.Errorf("After one move, cursor should be at 1, got %d", m.SelectedIndex)
	}

	// Task A should now be at index 1 (swapped with Task B)
	if m.FileModel.Todos[1].Text != "Task A" {
		t.Errorf("Task A should be at index 1 after one move, found %s", m.FileModel.Todos[1].Text)
	}

	// Task B should now be at index 0
	if m.FileModel.Todos[0].Text != "Task B" {
		t.Errorf("Task B should be at index 0 after swap, found %s", m.FileModel.Todos[0].Text)
	}
}

func TestHandleMoveKey_WithFilterDone_MoveUp(t *testing.T) {
	content := `- [ ] Task A
- [x] Task B
- [x] Task C
- [ ] Task D`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.MoveMode = true
	m.FilterDone = true
	m.SelectedIndex = 3 // Task D (visible)

	// Visible: [0, 3] (Task A, Task D)
	// With granular movement, one 'k' moves one position (swaps with Task C)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	result, _ := m.handleMoveKey(msg)
	m = result.(Model)

	// Verify cursor moved to index 2 (granular movement)
	if m.SelectedIndex != 2 {
		t.Errorf("After one move up, cursor should be at 2, got %d", m.SelectedIndex)
	}

	// Task D should now be at index 2 (swapped with Task C)
	if m.FileModel.Todos[2].Text != "Task D" {
		t.Errorf("Task D should be at index 2 after one move, found %s", m.FileModel.Todos[2].Text)
	}

	// Task C should now be at index 3
	if m.FileModel.Todos[3].Text != "Task C" {
		t.Errorf("Task C should be at index 3 after swap, found %s", m.FileModel.Todos[3].Text)
	}
}

func TestHandleMoveKey_WithFilterDone_ConsecutiveMoves(t *testing.T) {
	content := `- [ ] Task A
- [x] Task B
- [ ] Task C
- [ ] Task D`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.MoveMode = true
	m.FilterDone = true
	m.SelectedIndex = 0 // Task A

	// Visible: [0, 2, 3] (Task A, Task C, Task D)
	// With granular movement, need 3 moves to get Task A to the end

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}

	// First move: Task A swaps with Task B (index 0 -> 1)
	result, _ := m.handleMoveKey(msg)
	m = result.(Model)
	if m.SelectedIndex != 1 {
		t.Errorf("After first move, cursor should be at 1, got %d", m.SelectedIndex)
	}

	// Second move: Task A swaps with Task C (index 1 -> 2)
	result, _ = m.handleMoveKey(msg)
	m = result.(Model)
	if m.SelectedIndex != 2 {
		t.Errorf("After second move, cursor should be at 2, got %d", m.SelectedIndex)
	}

	// Third move: Task A swaps with Task D (index 2 -> 3)
	result, _ = m.handleMoveKey(msg)
	m = result.(Model)
	if m.SelectedIndex != 3 {
		t.Errorf("After third move, cursor should be at 3, got %d", m.SelectedIndex)
	}

	// Task A should now be at the end
	if m.FileModel.Todos[3].Text != "Task A" {
		t.Errorf("Task A should be at index 3 after three moves, found %s", m.FileModel.Todos[3].Text)
	}
}

func TestHandleMoveKey_WithFilterDone_MoveDownThenUp(t *testing.T) {
	content := `- [ ] Task A
- [x] Task B
- [ ] Task C`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.MoveMode = true
	m.FilterDone = true
	m.SelectedIndex = 0 // Task A

	// Visible: [0, 2] (Task A, Task C)

	// Move down
	msgDown := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	result, _ := m.handleMoveKey(msgDown)
	m = result.(Model)

	// Move back up
	msgUp := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	result, _ = m.handleMoveKey(msgUp)
	m = result.(Model)

	if !m.isTodoVisible(m.SelectedIndex) {
		t.Errorf("After move down then up, cursor at %d is not visible", m.SelectedIndex)
	}

	// Task A should be back at the first visible position
	visible := m.getVisibleTodos()
	firstVisibleIdx := visible[0]

	if m.FileModel.Todos[firstVisibleIdx].Text != "Task A" {
		t.Errorf("Task A should be first visible after move down then up, but found %s",
			m.FileModel.Todos[firstVisibleIdx].Text)
	}
}

func TestHandleMoveKey_CursorStaysOnMovedItem(t *testing.T) {
	content := `- [ ] Task A
- [x] Task B
- [ ] Task C
- [x] Task D
- [ ] Task E`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.MoveMode = true
	m.FilterDone = true
	m.SelectedIndex = 0 // Task A

	// Move down
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	result, _ := m.handleMoveKey(msg)
	m = result.(Model)

	// The cursor should be on Task A (the moved item), not some other item
	if m.FileModel.Todos[m.SelectedIndex].Text != "Task A" {
		t.Errorf("Cursor should stay on moved item 'Task A', but is on '%s' at index %d",
			m.FileModel.Todos[m.SelectedIndex].Text, m.SelectedIndex)
	}
}

func TestHandleMoveKey_WithTagFilter_MoveDown(t *testing.T) {
	content := `- [ ] Task A #work
- [ ] Task B #home
- [ ] Task C #work`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.MoveMode = true
	m.FilteredTags = []string{"work"}
	m.SelectedIndex = 0 // Task A #work

	// Visible: [0, 2] (Task A, Task C - both have #work)
	// With granular movement, one 'j' moves one position (swaps with Task B)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	result, _ := m.handleMoveKey(msg)
	m = result.(Model)

	// Cursor should be at index 1 (granular movement)
	if m.SelectedIndex != 1 {
		t.Errorf("After one move, cursor should be at 1, got %d", m.SelectedIndex)
	}

	// Task A should now be at index 1 (swapped with Task B)
	if m.FileModel.Todos[1].Text != "Task A #work" {
		t.Errorf("Task A should be at index 1 after one move, found %s", m.FileModel.Todos[1].Text)
	}

	// Task B should now be at index 0
	if m.FileModel.Todos[0].Text != "Task B #home" {
		t.Errorf("Task B should be at index 0 after swap, found %s", m.FileModel.Todos[0].Text)
	}
}

func TestHandleMoveKey_NoFilterActive_NormalBehavior(t *testing.T) {
	content := `- [ ] Task A
- [x] Task B
- [ ] Task C`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.MoveMode = true
	// No filters active
	m.SelectedIndex = 0

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	result, _ := m.handleMoveKey(msg)
	m = result.(Model)

	// Should move to index 1 (adjacent position, not skipping)
	if m.SelectedIndex != 1 {
		t.Errorf("Without filters, should move to adjacent position 1, got %d", m.SelectedIndex)
	}

	if m.FileModel.Todos[1].Text != "Task A" {
		t.Errorf("Task A should be at index 1, found %s", m.FileModel.Todos[1].Text)
	}
}

// ==================== Visibility Index Helpers Tests ====================

func TestGetVisiblePosition(t *testing.T) {
	content := `- [ ] Task A
- [x] Task B
- [ ] Task C
- [x] Task D
- [ ] Task E`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.FilterDone = true

	// Visible: [0, 2, 4] (Task A, C, E at visible positions 0, 1, 2)
	visible := m.getVisibleTodos()

	if len(visible) != 3 {
		t.Errorf("Expected 3 visible todos, got %d", len(visible))
	}

	// Verify the mapping
	expectedVisible := []int{0, 2, 4}
	for i, expected := range expectedVisible {
		if visible[i] != expected {
			t.Errorf("Visible position %d: expected array index %d, got %d", i, expected, visible[i])
		}
	}
}

func TestMovePreservesVisibleOrder(t *testing.T) {
	content := `- [ ] Task 1
- [x] Done A
- [ ] Task 2
- [x] Done B
- [ ] Task 3`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.MoveMode = true
	m.FilterDone = true

	// Initial visible order: Task 1, Task 2, Task 3 (at indices 0, 2, 4)

	// Get initial visible order
	getVisibleTexts := func() []string {
		var texts []string
		for _, idx := range m.getVisibleTodos() {
			texts = append(texts, m.FileModel.Todos[idx].Text)
		}
		return texts
	}

	initial := getVisibleTexts()
	if len(initial) != 3 {
		t.Fatalf("Expected 3 visible items, got %d", len(initial))
	}

	// Move Task 1 down once (granular: swaps with Done A at index 1)
	m.SelectedIndex = m.getVisibleTodos()[0] // First visible (index 0)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	result, _ := m.handleMoveKey(msg)
	m = result.(Model)

	// After granular move: Done A(0), Task 1(1), Task 2(2), Done B(3), Task 3(4)
	// Visible order is still: Task 1, Task 2, Task 3 (Task 1 just moved past hidden Done A)
	after := getVisibleTexts()

	// With granular movement, visible order stays the same after one move
	// because Task 1 only swapped with a hidden item (Done A)
	expected := []string{"Task 1", "Task 2", "Task 3"}
	for i, exp := range expected {
		if after[i] != exp {
			t.Errorf("After move, visible position %d: expected %s, got %s", i, exp, after[i])
		}
	}

	// Task 1 should now be at index 1
	if m.FileModel.Todos[1].Text != "Task 1" {
		t.Errorf("Task 1 should be at index 1, found %s", m.FileModel.Todos[1].Text)
	}
}
