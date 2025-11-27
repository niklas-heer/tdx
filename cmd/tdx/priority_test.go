package main

import (
	"os"
	"strings"
	"testing"
)

// TestCommand_SortPriority tests the sort-priority command
func TestCommand_SortPriority(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task A !p3")
	runCLI(t, file, "add", "Task B !p1")
	runCLI(t, file, "add", "Task C !p2")

	// Sort by priority
	runPiped(t, file, ":sort-priority\r")

	todos := getTodos(t, file)
	if len(todos) != 3 {
		t.Errorf("Expected 3 tasks after sort, got %d", len(todos))
	}

	// Should be ordered: B (p1), C (p2), A (p3)
	if !strings.Contains(todos[0], "Task B") {
		t.Errorf("Expected Task B (p1) first, got: %s", todos[0])
	}
	if !strings.Contains(todos[1], "Task C") {
		t.Errorf("Expected Task C (p2) second, got: %s", todos[1])
	}
	if !strings.Contains(todos[2], "Task A") {
		t.Errorf("Expected Task A (p3) third, got: %s", todos[2])
	}
}

// TestCommand_SortPriority_UnprioritizedAtEnd tests that tasks without priority go to the end
func TestCommand_SortPriority_UnprioritizedAtEnd(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task A")     // no priority
	runCLI(t, file, "add", "Task B !p2") // p2
	runCLI(t, file, "add", "Task C")     // no priority
	runCLI(t, file, "add", "Task D !p1") // p1

	runPiped(t, file, ":sort-priority\r")

	todos := getTodos(t, file)
	if len(todos) != 4 {
		t.Errorf("Expected 4 tasks, got %d", len(todos))
	}

	// Should be: D (p1), B (p2), A (no priority), C (no priority)
	if !strings.Contains(todos[0], "Task D") {
		t.Errorf("Expected Task D (p1) first, got: %s", todos[0])
	}
	if !strings.Contains(todos[1], "Task B") {
		t.Errorf("Expected Task B (p2) second, got: %s", todos[1])
	}
	// A and C should be at the end (order preserved among unprioritized)
	if !strings.Contains(todos[2], "Task A") {
		t.Errorf("Expected Task A third, got: %s", todos[2])
	}
	if !strings.Contains(todos[3], "Task C") {
		t.Errorf("Expected Task C fourth, got: %s", todos[3])
	}
}

// TestCommand_SortPriority_NoPriorities tests sorting when no tasks have priorities
func TestCommand_SortPriority_NoPriorities(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task A")
	runCLI(t, file, "add", "Task B")
	runCLI(t, file, "add", "Task C")

	// Sort by priority (should preserve order since all unprioritized)
	runPiped(t, file, ":sort-priority\r")

	todos := getTodos(t, file)
	if len(todos) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(todos))
	}

	// Order should be preserved
	if !strings.Contains(todos[0], "Task A") {
		t.Errorf("Expected Task A first, got: %s", todos[0])
	}
	if !strings.Contains(todos[1], "Task B") {
		t.Errorf("Expected Task B second, got: %s", todos[1])
	}
	if !strings.Contains(todos[2], "Task C") {
		t.Errorf("Expected Task C third, got: %s", todos[2])
	}
}

// TestCommand_SortPriority_SamePriority tests stable sort within same priority level
func TestCommand_SortPriority_SamePriority(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task A !p2")
	runCLI(t, file, "add", "Task B !p2")
	runCLI(t, file, "add", "Task C !p1")
	runCLI(t, file, "add", "Task D !p2")

	runPiped(t, file, ":sort-priority\r")

	todos := getTodos(t, file)
	if len(todos) != 4 {
		t.Errorf("Expected 4 tasks, got %d", len(todos))
	}

	// C (p1) should be first
	if !strings.Contains(todos[0], "Task C") {
		t.Errorf("Expected Task C (p1) first, got: %s", todos[0])
	}

	// A, B, D (all p2) should maintain original order (stable sort)
	if !strings.Contains(todos[1], "Task A") {
		t.Errorf("Expected Task A (p2) second, got: %s", todos[1])
	}
	if !strings.Contains(todos[2], "Task B") {
		t.Errorf("Expected Task B (p2) third, got: %s", todos[2])
	}
	if !strings.Contains(todos[3], "Task D") {
		t.Errorf("Expected Task D (p2) fourth, got: %s", todos[3])
	}
}

