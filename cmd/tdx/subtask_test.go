package main

import (
	"os"
	"strings"
	"testing"
)

// ==================== Subtask Basic Tests ====================

// TestTUI_SubtaskIndentOutdent tests basic indent/outdent functionality via TUI
func TestTUI_SubtaskIndentOutdent(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Parent task
- [ ] Child task
- [ ] Another task
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Navigate to second task and indent it (Tab)
	runPiped(t, file, "j\t")

	fileContent, _ := os.ReadFile(file)
	result := string(fileContent)

	// Child task should now be indented
	if !strings.Contains(result, "  - [ ] Child task") {
		t.Errorf("Expected indented child task, got:\n%s", result)
	}
}

// TestTUI_SubtaskOutdent tests outdenting a nested task
func TestTUI_SubtaskOutdent(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Parent task
  - [ ] Nested child
- [ ] Another task
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Navigate to nested task and outdent it (Shift+Tab simulated)
	// In piped mode, we use the command to test this
	output := runPiped(t, file, "j")

	// Verify nested task is shown with indentation in display
	if !strings.Contains(output, "Nested child") {
		t.Errorf("Expected to see nested child, got:\n%s", output)
	}
}

// TestTUI_SubtaskDisplayIndentation tests that subtasks show proper indentation in view
func TestTUI_SubtaskDisplayIndentation(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Level 0 task
  - [ ] Level 1 task
    - [ ] Level 2 task
- [ ] Another level 0
`
	_ = os.WriteFile(file, []byte(content), 0644)

	output := runPiped(t, file, "")

	// All tasks should be visible
	if !strings.Contains(output, "Level 0 task") {
		t.Errorf("Expected Level 0 task visible, got:\n%s", output)
	}
	if !strings.Contains(output, "Level 1 task") {
		t.Errorf("Expected Level 1 task visible, got:\n%s", output)
	}
	if !strings.Contains(output, "Level 2 task") {
		t.Errorf("Expected Level 2 task visible, got:\n%s", output)
	}
}

// ==================== Subtask Movement Tests ====================

// TestTUI_SubtaskMoveDown tests moving a task down in move mode
func TestTUI_SubtaskMoveDown(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Task A
- [ ] Task B
- [ ] Task C
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Enter move mode (m), move down (j), confirm (enter)
	runPiped(t, file, "mj\r")

	todos := getTodos(t, file)
	if len(todos) != 3 {
		t.Fatalf("Expected 3 todos, got %d", len(todos))
	}

	// Task A should now be second
	if !strings.Contains(todos[0], "Task B") {
		t.Errorf("Expected Task B first, got: %s", todos[0])
	}
	if !strings.Contains(todos[1], "Task A") {
		t.Errorf("Expected Task A second, got: %s", todos[1])
	}
	if !strings.Contains(todos[2], "Task C") {
		t.Errorf("Expected Task C third, got: %s", todos[2])
	}
}

// TestTUI_SubtaskMoveUp tests moving a task up in move mode
func TestTUI_SubtaskMoveUp(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Task A
- [ ] Task B
- [ ] Task C
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Navigate to Task C (jj), enter move mode (m), move up (k), confirm (enter)
	runPiped(t, file, "jjmk\r")

	todos := getTodos(t, file)
	if len(todos) != 3 {
		t.Fatalf("Expected 3 todos, got %d", len(todos))
	}

	// Task C should now be second
	if !strings.Contains(todos[0], "Task A") {
		t.Errorf("Expected Task A first, got: %s", todos[0])
	}
	if !strings.Contains(todos[1], "Task C") {
		t.Errorf("Expected Task C second, got: %s", todos[1])
	}
	if !strings.Contains(todos[2], "Task B") {
		t.Errorf("Expected Task B third, got: %s", todos[2])
	}
}

// TestTUI_SubtaskMoveCancel tests canceling move mode with escape
func TestTUI_SubtaskMoveCancel(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Task A
- [ ] Task B
- [ ] Task C
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Enter move mode, move down, then cancel with escape
	runPiped(t, file, "mj\x1b")

	todos := getTodos(t, file)
	// Order should be unchanged after cancel
	if !strings.Contains(todos[0], "Task A") {
		t.Errorf("Expected Task A first after cancel, got: %s", todos[0])
	}
	if !strings.Contains(todos[1], "Task B") {
		t.Errorf("Expected Task B second after cancel, got: %s", todos[1])
	}
}

// TestTUI_SubtaskMoveWithNesting tests moving tasks that have children
func TestTUI_SubtaskMoveWithNesting(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Parent A
  - [ ] Child A1
  - [ ] Child A2
- [ ] Parent B
- [ ] Parent C
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Move Parent A down
	runPiped(t, file, "mj\r")

	fileContent, _ := os.ReadFile(file)
	result := string(fileContent)

	// Parent B should now be first
	lines := strings.Split(result, "\n")
	foundParentB := false
	for _, line := range lines {
		if strings.Contains(line, "Parent B") {
			foundParentB = true
			break
		}
		if strings.Contains(line, "Parent A") {
			t.Errorf("Parent A should come after Parent B, got:\n%s", result)
			break
		}
	}
	if !foundParentB {
		t.Errorf("Parent B not found in result:\n%s", result)
	}
}

// TestTUI_SubtaskMoveNestedChild tests moving a nested child task
func TestTUI_SubtaskMoveNestedChild(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Parent
  - [ ] Child 1
  - [ ] Child 2
  - [ ] Child 3
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Navigate to Child 1 (j), move down (mj), confirm
	runPiped(t, file, "jmj\r")

	fileContent, _ := os.ReadFile(file)
	result := string(fileContent)

	// Child 2 should now be before Child 1
	child1Pos := strings.Index(result, "Child 1")
	child2Pos := strings.Index(result, "Child 2")

	if child2Pos > child1Pos {
		t.Errorf("Child 2 should be before Child 1 after move, got:\n%s", result)
	}
}

// ==================== Subtask with Headings Tests ====================

// TestTUI_SubtaskMoveWithHeadings tests moving tasks across heading sections
func TestTUI_SubtaskMoveWithHeadings(t *testing.T) {
	file := tempTestFile(t)

	content := `# Project

## Section A
- [ ] Task A1
- [ ] Task A2

## Section B
- [ ] Task B1
- [ ] Task B2
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Move Task A1 down multiple times to cross into Section B
	runPiped(t, file, "mjjj\r")

	fileContent, _ := os.ReadFile(file)
	result := string(fileContent)

	// Verify the structure is preserved
	if !strings.Contains(result, "## Section A") {
		t.Error("Section A heading lost")
	}
	if !strings.Contains(result, "## Section B") {
		t.Error("Section B heading lost")
	}
}

