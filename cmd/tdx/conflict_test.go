package main

import (
	"os"
	"testing"
	"time"

	"github.com/niklas-heer/tdx/internal/markdown"
)

// TestConflictDetection_ExternalModification tests that external file modifications are detected
func TestConflictDetection_ExternalModification(t *testing.T) {
	content := `# Tasks

Important paragraph that should not disappear.

- [ ] Task one
- [ ] Task two

Another important paragraph.
`

	tmpFile := tempTestFile(t)
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write initial content: %v", err)
	}

	// Load file (simulating TUI opening it)
	loadedFM, err := markdown.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load file: %v", err)
	}

	// Verify initial state
	if len(loadedFM.Todos) != 2 {
		t.Fatalf("Expected 2 todos, got %d", len(loadedFM.Todos))
	}

	// Wait to ensure different modification time
	time.Sleep(1100 * time.Millisecond)

	// Simulate external process modifying the file
	externalContent := `# Tasks

Important paragraph that should not disappear.

EXTERNAL EDIT: New paragraph added by another process!

- [ ] Task one
- [ ] Task two

Another important paragraph.
`
	if err := os.WriteFile(tmpFile, []byte(externalContent), 0644); err != nil {
		t.Fatalf("Failed to write external modification: %v", err)
	}

	// Simulate TUI making a change (toggle task)
	_ = loadedFM.UpdateTodoItem(0, "Task one", true)

	// Try to write - should detect conflict
	err = markdown.WriteFile(tmpFile, loadedFM)
	if err == nil {
		t.Fatal("Expected conflict detection error, but write succeeded")
	}

	// Verify error message mentions external modification
	expectedMsg := "file changed externally"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error '%s', got: %v", expectedMsg, err)
	}

	// Verify file still has external changes (not overwritten)
	diskContent, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	diskStr := string(diskContent)
	if !contains(diskStr, "EXTERNAL EDIT") {
		t.Error("External edit was lost - file was overwritten despite conflict!")
	}
	if !contains(diskStr, "Important paragraph") {
		t.Error("Original content was lost")
	}
}

// TestConflictDetection_ReloadAfterConflict tests the reload workflow
func TestConflictDetection_ReloadAfterConflict(t *testing.T) {
	content := `# Tasks

- [ ] Task one
- [ ] Task two
`

	tmpFile := tempTestFile(t)
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write initial content: %v", err)
	}

	// Load file
	fm, err := markdown.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load file: %v", err)
	}

	// Make a change
	_ = fm.UpdateTodoItem(0, "Task one", true)

	// External modification
	time.Sleep(1100 * time.Millisecond)
	externalContent := `# Tasks

- [ ] Task one
- [ ] Task two
- [ ] Task three (added externally)
`
	os.WriteFile(tmpFile, []byte(externalContent), 0644)

	// Try to write - should fail
	err = markdown.WriteFile(tmpFile, fm)
	if err == nil {
		t.Fatal("Expected conflict error")
	}

	// Reload from disk (simulating :reload command)
	reloadedFM, err := markdown.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to reload: %v", err)
	}

	// Verify reloaded state has external changes
	if len(reloadedFM.Todos) != 3 {
		t.Errorf("Expected 3 todos after reload, got %d", len(reloadedFM.Todos))
	}
	if reloadedFM.Todos[2].Text != "Task three (added externally)" {
		t.Error("External todo not found after reload")
	}
	// Original change should not be there
	if reloadedFM.Todos[0].Checked {
		t.Error("Local change persisted after reload (should be discarded)")
	}
}

// TestNoConflictDetection_NoExternalChange tests that writes succeed when file hasn't changed
func TestNoConflictDetection_NoExternalChange(t *testing.T) {
	content := `# Tasks

Paragraph to preserve.

- [ ] Task one
`

	tmpFile := tempTestFile(t)
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write initial content: %v", err)
	}

	// Load file
	fm, err := markdown.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load file: %v", err)
	}

	// Make a change
	_ = fm.UpdateTodoItem(0, "Task one", true)

	// Write should succeed (no external modification)
	err = markdown.WriteFile(tmpFile, fm)
	if err != nil {
		t.Fatalf("Expected write to succeed, got error: %v", err)
	}

	// Verify content
	diskContent, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	diskStr := string(diskContent)
	if !contains(diskStr, "[x] Task one") {
		t.Error("Task was not toggled")
	}
	if !contains(diskStr, "Paragraph to preserve") {
		t.Error("Paragraph was lost")
	}
}

// TestConflictDetection_MultipleWrites tests that modTime is updated after successful writes
func TestConflictDetection_MultipleWrites(t *testing.T) {
	content := `# Tasks

- [ ] Task one
- [ ] Task two
`

	tmpFile := tempTestFile(t)
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write initial content: %v", err)
	}

	// Load file
	fm, err := markdown.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load file: %v", err)
	}

	// First write
	_ = fm.UpdateTodoItem(0, "Task one", true)
	err = markdown.WriteFile(tmpFile, fm)
	if err != nil {
		t.Fatalf("First write failed: %v", err)
	}

	// Second write should also succeed (modTime updated after first write)
	time.Sleep(1100 * time.Millisecond) // Ensure different modTime
	_ = fm.UpdateTodoItem(1, "Task two", true)
	err = markdown.WriteFile(tmpFile, fm)
	if err != nil {
		t.Fatalf("Second write failed: %v", err)
	}

	// Verify both changes persisted
	diskContent, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	diskStr := string(diskContent)
	if !contains(diskStr, "[x] Task one") {
		t.Error("First task not toggled")
	}
	if !contains(diskStr, "[x] Task two") {
		t.Error("Second task not toggled")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