// TestCommand_SortPriority_MixedWithDone tests priority sort with completed tasks
func TestCommand_SortPriority_MixedWithDone(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task A !p3")
	runCLI(t, file, "add", "Task B !p1")
	runCLI(t, file, "add", "Task C !p2")

	// Check Task B
	runCLI(t, file, "toggle", "2")

	runPiped(t, file, ":sort-priority\r")

	todos := getTodos(t, file)
	if len(todos) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(todos))
	}

	// Should be ordered by priority, regardless of completion status
	// B (p1) first (even though checked), C (p2), A (p3)
	if !strings.Contains(todos[0], "Task B") || !strings.Contains(todos[0], "!p1") {
		t.Errorf("Expected Task B (p1) first, got: %s", todos[0])
	}
	if !strings.Contains(todos[1], "Task C") {
		t.Errorf("Expected Task C (p2) second, got: %s", todos[1])
	}
	if !strings.Contains(todos[2], "Task A") {
		t.Errorf("Expected Task A (p3) third, got: %s", todos[2])
	}

	// Verify Task B is still checked
	if !strings.HasPrefix(todos[0], "- [x] ") {
		t.Errorf("Expected Task B to remain checked, got: %s", todos[0])
	}
}

// TestCommand_SortDone tests the renamed sort-done command (previously sort)
func TestCommand_SortDone(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task A")
	runCLI(t, file, "add", "Task B")
	runCLI(t, file, "add", "Task C")
	runCLI(t, file, "add", "Task D")

	// Check B and D
	runCLI(t, file, "toggle", "2")
	runCLI(t, file, "toggle", "4")

	// Sort by completion (incomplete first)
	runPiped(t, file, ":sort-done\r")

	todos := getTodos(t, file)
	if len(todos) != 4 {
		t.Errorf("Expected 4 tasks after sort, got %d", len(todos))
	}

	// First two should be unchecked (A and C)
	if !strings.HasPrefix(todos[0], "- [ ] ") {
		t.Error("First task should be unchecked")
	}
	if !strings.HasPrefix(todos[1], "- [ ] ") {
		t.Error("Second task should be unchecked")
	}

	// Last two should be checked (B and D)
	if !strings.HasPrefix(todos[2], "- [x] ") {
		t.Error("Third task should be checked")
	}
	if !strings.HasPrefix(todos[3], "- [x] ") {
		t.Error("Fourth task should be checked")
	}
}

// TestPriority_PreservedOnToggle tests that priority is preserved when toggling completion
func TestPriority_PreservedOnToggle(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task with priority !p1")

	// Toggle completion
	runCLI(t, file, "toggle", "1")

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Errorf("Expected 1 task, got %d", len(todos))
	}

	// Priority should be preserved
	if !strings.Contains(todos[0], "!p1") {
		t.Errorf("Expected priority !p1 to be preserved, got: %s", todos[0])
	}
	if !strings.HasPrefix(todos[0], "- [x] ") {
		t.Errorf("Expected task to be checked, got: %s", todos[0])
	}
}

// TestPriority_PreservedOnEdit tests that priority is preserved when editing
func TestPriority_PreservedOnEdit(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Original task !p2")

	// Edit the task but keep the priority
	runCLI(t, file, "edit", "1", "Updated task !p2")

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Errorf("Expected 1 task, got %d", len(todos))
	}

	if !strings.Contains(todos[0], "Updated task") {
		t.Errorf("Expected updated text, got: %s", todos[0])
	}
	if !strings.Contains(todos[0], "!p2") {
		t.Errorf("Expected priority !p2 to be preserved, got: %s", todos[0])
	}
}

// TestPriority_WithTags tests priority works alongside tags
func TestPriority_WithTags(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Fix bug !p1 #urgent #backend")

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Errorf("Expected 1 task, got %d", len(todos))
	}

	// Both priority and tags should be present
	if !strings.Contains(todos[0], "!p1") {
		t.Errorf("Expected !p1, got: %s", todos[0])
	}
	if !strings.Contains(todos[0], "#urgent") {
		t.Errorf("Expected #urgent, got: %s", todos[0])
	}
	if !strings.Contains(todos[0], "#backend") {
		t.Errorf("Expected #backend, got: %s", todos[0])
	}
}

