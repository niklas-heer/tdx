package main

import (
	"os"
	"strings"
	"testing"
)

// TestMoveWithFilterDone tests that moving todos works correctly when filter-done is active
// With visible-swap movement, one 'j' swaps with the next VISIBLE item
func TestMoveWithFilterDone(t *testing.T) {
	file := tempTestFile(t)

	// Create initial file with mix of completed and incomplete todos
	initial := `# Test

- [ ] Task 1
- [x] Task 2 (completed)
- [ ] Task 3
- [x] Task 4 (completed)
- [ ] Task 5
`
	os.WriteFile(file, []byte(initial), 0644)

	// Enable filter-done, move to Task 3 (j), then move Task 3 down once
	// Visible: Task 1, Task 3, Task 5
	// After swap: Task 1 swaps with Task 3 → Task 3, Task 2, Task 1, Task 4, Task 5
	// Then we move to Task 1 (now at original Task 3 position) and swap with Task 5
	// Actually: we move to Task 3, then swap with Task 5
	// Result visible order: Task 1, Task 5, Task 3
	runPiped(t, file, ":filter-done\rjmj\r")

	content := readTestFile(t, file)
	todos := getTodos(t, file)

	if len(todos) != 5 {
		t.Fatalf("Expected 5 todos, got %d", len(todos))
	}

	t.Logf("After visible-swap:\n%s", content)

	// Verify visible order changed: Task 3 swapped with Task 5
	// Array order: Task 1, Task 2, Task 5, Task 4, Task 3
	visible := []string{}
	for _, todo := range todos {
		if strings.Contains(todo, "[ ]") {
			visible = append(visible, todo)
		}
	}

	if len(visible) != 3 {
		t.Fatalf("Expected 3 visible todos, got %d", len(visible))
	}

	// Visible order should have Task 3 at the end (swapped with Task 5)
	if !strings.Contains(visible[0], "Task 1") {
		t.Errorf("First visible should be Task 1, got: %s", visible[0])
	}
	if !strings.Contains(visible[1], "Task 5") {
		t.Errorf("Second visible should be Task 5 (swapped), got: %s", visible[1])
	}
	if !strings.Contains(visible[2], "Task 3") {
		t.Errorf("Third visible should be Task 3 (swapped), got: %s", visible[2])
	}
}

// TestMoveWithFilterDone_SavesProperly tests that moves are persisted to disk
func TestMoveWithFilterDone_SavesProperly(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Test

- [ ] First
- [x] Done
- [ ] Second
`
	os.WriteFile(file, []byte(initial), 0644)

	// Enable filter-done, move First down (swaps with Second - next visible)
	runPiped(t, file, ":filter-done\rmj\r")

	todos := getTodos(t, file)

	if len(todos) != 3 {
		t.Fatalf("Expected 3 todos, got %d", len(todos))
	}

	// Should be: Second, Done, First (First swapped with Second)
	if !strings.Contains(todos[0], "Second") {
		t.Errorf("First todo should be 'Second', got: %s", todos[0])
	}
	if !strings.Contains(todos[1], "Done") {
		t.Errorf("Second todo should be 'Done' (hidden, unchanged), got: %s", todos[1])
	}
	if !strings.Contains(todos[2], "First") {
		t.Errorf("Third todo should be 'First', got: %s", todos[2])
	}
}

// TestMoveWithFilterDone_SingleMove tests that a single visible-swap works correctly
func TestMoveWithFilterDone_SingleMove(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Test

- [ ] First
- [x] Done
- [ ] Second
`
	os.WriteFile(file, []byte(initial), 0644)

	// Enable filter-done, move First down ONCE (visible-swap with Second)
	runPiped(t, file, ":filter-done\rmj\r")

	todos := getTodos(t, file)

	if len(todos) != 3 {
		t.Fatalf("Expected 3 todos, got %d", len(todos))
	}

	// Should be: Second, Done, First (First swapped with Second - next visible)
	if !strings.Contains(todos[0], "Second") {
		t.Errorf("First todo should be 'Second', got: %s", todos[0])
	}
	if !strings.Contains(todos[1], "Done") {
		t.Errorf("Second todo should be 'Done', got: %s", todos[1])
	}
	if !strings.Contains(todos[2], "First") {
		t.Errorf("Third todo should be 'First', got: %s", todos[2])
	}
}

