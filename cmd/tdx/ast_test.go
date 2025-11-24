package main

import (
	"os"
	"strings"
	"testing"

	"github.com/niklas-heer/tdx/internal/markdown"
)

// TestAST_RoundtripParsing tests that parse → serialize → parse is stable
func TestAST_RoundtripParsing(t *testing.T) {
	initial := `# Todos

- [ ] First task
- [x] Second task
- [ ] Third task`

	// Parse once
	fm1 := markdown.ParseMarkdown(initial)
	content1 := markdown.SerializeMarkdown(fm1)

	// Parse again
	fm2 := markdown.ParseMarkdown(content1)
	content2 := markdown.SerializeMarkdown(fm2)

	// Second and subsequent serializations should be identical
	if content1 != content2 {
		t.Errorf("Roundtrip not stable:\nFirst:\n%s\nSecond:\n%s", content1, content2)
	}

	// Should have same number of todos
	if len(fm1.Todos) != len(fm2.Todos) {
		t.Errorf("Todo count changed: %d -> %d", len(fm1.Todos), len(fm2.Todos))
	}
}

// TestAST_PreservesStructure tests that AST preserves document structure
func TestAST_PreservesStructure(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Main Heading

Intro paragraph.

## Section 1

- [ ] Task in section 1

Some text.

## Section 2

- [ ] Task in section 2

More text.`

	os.WriteFile(file, []byte(initial), 0644)

	// Toggle first task
	runCLI(t, file, "toggle", "1")

	content, _ := os.ReadFile(file)
	result := string(content)

	// All structure should be preserved
	headingCount := strings.Count(result, "# Main Heading") +
		strings.Count(result, "## Section 1") +
		strings.Count(result, "## Section 2")

	if headingCount != 3 {
		t.Errorf("Expected 3 headings, found %d", headingCount)
	}

	if !strings.Contains(result, "Intro paragraph.") {
		t.Error("Lost intro paragraph")
	}
	if !strings.Contains(result, "Some text.") {
		t.Error("Lost 'Some text.'")
	}
	if !strings.Contains(result, "More text.") {
		t.Error("Lost 'More text.'")
	}
}

// TestAST_HandlesComplexNesting tests AST with complex nesting
func TestAST_HandlesComplexNesting(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Todos

## Active

### High Priority

- [ ] Important task

### Low Priority

- [ ] Less important

## Done

- [x] Completed task`

	os.WriteFile(file, []byte(initial), 0644)

	todos := getTodos(t, file)
	if len(todos) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(todos))
	}

	// Add a task - should preserve all structure
	runCLI(t, file, "add", "New task")

	content, _ := os.ReadFile(file)
	result := string(content)

	if !strings.Contains(result, "### High Priority") {
		t.Error("Lost nested heading")
	}
	if !strings.Contains(result, "## Active") {
		t.Error("Lost section heading")
	}
}

// TestAST_MultipleOperationsSequence tests sequential operations
func TestAST_MultipleOperationsSequence(t *testing.T) {
	file := tempTestFile(t)

	// Start fresh
	runCLI(t, file, "add", "Task 1")
	runCLI(t, file, "add", "Task 2")
	runCLI(t, file, "add", "Task 3")

	// Toggle, edit, toggle, delete sequence
	runCLI(t, file, "toggle", "1")
	runCLI(t, file, "edit", "2", "Modified task 2")
	runCLI(t, file, "toggle", "3")
	runCLI(t, file, "delete", "1")

	todos := getTodos(t, file)
	if len(todos) != 2 {
		t.Errorf("Expected 2 tasks after delete, got %d", len(todos))
	}

	if !strings.Contains(todos[0], "Modified task 2") {
		t.Error("Edit was lost")
	}
	if !strings.HasPrefix(todos[1], "- [x] ") {
		t.Error("Toggle was lost")
	}
}

// TestAST_AddMultipleTasks tests adding many tasks in sequence
func TestAST_AddMultipleTasks(t *testing.T) {
	file := tempTestFile(t)

	for i := 1; i <= 10; i++ {
		runCLI(t, file, "add", "Task "+string(rune('0'+i)))
	}

	todos := getTodos(t, file)
	if len(todos) != 10 {
		t.Errorf("Expected 10 tasks, got %d", len(todos))
	}

	// Verify order is preserved
	for i := 1; i <= 10; i++ {
		expected := "Task " + string(rune('0'+i))
		if !strings.Contains(todos[i-1], expected) {
			t.Errorf("Task %d wrong or out of order: %s", i, todos[i-1])
		}
	}
}

// TestAST_ToggleBackAndForth tests toggling same task multiple times
func TestAST_ToggleBackAndForth(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Toggle test")

	// Toggle 5 times
	for i := 0; i < 5; i++ {
		runCLI(t, file, "toggle", "1")
	}

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Errorf("Expected 1 task, got %d", len(todos))
	}

	// Should be checked (odd number of toggles)
	if !strings.HasPrefix(todos[0], "- [x] ") {
		t.Error("Task should be checked after 5 toggles")
	}

	// Toggle once more
	runCLI(t, file, "toggle", "1")
	todos = getTodos(t, file)

	// Should be unchecked
	if !strings.HasPrefix(todos[0], "- [ ] ") {
		t.Error("Task should be unchecked after 6 toggles")
	}
}