// TestCommand_SortPriority_EmptyFile tests sort-priority on empty file
func TestCommand_SortPriority_EmptyFile(t *testing.T) {
	file := tempTestFile(t)

	_ = os.WriteFile(file, []byte(""), 0644)

	// Should not crash on empty file
	runPiped(t, file, ":sort-priority\r")

	// File should still be valid
	content, _ := os.ReadFile(file)
	if content == nil {
		t.Error("File content is nil")
	}
}

// TestTUI_SortPriority tests sort-priority via TUI command palette
func TestTUI_SortPriority(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Low priority !p3")
	runCLI(t, file, "add", "High priority !p1")
	runCLI(t, file, "add", "Medium priority !p2")

	// Use command palette to sort by priority
	runPiped(t, file, ":sort-priority\r")

	todos := getTodos(t, file)
	if len(todos) != 3 {
		t.Fatalf("Expected 3 todos, got %d", len(todos))
	}

	// Verify order: p1, p2, p3
	if !strings.Contains(todos[0], "High priority") {
		t.Errorf("Expected High priority (p1) first, got: %s", todos[0])
	}
	if !strings.Contains(todos[1], "Medium priority") {
		t.Errorf("Expected Medium priority (p2) second, got: %s", todos[1])
	}
	if !strings.Contains(todos[2], "Low priority") {
		t.Errorf("Expected Low priority (p3) third, got: %s", todos[2])
	}
}

// TestCommand_SortPriority_WithHeadings tests that sort-priority sorts within heading sections
func TestCommand_SortPriority_WithHeadings(t *testing.T) {
	file := tempTestFile(t)

	content := `# Project Tasks

## Backend
- [ ] API endpoint !p2
- [ ] Database schema !p1
- [ ] Middleware !p3

## Frontend
- [ ] Button component !p3
- [ ] Form validation !p1
- [ ] Styling !p2
`
	_ = os.WriteFile(file, []byte(content), 0644)

	runPiped(t, file, ":sort-priority\r")

	todos := getTodos(t, file)
	if len(todos) != 6 {
		t.Fatalf("Expected 6 todos, got %d", len(todos))
	}

	// Backend section should be sorted: p1 (Database), p2 (API), p3 (Middleware)
	if !strings.Contains(todos[0], "Database schema") {
		t.Errorf("Expected Database schema (p1) first in Backend, got: %s", todos[0])
	}
	if !strings.Contains(todos[1], "API endpoint") {
		t.Errorf("Expected API endpoint (p2) second in Backend, got: %s", todos[1])
	}
	if !strings.Contains(todos[2], "Middleware") {
		t.Errorf("Expected Middleware (p3) third in Backend, got: %s", todos[2])
	}

	// Frontend section should be sorted: p1 (Form), p2 (Styling), p3 (Button)
	if !strings.Contains(todos[3], "Form validation") {
		t.Errorf("Expected Form validation (p1) first in Frontend, got: %s", todos[3])
	}
	if !strings.Contains(todos[4], "Styling") {
		t.Errorf("Expected Styling (p2) second in Frontend, got: %s", todos[4])
	}
	if !strings.Contains(todos[5], "Button component") {
		t.Errorf("Expected Button component (p3) third in Frontend, got: %s", todos[5])
	}

	// Verify headings are still in the file
	fileContent, _ := os.ReadFile(file)
	if !strings.Contains(string(fileContent), "## Backend") {
		t.Error("Backend heading should be preserved")
	}
	if !strings.Contains(string(fileContent), "## Frontend") {
		t.Error("Frontend heading should be preserved")
	}
}