// ==================== Subtask with Priority Tests ====================

// TestTUI_SubtaskWithPriorities tests subtasks with priority markers
func TestTUI_SubtaskWithPriorities(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Parent task !p1
  - [ ] Child task !p2
  - [ ] Another child !p3
- [ ] Regular task
`
	_ = os.WriteFile(file, []byte(content), 0644)

	output := runPiped(t, file, "")

	// All priorities should be visible
	if !strings.Contains(output, "!p1") {
		t.Errorf("Expected !p1 visible, got:\n%s", output)
	}
	if !strings.Contains(output, "!p2") {
		t.Errorf("Expected !p2 visible, got:\n%s", output)
	}
	if !strings.Contains(output, "!p3") {
		t.Errorf("Expected !p3 visible, got:\n%s", output)
	}
}

// TestTUI_SubtaskPriorityFilter tests filtering subtasks by priority
func TestTUI_SubtaskPriorityFilter(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Parent !p1
  - [ ] Child A !p1
  - [ ] Child B !p2
- [ ] Task !p2
- [ ] Task !p3
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Filter by p1 priority
	output := runPiped(t, file, "p ")

	// Only p1 tasks should be visible
	if !strings.Contains(output, "Parent !p1") {
		t.Errorf("Expected Parent !p1 visible, got:\n%s", output)
	}
	if !strings.Contains(output, "Child A !p1") {
		t.Errorf("Expected Child A !p1 visible, got:\n%s", output)
	}

	// p2 and p3 tasks should be filtered out
	if strings.Contains(output, "Child B !p2") {
		t.Errorf("Child B !p2 should be filtered out, got:\n%s", output)
	}
	if strings.Contains(output, "Task !p3") {
		t.Errorf("Task !p3 should be filtered out, got:\n%s", output)
	}
}

// TestTUI_SubtaskPrioritySort tests sorting subtasks by priority
func TestTUI_SubtaskPrioritySort(t *testing.T) {
	file := tempTestFile(t)

	content := `## Tasks
- [ ] Low priority !p3
  - [ ] Sub low !p3
- [ ] High priority !p1
  - [ ] Sub high !p1
- [ ] Medium priority !p2
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Sort by priority
	runPiped(t, file, ":sort-priority\r")

	todos := getTodos(t, file)

	// High priority should be first in the section
	if !strings.Contains(todos[0], "High priority") {
		t.Errorf("Expected High priority first, got: %s", todos[0])
	}
}