// TestAST_EditMultipleTimes tests editing same task multiple times
func TestAST_EditMultipleTimes(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Original")

	edits := []string{"Edit 1", "Edit 2", "Edit 3", "Final edit"}
	for _, edit := range edits {
		runCLI(t, file, "edit", "1", edit)
	}

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Errorf("Expected 1 task, got %d", len(todos))
	}

	if !strings.Contains(todos[0], "Final edit") {
		t.Errorf("Expected 'Final edit', got: %s", todos[0])
	}

	// Should not contain old text
	if strings.Contains(todos[0], "Original") {
		t.Error("Old text 'Original' still present")
	}
	if strings.Contains(todos[0], "Edit 1") {
		t.Error("Old text 'Edit 1' still present")
	}
}

// TestAST_DeleteFromDifferentPositions tests deleting from start, middle, end
func TestAST_DeleteFromDifferentPositions(t *testing.T) {
	file := tempTestFile(t)

	// Setup: 5 tasks
	for i := 1; i <= 5; i++ {
		runCLI(t, file, "add", "Task "+string(rune('A'+i-1)))
	}

	// Delete from end (task 5)
	runCLI(t, file, "delete", "5")
	todos := getTodos(t, file)
	if len(todos) != 4 {
		t.Errorf("After delete from end: expected 4 tasks, got %d", len(todos))
	}

	// Delete from middle (task 2)
	runCLI(t, file, "delete", "2")
	todos = getTodos(t, file)
	if len(todos) != 3 {
		t.Errorf("After delete from middle: expected 3 tasks, got %d", len(todos))
	}

	// Delete from beginning (task 1)
	runCLI(t, file, "delete", "1")
	todos = getTodos(t, file)
	if len(todos) != 2 {
		t.Errorf("After delete from beginning: expected 2 tasks, got %d", len(todos))
	}

	// Should have Task C and D remaining
	if !strings.Contains(todos[0], "Task C") {
		t.Errorf("Expected Task C, got: %s", todos[0])
	}
	if !strings.Contains(todos[1], "Task D") {
		t.Errorf("Expected Task D, got: %s", todos[1])
	}
}

// TestAST_MixedOperations tests realistic usage with mixed operations
func TestAST_MixedOperations(t *testing.T) {
	file := tempTestFile(t)

	// Realistic workflow
	runCLI(t, file, "add", "Buy groceries")
	runCLI(t, file, "add", "Write report")
	runCLI(t, file, "add", "Call dentist")
	runCLI(t, file, "toggle", "1") // Done with groceries
	runCLI(t, file, "add", "Fix bug")
	runCLI(t, file, "edit", "2", "Write quarterly report") // More specific
	runCLI(t, file, "toggle", "3")                         // Called dentist
	runCLI(t, file, "delete", "1")                         // Remove completed grocery task

	todos := getTodos(t, file)
	if len(todos) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(todos))
	}

	// Verify state
	if !strings.Contains(todos[0], "quarterly report") {
		t.Error("Edit not preserved")
	}
	if !strings.HasPrefix(todos[1], "- [x] ") {
		t.Error("Toggle not preserved")
	}
	if !strings.Contains(todos[2], "Fix bug") {
		t.Error("New task not found")
	}
}

// TestAST_PreservesTaskOrderAfterOperations tests that task order is stable
func TestAST_PreservesTaskOrderAfterOperations(t *testing.T) {
	file := tempTestFile(t)

	tasks := []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon"}
	for _, task := range tasks {
		runCLI(t, file, "add", task)
	}

	// Toggle some tasks (shouldn't affect order)
	runCLI(t, file, "toggle", "2")
	runCLI(t, file, "toggle", "4")

	todos := getTodos(t, file)

	// Verify order is unchanged
	expectedOrder := []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon"}
	for i, expected := range expectedOrder {
		if !strings.Contains(todos[i], expected) {
			t.Errorf("Task %d: expected %s, got %s", i, expected, todos[i])
		}
	}
}

// TestAST_HandlesEmptyLines tests that empty lines don't break AST
func TestAST_HandlesEmptyLines(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Todos


- [ ] Task with blank lines above




- [ ] Task with many blank lines above

- [ ] Task with single blank line above`

	os.WriteFile(file, []byte(initial), 0644)

	todos := getTodos(t, file)
	if len(todos) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(todos))
	}

	// Edit should work fine
	runCLI(t, file, "edit", "2", "Modified")

	todos = getTodos(t, file)
	if !strings.Contains(todos[1], "Modified") {
		t.Error("Edit failed with blank lines present")
	}
}