// TestMoveWithFilterDone_CrossingHeadings verifies that moves work across different sections
func TestMoveWithFilterDone_CrossingHeadings(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Test

## Section 1

- [ ] Task A
- [x] Task B (done)

## Section 2

- [ ] Task C
- [x] Task D (done)
`
	os.WriteFile(file, []byte(initial), 0644)

	// Enable filter-done, select Task A, move down
	// Visible: Task A, Task C
	// After swap: Task C's position has Task A, Task A's position has Task C
	runPiped(t, file, ":filter-done\rmj\r")

	content := readTestFile(t, file)
	t.Logf("After visible-swap across sections:\n%s", content)

	// Task A and Task C should have swapped positions
	// The hidden tasks stay in their original array positions
	todos := getTodos(t, file)
	visible := []string{}
	for _, todo := range todos {
		if strings.Contains(todo, "[ ]") {
			visible = append(visible, todo)
		}
	}

	if len(visible) != 2 {
		t.Fatalf("Expected 2 visible todos, got %d", len(visible))
	}

	// Visible order should be: Task C, Task A (swapped)
	if !strings.Contains(visible[0], "Task C") {
		t.Errorf("First visible should be Task C, got: %s", visible[0])
	}
	if !strings.Contains(visible[1], "Task A") {
		t.Errorf("Second visible should be Task A, got: %s", visible[1])
	}
}

// TestMoveWithFilterDone_IntoEmptySection tests swap behavior when sections have
// only hidden tasks
func TestMoveWithFilterDone_IntoEmptySection(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Test

## Section A
- [ ] Task A

## Section B (all done)
- [x] Done B1
- [x] Done B2

## Section C
- [ ] Task C
`
	os.WriteFile(file, []byte(initial), 0644)

	// With visible-swap: Task A swaps with Task C (next visible)
	// Section B's completed tasks stay in place
	runPiped(t, file, ":filter-done\rmj\r")

	content := readTestFile(t, file)
	t.Logf("After visible-swap:\n%s", content)

	todos := getTodos(t, file)
	visible := []string{}
	for _, todo := range todos {
		if strings.Contains(todo, "[ ]") {
			visible = append(visible, todo)
		}
	}

	// Visible order should be: Task C, Task A (swapped)
	if len(visible) != 2 {
		t.Fatalf("Expected 2 visible todos, got %d", len(visible))
	}

	if !strings.Contains(visible[0], "Task C") {
		t.Errorf("First visible should be Task C, got: %s", visible[0])
	}
	if !strings.Contains(visible[1], "Task A") {
		t.Errorf("Second visible should be Task A, got: %s", visible[1])
	}
}

// TestMoveWithFilterDone_MoveUpSwapsWithPreviousVisible tests moving up
func TestMoveWithFilterDone_MoveUpSwapsWithPreviousVisible(t *testing.T) {
	file := tempTestFile(t)

	initial := `- [ ] Task A
- [x] Hidden 1
- [x] Hidden 2
- [ ] Task B
`
	os.WriteFile(file, []byte(initial), 0644)

	// Enable filter-done, move to Task B (j), then move it up
	// Visible: Task A, Task B
	// After swap: Task B, Task A
	runPiped(t, file, ":filter-done\rjmk\r")

	content := readTestFile(t, file)
	t.Logf("After move up:\n%s", content)

	todos := getTodos(t, file)
	visible := []string{}
	for _, todo := range todos {
		if strings.Contains(todo, "[ ]") {
			visible = append(visible, todo)
		}
	}

	// Visible order should be: Task B, Task A (swapped)
	if len(visible) != 2 {
		t.Fatalf("Expected 2 visible todos, got %d", len(visible))
	}

	if !strings.Contains(visible[0], "Task B") {
		t.Errorf("First visible should be Task B, got: %s", visible[0])
	}
	if !strings.Contains(visible[1], "Task A") {
		t.Errorf("Second visible should be Task A, got: %s", visible[1])
	}
}

// TestMoveWithFilterDone_Reversible tests that down then up returns to original
func TestMoveWithFilterDone_Reversible(t *testing.T) {
	file := tempTestFile(t)

	initial := `- [ ] A
- [x] Hidden
- [ ] B
- [ ] C
`
	os.WriteFile(file, []byte(initial), 0644)

	initialTodos := getTodos(t, file)

	// Enable filter-done, move down then up (in single session)
	runPiped(t, file, ":filter-done\rmjk\r")

	afterTodos := getTodos(t, file)

	// Should return to original order
	if len(initialTodos) != len(afterTodos) {
		t.Fatalf("Todo count changed: %d -> %d", len(initialTodos), len(afterTodos))
	}
	for i := range initialTodos {
		if initialTodos[i] != afterTodos[i] {
			t.Errorf("Todo %d changed: %s -> %s", i, initialTodos[i], afterTodos[i])
		}
	}
}
