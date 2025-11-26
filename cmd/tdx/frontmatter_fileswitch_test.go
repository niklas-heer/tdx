package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/niklas-heer/tdx/internal/config"
)

// TestFrontmatterPreservation_FileSwitchWithRecentFiles tests that switching files
// via the recent files overlay doesn't corrupt frontmatter in either file
func TestFrontmatterPreservation_FileSwitchWithRecentFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Override config dir for testing
	config.SetConfigDirForTesting(tmpDir)
	defer config.ResetConfigDirForTesting()

	file1 := filepath.Join(tmpDir, "file1.md")
	file2 := filepath.Join(tmpDir, "file2.md")

	content1 := `---
filter-done: true
show-headings: true
max-visible: 10
---

# File 1

- [ ] Task 1 in file 1
- [ ] Task 2 in file 1
`

	content2 := `---
filter-done: false
word-wrap: true
---

# File 2

- [ ] Task 1 in file 2
- [ ] Task 2 in file 2
`

	os.WriteFile(file1, []byte(content1), 0644)
	os.WriteFile(file2, []byte(content2), 0644)

	// Open file1 to add it to recent
	runPiped(t, file1, "\x1b")

	// Open file2, navigate, then switch to file1 via recent files
	// r (recent files), enter (select first recent file)
	runPiped(t, file2, "jj r\r\x1b")

	// Check file1 still has frontmatter
	content := readTestFile(t, file1)
	if !strings.HasPrefix(content, "---\n") {
		t.Errorf("File1 should still have frontmatter, got:\n%s", content)
	}
	if !strings.Contains(content, "filter-done: true") {
		t.Errorf("File1 should have filter-done: true, got:\n%s", content)
	}
	if !strings.Contains(content, "show-headings: true") {
		t.Errorf("File1 should have show-headings: true, got:\n%s", content)
	}
	if !strings.Contains(content, "max-visible: 10") {
		t.Errorf("File1 should have max-visible: 10, got:\n%s", content)
	}

	// Check file2 still has frontmatter
	content = readTestFile(t, file2)
	if !strings.HasPrefix(content, "---\n") {
		t.Errorf("File2 should still have frontmatter, got:\n%s", content)
	}
	if !strings.Contains(content, "filter-done: false") {
		t.Errorf("File2 should have filter-done: false, got:\n%s", content)
	}
	if !strings.Contains(content, "word-wrap: true") {
		t.Errorf("File2 should have word-wrap: true, got:\n%s", content)
	}
}

// TestFrontmatterPreservation_MoveModeCancel tests that canceling move mode
// doesn't corrupt frontmatter
func TestFrontmatterPreservation_MoveModeCancel(t *testing.T) {
	file := tempTestFile(t)

	initial := `---
filter-done: true
show-headings: true
max-visible: 5
---

- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
`
	os.WriteFile(file, []byte(initial), 0644)

	// Enter move mode, move down, then cancel with Escape
	runPiped(t, file, "mj\x1b")

	assertFrontmatterExists(t, file, map[string]string{
		"filter-done":   "true",
		"show-headings": "true",
		"max-visible":   "5",
	})

	// Verify tasks are in original order (move was canceled)
	todos := getTodos(t, file)
	if len(todos) != 3 {
		t.Fatalf("Should have 3 todos, got %d", len(todos))
	}
}

// TestFrontmatterPreservation_MoveModeConfirm tests that confirming move mode
// preserves frontmatter
func TestFrontmatterPreservation_MoveModeConfirm(t *testing.T) {
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

	// Enter move mode, move down, then confirm with Enter
	runPiped(t, file, "mj\r")

	assertFrontmatterExists(t, file, map[string]string{
		"filter-done":   "true",
		"show-headings": "true",
	})

	// Verify tasks were reordered
	todos := getTodos(t, file)
	if len(todos) != 3 {
		t.Fatalf("Should have 3 todos, got %d", len(todos))
	}
	if !strings.Contains(todos[0], "Task 2") {
		t.Errorf("First task should be Task 2 after move, got: %s", todos[0])
	}
}

// TestFrontmatterPreservation_FileSwitchDuringMoveMode tests switching files
// while in move mode (should cancel move and preserve frontmatter)
func TestFrontmatterPreservation_FileSwitchDuringMoveMode(t *testing.T) {
	tmpDir := t.TempDir()

	// Override config dir for testing
	config.SetConfigDirForTesting(tmpDir)
	defer config.ResetConfigDirForTesting()

	file1 := filepath.Join(tmpDir, "file1.md")
	file2 := filepath.Join(tmpDir, "file2.md")

	content1 := `---
filter-done: true
show-headings: true
---

- [ ] File1 Task 1
- [ ] File1 Task 2
`

	content2 := `---
word-wrap: false
max-visible: 20
---

- [ ] File2 Task 1
- [ ] File2 Task 2
`

	os.WriteFile(file1, []byte(content1), 0644)
	os.WriteFile(file2, []byte(content2), 0644)

	// Open file1 to add it to recent
	runPiped(t, file1, "\x1b")

	// Open file2, enter move mode, move down, then open recent files and switch
	// This should cancel move mode implicitly
	runPiped(t, file2, "mjr\r\x1b")

	// Check file1 still has frontmatter
	content := readTestFile(t, file1)
	if !strings.HasPrefix(content, "---\n") {
		t.Errorf("File1 should still have frontmatter, got:\n%s", content)
	}

	// Check file2 still has frontmatter (move should not have been saved)
	content = readTestFile(t, file2)
	if !strings.HasPrefix(content, "---\n") {
		t.Errorf("File2 should still have frontmatter, got:\n%s", content)
	}
	if !strings.Contains(content, "word-wrap: false") {
		t.Errorf("File2 should have word-wrap: false, got:\n%s", content)
	}
	if !strings.Contains(content, "max-visible: 20") {
		t.Errorf("File2 should have max-visible: 20, got:\n%s", content)
	}
}

