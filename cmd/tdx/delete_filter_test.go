package main

import (
	"os"
	"strings"
	"testing"
)

// TestDelete_WithFilterDone_TopItem tests deleting the top visible item
// when there are hidden items above it
func TestDelete_WithFilterDone_TopItem(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
---

- [x] Done task 1
- [x] Done task 2
- [ ] Task A
- [ ] Task B
- [ ] Task C
`
	os.WriteFile(file, []byte(initial), 0644)

	t.Log("Initial state with filter-done metadata:")
	t.Log(initial)

	// Delete the first visible item (Task A)
	// Cursor should start on Task A, delete it, then move to Task B
	runPiped(t, file, "d")

	todos := getTodos(t, file)

	// Should have 4 todos left (2 done, 2 active)
	if len(todos) != 4 {
		t.Fatalf("Expected 4 todos after delete, got %d", len(todos))
	}

	// Task A should be deleted
	for _, todo := range todos {
		if strings.Contains(todo, "Task A") {
			t.Errorf("Task A should be deleted, but found: %s", todo)
		}
	}

	// Verify remaining tasks
	if !strings.Contains(todos[0], "[x]") || !strings.Contains(todos[0], "Done task 1") {
		t.Errorf("First todo should be Done task 1, got: %s", todos[0])
	}
	if !strings.Contains(todos[2], "[ ]") || !strings.Contains(todos[2], "Task B") {
		t.Errorf("Third todo should be Task B (first visible after delete), got: %s", todos[2])
	}
}

// TestDelete_WithFilterDone_MiddleItem tests deleting a middle visible item
func TestDelete_WithFilterDone_MiddleItem(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
---

- [x] Done task 1
- [ ] Task A
- [x] Done task 2
- [ ] Task B
- [ ] Task C
`
	os.WriteFile(file, []byte(initial), 0644)

	// Move to Task B (second visible item), delete it, then toggle
	// This verifies cursor moved to Task C after delete
	runPiped(t, file, "jd ")

	todos := getTodos(t, file)

	// Should have 4 todos left
	if len(todos) != 4 {
		t.Fatalf("Expected 4 todos after delete, got %d", len(todos))
	}

	// Task B should be deleted
	for _, todo := range todos {
		if strings.Contains(todo, "Task B") {
			t.Errorf("Task B should be deleted, but found: %s", todo)
		}
	}

	// Task C should be checked (cursor moved there after delete, then we toggled it)
	if !strings.Contains(todos[3], "[x]") || !strings.Contains(todos[3], "Task C") {
		t.Errorf("Task C should be checked (cursor moved there after delete), got: %s", todos[3])
	}
}

// TestDelete_WithFilterDone_LastVisibleItem tests deleting the last visible item
func TestDelete_WithFilterDone_LastVisibleItem(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
---

- [ ] Task A
- [ ] Task B
- [x] Done task 1
- [x] Done task 2
`
	os.WriteFile(file, []byte(initial), 0644)

	// Move to Task B (last visible) and delete
	runPiped(t, file, "jd")

	todos := getTodos(t, file)

	// Should have 3 todos left
	if len(todos) != 3 {
		t.Fatalf("Expected 3 todos after delete, got %d", len(todos))
	}

	// Task B should be deleted
	for _, todo := range todos {
		if strings.Contains(todo, "Task B") {
			t.Errorf("Task B should be deleted, but found: %s", todo)
		}
	}

	// Cursor should move back to Task A (previous visible)
	// Verify by toggling
	runPiped(t, file, " ")

	todos = getTodos(t, file)
	if !strings.Contains(todos[0], "[x]") || !strings.Contains(todos[0], "Task A") {
		t.Errorf("Task A should be checked (cursor moved there after delete), got: %s", todos[0])
	}
}

// TestDelete_WithFilterDone_OnlyVisibleItem tests deleting when there's only one visible item
func TestDelete_WithFilterDone_OnlyVisibleItem(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
---

- [x] Done task 1
- [ ] Task A
- [x] Done task 2
`
	os.WriteFile(file, []byte(initial), 0644)

	// Delete the only visible item
	runPiped(t, file, "d")

	todos := getTodos(t, file)

	// Should have 2 todos left (both done)
	if len(todos) != 2 {
		t.Fatalf("Expected 2 todos after delete, got %d", len(todos))
	}

	// Task A should be deleted
	for _, todo := range todos {
		if strings.Contains(todo, "Task A") {
			t.Errorf("Task A should be deleted, but found: %s", todo)
		}
	}

	// Only hidden items remain
	for _, todo := range todos {
		if !strings.Contains(todo, "[x]") {
			t.Errorf("All remaining todos should be completed, got: %s", todo)
		}
	}
}

// TestDelete_WithFilterDone_MultipleHiddenAtTop tests the specific issue:
// deleting top visible item when multiple hidden items are above
func TestDelete_WithFilterDone_MultipleHiddenAtTop(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
---

- [x] Hidden 1
- [x] Hidden 2
- [x] Hidden 3
- [ ] Visible 1
- [ ] Visible 2
- [ ] Visible 3
`
	os.WriteFile(file, []byte(initial), 0644)

	t.Log("Testing delete with multiple hidden items at top")
	t.Log("Cursor should start on 'Visible 1'")

	// Delete first visible item
	runPiped(t, file, "d")

	todos := getTodos(t, file)

	// Should have 5 todos left
	if len(todos) != 5 {
		t.Fatalf("Expected 5 todos after delete, got %d", len(todos))
	}

	// Visible 1 should be deleted
	for _, todo := range todos {
		if strings.Contains(todo, "Visible 1") {
			t.Errorf("Visible 1 should be deleted, but found: %s", todo)
		}
	}

	// Cursor should now be on Visible 2 (next visible after delete)
	// Verify by toggling - Visible 2 should be checked
	runPiped(t, file, " ")

	todos = getTodos(t, file)

	// Find Visible 2 and check it's checked
	foundVisible2 := false
	for _, todo := range todos {
		if strings.Contains(todo, "Visible 2") {
			foundVisible2 = true
			if !strings.Contains(todo, "[x]") {
				t.Errorf("Visible 2 should be checked (cursor on it after delete), got: %s", todo)
			}
		}
	}

	if !foundVisible2 {
		t.Error("Visible 2 not found in todos")
	}
}

// TestDelete_NoFilter_NormalBehavior tests that delete works normally without filters
func TestDelete_NoFilter_NormalBehavior(t *testing.T) {
	file := tempTestFile(t)

	initial := `- [x] Done task
- [ ] Task A
- [ ] Task B
`
	os.WriteFile(file, []byte(initial), 0644)

	// Delete first item (no filter, so it's at index 0)
	runPiped(t, file, "d")

	todos := getTodos(t, file)

	// Should have 2 todos left
	if len(todos) != 2 {
		t.Fatalf("Expected 2 todos after delete, got %d", len(todos))
	}

	// Done task should be deleted
	for _, todo := range todos {
		if strings.Contains(todo, "Done task") {
			t.Errorf("Done task should be deleted, but found: %s", todo)
		}
	}

	// First todo should now be Task A
	if !strings.Contains(todos[0], "Task A") {
		t.Errorf("First todo should be Task A, got: %s", todos[0])
	}
}