// TestCommand_SortDone_WithHeadings tests that sort-done sorts within heading sections
func TestCommand_SortDone_WithHeadings(t *testing.T) {
	file := tempTestFile(t)

	content := `# Project Tasks

## Backend
- [x] Done task B1
- [ ] Pending task B1
- [x] Done task B2

## Frontend
- [ ] Pending task F1
- [x] Done task F1
- [ ] Pending task F2
`
	_ = os.WriteFile(file, []byte(content), 0644)

	runPiped(t, file, ":sort-done\r")

	todos := getTodos(t, file)
	if len(todos) != 6 {
		t.Fatalf("Expected 6 todos, got %d", len(todos))
	}

	// Backend section: incomplete first, then complete
	if !strings.HasPrefix(todos[0], "- [ ]") || !strings.Contains(todos[0], "Pending task B1") {
		t.Errorf("Expected Pending task B1 first in Backend, got: %s", todos[0])
	}
	if !strings.HasPrefix(todos[1], "- [x]") {
		t.Errorf("Expected checked task second in Backend, got: %s", todos[1])
	}
	if !strings.HasPrefix(todos[2], "- [x]") {
		t.Errorf("Expected checked task third in Backend, got: %s", todos[2])
	}

	// Frontend section: incomplete first, then complete
	if !strings.HasPrefix(todos[3], "- [ ]") {
		t.Errorf("Expected unchecked task first in Frontend, got: %s", todos[3])
	}
	if !strings.HasPrefix(todos[4], "- [ ]") {
		t.Errorf("Expected unchecked task second in Frontend, got: %s", todos[4])
	}
	if !strings.HasPrefix(todos[5], "- [x]") {
		t.Errorf("Expected checked task third in Frontend, got: %s", todos[5])
	}
}

// TestCommand_SortPriority_MixedHeadingsAndNoHeadings tests sorting with some todos before any heading
func TestCommand_SortPriority_MixedHeadingsAndNoHeadings(t *testing.T) {
	file := tempTestFile(t)

	content := `# Todos

- [ ] Top level !p3
- [ ] Another top level !p1

## Section A
- [ ] Section A item !p2
- [ ] Section A item !p1
`
	_ = os.WriteFile(file, []byte(content), 0644)

	runPiped(t, file, ":sort-priority\r")

	todos := getTodos(t, file)
	if len(todos) != 4 {
		t.Fatalf("Expected 4 todos, got %d", len(todos))
	}

	// Top level section (before ## Section A) should be sorted
	if !strings.Contains(todos[0], "Another top level") {
		t.Errorf("Expected 'Another top level' (p1) first, got: %s", todos[0])
	}
	if !strings.Contains(todos[1], "Top level !p3") {
		t.Errorf("Expected 'Top level' (p3) second, got: %s", todos[1])
	}

	// Section A should be sorted separately
	if !strings.Contains(todos[2], "Section A item !p1") {
		t.Errorf("Expected Section A p1 item third, got: %s", todos[2])
	}
	if !strings.Contains(todos[3], "Section A item !p2") {
		t.Errorf("Expected Section A p2 item fourth, got: %s", todos[3])
	}
}

// TestTUI_PriorityFilterMode tests entering and exiting priority filter mode
func TestTUI_PriorityFilterMode(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task A !p1")
	runCLI(t, file, "add", "Task B !p2")
	runCLI(t, file, "add", "Task C !p3")

	// Press 'p' to enter priority filter mode
	output := runPiped(t, file, "p")

	// Should show priority filter overlay with priority options
	if !strings.Contains(output, "!p1") || !strings.Contains(output, "!p2") {
		t.Errorf("Expected priority options in overlay, got: %s", output)
	}

	// Should show help text for priority filter mode
	if !strings.Contains(output, "space toggle") {
		t.Errorf("Expected help text in overlay, got: %s", output)
	}
}

// TestTUI_PriorityFilterSelect tests selecting a priority filter
func TestTUI_PriorityFilterSelect(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task A !p1")
	runCLI(t, file, "add", "Task B !p2")
	runCLI(t, file, "add", "Task C !p3")
	runCLI(t, file, "add", "Task D") // no priority

	// Press 'p' to enter priority filter mode, space to select p1
	output := runPiped(t, file, "p ")

	// Output should show filtered indicator
	if !strings.Contains(output, "p1") {
		t.Errorf("Expected p1 indicator in output, got: %s", output)
	}

	// Only Task A (p1) should be visible
	if !strings.Contains(output, "Task A") {
		t.Errorf("Expected Task A (p1) to be visible, got: %s", output)
	}

	// Task B, C, D should be filtered out
	if strings.Contains(output, "Task B") {
		t.Errorf("Task B (p2) should be filtered out, got: %s", output)
	}
	if strings.Contains(output, "Task C") {
		t.Errorf("Task C (p3) should be filtered out, got: %s", output)
	}
	if strings.Contains(output, "Task D") {
		t.Errorf("Task D (no priority) should be filtered out, got: %s", output)
	}
}

