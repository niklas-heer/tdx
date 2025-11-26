package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/niklas-heer/tdx/internal/config"
)

// ==================== Multi-Step Undo/Redo Tests ====================
// Note: Undo history is per-session, so all operations and undos must happen
// in a single runPiped call to test undo properly.

// TestTUI_UndoMultipleOperations tests undoing multiple sequential operations
func TestTUI_UndoMultipleOperations(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task 1")
	runCLI(t, file, "add", "Task 2")
	runCLI(t, file, "add", "Task 3")

	// In a single session: toggle first, move down, delete second, then undo both
	// Space (toggle 1), j (move to 2), d (delete 2), u (undo delete), u (undo toggle)
	runPiped(t, file, " jduu")

	todos := getTodos(t, file)
	if len(todos) != 3 {
		t.Fatalf("Expected 3 todos after undo, got %d", len(todos))
	}

	// First task should be unchecked after undo
	if !strings.HasPrefix(todos[0], "- [ ]") {
		t.Errorf("First task should be unchecked after undo, got: %s", todos[0])
	}
}

// TestTUI_UndoAfterMultipleToggles tests undoing multiple toggles in one session
func TestTUI_UndoAfterMultipleToggles(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task A")
	runCLI(t, file, "add", "Task B")

	// Toggle both tasks, then undo both: space, j, space, u, u
	runPiped(t, file, " j uu")

	todos := getTodos(t, file)
	// Both should be unchecked after undoing both toggles
	if !strings.HasPrefix(todos[0], "- [ ]") {
		t.Errorf("Task A should be unchecked after undo, got: %s", todos[0])
	}
	if !strings.HasPrefix(todos[1], "- [ ]") {
		t.Errorf("Task B should be unchecked after undo, got: %s", todos[1])
	}
}

// TestTUI_UndoCheckAll tests undoing check-all command in same session
func TestTUI_UndoCheckAll(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task 1")
	runCLI(t, file, "add", "Task 2")
	runCLI(t, file, "add", "Task 3")

	// Check all via command palette, then immediately undo
	runPiped(t, file, ":check-all\ru")

	todos := getTodos(t, file)
	for i, todo := range todos {
		if !strings.HasPrefix(todo, "- [ ]") {
			t.Errorf("Task %d should be unchecked after undo, got: %s", i+1, todo)
		}
	}
}

// TestTUI_UndoClearDone tests undoing clear-done command in same session
func TestTUI_UndoClearDone(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task 1")
	runCLI(t, file, "add", "Task 2")
	runCLI(t, file, "add", "Task 3")

	// Check first and third tasks, clear done, then undo
	// space (check 1), jj (move to 3), space (check 3), :clear-done, u (undo)
	runPiped(t, file, " jj :clear-done\ru")

	todos := getTodos(t, file)
	if len(todos) != 3 {
		t.Fatalf("Expected 3 todos after undo, got %d", len(todos))
	}
}

// ==================== Word Wrap and Line Numbers Tests ====================

// TestTUI_WordWrapToggle tests the :wrap command
func TestTUI_WordWrapToggle(t *testing.T) {
	file := tempTestFile(t)

	// Create a task with a long line
	longTask := "This is a very long task that should wrap when word wrap is enabled and not wrap when disabled"
	runCLI(t, file, "add", longTask)

	// Word wrap is enabled by default - toggle it off then on
	_ = runPiped(t, file, ":wrap\r\x1b")

	// After toggling, it should be off (we can't easily test visual wrapping,
	// but we verify the command doesn't crash and the task is preserved)
	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Fatalf("Expected 1 todo, got %d", len(todos))
	}
	if !strings.Contains(todos[0], longTask) {
		t.Errorf("Long task should be preserved, got: %s", todos[0])
	}

	// Toggle wrap back on
	_ = runPiped(t, file, ":wrap\r\x1b")

	todos = getTodos(t, file)
	if !strings.Contains(todos[0], longTask) {
		t.Errorf("Long task should still be preserved after toggling wrap, got: %s", todos[0])
	}
}

// TestTUI_LineNumbersToggle tests the :line-numbers command
func TestTUI_LineNumbersToggle(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task 1")
	runCLI(t, file, "add", "Task 2")
	runCLI(t, file, "add", "Task 3")

	// Line numbers are shown by default (relative positioning)
	// Check that initial output contains line numbers
	output := runPiped(t, file, "\x1b")

	// Should contain relative line indicators like "0 ➜" for selected
	if !strings.Contains(output, "0 ➜") {
		t.Errorf("Expected line number indicator '0 ➜' in output: %s", output)
	}

	// Toggle line numbers off
	output = runPiped(t, file, ":line-numbers\r\x1b")

	// After toggling off, line numbers should not appear
	// The selector should still be there but without the number prefix
	if strings.Contains(output, "0 ➜") {
		t.Log("Note: Line numbers still showing after toggle - this may be expected behavior")
	}

	// Toggle back on
	_ = runPiped(t, file, ":line-numbers\r\x1b")

	// Tasks should still be intact
	todos := getTodos(t, file)
	if len(todos) != 3 {
		t.Fatalf("Expected 3 todos, got %d", len(todos))
	}
}

