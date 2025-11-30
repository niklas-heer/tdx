package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/niklas-heer/tdx/internal/markdown"
)

// testEmptyModel creates a test model with no todos (empty file)
func testEmptyModel() Model {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{},
	}
	return New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
}

func TestEmptyFile_PressN_ShowsInputField(t *testing.T) {
	m := testEmptyModel()

	// Verify we start with no todos
	if len(m.FileModel.Todos) != 0 {
		t.Fatalf("Expected 0 todos, got %d", len(m.FileModel.Todos))
	}

	// Press N to add new task at end
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}}
	result, _ := m.Update(msg)
	m = result.(Model)

	// Verify we're in input mode
	if !m.InputMode {
		t.Error("Expected InputMode to be true after pressing N")
	}

	// Verify InsertAfterCursor is false (N appends at end)
	if m.InsertAfterCursor {
		t.Error("Expected InsertAfterCursor to be false for N key")
	}

	// Render the view and check that input field is visible
	view := m.View()

	// The view should contain the input field indicator (checkbox for new task)
	// When in input mode, we should see "[ ]" for the new task being entered
	if !strings.Contains(view, "[ ]") {
		t.Errorf("Expected view to contain input field checkbox '[ ]', got:\n%s", view)
	}

	// Should NOT show "No todos" message when in input mode
	if strings.Contains(view, "No todos") {
		t.Errorf("Should not show 'No todos' message when in input mode, got:\n%s", view)
	}
}

func TestEmptyFile_PressSmallN_ShowsInputField(t *testing.T) {
	m := testEmptyModel()

	// Press n to add new task after cursor
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	result, _ := m.Update(msg)
	m = result.(Model)

	// Verify we're in input mode
	if !m.InputMode {
		t.Error("Expected InputMode to be true after pressing n")
	}

	// Verify InsertAfterCursor is true (n inserts after cursor)
	if !m.InsertAfterCursor {
		t.Error("Expected InsertAfterCursor to be true for n key")
	}

	// Render the view and check that input field is visible
	view := m.View()

	// The view should contain the input field
	if !strings.Contains(view, "[ ]") {
		t.Errorf("Expected view to contain input field checkbox '[ ]', got:\n%s", view)
	}
}

func TestEmptyFile_InputMode_CanTypeAndSubmit(t *testing.T) {
	m := testEmptyModel()

	// Press N to add new task
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}}
	result, _ := m.Update(msg)
	m = result.(Model)

	// Type some text
	for _, r := range "My first task" {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		result, _ = m.Update(msg)
		m = result.(Model)
	}

	// Verify input buffer has the text
	if m.InputBuffer != "My first task" {
		t.Errorf("Expected InputBuffer to be 'My first task', got '%s'", m.InputBuffer)
	}

	// Press Enter to submit
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	result, _ = m.Update(enterMsg)
	m = result.(Model)

	// Verify input mode is off
	if m.InputMode {
		t.Error("Expected InputMode to be false after pressing Enter")
	}

	// Verify we now have one todo
	if len(m.FileModel.Todos) != 1 {
		t.Fatalf("Expected 1 todo after submission, got %d", len(m.FileModel.Todos))
	}

	// Verify the todo text
	if m.FileModel.Todos[0].Text != "My first task" {
		t.Errorf("Expected todo text 'My first task', got '%s'", m.FileModel.Todos[0].Text)
	}
}