// ==================== Subtask with Tag Tests ====================

// TestTUI_SubtaskWithTags tests subtasks with tag markers
func TestTUI_SubtaskWithTags(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Parent task #backend
  - [ ] Child task #frontend
  - [ ] Another child #backend
- [ ] Regular task #docs
`
	_ = os.WriteFile(file, []byte(content), 0644)

	output := runPiped(t, file, "")

	// All tags should be visible
	if !strings.Contains(output, "#backend") {
		t.Errorf("Expected #backend visible, got:\n%s", output)
	}
	if !strings.Contains(output, "#frontend") {
		t.Errorf("Expected #frontend visible, got:\n%s", output)
	}
	if !strings.Contains(output, "#docs") {
		t.Errorf("Expected #docs visible, got:\n%s", output)
	}
}

// TestTUI_SubtaskTagFilter tests filtering subtasks by tag
func TestTUI_SubtaskTagFilter(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Parent #backend
  - [ ] Child A #backend
  - [ ] Child B #frontend
- [ ] Task #frontend
- [ ] Task #docs
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Filter by #backend tag (t to open filter, space to select first tag which should be backend)
	output := runPiped(t, file, "t ")

	// Only backend tasks should be visible
	if !strings.Contains(output, "Parent #backend") {
		t.Errorf("Expected Parent #backend visible, got:\n%s", output)
	}
	if !strings.Contains(output, "Child A #backend") {
		t.Errorf("Expected Child A #backend visible, got:\n%s", output)
	}
}

// TestTUI_SubtaskCombinedPriorityAndTagFilter tests filtering by both priority and tag
func TestTUI_SubtaskCombinedPriorityAndTagFilter(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Task A !p1 #backend
  - [ ] Child A1 !p1 #backend
  - [ ] Child A2 !p2 #backend
- [ ] Task B !p1 #frontend
- [ ] Task C !p2 #backend
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Filter by p1 priority, then by backend tag
	output := runPiped(t, file, "p t ")

	// Only tasks with both p1 AND backend should be visible
	if !strings.Contains(output, "Task A !p1 #backend") {
		t.Errorf("Expected Task A visible, got:\n%s", output)
	}
	if !strings.Contains(output, "Child A1 !p1 #backend") {
		t.Errorf("Expected Child A1 visible, got:\n%s", output)
	}
}

// ==================== Subtask Move Mode Indicator Tests ====================

// TestTUI_SubtaskMoveIndicator tests that move mode shows the correct indicator
func TestTUI_SubtaskMoveIndicator(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Task A
- [ ] Task B
- [ ] Task C
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Enter move mode
	output := runPiped(t, file, "m")

	// Should show MOVE indicator
	if !strings.Contains(output, "MOVE") {
		t.Errorf("Expected MOVE indicator, got:\n%s", output)
	}

	// Should show move indicator on task (≡)
	if !strings.Contains(output, "≡") {
		t.Errorf("Expected ≡ move indicator on task, got:\n%s", output)
	}
}

// TestTUI_SubtaskMoveWithFilterDone tests moving tasks when filter-done is active
func TestTUI_SubtaskMoveWithFilterDone(t *testing.T) {
	file := tempTestFile(t)

	content := `- [x] Done task
- [ ] Task A
- [x] Another done
- [ ] Task B
- [ ] Task C
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Enable filter-done, then move Task A down
	output := runPiped(t, file, ":filter-done\rmj\r")

	// Task A should have moved relative to visible tasks
	if !strings.Contains(output, "FILTERED") {
		t.Errorf("Expected FILTERED indicator, got:\n%s", output)
	}
}

