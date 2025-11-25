package main

import (
	"os"
	"testing"
)

// TestTUI_MoveWithCompletedItem tests moving with a completed item in between
// With granular movement, one 'j' moves one position (swaps with adjacent item)
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

	// Move A down (should swap with B - one position at a time)
	// Sequence: m (move mode), j (swap down), enter (confirm)
	runPiped(t, file, "mj\r")

	content := readTestFile(t, file)
	t.Logf("After move:\n%s", content)

	todos := getTodos(t, file)

	// Expected: B, A, C (A and B swapped - granular movement)
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
// With granular movement, one 'j' moves one position even with filter-done active
func TestTUI_MoveWithFilterDone_ReallySimple(t *testing.T) {
	file := tempTestFile(t)

	// Use CLI to create clear starting state
	runCLI(t, file, "add", "A")
	runCLI(t, file, "add", "B")
	runCLI(t, file, "add", "C")

	// Mark B complete
	runPiped(t, file, "j ")

	t.Log("Initial state: A, [x]B, C")

	// Enable filter-done, then move A down ONCE
	// With granular movement, A swaps with B (one position)
	runPiped(t, file, ":filter-done\rmj\r")

	content := readTestFile(t, file)
	t.Logf("After filter-done move (one position):\n%s", content)

	todos := getTodos(t, file)

	if len(todos) != 3 {
		t.Fatalf("Expected 3 todos, got %d", len(todos))
	}

	// Expected: B, A, C (A moved one position, swapped with B)
	// This is the same as without filter - granular movement is consistent
	if todos[0] != "- [x] B" {
		t.Errorf("First todo should be '[x] B', got: %s", todos[0])
	}
	if todos[1] != "- [ ] A" {
		t.Errorf("Second todo should be '[ ] A', got: %s", todos[1])
	}
	if todos[2] != "- [ ] C" {
		t.Errorf("Third todo should be '[ ] C', got: %s", todos[2])
	}
}

// TestTUI_MoveWithFilterDone_TwoMoves tests moving twice to get past hidden item
func TestTUI_MoveWithFilterDone_TwoMoves(t *testing.T) {
	file := tempTestFile(t)

	// Use CLI to create clear starting state
	runCLI(t, file, "add", "A")
	runCLI(t, file, "add", "B")
	runCLI(t, file, "add", "C")

	// Mark B complete
	runPiped(t, file, "j ")

	t.Log("Initial state: A, [x]B, C")

	// Enable filter-done, then move A down TWICE to get past B
	runPiped(t, file, ":filter-done\rmjj\r")

	content := readTestFile(t, file)
	t.Logf("After filter-done move (two positions):\n%s", content)

	todos := getTodos(t, file)

	if len(todos) != 3 {
		t.Fatalf("Expected 3 todos, got %d", len(todos))
	}

	// Expected: B, C, A (A moved two positions)
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
