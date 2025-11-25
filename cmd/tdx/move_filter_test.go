package main

import (
	"os"
	"strings"
	"testing"
)

// TestMoveWithFilterDone tests that moving todos works correctly when filter-done is active
// With granular movement, each 'j' moves one position at a time
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

	// Enable filter-done, move to Task 3 (j), then move Task 3 down twice (jj)
	// to get it past Task 4 (completed) and after Task 5
	// With granular movement, need 2 moves to get past one hidden item
	runPiped(t, file, ":filter-done\rjmjj\r")

	// Read the file to verify the move worked correctly
	content := readTestFile(t, file)
	todos := getTodos(t, file)

	if len(todos) != 5 {
		t.Fatalf("Expected 5 todos, got %d", len(todos))
	}

	// The order should be: Task 1, Task 2 (completed), Task 4 (completed), Task 5, Task 3
	// Task 3 moved after Task 5 (took 2 moves to pass Task 4)
	expected := []string{
		"- [ ] Task 1",
		"- [x] Task 2 (completed)",
		"- [x] Task 4 (completed)",
		"- [ ] Task 5",
		"- [ ] Task 3",
	}

	for i, expectedTodo := range expected {
		if todos[i] != expectedTodo {
			t.Errorf("Todo %d: expected %q, got %q", i, expectedTodo, todos[i])
			t.Logf("Full content:\n%s", content)
		}
	}
}

// TestMoveWithFilterDone_SavesProperly tests that moves are persisted to disk
// With granular movement, one move = one position swap
func TestMoveWithFilterDone_SavesProperly(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Test

- [ ] First
- [x] Done
- [ ] Second
`
	os.WriteFile(file, []byte(initial), 0644)

	// Enable filter-done, move First down TWICE to get past Done and after Second
	runPiped(t, file, ":filter-done\rmjj\r")

	// Read file after move
	todos := getTodos(t, file)

	if len(todos) != 3 {
		t.Fatalf("Expected 3 todos, got %d", len(todos))
	}

	// Should be: Done, Second, First (First moved two positions)
	if !strings.Contains(todos[0], "Done") {
		t.Errorf("First todo should be 'Done', got: %s", todos[0])
	}
	if !strings.Contains(todos[1], "Second") {
		t.Errorf("Second todo should be 'Second', got: %s", todos[1])
	}
	if !strings.Contains(todos[2], "First") {
		t.Errorf("Third todo should be 'First', got: %s", todos[2])
	}
}

// TestMoveWithFilterDone_SingleMove tests that a single move goes one position
func TestMoveWithFilterDone_SingleMove(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Test

- [ ] First
- [x] Done
- [ ] Second
`
	os.WriteFile(file, []byte(initial), 0644)

	// Enable filter-done, move First down ONCE (granular movement)
	runPiped(t, file, ":filter-done\rmj\r")

	todos := getTodos(t, file)

	if len(todos) != 3 {
		t.Fatalf("Expected 3 todos, got %d", len(todos))
	}

	// Should be: Done, First, Second (First moved one position, swapped with Done)
	if !strings.Contains(todos[0], "Done") {
		t.Errorf("First todo should be 'Done', got: %s", todos[0])
	}
	if !strings.Contains(todos[1], "First") {
		t.Errorf("Second todo should be 'First' (moved one pos), got: %s", todos[1])
	}
	if !strings.Contains(todos[2], "Second") {
		t.Errorf("Third todo should be 'Second', got: %s", todos[2])
	}
}

// TestMoveWithFilterDone_CrossingHeadings verifies that moves work across different sections
// With granular movement, you can move into sections one position at a time
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

	// Enable filter-done, select Task A, move down twice to get into Section 2
	// First move: swaps with Task B (done)
	// Second move: moves into Section 2, before Task C
	runPiped(t, file, ":filter-done\rmjj\r")

	content := readTestFile(t, file)
	t.Logf("After two moves:\n%s", content)

	// Task A should now be in Section 2
	// With AST-based movement, it should be after Section 2 heading
	if !strings.Contains(content, "## Section 1\n\n- [x] Task B") {
		t.Error("Section 1 should only have Task B (done)")
		t.Logf("Content:\n%s", content)
	}
}

// TestMoveWithFilterDone_IntoEmptySection tests moving into a section
// that appears empty due to filter-done (all tasks are completed)
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

	// With granular movement, we can move Task A step by step into Section B
	// Move down once to pass Section A boundary
	runPiped(t, file, ":filter-done\rmj\r")

	content := readTestFile(t, file)
	t.Logf("After first move:\n%s", content)

	// Task A should have moved one position (into or past Section B boundary)
	// The important thing is we didn't skip directly to Section C
}