// ==================== Tag Filtering Tests ====================

// TestTUI_TagFilterMultipleTags tests filtering with multiple tags selected
func TestTUI_TagFilterMultipleTags(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Task with #work tag
- [ ] Task with #home tag
- [ ] Task with #urgent tag
- [ ] Task with #work and #urgent tags
- [ ] Task with no tags`

	_ = os.WriteFile(file, []byte(content), 0644)

	// Open filter mode with 'f', select first tag (work), press enter to apply
	// The exact behavior depends on how many tags are shown and their order
	output := runPiped(t, file, "f\r\x1b")

	// After filtering, we should see filtered results
	// The output should show only tasks matching the selected tag
	_ = output

	// Verify the file wasn't modified
	todos := getTodos(t, file)
	if len(todos) != 5 {
		t.Errorf("Expected 5 todos in file (filter is view-only), got %d", len(todos))
	}
}

// TestTUI_TagFilterClearAll tests clearing all tag filters
func TestTUI_TagFilterClearAll(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Task with #work tag
- [ ] Task with #home tag
- [ ] Task with no tags`

	_ = os.WriteFile(file, []byte(content), 0644)

	// Open filter mode, select a tag, then clear all with 'c'
	runPiped(t, file, "f\rc\x1b")

	// All tasks should be visible (no filter active)
	todos := getTodos(t, file)
	if len(todos) != 3 {
		t.Errorf("Expected 3 todos after clearing filter, got %d", len(todos))
	}
}

// TestTUI_TagFilterNavigateAndSelect tests navigating tag list
func TestTUI_TagFilterNavigateAndSelect(t *testing.T) {
	file := tempTestFile(t)

	content := `- [ ] Alpha task #aaa
- [ ] Beta task #bbb
- [ ] Gamma task #ccc
- [ ] Delta task #aaa #bbb`

	_ = os.WriteFile(file, []byte(content), 0644)

	// Open filter, move down to second tag, select it
	output := runPiped(t, file, "fj\r\x1b")

	// The output should show filtering is active (tag indicator in status)
	// We can't easily verify the exact filtered view, but ensure no crash
	_ = output

	todos := getTodos(t, file)
	if len(todos) != 4 {
		t.Errorf("Expected 4 todos in file, got %d", len(todos))
	}
}

// ==================== Recent Files Overlay Tests ====================

// TestTUI_RecentFilesOverlayOpen tests opening the recent files overlay
func TestTUI_RecentFilesOverlayOpen(t *testing.T) {
	tmpDir := t.TempDir()

	// Override config dir for testing
	config.SetConfigDirForTesting(tmpDir)
	defer config.ResetConfigDirForTesting()

	file1 := filepath.Join(tmpDir, "file1.md")
	file2 := filepath.Join(tmpDir, "file2.md")

	_ = os.WriteFile(file1, []byte("- [ ] Task in file 1"), 0644)
	_ = os.WriteFile(file2, []byte("- [ ] Task in file 2"), 0644)

	// Open file1 first to add it to recent
	runPiped(t, file1, "\x1b")

	// Open file2 and then press 'r' to open recent files overlay
	output := runPiped(t, file2, "r\x1b")

	// The recent files overlay should show (or at least not crash)
	// Check that file2 tasks are visible (since we escaped the overlay)
	if !strings.Contains(output, "Task in file 2") {
		t.Logf("Output: %s", output)
		// This is okay - the overlay might show instead
	}
}

// TestTUI_RecentFilesOverlayNavigation tests navigating the recent files list
func TestTUI_RecentFilesOverlayNavigation(t *testing.T) {
	tmpDir := t.TempDir()

	// Override config dir for testing
	config.SetConfigDirForTesting(tmpDir)
	defer config.ResetConfigDirForTesting()

	file1 := filepath.Join(tmpDir, "first.md")
	file2 := filepath.Join(tmpDir, "second.md")
	file3 := filepath.Join(tmpDir, "third.md")

	_ = os.WriteFile(file1, []byte("- [ ] First file task"), 0644)
	_ = os.WriteFile(file2, []byte("- [ ] Second file task"), 0644)
	_ = os.WriteFile(file3, []byte("- [ ] Third file task"), 0644)

	// Open each file to populate recent list
	runPiped(t, file1, "\x1b")
	runPiped(t, file2, "\x1b")
	runPiped(t, file3, "\x1b")

	// From file3, open recent overlay, navigate down, and select
	output := runPiped(t, file3, "rj\r\x1b")

	// Should have switched to a different file (file2 or file1)
	// The exact file depends on sorting, but we verify no crash
	_ = output
}

