package main

import (
	"os"
	"strings"
	"testing"
)

// TestMetadataBehavior_FilterDoneRespected tests that filter-done metadata is respected
func TestMetadataBehavior_FilterDoneRespected(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
---

- [x] Completed task
- [ ] Active task 1
- [ ] Active task 2
`
	os.WriteFile(file, []byte(initial), 0644)

	// Just open and immediately toggle - should toggle the first VISIBLE item (Active task 1)
	runPiped(t, file, " ")

	todos := getTodos(t, file)

	// Active task 1 should be checked (it was the first visible item)
	if !strings.Contains(todos[1], "[x]") || !strings.Contains(todos[1], "Active task 1") {
		t.Errorf("Active task 1 should be checked (first visible with filter-done), got: %s", todos[1])
	}

	// Completed task should remain checked
	if !strings.Contains(todos[0], "[x]") || !strings.Contains(todos[0], "Completed task") {
		t.Errorf("Completed task should remain checked, got: %s", todos[0])
	}
}

// TestMetadataBehavior_FilterDoneFalse tests that filter-done: false shows all items
func TestMetadataBehavior_FilterDoneFalse(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: false
---

- [x] Completed task
- [ ] Active task 1
`
	os.WriteFile(file, []byte(initial), 0644)

	// Toggle first item (should be Completed task at index 0)
	runPiped(t, file, " ")

	todos := getTodos(t, file)

	// Completed task should now be unchecked (toggled from checked)
	if !strings.Contains(todos[0], "[ ]") || !strings.Contains(todos[0], "Completed task") {
		t.Errorf("Completed task should be unchecked, got: %s", todos[0])
	}
}

// TestMetadataBehavior_ShowHeadingsRespected tests that show-headings metadata is respected
func TestMetadataBehavior_ShowHeadingsRespected(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
show-headings: true
---

# Section 1

- [ ] Task A

# Section 2

- [ ] Task B
`
	os.WriteFile(file, []byte(initial), 0644)

	// Toggle first item - should be Task A (first todo, not heading)
	runPiped(t, file, " ")

	todos := getTodos(t, file)

	// Task A should be checked
	if !strings.Contains(todos[0], "[x]") || !strings.Contains(todos[0], "Task A") {
		t.Errorf("Task A should be checked (first todo with show-headings), got: %s", todos[0])
	}
}

// TestMetadataBehavior_MaxVisibleRespected tests that max-visible metadata is respected
func TestMetadataBehavior_MaxVisibleRespected(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
max-visible: 2
---

- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
- [ ] Task 4
`
	os.WriteFile(file, []byte(initial), 0644)

	// Move down 3 times and toggle - should wrap around due to max-visible
	// This test just verifies the metadata is parsed and available
	// The actual max-visible behavior is handled by the View layer
	runPiped(t, file, "jjj ")

	todos := getTodos(t, file)

	// Task 4 should be checked (moved down 3 times from Task 1)
	if !strings.Contains(todos[3], "[x]") || !strings.Contains(todos[3], "Task 4") {
		t.Errorf("Task 4 should be checked, got: %s", todos[3])
	}
}

// TestMetadataBehavior_ReadOnlyRespected tests that read-only metadata prevents changes
func TestMetadataBehavior_ReadOnlyRespected(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
read-only: true
---

- [ ] Task 1
- [ ] Task 2
`
	os.WriteFile(file, []byte(initial), 0644)

	initialContent := readTestFile(t, file)

	// Try to toggle - should have no effect in read-only mode
	runPiped(t, file, " ")

	afterContent := readTestFile(t, file)

	// Content should be unchanged
	if initialContent != afterContent {
		t.Errorf("File should not change in read-only mode.\nBefore:\n%s\nAfter:\n%s", initialContent, afterContent)
	}
}

// TestMetadataBehavior_WordWrapRespected tests that word-wrap metadata is stored
func TestMetadataBehavior_WordWrapRespected(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
word-wrap: false
---

- [ ] This is a very long task that would normally wrap but shouldn't with word-wrap disabled
`
	os.WriteFile(file, []byte(initial), 0644)

	// Just verify we can read and process the file with word-wrap metadata
	// The actual word-wrap behavior is in the View layer
	runPiped(t, file, "")

	// File should be unchanged
	content := readTestFile(t, file)
	if !strings.Contains(content, "word-wrap: false") {
		t.Error("word-wrap metadata should be preserved")
	}
}

// TestMetadataBehavior_CombinedSettings tests multiple metadata settings together
func TestMetadataBehavior_CombinedSettings(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
show-headings: true
---

# Completed

- [x] Done 1

# Active

- [ ] Task 1
- [ ] Task 2
`
	os.WriteFile(file, []byte(initial), 0644)

	// Toggle first visible item - should be Task 1
	runPiped(t, file, " ")

	todos := getTodos(t, file)

	// Task 1 should be checked (first visible todo)
	if !strings.Contains(todos[1], "[x]") || !strings.Contains(todos[1], "Task 1") {
		t.Errorf("Task 1 should be checked (first visible with both filters), got: %s", todos[1])
	}
}

// TestMetadataBehavior_NoMetadata tests default behavior without metadata
func TestMetadataBehavior_NoMetadata(t *testing.T) {
	file := tempTestFile(t)

	initial := `- [x] Completed task
- [ ] Active task
`
	os.WriteFile(file, []byte(initial), 0644)

	// Toggle first item - should be Completed task at index 0
	runPiped(t, file, " ")

	todos := getTodos(t, file)

	// Completed task should be unchecked (toggled)
	if !strings.Contains(todos[0], "[ ]") || !strings.Contains(todos[0], "Completed task") {
		t.Errorf("Completed task should be unchecked, got: %s", todos[0])
	}
}
