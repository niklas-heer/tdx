package main

import (
	"os"
	"strings"
	"testing"
)

// Helper function to verify frontmatter exists in file
func assertFrontmatterExists(t *testing.T, file string, expectedSettings map[string]string) {
	t.Helper()
	content := readTestFile(t, file)

	if !strings.HasPrefix(content, "---\n") {
		t.Errorf("File should start with frontmatter delimiter '---', got:\n%s", content[:50])
	}

	for key, value := range expectedSettings {
		expected := key + ": " + value
		if !strings.Contains(content, expected) {
			t.Errorf("Frontmatter should contain '%s', full content:\n%s", expected, content)
		}
	}

	// Count frontmatter delimiters - should have exactly 2
	count := strings.Count(content, "---\n")
	if count < 2 {
		t.Errorf("Frontmatter should have opening and closing '---', found %d, content:\n%s", count, content)
	}
}

// TestFrontmatterPreservation_Toggle tests that toggling preserves frontmatter
func TestFrontmatterPreservation_Toggle(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
show-headings: true
---

- [ ] Task 1
- [ ] Task 2
`
	os.WriteFile(file, []byte(initial), 0644)

	// Toggle first task
	runPiped(t, file, " ")

	assertFrontmatterExists(t, file, map[string]string{
		"filter-done":   "true",
		"show-headings": "true",
	})

	todos := getTodos(t, file)
	if len(todos) != 2 {
		t.Errorf("Should have 2 todos after toggle, got %d", len(todos))
	}
}

// TestFrontmatterPreservation_MultipleToggles tests multiple toggle operations
func TestFrontmatterPreservation_MultipleToggles(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
word-wrap: false
max-visible: 10
---

- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
`
	os.WriteFile(file, []byte(initial), 0644)

	// Toggle multiple times
	runPiped(t, file, " j j ")

	assertFrontmatterExists(t, file, map[string]string{
		"filter-done": "true",
		"word-wrap":   "false",
		"max-visible": "10",
	})
}

// TestFrontmatterPreservation_Delete tests that deleting preserves frontmatter
func TestFrontmatterPreservation_Delete(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
show-headings: true
---

- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
`
	os.WriteFile(file, []byte(initial), 0644)

	// Delete first task
	runPiped(t, file, "d")

	assertFrontmatterExists(t, file, map[string]string{
		"filter-done":   "true",
		"show-headings": "true",
	})

	todos := getTodos(t, file)
	if len(todos) != 2 {
		t.Errorf("Should have 2 todos after delete, got %d", len(todos))
	}
}

// TestFrontmatterPreservation_DeleteMultiple tests deleting multiple items
func TestFrontmatterPreservation_DeleteMultiple(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
read-only: false
---

- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
- [ ] Task 4
`
	os.WriteFile(file, []byte(initial), 0644)

	// Delete multiple tasks
	runPiped(t, file, "djd")

	assertFrontmatterExists(t, file, map[string]string{
		"filter-done": "true",
		"read-only":   "false",
	})

	todos := getTodos(t, file)
	if len(todos) != 2 {
		t.Errorf("Should have 2 todos after deleting 2, got %d", len(todos))
	}
}

// TestFrontmatterPreservation_Add tests that adding items preserves frontmatter
func TestFrontmatterPreservation_Add(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
---

- [ ] Task 1
`
	os.WriteFile(file, []byte(initial), 0644)

	// Add a new task
	runPiped(t, file, "nNew Task\r")

	assertFrontmatterExists(t, file, map[string]string{
		"filter-done": "true",
	})

	todos := getTodos(t, file)
	if len(todos) != 2 {
		t.Errorf("Should have 2 todos after add, got %d", len(todos))
	}
}

// TestFrontmatterPreservation_Edit tests that editing preserves frontmatter
func TestFrontmatterPreservation_Edit(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
word-wrap: true
---

- [ ] Original Task
`
	os.WriteFile(file, []byte(initial), 0644)

	// Edit the task
	runPiped(t, file, "e\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7fEdited Task\r")

	assertFrontmatterExists(t, file, map[string]string{
		"filter-done": "true",
		"word-wrap":   "true",
	})

	todos := getTodos(t, file)
	if !strings.Contains(todos[0], "Edited Task") {
		t.Errorf("Task should be edited, got: %s", todos[0])
	}
}

// TestFrontmatterPreservation_Move tests that moving items preserves frontmatter
func TestFrontmatterPreservation_Move(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
show-headings: false
---

- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
`
	os.WriteFile(file, []byte(initial), 0644)

	// Move first task down
	runPiped(t, file, "mj\r")

	assertFrontmatterExists(t, file, map[string]string{
		"filter-done":   "true",
		"show-headings": "false",
	})

	todos := getTodos(t, file)
	if len(todos) != 3 {
		t.Errorf("Should still have 3 todos after move, got %d", len(todos))
	}
}

// TestFrontmatterPreservation_ClearDone tests that :clear-done preserves frontmatter
func TestFrontmatterPreservation_ClearDone(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
max-visible: 5
---

- [x] Done Task 1
- [x] Done Task 2
- [ ] Active Task
`
	os.WriteFile(file, []byte(initial), 0644)

	// Clear done tasks
	runPiped(t, file, ":clear-done\r")

	assertFrontmatterExists(t, file, map[string]string{
		"filter-done": "true",
		"max-visible": "5",
	})

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Errorf("Should have 1 todo after clear-done, got %d", len(todos))
	}
}

