package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/niklas-heer/tdx/internal/markdown"
)

// Helper to create a todo with a due date
func todoWithDueDate(text string, daysFromNow int) string {
	date := time.Now().AddDate(0, 0, daysFromNow)
	return text + " @due(" + date.Format("2006-01-02") + ")"
}

func TestDueFilter_OverdueFilter(t *testing.T) {
	// Create todos with various due dates
	content := `# Todos

- [ ] ` + todoWithDueDate("Overdue task", -2) + `
- [ ] ` + todoWithDueDate("Due today", 0) + `
- [ ] ` + todoWithDueDate("Due tomorrow", 1) + `
- [ ] Task without due date
`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	// Verify initial state - 4 todos
	if len(m.FileModel.Todos) != 4 {
		t.Fatalf("Expected 4 todos, got %d", len(m.FileModel.Todos))
	}

	// Set overdue filter
	m.FilteredDueDate = "overdue"
	m.InvalidateDocumentTree()

	// Get visible todos
	visible := m.getVisibleTodos()
	if len(visible) != 1 {
		t.Errorf("Expected 1 visible todo with overdue filter, got %d", len(visible))
	}

	// Verify it's the overdue task
	if len(visible) > 0 {
		todo := m.FileModel.Todos[visible[0]]
		if !strings.Contains(todo.Text, "Overdue task") {
			t.Errorf("Expected overdue task, got %q", todo.Text)
		}
	}
}

func TestDueFilter_TodayFilter(t *testing.T) {
	content := `# Todos

- [ ] ` + todoWithDueDate("Overdue task", -2) + `
- [ ] ` + todoWithDueDate("Due today", 0) + `
- [ ] ` + todoWithDueDate("Due tomorrow", 1) + `
- [ ] Task without due date
`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	// Set today filter
	m.FilteredDueDate = "today"
	m.InvalidateDocumentTree()

	// Get visible todos
	visible := m.getVisibleTodos()
	if len(visible) != 1 {
		t.Errorf("Expected 1 visible todo with today filter, got %d", len(visible))
	}

	// Verify it's today's task
	if len(visible) > 0 {
		todo := m.FileModel.Todos[visible[0]]
		if !strings.Contains(todo.Text, "Due today") {
			t.Errorf("Expected today's task, got %q", todo.Text)
		}
	}
}

func TestDueFilter_WeekFilter(t *testing.T) {
	content := `# Todos

- [ ] ` + todoWithDueDate("Overdue task", -2) + `
- [ ] ` + todoWithDueDate("Due today", 0) + `
- [ ] ` + todoWithDueDate("Due in 3 days", 3) + `
- [ ] ` + todoWithDueDate("Due in 10 days", 10) + `
- [ ] Task without due date
`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	// Set week filter
	m.FilteredDueDate = "week"
	m.InvalidateDocumentTree()

	// Get visible todos - should include today and within 7 days
	visible := m.getVisibleTodos()
	if len(visible) != 2 {
		t.Errorf("Expected 2 visible todos with week filter (today + 3 days), got %d", len(visible))
	}
}

func TestDueFilter_AllFilter(t *testing.T) {
	content := `# Todos

- [ ] ` + todoWithDueDate("Task with due date 1", -2) + `
- [ ] ` + todoWithDueDate("Task with due date 2", 5) + `
- [ ] Task without due date
`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	// Set all filter (any due date)
	m.FilteredDueDate = "all"
	m.InvalidateDocumentTree()

	// Get visible todos - should include only those with due dates
	visible := m.getVisibleTodos()
	if len(visible) != 2 {
		t.Errorf("Expected 2 visible todos with 'all' filter, got %d", len(visible))
	}
}

func TestDueFilter_NoFilter(t *testing.T) {
	content := `# Todos

- [ ] ` + todoWithDueDate("Task with due date", 5) + `
- [ ] Task without due date
`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	// No filter set (empty string)
	m.FilteredDueDate = ""
	m.InvalidateDocumentTree()

	// Get visible todos - should show all
	visible := m.getVisibleTodos()
	if len(visible) != 2 {
		t.Errorf("Expected 2 visible todos with no filter, got %d", len(visible))
	}
}