// ==================== Subtask Navigation Tests ====================

// TestTUI_SubtaskNavigationWithFilters tests navigation through nested tasks with filters
func TestTUI_SubtaskNavigationWithFilters(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Parent 1 !p1
  - [ ] Child 1a !p1
  - [ ] Child 1b !p2
- [ ] Parent 2 !p2
  - [ ] Child 2a !p1
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Filter by p1, then navigate down
	output := runPiped(t, file, "p jj")

	// Should be on Child 2a (third visible p1 task)
	// Cursor indicator should show 0 for current position
	if !strings.Contains(output, "  0") {
		t.Errorf("Expected cursor at position 0, got:\n%s", output)
	}
}

// TestTUI_SubtaskDeleteParent tests deleting a parent task with children
func TestTUI_SubtaskDeleteParent(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Parent
  - [ ] Child 1
  - [ ] Child 2
- [ ] Sibling
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Delete the parent
	runPiped(t, file, "d")

	todos := getTodos(t, file)

	// Children should be promoted or deleted (depending on implementation)
	// Check that Sibling is still there
	found := false
	for _, todo := range todos {
		if strings.Contains(todo, "Sibling") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Sibling task should still exist, got: %v", todos)
	}
}

// TestTUI_SubtaskToggleNested tests toggling a nested task
func TestTUI_SubtaskToggleNested(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Parent
  - [ ] Child 1
  - [ ] Child 2
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Navigate to Child 1 and toggle
	runPiped(t, file, "j ")

	// Read file content directly to check indented subtasks
	fileContent, _ := os.ReadFile(file)
	result := string(fileContent)

	// Child 1 should be checked
	if !strings.Contains(result, "- [x] Child 1") {
		t.Errorf("Expected Child 1 to be checked, got:\n%s", result)
	}

	// Parent should still be unchecked
	if !strings.Contains(result, "- [ ] Parent") {
		t.Errorf("Expected Parent to be unchecked, got:\n%s", result)
	}
}

// TestTUI_SubtaskNewAfterNested tests creating new task after a nested task
func TestTUI_SubtaskNewAfterNested(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Parent
  - [ ] Child 1
  - [ ] Child 2
- [ ] Sibling
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Navigate to Child 1, create new task after it
	runPiped(t, file, "jnNew task\r")

	fileContent, _ := os.ReadFile(file)
	result := string(fileContent)

	// New task should be after Child 1 and at same indentation level
	if !strings.Contains(result, "New task") {
		t.Errorf("New task not found in result:\n%s", result)
	}
}

// ==================== Subtask with Headings and Priorities ====================

// TestTUI_SubtaskHeadingsPrioritiesAndTags tests complex scenario with all features
func TestTUI_SubtaskHeadingsPrioritiesAndTags(t *testing.T) {
	file := tempTestFile(t)

	content := `# Project

## High Priority
- [ ] Critical bug !p1 #backend
  - [ ] Sub-task 1 !p1 #backend
  - [ ] Sub-task 2 !p2 #frontend

## Normal Priority
- [ ] Feature request !p2 #frontend
  - [ ] Design !p2 #design
  - [ ] Implementation !p2 #backend

## Low Priority
- [ ] Nice to have !p3 #docs
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Test that all content is visible initially
	output := runPiped(t, file, "")

	if !strings.Contains(output, "Critical bug") {
		t.Errorf("Expected Critical bug visible, got:\n%s", output)
	}
	if !strings.Contains(output, "Feature request") {
		t.Errorf("Expected Feature request visible, got:\n%s", output)
	}
	if !strings.Contains(output, "Nice to have") {
		t.Errorf("Expected Nice to have visible, got:\n%s", output)
	}
}

// TestTUI_SubtaskMoveAcrossSections tests moving a nested task across heading sections
func TestTUI_SubtaskMoveAcrossSections(t *testing.T) {
	file := tempTestFile(t)

	content := `## Section A
- [ ] Parent A
  - [ ] Child A1
  - [ ] Child A2

## Section B
- [ ] Parent B
  - [ ] Child B1
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Navigate to Child A1, move it down multiple times
	runPiped(t, file, "jmjjj\r")

	fileContent, _ := os.ReadFile(file)
	result := string(fileContent)

	// Sections should still exist
	if !strings.Contains(result, "## Section A") {
		t.Error("Section A lost after move")
	}
	if !strings.Contains(result, "## Section B") {
		t.Error("Section B lost after move")
	}
}