// TestFrontmatterPreservation_CheckAll tests that :check-all preserves frontmatter
func TestFrontmatterPreservation_CheckAll(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: false
word-wrap: true
---

- [ ] Task 1
- [ ] Task 2
`
	os.WriteFile(file, []byte(initial), 0644)

	// Check all tasks
	runPiped(t, file, ":check-all\r")

	assertFrontmatterExists(t, file, map[string]string{
		"filter-done": "false",
		"word-wrap":   "true",
	})

	todos := getTodos(t, file)
	for i, todo := range todos {
		if !strings.Contains(todo, "[x]") {
			t.Errorf("Todo %d should be checked, got: %s", i, todo)
		}
	}
}

// TestFrontmatterPreservation_UncheckAll tests that :uncheck-all preserves frontmatter
func TestFrontmatterPreservation_UncheckAll(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
---

- [x] Task 1
- [x] Task 2
`
	os.WriteFile(file, []byte(initial), 0644)

	// Uncheck all tasks
	runPiped(t, file, ":uncheck-all\r")

	assertFrontmatterExists(t, file, map[string]string{
		"filter-done": "true",
	})

	todos := getTodos(t, file)
	for i, todo := range todos {
		if !strings.Contains(todo, "[ ]") {
			t.Errorf("Todo %d should be unchecked, got: %s", i, todo)
		}
	}
}

// TestFrontmatterPreservation_Sort tests that :sort preserves frontmatter
func TestFrontmatterPreservation_Sort(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
show-headings: true
---

- [ ] Zebra
- [ ] Apple
- [ ] Banana
`
	os.WriteFile(file, []byte(initial), 0644)

	// Sort tasks
	runPiped(t, file, ":sort\r")

	assertFrontmatterExists(t, file, map[string]string{
		"filter-done":   "true",
		"show-headings": "true",
	})

	todos := getTodos(t, file)
	if len(todos) != 3 {
		t.Errorf("Should have 3 todos after sort, got %d", len(todos))
	}
}

// TestFrontmatterPreservation_ComplexOperations tests a series of operations
func TestFrontmatterPreservation_ComplexOperations(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
show-headings: true
word-wrap: false
max-visible: 10
read-only: false
---

- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
`
	os.WriteFile(file, []byte(initial), 0644)

	// Perform multiple operations in sequence
	runPiped(t, file, " ")      // Toggle first
	runPiped(t, file, "jd")     // Move down and delete
	runPiped(t, file, "nNew\r") // Add new task

	assertFrontmatterExists(t, file, map[string]string{
		"filter-done":   "true",
		"show-headings": "true",
		"word-wrap":     "false",
		"max-visible":   "10",
		"read-only":     "false",
	})
}

// TestFrontmatterPreservation_WithHeadings tests operations with headings
func TestFrontmatterPreservation_WithHeadings(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
show-headings: true
---

# Section 1

- [ ] Task 1

# Section 2

- [ ] Task 2
`
	os.WriteFile(file, []byte(initial), 0644)

	// Toggle and delete
	runPiped(t, file, " jd")

	assertFrontmatterExists(t, file, map[string]string{
		"filter-done":   "true",
		"show-headings": "true",
	})

	// Verify headings are still there
	content := readTestFile(t, file)
	if !strings.Contains(content, "# Section 1") {
		t.Error("Section 1 heading should be preserved")
	}
	if !strings.Contains(content, "# Section 2") {
		t.Error("Section 2 heading should be preserved")
	}
}

// TestFrontmatterPreservation_EmptyFile tests operations that delete all todos
func TestFrontmatterPreservation_EmptyFile(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
---

- [ ] Only Task
`
	os.WriteFile(file, []byte(initial), 0644)

	// Delete the only task
	runPiped(t, file, "d")

	assertFrontmatterExists(t, file, map[string]string{
		"filter-done": "true",
	})

	todos := getTodos(t, file)
	if len(todos) != 0 {
		t.Errorf("Should have 0 todos after deleting all, got %d", len(todos))
	}
}

// TestFrontmatterPreservation_CLIOperations tests CLI operations preserve frontmatter
func TestFrontmatterPreservation_CLIOperations(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
---

- [ ] Task 1
`
	os.WriteFile(file, []byte(initial), 0644)

	// Use CLI to add a task
	runCLI(t, file, "add", "New CLI Task")

	assertFrontmatterExists(t, file, map[string]string{
		"filter-done": "true",
	})

	todos := getTodos(t, file)
	if len(todos) != 2 {
		t.Errorf("Should have 2 todos after CLI add, got %d", len(todos))
	}
}

// TestFrontmatterPreservation_ToggleCLI tests CLI toggle preserves frontmatter
func TestFrontmatterPreservation_ToggleCLI(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
word-wrap: false
---

- [ ] Task 1
- [ ] Task 2
`
	os.WriteFile(file, []byte(initial), 0644)

	// Use CLI to toggle
	runCLI(t, file, "toggle", "1")

	assertFrontmatterExists(t, file, map[string]string{
		"filter-done": "true",
		"word-wrap":   "false",
	})
}
