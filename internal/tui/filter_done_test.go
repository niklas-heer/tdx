package tui

import (
	"strings"
	"testing"

	"github.com/niklas-heer/tdx/internal/markdown"
)

func TestFilterDone_CheckAllThenToggleFilter(t *testing.T) {
	// This test reproduces a bug where:
	// 1. Start with multiple todos (some done, some not)
	// 2. Run check-all to mark all as complete
	// 3. Toggle filter-done ON (hides completed)
	// 4. Toggle filter-done OFF (should show all again)
	// Bug: After step 4, not all todos are visible

	content := `# Todos

- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
- [ ] Task 4
- [ ] Task 5
`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	// Verify initial state - 5 todos visible
	if len(m.FileModel.Todos) != 5 {
		t.Fatalf("Expected 5 todos, got %d", len(m.FileModel.Todos))
	}

	// Step 1: Run check-all command
	for _, cmd := range m.Commands {
		if cmd.Name == "check-all" {
			cmd.Handler(&m)
			break
		}
	}

	// Verify all are now checked
	for i, todo := range m.FileModel.Todos {
		if !todo.Checked {
			t.Errorf("Todo %d should be checked after check-all", i)
		}
	}

	// Step 2: Toggle filter-done ON
	for _, cmd := range m.Commands {
		if cmd.Name == "filter-done" {
			cmd.Handler(&m)
			break
		}
	}

	if !m.FilterDone {
		t.Fatal("FilterDone should be true after first toggle")
	}

	// With filter on, no todos should be visible in view
	view := m.View()
	// All todos are done, so none should show with filter on
	// (They're hidden, but we should not crash)

	// Step 3: Toggle filter-done OFF
	for _, cmd := range m.Commands {
		if cmd.Name == "filter-done" {
			cmd.Handler(&m)
			break
		}
	}

	if m.FilterDone {
		t.Fatal("FilterDone should be false after second toggle")
	}

	// Step 4: All 5 todos should now be visible again
	view = m.View()

	// Count how many tasks appear in the view
	visibleCount := 0
	for i := 1; i <= 5; i++ {
		taskName := "Task " + string(rune('0'+i))
		if strings.Contains(view, taskName) {
			visibleCount++
		}
	}

	if visibleCount != 5 {
		t.Errorf("Expected 5 todos visible after toggling filter off, got %d", visibleCount)
		t.Logf("View:\n%s", view)
	}
}

func TestFilterDone_LargerListWithMaxVisible(t *testing.T) {
	// Test with a larger list and max_visible set - this might expose scrolling issues
	content := `# Todos

- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
- [ ] Task 4
- [ ] Task 5
- [ ] Task 6
- [ ] Task 7
- [ ] Task 8
- [ ] Task 9
- [ ] Task 10
`
	fm := markdown.ParseMarkdown(content)
	// Set max visible to 5 to enable scrolling
	m := New("/tmp/test.md", fm, false, false, 5, testConfig(), testStyles(), "")
	m.TermWidth = 80

	// Verify initial state
	if len(m.FileModel.Todos) != 10 {
		t.Fatalf("Expected 10 todos, got %d", len(m.FileModel.Todos))
	}

	// Check-all
	for _, cmd := range m.Commands {
		if cmd.Name == "check-all" {
			cmd.Handler(&m)
			break
		}
	}

	// Toggle filter-done ON (should hide all since all are done)
	for _, cmd := range m.Commands {
		if cmd.Name == "filter-done" {
			cmd.Handler(&m)
			break
		}
	}

	// Toggle filter-done OFF
	for _, cmd := range m.Commands {
		if cmd.Name == "filter-done" {
			cmd.Handler(&m)
			break
		}
	}

	// Get the view
	view := m.View()

	// Check that we can see todos (with max_visible=5, we should see at least some)
	foundTasks := 0
	for i := 1; i <= 10; i++ {
		taskName := "Task " + string(rune('0'+i))
		if i >= 10 {
			taskName = "Task 10"
		}
		if strings.Contains(view, taskName) {
			foundTasks++
		}
	}

	// With max_visible=5 we should see at least 5 tasks
	if foundTasks < 5 {
		t.Errorf("Expected at least 5 visible tasks, found %d", foundTasks)
		t.Logf("View:\n%s", view)
	}

	// More importantly - check that getVisibleTodos returns all 10
	visible := m.getVisibleTodos()
	if len(visible) != 10 {
		t.Errorf("getVisibleTodos should return 10, got %d", len(visible))
	}
}

func TestFilterDone_ToggleOffInvalidatesTree(t *testing.T) {
	content := `# Todos

- [x] Done 1
- [ ] Todo 1
- [x] Done 2
- [ ] Todo 2
`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	// Toggle filter-done ON
	m.FilterDone = true
	m.InvalidateDocumentTree()

	// Get visible todos (should be 2 - only unchecked)
	visible := m.getVisibleTodos()
	if len(visible) != 2 {
		t.Errorf("With filter on, expected 2 visible, got %d", len(visible))
	}

	// Toggle filter-done OFF
	m.FilterDone = false
	m.InvalidateDocumentTree() // This should be called by the command handler

	// Get visible todos (should be all 4)
	visible = m.getVisibleTodos()
	if len(visible) != 4 {
		t.Errorf("With filter off, expected 4 visible, got %d", len(visible))
	}
}