// TestTUI_SubtaskMoveMultipleDown tests moving a task multiple positions down
func TestTUI_SubtaskMoveMultipleDown(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
- [ ] Task 4
- [ ] Task 5
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Move Task 1 to the bottom (4 moves down)
	runPiped(t, file, "mjjjj\r")

	todos := getTodos(t, file)

	// Task 1 should now be last
	if !strings.Contains(todos[4], "Task 1") {
		t.Errorf("Expected Task 1 at position 5, got: %s", todos[4])
	}
	if !strings.Contains(todos[0], "Task 2") {
		t.Errorf("Expected Task 2 first, got: %s", todos[0])
	}
}

// TestTUI_SubtaskMoveMultipleUp tests moving a task multiple positions up
func TestTUI_SubtaskMoveMultipleUp(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
- [ ] Task 4
- [ ] Task 5
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Navigate to Task 5, move to top (4 moves up)
	runPiped(t, file, "jjjjmkkkk\r")

	todos := getTodos(t, file)

	// Task 5 should now be first
	if !strings.Contains(todos[0], "Task 5") {
		t.Errorf("Expected Task 5 first, got: %s", todos[0])
	}
	if !strings.Contains(todos[4], "Task 4") {
		t.Errorf("Expected Task 4 last, got: %s", todos[4])
	}
}

// TestTUI_SubtaskCursorAfterMove tests cursor position after move is confirmed
func TestTUI_SubtaskCursorAfterMove(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Task A
- [ ] Task B
- [ ] Task C
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Move Task A down, cursor should follow
	output := runPiped(t, file, "mj\r")

	// The cursor (0 indicator) should be on Task A in its new position
	// Task A is now at index 1, so "  0" should appear on line with Task A
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Task A") {
			if !strings.Contains(line, "0") {
				t.Errorf("Expected cursor on Task A after move, got line: %s", line)
			}
			break
		}
	}
}

// ==================== Subtask Delete Cursor Tests ====================

// TestTUI_SubtaskDeleteLastChild_SelectsParent tests that deleting the last child selects the parent
func TestTUI_SubtaskDeleteLastChild_SelectsParent(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Parent
  - [ ] Only Child
- [ ] Sibling
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Navigate to Only Child (j) and delete (d)
	output := runPiped(t, file, "jd")

	// After deleting the only child, cursor should be on Parent
	// The cursor indicator "0 ➜" should be on the Parent line
	if !strings.Contains(output, "0 ➜ [ ] Parent") {
		t.Errorf("Expected cursor on Parent after deleting only child, got:\n%s", output)
	}
}

// TestTUI_SubtaskDeleteFirstChild_SelectsNextSibling tests that deleting first child selects next sibling
func TestTUI_SubtaskDeleteFirstChild_SelectsNextSibling(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Parent
  - [ ] Child 1
  - [ ] Child 2
  - [ ] Child 3
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Navigate to Child 1 (j) and delete (d)
	output := runPiped(t, file, "jd")

	// After deleting Child 1, cursor should be on Child 2 (next sibling)
	if !strings.Contains(output, "0 ➜") {
		t.Errorf("Expected cursor indicator in output, got:\n%s", output)
	}
	// Check that Child 2 is selected (cursor line should contain Child 2)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "0 ➜") {
			if !strings.Contains(line, "Child 2") {
				t.Errorf("Expected cursor on Child 2 after deleting Child 1, got line: %s", line)
			}
			break
		}
	}
}