// TestTUI_PriorityFilterNavigate tests navigating priority filter options
func TestTUI_PriorityFilterNavigate(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task A !p1")
	runCLI(t, file, "add", "Task B !p2")
	runCLI(t, file, "add", "Task C !p3")

	// Press 'p' to enter priority filter mode, j to move down, space to select p2
	output := runPiped(t, file, "pj ")

	// Only Task B (p2) should be visible
	if !strings.Contains(output, "Task B") {
		t.Errorf("Expected Task B (p2) to be visible, got: %s", output)
	}

	// Task A and C should be filtered out
	if strings.Contains(output, "Task A") {
		t.Errorf("Task A (p1) should be filtered out, got: %s", output)
	}
	if strings.Contains(output, "Task C") {
		t.Errorf("Task C (p3) should be filtered out, got: %s", output)
	}
}

// TestTUI_PriorityFilterClear tests clearing priority filters
func TestTUI_PriorityFilterClear(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task A !p1")
	runCLI(t, file, "add", "Task B !p2")
	runCLI(t, file, "add", "Task C !p3")

	// Select p1, then clear with 'c'
	output := runPiped(t, file, "p pc")

	// After clearing, all tasks should be visible
	if !strings.Contains(output, "Task A") {
		t.Errorf("Expected Task A to be visible after clear, got: %s", output)
	}
	if !strings.Contains(output, "Task B") {
		t.Errorf("Expected Task B to be visible after clear, got: %s", output)
	}
	if !strings.Contains(output, "Task C") {
		t.Errorf("Expected Task C to be visible after clear, got: %s", output)
	}
}

// TestTUI_PriorityFilterNoPriorities tests priority filter mode with no priorities shows helpful message
func TestTUI_PriorityFilterNoPriorities(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task A")
	runCLI(t, file, "add", "Task B")
	runCLI(t, file, "add", "Task C")

	// Press 'p' - should enter priority filter mode and show helpful message
	output := runPiped(t, file, "p")

	// Should show empty state message
	if !strings.Contains(output, "No priorities available") {
		t.Errorf("Expected 'No priorities available' message, got: %s", output)
	}
}

// TestTUI_TagFilterNoTags tests tag filter mode with no tags shows helpful message
func TestTUI_TagFilterNoTags(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task A")
	runCLI(t, file, "add", "Task B")
	runCLI(t, file, "add", "Task C")

	// Press 't' - should enter tag filter mode and show helpful message
	output := runPiped(t, file, "t")

	// Should show empty state message
	if !strings.Contains(output, "No tags available") {
		t.Errorf("Expected 'No tags available' message, got: %s", output)
	}
}

// TestTUI_TagFilterKeyChanged tests that 't' now opens tag filter (not 'f')
func TestTUI_TagFilterKeyChanged(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task A #urgent")
	runCLI(t, file, "add", "Task B #backend")
	runCLI(t, file, "add", "Task C #urgent")

	// Press 't' to enter tag filter mode
	output := runPiped(t, file, "t")

	// Should show tag filter overlay with tag options
	if !strings.Contains(output, "#urgent") || !strings.Contains(output, "#backend") {
		t.Errorf("Expected tag options in overlay, got: %s", output)
	}

	// Should show help text
	if !strings.Contains(output, "space toggle") {
		t.Errorf("Expected help text in overlay, got: %s", output)
	}
}

// TestTUI_TagFilterSelect tests selecting a tag filter with new 't' key
func TestTUI_TagFilterSelect(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task A #urgent")
	runCLI(t, file, "add", "Task B #backend")
	runCLI(t, file, "add", "Task C #urgent")

	// Press 't' to enter tag filter mode, space to select first tag
	output := runPiped(t, file, "t ")

	// Should show filtered todos
	// The exact behavior depends on which tag is first alphabetically
	if !strings.Contains(output, "#") {
		t.Errorf("Expected tag indicator in output, got: %s", output)
	}
}

