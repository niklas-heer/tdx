package main

import (
	"os"
	"strings"
	"testing"
)

// TestEdgeCase_EmptyFile tests operations on empty file
func TestEdgeCase_EmptyFile(t *testing.T) {
	file := tempTestFile(t)
	os.WriteFile(file, []byte(""), 0644)

	// Add to empty file
	runCLI(t, file, "add", "First task")

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Errorf("Expected 1 todo, got %d", len(todos))
	}
}

// TestEdgeCase_NoTasks tests file with only non-task content
func TestEdgeCase_NoTasks(t *testing.T) {
	file := tempTestFile(t)

	initial := `# My Document

This is just regular text.

No tasks here.`

	os.WriteFile(file, []byte(initial), 0644)

	// Add task to file with no tasks
	runCLI(t, file, "add", "First task")

	content, _ := os.ReadFile(file)
	result := string(content)

	// Original content should be preserved
	if !strings.Contains(result, "This is just regular text.") {
		t.Error("Lost original content")
	}
	if !strings.Contains(result, "No tasks here.") {
		t.Error("Lost original content")
	}
	if !strings.Contains(result, "- [ ] First task") {
		t.Error("Task not added")
	}
}

// TestEdgeCase_TaskWithLinks tests tasks containing markdown links
func TestEdgeCase_TaskWithLinks(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Check [documentation](https://example.com)")

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Fatalf("Expected 1 todo, got %d", len(todos))
	}

	if !strings.Contains(todos[0], "[documentation](https://example.com)") {
		t.Errorf("Link not preserved in task: %s", todos[0])
	}

	// Toggle should preserve link
	runCLI(t, file, "toggle", "1")
	todos = getTodos(t, file)
	if !strings.Contains(todos[0], "[documentation](https://example.com)") {
		t.Errorf("Link lost after toggle: %s", todos[0])
	}
}

// TestEdgeCase_TaskWithCodeSpans tests tasks with inline code
func TestEdgeCase_TaskWithCodeSpans(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Fix `bug` in `main.go`")

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Fatalf("Expected 1 todo, got %d", len(todos))
	}

	if !strings.Contains(todos[0], "`bug`") || !strings.Contains(todos[0], "`main.go`") {
		t.Errorf("Code spans not preserved: %s", todos[0])
	}
}

// TestEdgeCase_TaskWithEmphasis tests tasks with bold and italic
func TestEdgeCase_TaskWithEmphasis(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "This is *important* and **critical**")

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Fatalf("Expected 1 todo, got %d", len(todos))
	}

	if !strings.Contains(todos[0], "*important*") {
		t.Errorf("Italic not preserved: %s", todos[0])
	}
	if !strings.Contains(todos[0], "**critical**") {
		t.Errorf("Bold not preserved: %s", todos[0])
	}
}

// TestEdgeCase_VeryLongTask tests tasks with very long text
func TestEdgeCase_VeryLongTask(t *testing.T) {
	file := tempTestFile(t)

	longText := strings.Repeat("This is a very long task description. ", 20)
	runCLI(t, file, "add", longText)

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Fatalf("Expected 1 todo, got %d", len(todos))
	}

	if !strings.Contains(todos[0], longText) {
		t.Error("Long text was truncated or modified")
	}
}

// TestEdgeCase_SpecialCharacters tests tasks with special characters
func TestEdgeCase_SpecialCharacters(t *testing.T) {
	file := tempTestFile(t)

	special := "Task with special chars: @#$%^&*()_+-=[]{}|;':\",./<>?"
	runCLI(t, file, "add", special)

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Fatalf("Expected 1 todo, got %d", len(todos))
	}

	if !strings.Contains(todos[0], "@#$%^&*()") {
		t.Error("Special characters not preserved")
	}
}

// TestEdgeCase_UnicodeAndEmoji tests tasks with unicode and emoji
func TestEdgeCase_UnicodeAndEmoji(t *testing.T) {
	file := tempTestFile(t)

	unicode := "Task with emoji 🎉 and unicode: 你好 世界 🚀"
	runCLI(t, file, "add", unicode)

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Fatalf("Expected 1 todo, got %d", len(todos))
	}

	if !strings.Contains(todos[0], "🎉") || !strings.Contains(todos[0], "🚀") {
		t.Error("Emoji not preserved")
	}
	if !strings.Contains(todos[0], "你好") {
		t.Error("Unicode not preserved")
	}
}

