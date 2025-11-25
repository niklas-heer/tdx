package main

import (
	"os"
	"strings"
	"testing"
)

// TestMoveWithFilterDone_Simple tests basic move with filter-done (no headings)
// With granular movement, each move goes one position at a time
func TestMoveWithFilterDone_Simple(t *testing.T) {
	file := tempTestFile(t)

	initial := `- [ ] A
- [x] B
- [ ] C
`
	os.WriteFile(file, []byte(initial), 0644)

	t.Log("Initial state:")
	t.Log(initial)

	// Enable filter-done, then move A down ONCE
	// Visible: A, C (B is hidden)
	// With granular movement, A moves one position (swaps with B)
	runPiped(t, file, ":filter-done\rmj\r")

	content := readTestFile(t, file)
	t.Logf("After one move down:\n%s", content)

	todos := getTodos(t, file)

	// Expected order: B, A, C (A moved one position, swapped with B)
	if !strings.Contains(todos[0], "B") {
		t.Errorf("First todo should be B, got: %s", todos[0])
	}
	if !strings.Contains(todos[1], "A") {
		t.Errorf("Second todo should be A (moved one position), got: %s", todos[1])
	}
	if !strings.Contains(todos[2], "C") {
		t.Errorf("Third todo should be C, got: %s", todos[2])
	}
}

// TestMoveWithFilterDone_TwoMoves tests moving twice to get past hidden item
func TestMoveWithFilterDone_TwoMoves(t *testing.T) {
	file := tempTestFile(t)

	initial := `- [ ] A
- [x] B
- [ ] C
`
	os.WriteFile(file, []byte(initial), 0644)

	// Enable filter-done, then move A down TWICE (in same session)
	// First move: A swaps with B → B, A, C
	// Second move: A swaps with C → B, C, A
	runPiped(t, file, ":filter-done\rmjj\r")

	content := readTestFile(t, file)
	t.Logf("After two moves down:\n%s", content)

	todos := getTodos(t, file)

	// Expected order: B, C, A (A moved two positions)
	if !strings.Contains(todos[0], "B") {
		t.Errorf("First todo should be B, got: %s", todos[0])
	}
	if !strings.Contains(todos[1], "C") {
		t.Errorf("Second todo should be C, got: %s", todos[1])
	}
	if !strings.Contains(todos[2], "A") {
		t.Errorf("Third todo should be A (moved two positions), got: %s", todos[2])
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
	// Should swap A with B
	runPiped(t, file, "mj\r")

	content := readTestFile(t, file)
	t.Logf("After move:\n%s", content)

	todos := getTodos(t, file)

	// Expected order: B, A, C
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

// TestMoveWithFilterDone_GranularIsConsistent tests that behavior is the same
// whether filter-done is active or not (granular movement in both cases)
func TestMoveWithFilterDone_GranularIsConsistent(t *testing.T) {
	// Test without filter
	file1 := tempTestFile(t)
	initial := `- [ ] A
- [x] B
- [ ] C
`
	os.WriteFile(file1, []byte(initial), 0644)
	runPiped(t, file1, "mj\r")
	withoutFilter := readTestFile(t, file1)

	// Test with filter
	file2 := tempTestFile(t)
	os.WriteFile(file2, []byte(initial), 0644)
	runPiped(t, file2, ":filter-done\rmj\r")
	withFilter := readTestFile(t, file2)

	// Both should produce the same result (A swapped with B)
	if withoutFilter != withFilter {
		t.Error("Move behavior should be consistent with or without filter")
		t.Logf("Without filter:\n%s", withoutFilter)
		t.Logf("With filter:\n%s", withFilter)
	}
}
