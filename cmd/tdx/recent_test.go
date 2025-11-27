package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/niklas-heer/tdx/internal/config"
)

// TestRecentCommandClear tests clearing recent files
func TestRecentCommandClear(t *testing.T) {
	tmpDir := t.TempDir()

	// Override config dir for testing to isolate from other tests
	config.SetConfigDirForTesting(tmpDir)
	defer config.ResetConfigDirForTesting()

	testFile := filepath.Join(tmpDir, "test.md")

	if err := os.WriteFile(testFile, []byte("- [ ] Task"), 0644); err != nil {
		t.Fatal(err)
	}

	// Add file to recent using TUI (open and quit with Escape)
	runPiped(t, testFile, "\x1b")

	// Clear recent files using the config API directly
	// (CLI commands require proper arg parsing which is complex with empty file path)
	if err := config.ClearRecentFiles(); err != nil {
		t.Fatalf("Failed to clear recent files: %v", err)
	}

	// Verify recent files are cleared
	recentFiles, err := config.LoadRecentFiles()
	if err != nil {
		t.Fatalf("Failed to load recent files: %v", err)
	}

	if len(recentFiles.Files) != 0 {
		t.Errorf("Expected 0 recent files after clear, got %d", len(recentFiles.Files))
	}
}

// TestRecentFilesCursorRestoration tests cursor position is restored
func TestRecentFilesCursorRestoration(t *testing.T) {
	tmpDir := t.TempDir()

	// Override config dir for testing to isolate from other tests
	config.SetConfigDirForTesting(tmpDir)
	defer config.ResetConfigDirForTesting()

	testFile := filepath.Join(tmpDir, "cursor_test.md")

	content := `- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
- [ ] Task 4
- [ ] Task 5`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Open file, move cursor down 3 times, then quit with Escape
	output := runPiped(t, testFile, "jjj\x1b")

	// The display uses relative positions where 0 = selected item
	// After moving down 3 times, we're on Task 4 which shows as "0 ➜" (selected)
	if !strings.Contains(output, "0 ➜ [ ] Task 4") {
		t.Errorf("Expected cursor on Task 4 (shown as 0 ➜) after 3 downs, got: %s", output)
	}

	// Open file again - cursor should be restored to Task 4 (still shown as 0 ➜)
	output = runPiped(t, testFile, "\x1b")

	if !strings.Contains(output, "0 ➜ [ ] Task 4") {
		t.Errorf("Expected cursor restored to Task 4 (shown as 0 ➜), got: %s", output)
	}
}

// TestRecentFilesCursorResetOnChange tests cursor is not restored if file changed
func TestRecentFilesCursorResetOnChange(t *testing.T) {
	tmpDir := t.TempDir()

	// Override config dir for testing to isolate from other tests
	config.SetConfigDirForTesting(tmpDir)
	defer config.ResetConfigDirForTesting()

	testFile := filepath.Join(tmpDir, "change_test.md")

	content := `- [ ] Task 1
- [ ] Task 2
- [ ] Task 3`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Open file, move cursor down 2 times, then quit with Escape
	output := runPiped(t, testFile, "jj\x1b")
	// Selected item always shows as "0 ➜" (relative position)
	if !strings.Contains(output, "0 ➜ [ ] Task 3") {
		t.Errorf("Expected cursor on Task 3 (shown as 0 ➜), got: %s", output)
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
	output = runPiped(t, testFile, "\x1b")

	if !strings.Contains(output, "0 ➜ [ ] New Task 1") {
		t.Errorf("Expected cursor reset to first task after file change, got: %s", output)
	}
}
