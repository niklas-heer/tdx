package main

import (
	"os"
	"strings"
	"testing"
)

// TestMoveWithFilterDone tests that moving todos works correctly when filter-done is active
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

	// Enable filter-done, then move Task 3 down
	// With filter-done, visible tasks are: Task 1, Task 3, Task 5
	// Moving Task 3 down moves it after Task 5 in the ACTUAL file
	runPiped(t, file, ":filter-done\rjmj\r")

	// Read the file to verify the move worked correctly
	content := readTestFile(t, file)
	todos := getTodos(t, file)

	if len(todos) != 5 {
		t.Fatalf("Expected 5 todos, got %d", len(todos))
	}

	// The order should be: Task 1, Task 2 (completed), Task 4 (completed), Task 5, Task 3
	// Task 3 moved after Task 5, completed tasks stay in place
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
func TestMoveWithFilterDone_SavesProperly(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Test

- [ ] First
- [x] Done
- [ ] Second
`
	os.WriteFile(file, []byte(initial), 0644)

	// Enable filter-done, move First down (moves it after Second)
	runPiped(t, file, ":filter-done\rmj\r")

	// Read file after move
	todos := getTodos(t, file)

	if len(todos) != 3 {
		t.Fatalf("Expected 3 todos, got %d", len(todos))
	}

	// Should be: Done, Second, First (First moved after Second, Done stays in place)
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
	// Moving Task A down should move it to Section 2 after Task C
	runPiped(t, file, ":filter-done\rmj\r")

	content := readTestFile(t, file)

	// Task A should have moved to Section 2, Task C stays in Section 2
	// Section 1 should only have Task B (done)
	if !strings.Contains(content, "## Section 1\n\n- [x] Task B") {
		t.Error("Section 1 should only have Task B (done)")
		t.Logf("Content:\n%s", content)
	}
	// Section 2 should have Task C, Task A (moved), Task D (done)
	if !strings.Contains(content, "## Section 2\n\n- [ ] Task C\n- [ ] Task A") {
		t.Error("Task A should have moved to Section 2 after Task C")
		t.Logf("Content:\n%s", content)
	}
}
