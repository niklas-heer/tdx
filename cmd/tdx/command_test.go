package main

import (
	"os"
	"strings"
	"testing"
)

// TestCommand_CheckAll tests the check-all command
func TestCommand_CheckAll(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task 1")
	runCLI(t, file, "add", "Task 2")
	runCLI(t, file, "add", "Task 3")

	// Use command palette to check all
	runPiped(t, file, ":check-all\r")

	todos := getTodos(t, file)
	for i, todo := range todos {
		if !strings.HasPrefix(todo, "- [x] ") {
			t.Errorf("Task %d not checked: %s", i+1, todo)
		}
	}
}

// TestCommand_UncheckAll tests the uncheck-all command
func TestCommand_UncheckAll(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task 1")
	runCLI(t, file, "add", "Task 2")
	runCLI(t, file, "add", "Task 3")

	// Check all tasks first
	runPiped(t, file, ":check-all\r")

	// Then uncheck all
	runPiped(t, file, ":uncheck-all\r")

	todos := getTodos(t, file)
	for i, todo := range todos {
		if !strings.HasPrefix(todo, "- [ ] ") {
			t.Errorf("Task %d not unchecked: %s", i+1, todo)
		}
	}
}

// TestCommand_ClearDone tests the clear-done command
func TestCommand_ClearDone(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task 1")
	runCLI(t, file, "add", "Task 2")
	runCLI(t, file, "add", "Task 3")
	runCLI(t, file, "add", "Task 4")

	// Check tasks 1 and 3
	runCLI(t, file, "toggle", "1")
	runCLI(t, file, "toggle", "3")

	// Clear done tasks
	runPiped(t, file, ":clear-done\r")

	todos := getTodos(t, file)
	if len(todos) != 2 {
		t.Errorf("Expected 2 tasks after clear-done, got %d", len(todos))
	}

	// Should only have Task 2 and Task 4
	if !strings.Contains(todos[0], "Task 2") {
		t.Errorf("Expected Task 2, got: %s", todos[0])
	}
	if !strings.Contains(todos[1], "Task 4") {
		t.Errorf("Expected Task 4, got: %s", todos[1])
	}
}

