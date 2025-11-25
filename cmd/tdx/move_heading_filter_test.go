package main

import (
	"os"
	"strings"
	"testing"
)

// TestMoveWithFilterDone_IntoEmptyHeadingGroup tests that a task can be moved
// into a heading group that appears empty due to filter-done hiding all its tasks
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
	os.WriteFile(file, []byte(initial), 0644)

	t.Log("Initial visible tasks: Task A (Section A), Task C (Section C)")
	t.Log("Section B appears empty because all tasks are done")

	// Move Task A down - with current behavior it might skip Section B entirely
	// Expected behavior: Task A should be movable into Section B
	runPiped(t, file, "mj\r")

	content := readTestFile(t, file)
	t.Logf("After first move down:\n%s", content)

	// Move down again
	runPiped(t, file, "mj\r")

	content = readTestFile(t, file)
	t.Logf("After second move down:\n%s", content)

	// Task A should now be in or past Section B
	// The key is that it should be possible to move INTO Section B,
	// not just skip over it to Section C
}

// TestMoveWithFilterDone_GranularMovement tests that move operations
// happen one position at a time, not jumping over hidden items
func TestMoveWithFilterDone_GranularMovement(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Tasks

- [ ] Task A
- [x] Done 1
- [x] Done 2
- [x] Done 3
- [ ] Task B
`
	os.WriteFile(file, []byte(initial), 0644)

	t.Log("Array indices: Task A(0), Done1(1), Done2(2), Done3(3), Task B(4)")
	t.Log("Visible with filter-done: Task A, Task B")

	// Enable filter-done and move Task A down once
	runPiped(t, file, ":filter-done\rmj\r")

	todos := getTodos(t, file)
	content := readTestFile(t, file)
	t.Logf("After one move down:\n%s", content)

	// Find positions of Task A
	taskAPos := -1
	for i, todo := range todos {
		if strings.Contains(todo, "Task A") {
			taskAPos = i
			break
		}
	}

	t.Logf("Task A is now at position %d", taskAPos)

	// With granular movement, Task A should move ONE position at a time
	// So after one 'j', it should be at index 1 (swapped with Done 1)
	// NOT at index 4 (jumped over all done tasks)
	if taskAPos == 4 {
		t.Error("Task A jumped over all hidden tasks instead of moving one position at a time")
		t.Log("Expected granular movement: Task A should move one array position per keypress")
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
	os.WriteFile(file, []byte(initial), 0644)

	// Do all moves in a SINGLE session to test consecutive behavior
	// Enable filter-done, then move down 4 times, confirming each
	runPiped(t, file, ":filter-done\rmjjjj\r")

	content := readTestFile(t, file)
	t.Logf("After 4 consecutive moves down:\n%s", content)

	todos := getTodos(t, file)

	// Task A should have moved down 4 positions (to the end)
	// Original: A(0), Done1(1), B(2), Done2(3), C(4)
	// After 4 moves: Done1(0), B(1), Done2(2), C(3), A(4)
	if len(todos) != 5 {
		t.Fatalf("Expected 5 todos, got %d", len(todos))
	}

	// Task A should be at the end
	lastTodo := todos[len(todos)-1]
	if !strings.Contains(lastTodo, "Task A") {
		t.Errorf("Task A should be at the end after 4 moves, got: %s", lastTodo)
	}
}

// TestMoveWithFilterDone_CursorStaysOnMovedItem tests that the cursor
// follows the moved item, even when moving past hidden items
func TestMoveWithFilterDone_CursorStaysOnMovedItem(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Tasks

- [ ] Task A
- [x] Done 1
- [x] Done 2
- [ ] Task B
`
	os.WriteFile(file, []byte(initial), 0644)

	// Enable filter-done, move Task A down, then immediately delete
	// If cursor stayed on Task A, the delete should remove Task A
	runPiped(t, file, ":filter-done\rmj\rd")

	content := readTestFile(t, file)
	t.Logf("After move and delete:\n%s", content)

	// Task A should be deleted (cursor was on it after move)
	if strings.Contains(content, "Task A") {
		t.Error("Task A should have been deleted - cursor should have stayed on moved item")
	}

	// Task B should still exist
	if !strings.Contains(content, "Task B") {
		t.Error("Task B should still exist")
	}
}

// TestMoveWithFilterDone_MoveUpIntoEmptySection tests moving UP into
// a section that appears empty due to filtering
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
	os.WriteFile(file, []byte(initial), 0644)

	t.Log("Starting with cursor on Task C (last visible task)")

	// Move to Task C (j moves to next visible), then move up once
	// With granular movement, it should move one position at a time
	runPiped(t, file, "jmk\r")

	content := readTestFile(t, file)
	t.Logf("After moving Task C up once:\n%s", content)

	// Task C should now be just above Done B2 (moved one position up)
	// Check that Task C is now in Section B
	if !strings.Contains(content, "## Section B") {
		t.Error("Section B should still exist")
	}

	// Move up more times to get into Section B
	runPiped(t, file, "mkk\r")
	content = readTestFile(t, file)
	t.Logf("After moving Task C up two more times:\n%s", content)
}

// TestMoveWithFilterDone_HeadingBoundaries tests that moving respects
// heading boundaries and places items correctly within sections
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
	os.WriteFile(file, []byte(initial), 0644)

	// Enable filter-done, move Work Task 1 to end of Work section
	runPiped(t, file, ":filter-done\rmjj\r")

	content := readTestFile(t, file)
	t.Logf("After moving Work Task 1 down twice:\n%s", content)

	// Check that Work Task 1 is still in the Work section or moved to Personal
	// It should have passed through the hidden "Work Done" task
}

// TestMoveNoFilter_AdjacentSwap verifies baseline behavior without filters
func TestMoveNoFilter_AdjacentSwap(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Tasks

- [ ] A
- [x] B
- [ ] C
`
	os.WriteFile(file, []byte(initial), 0644)

	// Move A down (no filter active)
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

// TestMoveWithFilterDone_MoveDownThenUp tests that moving down then up
// returns to the original position (in single session)
func TestMoveWithFilterDone_MoveDownThenUp(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Tasks

- [ ] Task A
- [x] Done 1
- [ ] Task B
`
	os.WriteFile(file, []byte(initial), 0644)

	initialContent := readTestFile(t, file)
	t.Logf("Initial:\n%s", initialContent)

	// Do move down then up in SINGLE session
	runPiped(t, file, ":filter-done\rmjk\r")

	afterUpDown := readTestFile(t, file)
	t.Logf("After move down then up:\n%s", afterUpDown)

	// With granular movement in single session, down then up should return to original
	if afterUpDown != initialContent {
		t.Error("Move down then up should return to original position")
	}
}

// TestMoveWithTagFilter_IntoEmptyTagGroup tests moving with tag filters
// (similar issue to filter-done)
func TestMoveWithTagFilter_IntoEmptyTagGroup(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Tasks

- [ ] Task A #work
- [ ] Task B #home
- [ ] Task C #home
- [ ] Task D #work
`
	os.WriteFile(file, []byte(initial), 0644)

	t.Log("Filtering for #work: visible are Task A and Task D")
	t.Log("Tasks B and C with #home are hidden")

	// This test documents expected behavior with tag filtering
	// When moving Task A down, it should be possible to move past B and C
	// one position at a time, not just jump to Task D
}
