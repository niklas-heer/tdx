package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRecentCommandClear tests clearing recent files
func TestRecentCommandClear(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")

	if err := os.WriteFile(testFile, []byte("- [ ] Task"), 0644); err != nil {
		t.Fatal(err)
	}

	// Add file to recent using TUI
	runPiped(t, testFile, " ")

	// Verify it's in recent using CLI
	output := runCLI(t, "", "recent")
	if !strings.Contains(output, "test.md") {
		t.Logf("Output: %s", output)
		t.Errorf("Expected test.md in recent files before clear")
	}

	// Clear recent files
	output = runCLI(t, "", "recent", "clear")
	if !strings.Contains(output, "cleared") {
		t.Errorf("Expected 'cleared' message, got: %s", output)
	}

	// Verify it's empty
	output = runCLI(t, "", "recent")
	if !strings.Contains(output, "No recent files") {
		t.Errorf("Expected 'No recent files' after clear, got: %s", output)
	}
}

// TestRecentFilesCursorRestoration tests cursor position is restored
func TestRecentFilesCursorRestoration(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "cursor_test.md")

	content := `- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
- [ ] Task 4
- [ ] Task 5`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Clear recent files
	runCLI(t, "", "recent", "clear")

	// Open file, move cursor down 3 times, then quit
	output := runPiped(t, testFile, "jjj ")

	// Verify we're on task 4 (index 3)
	if !strings.Contains(output, "3 ➜ [ ] Task 4") {
		t.Errorf("Expected cursor on Task 4 after 3 downs, got: %s", output)
	}

	// Open file again - cursor should be restored to position 3
	output = runPiped(t, testFile, " ")

	if !strings.Contains(output, "3 ➜ [ ] Task 4") {
		t.Errorf("Expected cursor restored to Task 4, got: %s", output)
	}
}

// TestRecentFilesCursorResetOnChange tests cursor is not restored if file changed
func TestRecentFilesCursorResetOnChange(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "change_test.md")

	content := `- [ ] Task 1
- [ ] Task 2
- [ ] Task 3`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Clear recent files
	runCLI(t, "", "recent", "clear")

	// Open file, move cursor down 2 times
	output := runPiped(t, testFile, "jj ")
	if !strings.Contains(output, "2 ➜ [ ] Task 3") {
		t.Errorf("Expected cursor on Task 3, got: %s", output)
	}

	// Modify the file
	newContent := `- [ ] New Task 1
- [ ] Task 1
- [ ] Task 2
- [ ] Task 3`

	if err := os.WriteFile(testFile, []byte(newContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Open file again - cursor should be at start (file changed)
	output = runPiped(t, testFile, " ")

	if !strings.Contains(output, "0 ➜ [ ] New Task 1") {
		t.Errorf("Expected cursor reset to first task after file change, got: %s", output)
	}
}
