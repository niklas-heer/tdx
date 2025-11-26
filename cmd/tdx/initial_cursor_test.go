package main

import (
	"os"
	"strings"
	"testing"
)

// TestInitialCursor_FilterDoneMetadata tests that cursor starts on first visible item
// when filter-done is set in metadata and completed tasks are at the top
func TestInitialCursor_FilterDoneMetadata(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
---

- [x] Done task 1
- [x] Done task 2
- [ ] Task A1
- [ ] Task A2
`
	_ = os.WriteFile(file, []byte(initial), 0644)

	t.Log("Initial state with filter-done metadata:")
	t.Log(initial)

	// Open TUI and toggle item at cursor (should be first visible item)
	// Space toggles the item at cursor position
	runPiped(t, file, " ")

	todos := getTodos(t, file)

	// The cursor should start on Task A1 (first visible item), not Done task 1 (index 0)
	// After toggle, Task A1 should be checked
	if !strings.Contains(todos[2], "[x]") || !strings.Contains(todos[2], "Task A1") {
		t.Errorf("Task A1 should be checked (cursor was on it), got: %s", todos[2])
	}

	// Done tasks should remain unchanged
	if !strings.Contains(todos[0], "[x]") || !strings.Contains(todos[0], "Done task 1") {
		t.Errorf("Done task 1 should remain checked, got: %s", todos[0])
	}
}

// TestInitialCursor_FilterDoneCompletedInMiddle tests cursor positioning when
// completed tasks are in the middle of the list
func TestInitialCursor_FilterDoneCompletedInMiddle(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
---

- [ ] Task A
- [x] Done task 1
- [x] Done task 2
- [ ] Task B
- [ ] Task C
`
	_ = os.WriteFile(file, []byte(initial), 0644)

	t.Log("Initial state with completed tasks in middle:")
	t.Log(initial)

	// Toggle item at cursor (should be Task A - first visible)
	runPiped(t, file, " ")

	todos := getTodos(t, file)

	// Task A should be checked (cursor started on it)
	if !strings.Contains(todos[0], "[x]") || !strings.Contains(todos[0], "Task A") {
		t.Errorf("Task A should be checked, got: %s", todos[0])
	}
}

// TestInitialCursor_FilterDoneAllCompleted tests cursor positioning when all tasks
// at the start are completed
func TestInitialCursor_FilterDoneAllCompleted(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
---

- [x] Done 1
- [x] Done 2
- [x] Done 3
- [ ] First incomplete
- [ ] Second incomplete
`
	_ = os.WriteFile(file, []byte(initial), 0644)

	// Just toggle without moving (cursor should start at First incomplete)
	runPiped(t, file, " ")

	todos := getTodos(t, file)

	// First incomplete should be checked (cursor started there and we toggled it)
	if !strings.Contains(todos[3], "[x]") || !strings.Contains(todos[3], "First incomplete") {
		t.Errorf("First incomplete should be checked (cursor started on it), got: %s", todos[3])
	}
	// Second incomplete should remain unchecked
	if !strings.Contains(todos[4], "[ ]") || !strings.Contains(todos[4], "Second incomplete") {
		t.Errorf("Second incomplete should remain unchecked, got: %s", todos[4])
	}
}

// TestInitialCursor_NoFilters tests that cursor starts at index 0 when no filters active
func TestInitialCursor_NoFilters(t *testing.T) {
	file := tempTestFile(t)

	initial := `- [x] Done task
- [ ] Task A
- [ ] Task B
`
	_ = os.WriteFile(file, []byte(initial), 0644)

	// Toggle item at cursor (should be index 0 - Done task)
	runPiped(t, file, " ")

	todos := getTodos(t, file)

	// Done task should now be unchecked (cursor started on it)
	if !strings.Contains(todos[0], "[ ]") || !strings.Contains(todos[0], "Done task") {
		t.Errorf("Done task should be unchecked (toggled from checked), got: %s", todos[0])
	}
}

// TestInitialCursor_ShowHeadingsMetadata tests cursor starts on first visible todo
// when show-headings is enabled in metadata
func TestInitialCursor_ShowHeadingsMetadata(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
show-headings: true
---

# Section 1

- [ ] Task A
- [ ] Task B

# Section 2

- [ ] Task C
`
	_ = os.WriteFile(file, []byte(initial), 0644)

	t.Log("Initial state with show-headings metadata:")
	t.Log(initial)

	// Toggle item at cursor (should be Task A - first todo)
	runPiped(t, file, " ")

	todos := getTodos(t, file)

	// Task A should be checked (cursor started on it, not on heading)
	if !strings.Contains(todos[0], "[x]") || !strings.Contains(todos[0], "Task A") {
		t.Errorf("Task A should be checked (cursor on first todo, not heading), got: %s", todos[0])
	}
}