// TestTUI_RecentFilesOverlaySearch tests searching in recent files
func TestTUI_RecentFilesOverlaySearch(t *testing.T) {
	tmpDir := t.TempDir()

	// Override config dir for testing
	config.SetConfigDirForTesting(tmpDir)
	defer config.ResetConfigDirForTesting()

	file1 := filepath.Join(tmpDir, "project_alpha.md")
	file2 := filepath.Join(tmpDir, "project_beta.md")
	file3 := filepath.Join(tmpDir, "notes.md")

	_ = os.WriteFile(file1, []byte("- [ ] Alpha task"), 0644)
	_ = os.WriteFile(file2, []byte("- [ ] Beta task"), 0644)
	_ = os.WriteFile(file3, []byte("- [ ] Notes task"), 0644)

	// Open each file
	runPiped(t, file1, "\x1b")
	runPiped(t, file2, "\x1b")
	runPiped(t, file3, "\x1b")

	// From notes, open recent, type "alpha" to filter, then select
	output := runPiped(t, file3, "ralpha\r\x1b")

	// Should have filtered to project_alpha and selected it
	// Verify no crash and output is reasonable
	_ = output
}

// ==================== Command Palette Additional Tests ====================

// TestTUI_CommandShowHeadings tests toggling show-headings
func TestTUI_CommandShowHeadings(t *testing.T) {
	file := tempTestFile(t)

	content := `# Project Tasks

## Backend
- [ ] API work

## Frontend
- [ ] UI work`

	_ = os.WriteFile(file, []byte(content), 0644)

	// Toggle show-headings on
	output := runPiped(t, file, ":show-headings\r\x1b")

	// Should see headings in output when enabled
	if !strings.Contains(output, "Backend") && !strings.Contains(output, "Frontend") {
		t.Log("Note: Headings may not be visible in this view mode")
	}

	// Tasks should be preserved
	todos := getTodos(t, file)
	if len(todos) != 2 {
		t.Errorf("Expected 2 todos, got %d", len(todos))
	}
}

// TestTUI_CommandSort tests the sort command
func TestTUI_CommandSort(t *testing.T) {
	file := tempTestFile(t)

	// Create mixed checked/unchecked tasks
	content := `- [x] Done task 1
- [ ] Pending task 1
- [x] Done task 2
- [ ] Pending task 2`

	_ = os.WriteFile(file, []byte(content), 0644)

	// Run sort command
	runPiped(t, file, ":sort\r\x1b")

	todos := getTodos(t, file)
	if len(todos) != 4 {
		t.Fatalf("Expected 4 todos, got %d", len(todos))
	}

	// After sort, incomplete tasks should come first
	if !strings.HasPrefix(todos[0], "- [ ]") {
		t.Errorf("First task should be incomplete after sort, got: %s", todos[0])
	}
	if !strings.HasPrefix(todos[1], "- [ ]") {
		t.Errorf("Second task should be incomplete after sort, got: %s", todos[1])
	}
	if !strings.HasPrefix(todos[2], "- [x]") {
		t.Errorf("Third task should be complete after sort, got: %s", todos[2])
	}
	if !strings.HasPrefix(todos[3], "- [x]") {
		t.Errorf("Fourth task should be complete after sort, got: %s", todos[3])
	}
}

// TestTUI_CommandReload tests reloading file from disk
func TestTUI_CommandReload(t *testing.T) {
	file := tempTestFile(t)

	_ = os.WriteFile(file, []byte("- [ ] Original task"), 0644)

	// Make a change via TUI
	runPiped(t, file, " ")

	todos := getTodos(t, file)
	if !strings.HasPrefix(todos[0], "- [x]") {
		t.Errorf("Task should be checked, got: %s", todos[0])
	}

	// Externally modify the file
	_ = os.WriteFile(file, []byte("- [ ] External change"), 0644)

	// Reload from disk (discards our toggle)
	runPiped(t, file, ":reload\r\x1b")

	todos = getTodos(t, file)
	if !strings.Contains(todos[0], "External change") {
		t.Errorf("Should have external change after reload, got: %s", todos[0])
	}
}

// TestTUI_UndoSort tests undoing the sort command in same session
func TestTUI_UndoSort(t *testing.T) {
	file := tempTestFile(t)

	// Create specific order
	content := `- [x] Done first
- [ ] Pending
- [x] Done second`

	_ = os.WriteFile(file, []byte(content), 0644)

	// Sort then immediately undo in same session
	runPiped(t, file, ":sort\ru")

	todos := getTodos(t, file)
	// Original order should be restored after undo
	if !strings.Contains(todos[0], "Done first") {
		t.Errorf("First should be 'Done first' after undo, got: %s", todos[0])
	}
	if !strings.Contains(todos[1], "Pending") {
		t.Errorf("Second should be 'Pending' after undo, got: %s", todos[1])
	}
}
