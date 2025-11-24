package main

import (
	"os"
	"testing"
)

// TestTUI_MoveWithCompletedItem tests moving with a completed item in between
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

	// Move A down (should swap with B)
	// Sequence: m (move mode), j (swap down), enter (confirm)
	runPiped(t, file, "mj\r")

	content := readTestFile(t, file)
	t.Logf("After move:\n%s", content)
	t.Logf("Swap should have been: A (index 0) with B (index 1)")
	t.Logf("But result suggests: A (index 0) with C (index 2)")

	todos := getTodos(t, file)

	// Expected: B, A, C (A and B swapped)
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
	// With filter active, visible items are: A, C
	// Move A down: A moves after C, B stays in place
	runPiped(t, file, ":filter-done\rmj\r")

	content := readTestFile(t, file)
	t.Logf("After filter-done move:\n%s", content)

	todos := getTodos(t, file)

	if len(todos) != 3 {
		t.Fatalf("Expected 3 todos, got %d", len(todos))
	}

	// Expected: B, C, A (A moved after C, B stays in original position)
	if todos[0] != "- [x] B" {
		t.Errorf("First todo should be '[x] B', got: %s", todos[0])
	}
	if todos[1] != "- [ ] C" {
		t.Errorf("Second todo should be '[ ] C', got: %s", todos[1])
	}
	if todos[2] != "- [ ] A" {
		t.Errorf("Third todo should be '[ ] A', got: %s", todos[2])
	}
}