// TestFrontmatterPreservation_MultipleFileSwitches tests multiple file switches
func TestFrontmatterPreservation_MultipleFileSwitches(t *testing.T) {
	tmpDir := t.TempDir()

	// Override config dir for testing
	config.SetConfigDirForTesting(tmpDir)
	defer config.ResetConfigDirForTesting()

	file1 := filepath.Join(tmpDir, "file1.md")
	file2 := filepath.Join(tmpDir, "file2.md")
	file3 := filepath.Join(tmpDir, "file3.md")

	content1 := `---
filter-done: true
---

- [ ] File1 Task
`

	content2 := `---
show-headings: true
---

- [ ] File2 Task
`

	content3 := `---
word-wrap: false
---

- [ ] File3 Task
`

	os.WriteFile(file1, []byte(content1), 0644)
	os.WriteFile(file2, []byte(content2), 0644)
	os.WriteFile(file3, []byte(content3), 0644)

	// Open all files to add them to recent
	runPiped(t, file1, "\x1b")
	runPiped(t, file2, "\x1b")
	runPiped(t, file3, "\x1b")

	// From file3, switch to file2, then to file1
	runPiped(t, file3, "r\r\x1b") // Switch to file2
	runPiped(t, file2, "r\r\x1b") // Switch to file1

	// Verify all files still have their frontmatter
	for _, tc := range []struct {
		file     string
		expected string
	}{
		{file1, "filter-done: true"},
		{file2, "show-headings: true"},
		{file3, "word-wrap: false"},
	} {
		content := readTestFile(t, tc.file)
		if !strings.HasPrefix(content, "---\n") {
			t.Errorf("%s should have frontmatter, got:\n%s", tc.file, content)
		}
		if !strings.Contains(content, tc.expected) {
			t.Errorf("%s should contain '%s', got:\n%s", tc.file, tc.expected, content)
		}
	}
}

// TestFrontmatterPreservation_EditThenSwitch tests editing a task then switching files
func TestFrontmatterPreservation_EditThenSwitch(t *testing.T) {
	tmpDir := t.TempDir()

	// Override config dir for testing
	config.SetConfigDirForTesting(tmpDir)
	defer config.ResetConfigDirForTesting()

	file1 := filepath.Join(tmpDir, "file1.md")
	file2 := filepath.Join(tmpDir, "file2.md")

	content1 := `---
filter-done: true
max-visible: 15
---

- [ ] Original Task in File1
`

	content2 := `---
show-headings: true
---

- [ ] Task in File2
`

	os.WriteFile(file1, []byte(content1), 0644)
	os.WriteFile(file2, []byte(content2), 0644)

	// Open file1 first
	runPiped(t, file1, "\x1b")

	// Open file2, edit the task, then switch to file1
	runPiped(t, file2, "e\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7fEdited Task\rr\r\x1b")

	// Check file1 frontmatter is preserved
	content := readTestFile(t, file1)
	if !strings.HasPrefix(content, "---\n") {
		t.Errorf("File1 should have frontmatter, got:\n%s", content)
	}
	if !strings.Contains(content, "filter-done: true") {
		t.Errorf("File1 should have filter-done: true, got:\n%s", content)
	}
	if !strings.Contains(content, "max-visible: 15") {
		t.Errorf("File1 should have max-visible: 15, got:\n%s", content)
	}

	// Check file2 frontmatter is preserved and edit was saved
	content = readTestFile(t, file2)
	if !strings.HasPrefix(content, "---\n") {
		t.Errorf("File2 should have frontmatter, got:\n%s", content)
	}
	if !strings.Contains(content, "show-headings: true") {
		t.Errorf("File2 should have show-headings: true, got:\n%s", content)
	}
	if !strings.Contains(content, "Edited Task") {
		t.Errorf("File2 should have edited task, got:\n%s", content)
	}
}

// TestFrontmatterPreservation_UndoAfterSwitch tests that undo after file switch
// doesn't corrupt frontmatter
func TestFrontmatterPreservation_ToggleThenSwitch(t *testing.T) {
	tmpDir := t.TempDir()

	// Override config dir for testing
	config.SetConfigDirForTesting(tmpDir)
	defer config.ResetConfigDirForTesting()

	file1 := filepath.Join(tmpDir, "file1.md")
	file2 := filepath.Join(tmpDir, "file2.md")

	content1 := `---
filter-done: true
---

- [ ] Task in File1
`

	content2 := `---
word-wrap: false
---

- [ ] Task in File2
`

	os.WriteFile(file1, []byte(content1), 0644)
	os.WriteFile(file2, []byte(content2), 0644)

	// Open file1 first
	runPiped(t, file1, "\x1b")

	// Open file2, toggle the task, then switch to file1
	runPiped(t, file2, " r\r\x1b")

	// Check file1 frontmatter is preserved
	content := readTestFile(t, file1)
	if !strings.Contains(content, "filter-done: true") {
		t.Errorf("File1 should have filter-done: true, got:\n%s", content)
	}

	// Check file2 frontmatter is preserved and toggle was saved
	content = readTestFile(t, file2)
	if !strings.Contains(content, "word-wrap: false") {
		t.Errorf("File2 should have word-wrap: false, got:\n%s", content)
	}
	if !strings.Contains(content, "[x]") {
		t.Errorf("File2 task should be checked, got:\n%s", content)
	}
}