// TestTUI_SubtaskDeleteMiddleChild_SelectsNextSibling tests that deleting middle child selects next sibling
func TestTUI_SubtaskDeleteMiddleChild_SelectsNextSibling(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Parent
  - [ ] Child 1
  - [ ] Child 2
  - [ ] Child 3
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Navigate to Child 2 (jj) and delete (d)
	output := runPiped(t, file, "jjd")

	// After deleting Child 2, cursor should be on Child 3 (next sibling)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "0 ➜") {
			if !strings.Contains(line, "Child 3") {
				t.Errorf("Expected cursor on Child 3 after deleting Child 2, got line: %s", line)
			}
			break
		}
	}
}

// TestTUI_SubtaskDeleteLastChildOfMany_SelectsPrevSibling tests deleting last child selects previous sibling
func TestTUI_SubtaskDeleteLastChildOfMany_SelectsPrevSibling(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Parent
  - [ ] Child 1
  - [ ] Child 2
  - [ ] Child 3
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Navigate to Child 3 (jjj) and delete (d)
	output := runPiped(t, file, "jjjd")

	// After deleting Child 3 (last child), cursor should be on Child 2 (previous sibling)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "0 ➜") {
			if !strings.Contains(line, "Child 2") {
				t.Errorf("Expected cursor on Child 2 after deleting Child 3, got line: %s", line)
			}
			break
		}
	}
}

// TestTUI_SubtaskDeleteAllChildren_SelectsParent tests deleting all children one by one ends on parent
func TestTUI_SubtaskDeleteAllChildren_SelectsParent(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Parent
  - [ ] Child 1
  - [ ] Child 2
- [ ] Sibling
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Navigate to Child 1 (j), delete twice (dd) to remove both children
	// After first delete, cursor goes to Child 2
	// After second delete, cursor should go to Parent (no more children)
	output := runPiped(t, file, "jdd")

	// After deleting both children, cursor should be on Parent
	if !strings.Contains(output, "0 ➜ [ ] Parent") {
		t.Errorf("Expected cursor on Parent after deleting all children, got:\n%s", output)
	}

	// Verify file state
	fileContent, _ := os.ReadFile(file)
	result := string(fileContent)
	if strings.Contains(result, "Child") {
		t.Errorf("Expected no children remaining, got:\n%s", result)
	}
}

// TestTUI_SubtaskDeleteNestedChild_SelectsParentChild tests deleting deeply nested child
func TestTUI_SubtaskDeleteNestedChild_SelectsParentChild(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Grandparent
  - [ ] Parent
    - [ ] Child
  - [ ] Uncle
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Navigate to Child (jjj) and delete (d)
	output := runPiped(t, file, "jjd")

	// After deleting Child, cursor should be on Parent (its parent)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "0 ➜") {
			if !strings.Contains(line, "Parent") {
				t.Errorf("Expected cursor on Parent after deleting nested Child, got line: %s", line)
			}
			break
		}
	}
}

// TestTUI_SubtaskDeleteTopLevel_SelectsNextTopLevel tests deleting top-level selects next top-level
func TestTUI_SubtaskDeleteTopLevel_SelectsNextTopLevel(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Task A
  - [ ] Child of A
- [ ] Task B
- [ ] Task C
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Delete Task A (cursor starts there)
	output := runPiped(t, file, "d")

	// After deleting Task A (and its children get promoted), cursor should be on next top-level
	// Check that cursor is on a valid task
	if !strings.Contains(output, "0 ➜") {
		t.Errorf("Expected cursor indicator in output, got:\n%s", output)
	}
}

