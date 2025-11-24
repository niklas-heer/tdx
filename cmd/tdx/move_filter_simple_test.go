package main

import (
	"os"
	"strings"
	"testing"
)

// TestMoveWithFilterDone_Simple tests basic move with filter-done (no headings)
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
	// Move A down moves it after C, B stays in place
	runPiped(t, file, ":filter-done\rmj\r")

	content := readTestFile(t, file)
	t.Logf("After move:\n%s", content)

	todos := getTodos(t, file)

	// Expected order: B, C, A (A moved after C, B unchanged)
	if !strings.Contains(todos[0], "B") {
		t.Errorf("First todo should be B (hidden, unchanged), got: %s", todos[0])
	}
	if !strings.Contains(todos[1], "C") {
		t.Errorf("Second todo should be C, got: %s", todos[1])
	}
	if !strings.Contains(todos[2], "A") {
		t.Errorf("Third todo should be A (moved), got: %s", todos[2])
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
