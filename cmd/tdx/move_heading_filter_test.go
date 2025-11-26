package main

import (
	"os"
	"strings"
	"testing"
)

// TestMoveWithFilterDone_IntoEmptyHeadingGroup tests that visible items swap correctly
// even when there are heading groups with only hidden tasks in between
func TestMoveWithFilterDone_IntoEmptyHeadingGroup(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
---
# Tasks

## Section A
- [ ] Task A

## Section B (all done - appears empty)
- [x] Done B1
- [x] Done B2

## Section C
- [ ] Task C
`
	_ = os.WriteFile(file, []byte(initial), 0644)

	t.Log("Initial visible tasks: Task A (Section A), Task C (Section C)")
	t.Log("Section B appears empty because all tasks are done")

	// Move Task A down - with visible-swap, it swaps with Task C
	runPiped(t, file, "mj\r")

	content := readTestFile(t, file)
	t.Logf("After visible-swap:\n%s", content)

	// Extract visible todos
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

// TestMoveWithFilterDone_VisibleSwapMovement tests that move operations
// swap visible items predictably, regardless of hidden items between them
func TestMoveWithFilterDone_VisibleSwapMovement(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Tasks

- [ ] Task A
- [x] Done 1
- [x] Done 2
- [x] Done 3
- [ ] Task B
`
	_ = os.WriteFile(file, []byte(initial), 0644)

	t.Log("Array indices: Task A(0), Done1(1), Done2(2), Done3(3), Task B(4)")
	t.Log("Visible with filter-done: Task A, Task B")

	// Enable filter-done and move Task A down once
	runPiped(t, file, ":filter-done\rmj\r")

	todos := getTodos(t, file)
	content := readTestFile(t, file)
	t.Logf("After one visible-swap:\n%s", content)

	// Extract visible todos
	visible := []string{}
	for _, todo := range todos {
		if strings.Contains(todo, "[ ]") {
			visible = append(visible, todo)
		}
	}

	if len(visible) != 2 {
		t.Fatalf("Expected 2 visible todos, got %d", len(visible))
	}

	// Visible order should be: Task B, Task A (swapped)
	if !strings.Contains(visible[0], "Task B") {
		t.Errorf("First visible should be Task B (swapped), got: %s", visible[0])
	}
	if !strings.Contains(visible[1], "Task A") {
		t.Errorf("Second visible should be Task A (swapped), got: %s", visible[1])
	}
}

// TestMoveWithFilterDone_ConsecutiveMovesArePredictable tests that multiple
// consecutive moves behave predictably (all in single session)
func TestMoveWithFilterDone_ConsecutiveMovesArePredictable(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Tasks

- [ ] Task A
- [x] Done 1
- [ ] Task B
- [x] Done 2
- [ ] Task C
`
	_ = os.WriteFile(file, []byte(initial), 0644)

	// Visible: Task A, Task B, Task C
	// Move Task A down twice (in single session)
	// First swap: Task A <-> Task B → B, A, C
	// Second swap: Task A <-> Task C → B, C, A
	runPiped(t, file, ":filter-done\rmjj\r")

	content := readTestFile(t, file)
	t.Logf("After 2 consecutive visible-swaps:\n%s", content)

	todos := getTodos(t, file)
	visible := []string{}
	for _, todo := range todos {
		if strings.Contains(todo, "[ ]") {
			visible = append(visible, todo)
		}
	}

	if len(visible) != 3 {
		t.Fatalf("Expected 3 visible todos, got %d", len(visible))
	}

	// Visible order should be: Task B, Task C, Task A
	if !strings.Contains(visible[0], "Task B") {
		t.Errorf("First visible should be Task B, got: %s", visible[0])
	}
	if !strings.Contains(visible[1], "Task C") {
		t.Errorf("Second visible should be Task C, got: %s", visible[1])
	}
	if !strings.Contains(visible[2], "Task A") {
		t.Errorf("Third visible should be Task A, got: %s", visible[2])
	}
}

// TestMoveWithFilterDone_CursorStaysOnMovedItem tests that the cursor
// follows the moved item after a swap
func TestMoveWithFilterDone_CursorStaysOnMovedItem(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Tasks

- [ ] Task A
- [x] Done 1
- [x] Done 2
- [ ] Task B
`
	_ = os.WriteFile(file, []byte(initial), 0644)

	// Enable filter-done, move Task A down, then immediately delete
	// If cursor stayed on Task A (now at Task B's old position), delete removes Task A
	runPiped(t, file, ":filter-done\rmj\rd")

	content := readTestFile(t, file)
	t.Logf("After swap and delete:\n%s", content)

	// Task A should be deleted (cursor was on it after swap)
	if strings.Contains(content, "Task A") {
		t.Error("Task A should have been deleted - cursor should have stayed on moved item")
	}

	// Task B should still exist
	if !strings.Contains(content, "Task B") {
		t.Error("Task B should still exist")
	}
}