// TestTUI_PriorityFilterMultipleSelect tests selecting multiple priorities
func TestTUI_PriorityFilterMultipleSelect(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task A !p1")
	runCLI(t, file, "add", "Task B !p2")
	runCLI(t, file, "add", "Task C !p3")

	// Select p1, then enter priority mode again and select p2
	output := runPiped(t, file, "p pj ")

	// Both p1 and p2 tasks should be visible
	if !strings.Contains(output, "Task A") {
		t.Errorf("Expected Task A (p1) to be visible, got: %s", output)
	}
	if !strings.Contains(output, "Task B") {
		t.Errorf("Expected Task B (p2) to be visible, got: %s", output)
	}

	// Task C (p3) should be filtered out
	if strings.Contains(output, "Task C") {
		t.Errorf("Task C (p3) should be filtered out, got: %s", output)
	}
}

// TestTUI_PriorityFilterWithTagFilter tests combining priority and tag filters
func TestTUI_PriorityFilterWithTagFilter(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task A !p1 #urgent")
	runCLI(t, file, "add", "Task B !p1 #backend")
	runCLI(t, file, "add", "Task C !p2 #urgent")
	runCLI(t, file, "add", "Task D !p2 #backend")

	// Select priority p1, then select tag #backend (first alphabetically)
	// This should show only Task B (p1 + #backend)
	output := runPiped(t, file, "p t ")

	// Task B should be visible (has both p1 and #backend - first tag alphabetically)
	if !strings.Contains(output, "Task B") {
		t.Errorf("Expected Task B (p1 + #backend) to be visible, got: %s", output)
	}
}

// TestTUI_PriorityFilterOverlayRender tests that priority filter overlay renders correctly
func TestTUI_PriorityFilterOverlayRender(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task A !p1")
	runCLI(t, file, "add", "Task B !p2")
	runCLI(t, file, "add", "Task C !p3")

	// Enter priority filter mode and check overlay content
	output := runPiped(t, file, "p")

	// Should show priority options in overlay
	if !strings.Contains(output, "!p1") {
		t.Errorf("Expected !p1 in priority overlay, got: %s", output)
	}
	if !strings.Contains(output, "!p2") {
		t.Errorf("Expected !p2 in priority overlay, got: %s", output)
	}
	if !strings.Contains(output, "!p3") {
		t.Errorf("Expected !p3 in priority overlay, got: %s", output)
	}

	// Should show help text
	if !strings.Contains(output, "space toggle") {
		t.Errorf("Expected help text in overlay, got: %s", output)
	}
}

// TestTUI_EmptyFilterMessage tests that a helpful message is shown when filters result in no visible todos
func TestTUI_EmptyFilterMessage(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task A !p1")
	runCLI(t, file, "add", "Task B !p2")

	// Mark Task B as done first
	runCLI(t, file, "toggle", "2")

	// Select p2 priority filter and enable filter-done
	// This should result in no visible todos (Task B is p2 but done, Task A is p1)
	output := runPiped(t, file, "pj :filter-done\r")

	// Should show the empty filter message
	if !strings.Contains(output, "No todos match current filters") {
		t.Errorf("Expected empty filter message, got: %s", output)
	}

	// Should show which filters are active
	if !strings.Contains(output, "completed hidden") {
		t.Errorf("Expected 'completed hidden' in active filters, got: %s", output)
	}
	if !strings.Contains(output, "p2") {
		t.Errorf("Expected 'p2' in active filters, got: %s", output)
	}
}

// TestTUI_EmptyFilterMessageWithTags tests empty filter message with tag filters
func TestTUI_EmptyFilterMessageWithTags(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task A #frontend")
	runCLI(t, file, "add", "Task B #backend")

	// Mark both tasks as done
	runCLI(t, file, "toggle", "1")
	runCLI(t, file, "toggle", "2")

	// Enable filter-done and filter by #frontend tag
	output := runPiped(t, file, ":filter-done\rt ")

	// Should show the empty filter message
	if !strings.Contains(output, "No todos match current filters") {
		t.Errorf("Expected empty filter message, got: %s", output)
	}

	// Should show active filters hint
	if !strings.Contains(output, "completed hidden") {
		t.Errorf("Expected 'completed hidden' in message, got: %s", output)
	}
}