func TestDueFilter_CombinedWithTagFilter(t *testing.T) {
	content := `# Todos

- [ ] ` + todoWithDueDate("Urgent task", 0) + ` #urgent
- [ ] ` + todoWithDueDate("Regular task", 0) + `
- [ ] Urgent without date #urgent
`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	// Set both filters
	m.FilteredDueDate = "today"
	m.FilteredTags = []string{"urgent"}
	m.InvalidateDocumentTree()

	// Should only show the urgent task due today
	visible := m.getVisibleTodos()
	if len(visible) != 1 {
		t.Errorf("Expected 1 visible todo with combined filters, got %d", len(visible))
	}

	if len(visible) > 0 {
		todo := m.FileModel.Todos[visible[0]]
		if !strings.Contains(todo.Text, "Urgent task") {
			t.Errorf("Expected urgent task, got %q", todo.Text)
		}
	}
}

func TestDueFilter_KeyBinding(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	// Verify initial state - not in due filter mode
	if m.DueFilterMode {
		t.Error("Should not be in due filter mode initially")
	}

	// Simulate pressing 'D'
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	m = newModel.(Model)

	// Should now be in due filter mode
	if !m.DueFilterMode {
		t.Error("Should be in due filter mode after pressing 'D'")
	}

	// Simulate pressing 'Esc' to exit
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(Model)

	// Should exit due filter mode
	if m.DueFilterMode {
		t.Error("Should not be in due filter mode after pressing 'Esc'")
	}
}

func TestDueFilter_SelectFilter(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	// Enter due filter mode
	m.DueFilterMode = true
	m.DueFilterCursor = 0 // "overdue" is first option

	// Simulate pressing Space to select
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = newModel.(Model)

	// Filter should be set
	if m.FilteredDueDate != "overdue" {
		t.Errorf("Expected filter to be 'overdue', got %q", m.FilteredDueDate)
	}

	// Should exit filter mode
	if m.DueFilterMode {
		t.Error("Should exit due filter mode after selection")
	}
}

func TestDueFilter_ClearFilter(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	// Set a filter
	m.FilteredDueDate = "today"
	m.DueFilterMode = true

	// Simulate pressing 'c' to clear
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = newModel.(Model)

	// Filter should be cleared
	if m.FilteredDueDate != "" {
		t.Errorf("Expected filter to be cleared, got %q", m.FilteredDueDate)
	}
}

func TestDueFilter_Navigation(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	// Enter due filter mode
	m.DueFilterMode = true
	m.DueFilterCursor = 0

	// Navigate down
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(Model)

	if m.DueFilterCursor != 1 {
		t.Errorf("Expected cursor at 1 after moving down, got %d", m.DueFilterCursor)
	}

	// Navigate up
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newModel.(Model)

	if m.DueFilterCursor != 0 {
		t.Errorf("Expected cursor at 0 after moving up, got %d", m.DueFilterCursor)
	}
}

func TestDueFilter_HasActiveFilters(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	// No filters active
	if m.hasActiveFilters() {
		t.Error("Should not have active filters initially")
	}

	// Set due date filter
	m.FilteredDueDate = "today"
	if !m.hasActiveFilters() {
		t.Error("Should have active filters after setting due date filter")
	}

	// Clear and set tag filter
	m.FilteredDueDate = ""
	m.FilteredTags = []string{"test"}
	if !m.hasActiveFilters() {
		t.Error("Should have active filters with tag filter")
	}
}

func TestDueFilter_ViewRendering(t *testing.T) {
	content := `# Todos

- [ ] ` + todoWithDueDate("Task with date", 5) + `
`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.TermWidth = 80

	// Render view - should contain the due date
	view := m.View()
	if !strings.Contains(view, "@due(") {
		t.Error("View should contain due date marker")
	}
}

func TestDueFilter_StatusBarIndicator(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.TermWidth = 80

	// Set due date filter
	m.FilteredDueDate = "today"

	// Render view - should show filter indicator
	view := m.View()
	if !strings.Contains(view, "today") {
		t.Error("Status bar should show 'today' filter indicator")
	}
}

func TestDueFilter_OverlayRendering(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.TermWidth = 80
	m.TermHeight = 24

	// Enter due filter mode
	m.DueFilterMode = true

	// Render view - should show overlay
	view := m.View()
	if !strings.Contains(view, "Overdue") {
		t.Error("Due filter overlay should contain 'Overdue' option")
	}
	if !strings.Contains(view, "Today") {
		t.Error("Due filter overlay should contain 'Today' option")
	}
	if !strings.Contains(view, "This Week") {
		t.Error("Due filter overlay should contain 'This Week' option")
	}
}