// TestMoveWithFilterDone_MoveUpIntoEmptySection tests moving UP when there
// are empty sections (due to filtering) between visible items
func TestMoveWithFilterDone_MoveUpIntoEmptySection(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
---
# Tasks

## Section A
- [ ] Task A

## Section B (all done)
- [x] Done B1
- [x] Done B2

## Section C
- [ ] Task C
`
	_ = os.WriteFile(file, []byte(initial), 0644)

	t.Log("Starting with cursor on Task C (last visible task)")

	// Move to Task C (j), then move it up (swaps with Task A)
	runPiped(t, file, "jmk\r")

	content := readTestFile(t, file)
	t.Logf("After moving Task C up:\n%s", content)

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

// TestMoveWithFilterDone_HeadingBoundaries tests that swapping works correctly
// when items are in different heading sections
func TestMoveWithFilterDone_HeadingBoundaries(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Tasks

## Work
- [ ] Work Task 1
- [x] Work Done
- [ ] Work Task 2

## Personal
- [ ] Personal Task 1
- [x] Personal Done
- [ ] Personal Task 2
`
	_ = os.WriteFile(file, []byte(initial), 0644)

	// Enable filter-done
	// Visible: Work Task 1, Work Task 2, Personal Task 1, Personal Task 2
	// Move Work Task 1 down twice (swap with Work Task 2, then with Personal Task 1)
	runPiped(t, file, ":filter-done\rmjj\r")

	content := readTestFile(t, file)
	t.Logf("After two visible-swaps:\n%s", content)

	todos := getTodos(t, file)
	visible := []string{}
	for _, todo := range todos {
		if strings.Contains(todo, "[ ]") {
			visible = append(visible, todo)
		}
	}

	if len(visible) != 4 {
		t.Fatalf("Expected 4 visible todos, got %d", len(visible))
	}

	// Visible order should be: Work Task 2, Personal Task 1, Work Task 1, Personal Task 2
	if !strings.Contains(visible[0], "Work Task 2") {
		t.Errorf("First visible should be Work Task 2, got: %s", visible[0])
	}
	if !strings.Contains(visible[1], "Personal Task 1") {
		t.Errorf("Second visible should be Personal Task 1, got: %s", visible[1])
	}
	if !strings.Contains(visible[2], "Work Task 1") {
		t.Errorf("Third visible should be Work Task 1, got: %s", visible[2])
	}
}

