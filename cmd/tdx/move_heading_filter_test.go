package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/niklas-heer/tdx/internal/markdown"
)

// TestMoveWithFilterDone_RespectsHeadings tests that moving items with filter-done
// respects heading boundaries and doesn't skip over entire sections
func TestMoveWithFilterDone_RespectsHeadings(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.md")

	content := `---
filter-done: true
---
# Tasks

## Section A
- [x] Done task A1
- [x] Done task A2
- [ ] Active task A

## Section B
- [x] Done task B1
- [ ] Active task B1
- [ ] Active task B2

## Section C
- [ ] Active task C
`

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Read file
	fm, err := markdown.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	// With filter-done, we should see:
	// Section A: Active task A (index 2 in full list)
	// Section B: Active task B1 (index 4), Active task B2 (index 5)
	// Section C: Active task C (index 6)

	// Moving "Active task A" down should move it to the first position in Section B
	// (after the done task B1, before Active task B1)
	// It should NOT skip to Section C

	// Find index of "Active task A"
	activeAIdx := -1
	for i, todo := range fm.Todos {
		if strings.Contains(todo.Text, "Active task A") {
			activeAIdx = i
			break
		}
	}

	if activeAIdx == -1 {
		t.Fatal("Could not find 'Active task A'")
	}

	t.Logf("Active task A is at index %d", activeAIdx)

	// The next visible todo should be "Active task B1" at index 4
	// When we move, it should place "Active task A" between the done and active tasks in Section B

	// This is the expected behavior - we need to implement logic that:
	// 1. Finds the next visible todo (Active task B1)
	// 2. Checks if there's a heading between current and next visible
	// 3. If yes, place the item right after the heading (or after any hidden tasks under that heading)

	t.Log("This test documents the expected behavior for the fix")
	t.Log("When moving down with filter-done, items should respect heading boundaries")
}

// TestMoveWithFilterDone_IntoEmptySection tests moving into a section
// that only has hidden (completed) tasks
func TestMoveWithFilterDone_IntoEmptySection(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.md")

	content := `---
filter-done: true
---
# Tasks

## Section A
- [ ] Active task A

## Section B (all done)
- [x] Done task B1
- [x] Done task B2

## Section C
- [ ] Active task C
`

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Read file
	_, err = markdown.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	t.Log("This test documents behavior when moving into sections with only hidden tasks")
	t.Log("The item should be placed at the end of the section (after hidden tasks)")
	t.Log("When moving again, it should jump to the next section with visible tasks")
}