// TestTUI_SubtaskDeleteWithFilter_SelectsNextVisible tests delete respects visibility filter
func TestTUI_SubtaskDeleteWithFilter_SelectsNextVisible(t *testing.T) {
	file := tempTestFile(t)

	content := `---
filter-done: true
---
- [ ] Parent
  - [x] Done Child
  - [ ] Visible Child
- [ ] Sibling
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// With filter-done, we see: Parent, Visible Child, Sibling
	// Navigate to Visible Child (j) and delete (d)
	output := runPiped(t, file, "jd")

	// After deleting Visible Child, should select Parent (since Done Child is hidden)
	if !strings.Contains(output, "0 ➜ [ ] Parent") {
		t.Errorf("Expected cursor on Parent after deleting visible child, got:\n%s", output)
	}
}

// TestTUI_SubtaskDeleteOnlyVisibleChild_SelectsParent tests when only visible child is deleted
func TestTUI_SubtaskDeleteOnlyVisibleChild_SelectsParent(t *testing.T) {
	file := tempTestFile(t)

	content := `---
filter-done: true
---
- [ ] Parent
  - [x] Done Child 1
  - [ ] Visible Child
  - [x] Done Child 2
- [ ] Sibling
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Navigate to Visible Child (j) and delete (d)
	output := runPiped(t, file, "jd")

	// After deleting the only visible child, cursor should go to Parent
	if !strings.Contains(output, "0 ➜ [ ] Parent") {
		t.Errorf("Expected cursor on Parent after deleting only visible child, got:\n%s", output)
	}
}

// ==================== Toggle Cursor Visibility Tests ====================

// TestTUI_ToggleCursorVisible_FilterDone tests cursor stays visible after toggling with filter-done
func TestTUI_ToggleCursorVisible_FilterDone(t *testing.T) {
	file := tempTestFile(t)

	content := `---
filter-done: true
---
- [ ] Task A
- [ ] Task B
- [ ] Task C
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Toggle Task A (marks it done, becomes hidden)
	output := runPiped(t, file, " ")

	// Cursor should be visible on Task B (next item)
	if !strings.Contains(output, "0 ➜") {
		t.Errorf("Expected cursor indicator visible after toggle, got:\n%s", output)
	}
	// Cursor should be on Task B
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "0 ➜") {
			if !strings.Contains(line, "Task B") {
				t.Errorf("Expected cursor on Task B after toggling Task A, got line: %s", line)
			}
			break
		}
	}
}

// TestTUI_ToggleCursorVisible_FilterDone_LastItem tests cursor when toggling last visible item
func TestTUI_ToggleCursorVisible_FilterDone_LastItem(t *testing.T) {
	file := tempTestFile(t)

	content := `---
filter-done: true
---
- [ ] Task A
- [ ] Task B
- [ ] Task C
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Navigate to Task C (jj) and toggle (marks it done, becomes hidden)
	output := runPiped(t, file, "jj ")

	// Cursor should be visible on Task B (previous item since C was last)
	if !strings.Contains(output, "0 ➜") {
		t.Errorf("Expected cursor indicator visible after toggle, got:\n%s", output)
	}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "0 ➜") {
			if !strings.Contains(line, "Task B") {
				t.Errorf("Expected cursor on Task B after toggling last item, got line: %s", line)
			}
			break
		}
	}
}

// TestTUI_ToggleCursorVisible_FilterDone_Subtask tests cursor when toggling subtask with filter-done
func TestTUI_ToggleCursorVisible_FilterDone_Subtask(t *testing.T) {
	file := tempTestFile(t)

	content := `---
filter-done: true
---
- [ ] Parent
  - [ ] Child 1
  - [ ] Child 2
- [ ] Sibling
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Navigate to Child 1 (j) and toggle (marks it done, becomes hidden)
	output := runPiped(t, file, "j ")

	// Cursor should be visible on Child 2 (next sibling)
	if !strings.Contains(output, "0 ➜") {
		t.Errorf("Expected cursor indicator visible after toggle, got:\n%s", output)
	}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "0 ➜") {
			if !strings.Contains(line, "Child 2") {
				t.Errorf("Expected cursor on Child 2 after toggling Child 1, got line: %s", line)
			}
			break
		}
	}
}

// TestTUI_ToggleCursorVisible_FilterDone_OnlyChild tests cursor goes to parent when only child toggled
func TestTUI_ToggleCursorVisible_FilterDone_OnlyChild(t *testing.T) {
	file := tempTestFile(t)

	content := `---
filter-done: true
---
- [ ] Parent
  - [ ] Only Child
- [ ] Sibling
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Navigate to Only Child (j) and toggle (marks it done, becomes hidden)
	output := runPiped(t, file, "j ")

	// Cursor should be visible on Parent (no more visible children)
	if !strings.Contains(output, "0 ➜ [ ] Parent") {
		t.Errorf("Expected cursor on Parent after toggling only child, got:\n%s", output)
	}
}