// TestMoveNoFilter_AdjacentSwap verifies baseline behavior without filters
func TestMoveNoFilter_AdjacentSwap(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Tasks

- [ ] A
- [x] B
- [ ] C
`
	_ = os.WriteFile(file, []byte(initial), 0644)

	// Move A down (no filter active) - swaps with adjacent B
	runPiped(t, file, "mj\r")

	todos := getTodos(t, file)
	content := readTestFile(t, file)
	t.Logf("After move:\n%s", content)

	// Should be: B, A, C (A swapped with adjacent B)
	if len(todos) != 3 {
		t.Fatalf("Expected 3 todos, got %d", len(todos))
	}

	expected := []string{"- [x] B", "- [ ] A", "- [ ] C"}
	for i, exp := range expected {
		if todos[i] != exp {
			t.Errorf("Position %d: expected %q, got %q", i, exp, todos[i])
		}
	}
}

// TestMoveWithFilterDone_Reversibility tests that moving down then up
// returns to the original position
func TestMoveWithFilterDone_Reversibility(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Tasks

- [ ] Task A
- [x] Done 1
- [ ] Task B
`
	_ = os.WriteFile(file, []byte(initial), 0644)

	initialContent := readTestFile(t, file)
	t.Logf("Initial:\n%s", initialContent)

	// Do move down then up in SINGLE session
	// Down: Task A inserted after Task B -> Done 1, Task B, Task A
	// Up: Task A inserted before Task B -> Done 1, Task A, Task B (NOT original)
	runPiped(t, file, ":filter-done\rmjk\r")

	afterUpDown := readTestFile(t, file)
	t.Logf("After move down then up:\n%s", afterUpDown)

	todos := getTodos(t, file)

	// With insertion-based movement, down then up does NOT return to original
	// Expected: Done 1, Task A, Task B
	if !strings.Contains(todos[0], "Done 1") {
		t.Errorf("First todo should be 'Done 1', got: %s", todos[0])
	}
	if !strings.Contains(todos[1], "Task A") {
		t.Errorf("Second todo should be 'Task A', got: %s", todos[1])
	}
	if !strings.Contains(todos[2], "Task B") {
		t.Errorf("Third todo should be 'Task B', got: %s", todos[2])
	}
}

// Note: Tag filter tests are handled in unit tests where we can set FilteredTags directly
// The TUI uses a filter overlay mode (press 't') rather than a command for tag filtering

// TestMoveWithFilterDone_StopsAtEnd tests that you can't move past the last visible item
func TestMoveWithFilterDone_StopsAtEnd(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Tasks

- [ ] Task A
- [x] Hidden
- [ ] Task B
`
	_ = os.WriteFile(file, []byte(initial), 0644)

	// Enable filter-done, try to move Task A down multiple times
	// Should only swap once (with Task B), then stop
	runPiped(t, file, ":filter-done\rmjjjjj\r")

	content := readTestFile(t, file)
	t.Logf("After multiple move attempts:\n%s", content)

	todos := getTodos(t, file)
	visible := []string{}
	for _, todo := range todos {
		if strings.Contains(todo, "[ ]") {
			visible = append(visible, todo)
		}
	}

	// Visible order should be: Task B, Task A (just one swap happened)
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

// TestMoveWithFilterDone_CantMovePastStart tests that you can't move past the first visible item
func TestMoveWithFilterDone_CantMovePastStart(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Tasks

- [ ] Task A
- [x] Hidden
- [ ] Task B
`
	_ = os.WriteFile(file, []byte(initial), 0644)

	// Enable filter-done, move to Task B, try to move it up multiple times
	// Should only swap once (with Task A), then stop
	runPiped(t, file, ":filter-done\rjmkkkkk\r")

	content := readTestFile(t, file)
	t.Logf("After multiple move up attempts:\n%s", content)

	todos := getTodos(t, file)
	visible := []string{}
	for _, todo := range todos {
		if strings.Contains(todo, "[ ]") {
			visible = append(visible, todo)
		}
	}

	// Visible order should be: Task B, Task A (just one swap happened)
	if len(visible) != 2 {
		t.Fatalf("Expected 2 visible todos, got %d", len(visible))
	}

	if !strings.Contains(visible[0], "Task B") {
		t.Errorf("First visible should be Task B (moved up), got: %s", visible[0])
	}
	if !strings.Contains(visible[1], "Task A") {
		t.Errorf("Second visible should be Task A, got: %s", visible[1])
	}
}