// TestEdgeCase_EmptyTaskText tests edge case of empty task text
func TestEdgeCase_EmptyTaskText(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "First task")

	// Try to edit to empty (should still work)
	runCLI(t, file, "edit", "1", "")

	todos := getTodos(t, file)
	// Task might be removed or kept as empty - either is acceptable
	if len(todos) > 1 {
		t.Error("Unexpected multiple tasks")
	}
}

// TestEdgeCase_SingleTask tests file with only one task
func TestEdgeCase_SingleTask(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Only task")

	// Toggle single task
	runCLI(t, file, "toggle", "1")
	todos := getTodos(t, file)
	if !strings.HasPrefix(todos[0], "- [x] ") {
		t.Error("Single task toggle failed")
	}

	// Delete single task
	runCLI(t, file, "delete", "1")
	todos = getTodos(t, file)
	if len(todos) != 0 {
		t.Errorf("Task not deleted, still have %d tasks", len(todos))
	}
}

// TestEdgeCase_ManyTasks tests file with many tasks
func TestEdgeCase_ManyTasks(t *testing.T) {
	file := tempTestFile(t)

	// Add 50 tasks
	for i := 1; i <= 50; i++ {
		runCLI(t, file, "add", "Task "+string(rune('A'+i%26)))
	}

	todos := getTodos(t, file)
	if len(todos) != 50 {
		t.Errorf("Expected 50 tasks, got %d", len(todos))
	}

	// Toggle task in middle
	runCLI(t, file, "toggle", "25")
	todos = getTodos(t, file)
	if len(todos) != 50 {
		t.Error("Lost tasks after toggle")
	}

	// Delete task from beginning
	runCLI(t, file, "delete", "1")
	todos = getTodos(t, file)
	if len(todos) != 49 {
		t.Errorf("Expected 49 tasks after delete, got %d", len(todos))
	}
}

// TestEdgeCase_NewlineInContent tests that newlines in surrounding content work
func TestEdgeCase_NewlineInContent(t *testing.T) {
	file := tempTestFile(t)

	initial := "# Todos\n\nParagraph 1\n\nParagraph 2\n\n- [ ] Task\n\nParagraph 3\n\nParagraph 4"
	os.WriteFile(file, []byte(initial), 0644)

	runCLI(t, file, "toggle", "1")

	content, _ := os.ReadFile(file)
	result := string(content)

	if !strings.Contains(result, "Paragraph 1") {
		t.Error("Lost paragraph 1")
	}
	if !strings.Contains(result, "Paragraph 4") {
		t.Error("Lost paragraph 4")
	}
}

// TestEdgeCase_OnlyHeader tests file with only header
func TestEdgeCase_OnlyHeader(t *testing.T) {
	file := tempTestFile(t)

	os.WriteFile(file, []byte("# Todos\n"), 0644)

	runCLI(t, file, "add", "First task")

	content, _ := os.ReadFile(file)
	result := string(content)

	if !strings.Contains(result, "# Todos") {
		t.Error("Lost header")
	}
	if !strings.Contains(result, "- [ ] First task") {
		t.Error("Task not added")
	}
}

// TestEdgeCase_TasksWithDifferentStatuses tests mixing checked and unchecked
func TestEdgeCase_TasksWithDifferentStatuses(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task 1")
	runCLI(t, file, "add", "Task 2")
	runCLI(t, file, "add", "Task 3")
	runCLI(t, file, "add", "Task 4")

	// Toggle 1 and 3
	runCLI(t, file, "toggle", "1")
	runCLI(t, file, "toggle", "3")

	todos := getTodos(t, file)
	if len(todos) != 4 {
		t.Fatalf("Expected 4 tasks, got %d", len(todos))
	}

	// Verify pattern: checked, unchecked, checked, unchecked
	if !strings.HasPrefix(todos[0], "- [x] ") {
		t.Error("Task 1 should be checked")
	}
	if !strings.HasPrefix(todos[1], "- [ ] ") {
		t.Error("Task 2 should be unchecked")
	}
	if !strings.HasPrefix(todos[2], "- [x] ") {
		t.Error("Task 3 should be checked")
	}
	if !strings.HasPrefix(todos[3], "- [ ] ") {
		t.Error("Task 4 should be unchecked")
	}
}