// TestCommand_ClearDoneWithContent tests that clear-done preserves non-task content
func TestCommand_ClearDoneWithContent(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Todos

Important note.

- [ ] Keep this
- [x] Remove this

Another note.

- [ ] Keep this too
- [x] Remove this too

Final note.`

	_ = os.WriteFile(file, []byte(initial), 0644)

	runPiped(t, file, ":clear-done\r")

	content, _ := os.ReadFile(file)
	result := string(content)

	// Notes should be preserved
	if !strings.Contains(result, "Important note.") {
		t.Error("Lost important note")
	}
	if !strings.Contains(result, "Another note.") {
		t.Error("Lost another note")
	}
	if !strings.Contains(result, "Final note.") {
		t.Error("Lost final note")
	}

	// Checked tasks should be gone
	if strings.Contains(result, "Remove this") {
		t.Error("Checked tasks not removed")
	}

	// Unchecked tasks should remain
	todos := getTodos(t, file)
	if len(todos) != 2 {
		t.Errorf("Expected 2 unchecked tasks, got %d", len(todos))
	}
}

// TestCommand_SortTasks tests the sort command
func TestCommand_SortTasks(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task A")
	runCLI(t, file, "add", "Task B")
	runCLI(t, file, "add", "Task C")
	runCLI(t, file, "add", "Task D")

	// Check B and D
	runCLI(t, file, "toggle", "2")
	runCLI(t, file, "toggle", "4")

	// Sort (incomplete first)
	runPiped(t, file, ":sort\r")

	todos := getTodos(t, file)
	if len(todos) != 4 {
		t.Errorf("Expected 4 tasks after sort, got %d", len(todos))
	}

	// First two should be unchecked (A and C)
	if !strings.HasPrefix(todos[0], "- [ ] ") {
		t.Error("First task should be unchecked")
	}
	if !strings.HasPrefix(todos[1], "- [ ] ") {
		t.Error("Second task should be unchecked")
	}

	// Last two should be checked (B and D)
	if !strings.HasPrefix(todos[2], "- [x] ") {
		t.Error("Third task should be checked")
	}
	if !strings.HasPrefix(todos[3], "- [x] ") {
		t.Error("Fourth task should be checked")
	}
}

// TestCommand_CheckAllThenClearDone tests combining commands
func TestCommand_CheckAllThenClearDone(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task 1")
	runCLI(t, file, "add", "Task 2")
	runCLI(t, file, "add", "Task 3")

	// Check all
	runPiped(t, file, ":check-all\r")

	todos := getTodos(t, file)
	if len(todos) != 3 {
		t.Fatalf("Expected 3 tasks after check-all, got %d", len(todos))
	}

	// Clear done
	runPiped(t, file, ":clear-done\r")

	todos = getTodos(t, file)
	if len(todos) != 0 {
		t.Errorf("Expected 0 tasks after clear-done, got %d", len(todos))
	}
}

// TestCommand_MixedStatus tests commands with mixed task statuses
func TestCommand_MixedStatus(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task 1")
	runCLI(t, file, "add", "Task 2")
	runCLI(t, file, "add", "Task 3")
	runCLI(t, file, "add", "Task 4")
	runCLI(t, file, "add", "Task 5")

	// Check 1, 3, 5
	runCLI(t, file, "toggle", "1")
	runCLI(t, file, "toggle", "3")
	runCLI(t, file, "toggle", "5")

	todos := getTodos(t, file)

	// Count checked and unchecked
	checkedCount := 0
	uncheckedCount := 0
	for _, todo := range todos {
		if strings.HasPrefix(todo, "- [x] ") {
			checkedCount++
		} else if strings.HasPrefix(todo, "- [ ] ") {
			uncheckedCount++
		}
	}

	if checkedCount != 3 {
		t.Errorf("Expected 3 checked tasks, got %d", checkedCount)
	}
	if uncheckedCount != 2 {
		t.Errorf("Expected 2 unchecked tasks, got %d", uncheckedCount)
	}

	// Uncheck all
	runPiped(t, file, ":uncheck-all\r")

	todos = getTodos(t, file)
	for _, todo := range todos {
		if !strings.HasPrefix(todo, "- [ ] ") {
			t.Error("Not all tasks unchecked after uncheck-all")
			break
		}
	}
}

// TestCommand_EmptyFile tests commands on empty file
func TestCommand_EmptyFile(t *testing.T) {
	file := tempTestFile(t)

	_ = os.WriteFile(file, []byte(""), 0644)

	// These should not crash on empty file
	runPiped(t, file, ":check-all\r")
	runPiped(t, file, ":uncheck-all\r")
	runPiped(t, file, ":clear-done\r")
	runPiped(t, file, ":sort\r")

	// File should still be valid (just empty or with header)
	content, _ := os.ReadFile(file)
	if content == nil {
		t.Error("File content is nil")
	}
}

// TestCommand_SingleTask tests commands with only one task
func TestCommand_SingleTask(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Only task")

	// Check all
	runPiped(t, file, ":check-all\r")
	todos := getTodos(t, file)
	if len(todos) != 1 || !strings.HasPrefix(todos[0], "- [x] ") {
		t.Error("check-all failed on single task")
	}

	// Uncheck all
	runPiped(t, file, ":uncheck-all\r")
	todos = getTodos(t, file)
	if len(todos) != 1 || !strings.HasPrefix(todos[0], "- [ ] ") {
		t.Error("uncheck-all failed on single task")
	}

	// Check it again
	runCLI(t, file, "toggle", "1")

	// Clear done
	runPiped(t, file, ":clear-done\r")
	todos = getTodos(t, file)
	if len(todos) != 0 {
		t.Error("clear-done failed on single task")
	}
}

// TestCommand_AllChecked tests commands when all tasks are checked
func TestCommand_AllChecked(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task 1")
	runCLI(t, file, "add", "Task 2")
	runCLI(t, file, "add", "Task 3")

	runPiped(t, file, ":check-all\r")

	// Check-all again (should be idempotent)
	runPiped(t, file, ":check-all\r")

	todos := getTodos(t, file)
	if len(todos) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(todos))
	}
	for _, todo := range todos {
		if !strings.HasPrefix(todo, "- [x] ") {
			t.Error("Not all tasks checked")
			break
		}
	}
}

// TestCommand_AllUnchecked tests commands when all tasks are unchecked
func TestCommand_AllUnchecked(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task 1")
	runCLI(t, file, "add", "Task 2")
	runCLI(t, file, "add", "Task 3")

	// Uncheck-all on already unchecked (should be idempotent)
	runPiped(t, file, ":uncheck-all\r")

	todos := getTodos(t, file)
	if len(todos) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(todos))
	}
	for _, todo := range todos {
		if !strings.HasPrefix(todo, "- [ ] ") {
			t.Error("Not all tasks unchecked")
			break
		}
	}

	// Clear-done on no completed tasks (should do nothing)
	runPiped(t, file, ":clear-done\r")
	todos = getTodos(t, file)
	if len(todos) != 3 {
		t.Error("clear-done removed unchecked tasks")
	}
}