// TestInitialCursor_FilterDoneAndHeadings tests cursor with both filters active
func TestInitialCursor_FilterDoneAndHeadings(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
show-headings: true
---

# Section 1

- [x] Done task
- [ ] Task A

# Section 2

- [ ] Task B
`
	_ = os.WriteFile(file, []byte(initial), 0644)

	// Toggle item at cursor (should be Task A - first visible todo)
	runPiped(t, file, " ")

	todos := getTodos(t, file)

	// Task A should be checked
	if !strings.Contains(todos[1], "[x]") || !strings.Contains(todos[1], "Task A") {
		t.Errorf("Task A should be checked, got: %s", todos[1])
	}

	// Done task should remain checked
	if !strings.Contains(todos[0], "[x]") || !strings.Contains(todos[0], "Done task") {
		t.Errorf("Done task should remain checked, got: %s", todos[0])
	}
}

// TestInitialCursor_OnlyCompletedTasks tests behavior when all tasks are completed
func TestInitialCursor_OnlyCompletedTasks(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
---

- [x] Done 1
- [x] Done 2
- [x] Done 3
`
	_ = os.WriteFile(file, []byte(initial), 0644)

	// Try to toggle (no visible items, cursor at index 0)
	// This will toggle the item at index 0 even though it's filtered
	runPiped(t, file, " ")

	todos := getTodos(t, file)

	// The first task will be toggled (unchecked) because cursor is at index 0
	// This is expected behavior - space toggles the item at cursor position
	if !strings.Contains(todos[0], "[ ]") {
		t.Errorf("First todo should be unchecked after toggle, got: %s", todos[0])
	}
	// Other tasks should remain checked
	if !strings.Contains(todos[1], "[x]") {
		t.Errorf("Second todo should remain checked, got: %s", todos[1])
	}
	if !strings.Contains(todos[2], "[x]") {
		t.Errorf("Third todo should remain checked, got: %s", todos[2])
	}
}

// TestInitialCursor_MixedVisibilityComplex tests complex scenario with mixed visibility
func TestInitialCursor_MixedVisibilityComplex(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
---

- [x] Completed task 1
- [ ] Active task 1
- [x] Completed task 2
- [ ] Active task 2
- [x] Completed task 3
- [ ] Active task 3
`
	_ = os.WriteFile(file, []byte(initial), 0644)

	// Move down twice and toggle
	// Should start at Active task 1, move to Active task 2, move to Active task 3, then toggle
	runPiped(t, file, "jj ")

	todos := getTodos(t, file)

	// Active task 3 should be checked (after two moves down and toggle)
	if !strings.Contains(todos[5], "[x]") || !strings.Contains(todos[5], "Active task 3") {
		t.Errorf("Active task 3 should be checked after jj+space, got: %s", todos[5])
	}

	// Active task 1 and 2 should remain unchecked
	if !strings.Contains(todos[1], "[ ]") || !strings.Contains(todos[1], "Active task 1") {
		t.Errorf("Active task 1 should remain unchecked, got: %s", todos[1])
	}
	if !strings.Contains(todos[3], "[ ]") || !strings.Contains(todos[3], "Active task 2") {
		t.Errorf("Active task 2 should remain unchecked, got: %s", todos[3])
	}
}
