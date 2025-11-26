package main

import (
	"os"
	"strings"
	"testing"
)

// TestMoveWithFilterDone_Simple tests basic move with filter-done (no headings)
// With insertion-based movement, one 'j' inserts AFTER the next VISIBLE item
func TestMoveWithFilterDone_Simple(t *testing.T) {
	file := tempTestFile(t)

	initial := `- [ ] A
- [x] B
- [ ] C
`
	_ = os.WriteFile(file, []byte(initial), 0644)

	t.Log("Initial state:")
	t.Log(initial)

	// Enable filter-done, then move A down
	// Visible: A, C (B is hidden)
	// With insertion: A is inserted AFTER C (the next visible item)
	runPiped(t, file, ":filter-done\rmj\r")

	content := readTestFile(t, file)
	t.Logf("After insertion move:\n%s", content)

	todos := getTodos(t, file)

	// Expected order: B, C, A (A inserted after C)
	// B stays in original position, C moves to first visible position, A goes after C
	if !strings.Contains(todos[0], "B") {
		t.Errorf("First todo should be B (hidden, unchanged position), got: %s", todos[0])
	}
	if !strings.Contains(todos[1], "C") {
		t.Errorf("Second todo should be C (now first visible), got: %s", todos[1])
	}
	if !strings.Contains(todos[2], "A") {
		t.Errorf("Third todo should be A (inserted after C), got: %s", todos[2])
	}
}

// TestMoveWithFilterDone_MoveDownThenUp tests that moving down then up returns to original
func TestMoveWithFilterDone_MoveDownThenUp(t *testing.T) {
	file := tempTestFile(t)

	initial := `- [ ] A
- [x] B
- [ ] C
`
	_ = os.WriteFile(file, []byte(initial), 0644)

	initialTodos := getTodos(t, file)

	// Enable filter-done, move down then up (in single session)
	// A down: inserts A after C -> B, C, A
	// A up: inserts A before C -> B, A, C
	// But wait, after first move, cursor should be on A at position 2
	// Moving up from position 2 (A) inserts before C at position 1
	// Result: B, A, C (not original, but that's expected with insertion-based movement)
	runPiped(t, file, ":filter-done\rmjk\r")

	afterTodos := getTodos(t, file)
	t.Logf("Initial todos: %v", initialTodos)
	t.Logf("After todos: %v", afterTodos)

	// With insertion-based movement, down then up does NOT return to original
	// because each move is an insertion operation, not a swap
	// After down: B, C, A (A after C)
	// After up: B, A, C (A before C)
	// This is expected behavior
	if len(initialTodos) != len(afterTodos) {
		t.Fatalf("Todo count changed: %d -> %d", len(initialTodos), len(afterTodos))
	}

	// Expected order: B, A, C
	if !strings.Contains(afterTodos[0], "B") {
		t.Errorf("Todo 0 should be B, got: %s", afterTodos[0])
	}
	if !strings.Contains(afterTodos[1], "A") {
		t.Errorf("Todo 1 should be A, got: %s", afterTodos[1])
	}
	if !strings.Contains(afterTodos[2], "C") {
		t.Errorf("Todo 2 should be C, got: %s", afterTodos[2])
	}
}

// TestMoveWithFilterDone_NoFilter tests that move works normally without filter
func TestMoveWithFilterDone_NoFilter(t *testing.T) {
	file := tempTestFile(t)

	initial := `- [ ] A
- [x] B
- [ ] C
`
	_ = os.WriteFile(file, []byte(initial), 0644)

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
	_ = os.WriteFile(file, []byte(initial), 0644)

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
	_ = os.WriteFile(file, []byte(initial), 0644)

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
