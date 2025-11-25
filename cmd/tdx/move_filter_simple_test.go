package main

import (
	"os"
	"strings"
	"testing"
)

// TestMoveWithFilterDone_Simple tests basic move with filter-done (no headings)
// With visible-swap movement, one 'j' swaps with the next VISIBLE item
func TestMoveWithFilterDone_Simple(t *testing.T) {
	file := tempTestFile(t)

	initial := `- [ ] A
- [x] B
- [ ] C
`
	os.WriteFile(file, []byte(initial), 0644)

	t.Log("Initial state:")
	t.Log(initial)

	// Enable filter-done, then move A down
	// Visible: A, C (B is hidden)
	// With visible-swap: A swaps with C, B stays in place
	runPiped(t, file, ":filter-done\rmj\r")

	content := readTestFile(t, file)
	t.Logf("After visible-swap move:\n%s", content)

	todos := getTodos(t, file)

	// Expected order: C, B, A (A swapped with C - the next visible item)
	if !strings.Contains(todos[0], "C") {
		t.Errorf("First todo should be C (swapped to front), got: %s", todos[0])
	}
	if !strings.Contains(todos[1], "B") {
		t.Errorf("Second todo should be B (hidden, unchanged), got: %s", todos[1])
	}
	if !strings.Contains(todos[2], "A") {
		t.Errorf("Third todo should be A (swapped to back), got: %s", todos[2])
	}
}

// TestMoveWithFilterDone_MoveDownThenUp tests that moving down then up returns to original
func TestMoveWithFilterDone_MoveDownThenUp(t *testing.T) {
	file := tempTestFile(t)

	initial := `- [ ] A
- [x] B
- [ ] C
`
	os.WriteFile(file, []byte(initial), 0644)

	initialTodos := getTodos(t, file)

	// Enable filter-done, move down then up (in single session)
	runPiped(t, file, ":filter-done\rmjk\r")

	afterTodos := getTodos(t, file)
	t.Logf("Initial todos: %v", initialTodos)
	t.Logf("After todos: %v", afterTodos)

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

// TestMoveWithFilterDone_NoFilter tests that move works normally without filter
func TestMoveWithFilterDone_NoFilter(t *testing.T) {
	file := tempTestFile(t)

	initial := `- [ ] A
- [x] B
- [ ] C
`
	os.WriteFile(file, []byte(initial), 0644)

	// Move A down (no filter)
	// Should swap A with B (adjacent in array)
	runPiped(t, file, "mj\r")

	content := readTestFile(t, file)
	t.Logf("After move:\n%s", content)

	todos := getTodos(t, file)

	// Expected order: B, A, C (A swapped with adjacent B)
	if !strings.Contains(todos[0], "B") {
		t.Errorf("First todo should be B, got: %s", todos[0])
	}
	if !strings.Contains(todos[1], "A") {
		t.Errorf("Second todo should be A, got: %s", todos[1])
	}
	if !strings.Contains(todos[2], "C") {
		t.Errorf("Third todo should be C, got: %s", todos[2])
	}
}

// TestMoveWithFilterDone_VisibleSwapIsPredictable tests that visible items swap predictably
func TestMoveWithFilterDone_VisibleSwapIsPredictable(t *testing.T) {
	file := tempTestFile(t)

	initial := `- [ ] Task 1
- [x] Done A
- [x] Done B
- [ ] Task 2
- [x] Done C
- [ ] Task 3
`
	os.WriteFile(file, []byte(initial), 0644)

	// Enable filter-done, move Task 1 down twice (in single session)
	// Visible: Task 1, Task 2, Task 3
	// After first swap: Task 2, Task 1, Task 3
	// After second swap: Task 2, Task 3, Task 1
	runPiped(t, file, ":filter-done\rmjj\r")

	content := readTestFile(t, file)
	t.Logf("After two visible-swaps:\n%s", content)

	// Extract visible todos
	todos := getTodos(t, file)
	var visible []string
	for _, todo := range todos {
		if strings.Contains(todo, "[ ]") {
			visible = append(visible, todo)
		}
	}

	if len(visible) != 3 {
		t.Fatalf("Expected 3 visible todos, got %d", len(visible))
	}

	// Visible order should be: Task 2, Task 3, Task 1
	if !strings.Contains(visible[0], "Task 2") {
		t.Errorf("First visible should be Task 2, got: %s", visible[0])
	}
	if !strings.Contains(visible[1], "Task 3") {
		t.Errorf("Second visible should be Task 3, got: %s", visible[1])
	}
	if !strings.Contains(visible[2], "Task 1") {
		t.Errorf("Third visible should be Task 1, got: %s", visible[2])
	}
}

// TestMoveWithFilterDone_CantMovePastEnd tests that you can't move past the last visible item
func TestMoveWithFilterDone_CantMovePastEnd(t *testing.T) {
	file := tempTestFile(t)

	initial := `- [ ] A
- [x] B
- [ ] C
`
	os.WriteFile(file, []byte(initial), 0644)

	// Enable filter-done, try to move A down 5 times (but only 1 visible item after)
	runPiped(t, file, ":filter-done\rmjjjjj\r")

	content := readTestFile(t, file)
	t.Logf("After multiple move attempts:\n%s", content)

	todos := getTodos(t, file)

	// A should be at the end (swapped with C, can't go further)
	// Expected: C, B, A
	if !strings.Contains(todos[2], "A") {
		t.Errorf("A should be at the end, got: %s", todos[2])
	}
}