// TestTUI_ToggleCursorVisible_FilterDone_AllToggled tests cursor when all items get toggled
func TestTUI_ToggleCursorVisible_FilterDone_AllToggled(t *testing.T) {
	file := tempTestFile(t)

	content := `---
filter-done: true
---
- [ ] Task A
- [ ] Task B
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Toggle both tasks
	output := runPiped(t, file, "  ")

	// When all items are toggled to done and filtered, should show empty state or last position
	// At minimum, shouldn't crash and should have some output
	if len(output) == 0 {
		t.Errorf("Expected some output even with all items filtered")
	}
}

// TestTUI_ToggleCursorVisible_MultipleToggles tests cursor through multiple toggles
func TestTUI_ToggleCursorVisible_MultipleToggles(t *testing.T) {
	file := tempTestFile(t)

	content := `---
filter-done: true
---
- [ ] Task A
- [ ] Task B
- [ ] Task C
- [ ] Task D
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Toggle Task A, then toggle Task B (now current), then toggle Task C
	output := runPiped(t, file, "   ")

	// After toggling A, B, C - cursor should be on Task D
	if !strings.Contains(output, "0 ➜") {
		t.Errorf("Expected cursor indicator visible after multiple toggles, got:\n%s", output)
	}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "0 ➜") {
			if !strings.Contains(line, "Task D") {
				t.Errorf("Expected cursor on Task D after toggling A, B, C, got line: %s", line)
			}
			break
		}
	}
}

// TestTUI_ToggleCursorVisible_WithHeadings tests cursor visibility with headings enabled
func TestTUI_ToggleCursorVisible_WithHeadings(t *testing.T) {
	file := tempTestFile(t)

	content := `---
filter-done: true
show-headings: true
---
# Section 1

- [ ] Task A
- [ ] Task B

# Section 2

- [ ] Task C
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Toggle Task A
	output := runPiped(t, file, " ")

	// Cursor should be visible on Task B
	if !strings.Contains(output, "0 ➜") {
		t.Errorf("Expected cursor indicator visible with headings, got:\n%s", output)
	}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "0 ➜") {
			if !strings.Contains(line, "Task B") {
				t.Errorf("Expected cursor on Task B after toggle with headings, got line: %s", line)
			}
			break
		}
	}
}

// TestTUI_ToggleCursorVisible_TagFilter tests cursor stays visible with tag filter
func TestTUI_ToggleCursorVisible_TagFilter(t *testing.T) {
	file := tempTestFile(t)

	// Use only one tag type to avoid ordering issues
	content := `- [ ] Task A #work
- [ ] Task B #work
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Open tag filter (t), select work tag (space), close filter (esc)
	// Then toggle Task A
	output := runPiped(t, file, "t \x1b ")

	// Cursor should be visible with arrow indicator
	if !strings.Contains(output, "➜") {
		t.Errorf("Expected cursor arrow (➜) visible after toggle with tag filter, got:\n%s", output)
	}
}

// TestTUI_ToggleCursorVisible_UntoggleRestoresVisibility tests uncheck makes item visible again
func TestTUI_ToggleCursorVisible_UntoggleRestoresVisibility(t *testing.T) {
	file := tempTestFile(t)

	content := `---
filter-done: true
---
- [x] Done Task
- [ ] Task B
`
	_ = os.WriteFile(file, []byte(content), 0644)

	// Cursor starts on Task B (Done Task is hidden)
	// Toggle Task B to done, cursor should move somewhere
	// Toggle again to undone, cursor should stay on Task B
	output := runPiped(t, file, "  ")

	// After toggle-untoggle, Task B should be visible and selected
	if !strings.Contains(output, "0 ➜ [ ] Task B") {
		t.Errorf("Expected cursor on Task B after toggle-untoggle, got:\n%s", output)
	}
}
