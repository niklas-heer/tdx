package main

import (
	"os"
	"testing"
)

// TestTUI_MoveWithCompletedItem tests moving with a completed item in between
// Without filters, movement swaps with adjacent item in the array
func TestTUI_MoveWithCompletedItem(t *testing.T) {
	file := tempTestFile(t)

	// Create file with B already completed
	initial := `# Todos

- [ ] A
- [x] B
- [ ] C
`
	os.WriteFile(file, []byte(initial), 0644)

	t.Log("Initial file:")
	t.Log(initial)

	// Move A down (should swap with B - adjacent in array)
	// Sequence: m (move mode), j (swap down), enter (confirm)
	runPiped(t, file, "mj\r")

	content := readTestFile(t, file)
	t.Logf("After move:\n%s", content)

	todos := getTodos(t, file)

	// Expected: B, A, C (A and B swapped - adjacent swap without filter)
	if len(todos) != 3 {
		t.Fatalf("Expected 3 todos, got %d", len(todos))
	}

	// B should be first (and checked)
	if todos[0] != "- [x] B" {
		t.Errorf("First todo should be '[x] B', got: %s", todos[0])
	}

	// A should be second
	if todos[1] != "- [ ] A" {
		t.Errorf("Second todo should be '[ ] A', got: %s", todos[1])
	}

	// C should be third
	if todos[2] != "- [ ] C" {
		t.Errorf("Third todo should be '[ ] C', got: %s", todos[2])
	}
}

// TestTUI_MoveWithFilterDone_ReallySimple tests the simplest filter-done move case
// With filter-done active, movement swaps with next VISIBLE item
func TestTUI_MoveWithFilterDone_ReallySimple(t *testing.T) {
	file := tempTestFile(t)

	// Use CLI to create clear starting state
	runCLI(t, file, "add", "A")
	runCLI(t, file, "add", "B")
	runCLI(t, file, "add", "C")

	// Mark B complete
	runPiped(t, file, "j ")

	t.Log("Initial state: A, [x]B, C")

	// Enable filter-done, then move A down
	// With visible-swap: A swaps with C (next visible), B stays in place
	// Result: C, B, A
	runPiped(t, file, ":filter-done\rmj\r")

	content := readTestFile(t, file)
	t.Logf("After filter-done visible-swap:\n%s", content)

	todos := getTodos(t, file)

	if len(todos) != 3 {
		t.Fatalf("Expected 3 todos, got %d", len(todos))
	}

	// Expected: C, B, A (A swapped with C - the next visible item)
	if todos[0] != "- [ ] C" {
		t.Errorf("First todo should be '[ ] C', got: %s", todos[0])
	}
	if todos[1] != "- [x] B" {
		t.Errorf("Second todo should be '[x] B', got: %s", todos[1])
	}
	if todos[2] != "- [ ] A" {
		t.Errorf("Third todo should be '[ ] A', got: %s", todos[2])
	}
}

// TestTUI_MoveWithFilterDone_MoveBackUp tests that moving down then up returns to original
func TestTUI_MoveWithFilterDone_MoveBackUp(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "A")
	runCLI(t, file, "add", "B")
	runCLI(t, file, "add", "C")

	// Mark B complete
	runPiped(t, file, "j ")

	initial := readTestFile(t, file)
	t.Logf("Initial:\n%s", initial)

	// Enable filter-done, move down then up (in single session)
	runPiped(t, file, ":filter-done\rmjk\r")

	after := readTestFile(t, file)
	t.Logf("After down then up:\n%s", after)

	// Should return to original state
	if initial != after {
		t.Error("Moving down then up should return to original position")
	}
}

// TestTUI_MoveWithFilterDone_VisibleOrderChanges verifies visually predictable movement
func TestTUI_MoveWithFilterDone_VisibleOrderChanges(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Todos

- [ ] Task 1
- [x] Hidden A
- [ ] Task 2
- [x] Hidden B
- [ ] Task 3
`
	os.WriteFile(file, []byte(initial), 0644)

	// Enable filter-done, move Task 1 down once
	// Visible order: Task 1, Task 2, Task 3
	// After swap: Task 2, Task 1, Task 3
	runPiped(t, file, ":filter-done\rmj\r")

	todos := getTodos(t, file)
	content := readTestFile(t, file)
	t.Logf("After one visible-swap:\n%s", content)

	// Verify visible order changed predictably
	visibleTodos := []string{}
	for _, todo := range todos {
		if todo[:6] == "- [ ] " {
			visibleTodos = append(visibleTodos, todo)
		}
	}

	if len(visibleTodos) != 3 {
		t.Fatalf("Expected 3 visible todos, got %d", len(visibleTodos))
	}

	// Visible order should be: Task 2, Task 1, Task 3
	if visibleTodos[0] != "- [ ] Task 2" {
		t.Errorf("First visible should be Task 2, got: %s", visibleTodos[0])
	}
	if visibleTodos[1] != "- [ ] Task 1" {
		t.Errorf("Second visible should be Task 1, got: %s", visibleTodos[1])
	}
	if visibleTodos[2] != "- [ ] Task 3" {
		t.Errorf("Third visible should be Task 3, got: %s", visibleTodos[2])
	}
}
