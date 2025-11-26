package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/niklas-heer/tdx/internal/config"
)

// TestFrontmatterBugRepro_FileSwitchMoveSequence reproduces the exact sequence
// that caused frontmatter deletion:
// 1. Open program
// 2. Press Help (?)
// 3. Press R (recent files)
// 4. Move to another file
// 5. Press R again
// 6. Move back to original file
// 7. Press M (move mode)
// 8. Move item into empty heading (filtered by filter-done)
// 9. Press Enter to confirm move
// 10. Press M again
// 11. Move it back
// 12. Press Enter again
// 13. Press M and cancel
func TestFrontmatterBugRepro_FileSwitchMoveSequence(t *testing.T) {
	tmpDir := t.TempDir()

	// Override config dir for testing
	config.SetConfigDirForTesting(tmpDir)
	defer config.ResetConfigDirForTesting()

	file1 := filepath.Join(tmpDir, "file1.md")
	file2 := filepath.Join(tmpDir, "file2.md")

	// File1 has filter-done: true with some completed items under headings
	content1 := `---
filter-done: true
max-visible: 0
show-headings: true
---
# Project Tasks

## Section A

- [x] Done task A1
- [x] Done task A2

## Section B

- [ ] Active task B1
- [ ] Active task B2

## Section C

- [x] Done task C1
`

	content2 := `---
filter-done: false
---

- [ ] Task in file 2
`

	os.WriteFile(file1, []byte(content1), 0644)
	os.WriteFile(file2, []byte(content2), 0644)

	// Add file2 to recent first
	runPiped(t, file2, "\x1b")

	// Now simulate the exact sequence on file1:
	// 1. Open file1
	// 2. ? (help) - then close with ? or esc
	// 3. r (recent files)
	// 4. Enter to switch to file2
	runPiped(t, file1, "?\x1br\r\x1b")

	// Check file1 still has frontmatter after first switch
	content := readTestFile(t, file1)
	if !strings.HasPrefix(content, "---\n") {
		t.Fatalf("File1 frontmatter lost after first file switch, got:\n%s", content)
	}

	// Add file1 to recent
	runPiped(t, file1, "\x1b")

	// 5. r (recent files from file2)
	// 6. Enter to switch back to file1
	runPiped(t, file2, "r\r\x1b")

	// Check file1 still has frontmatter
	content = readTestFile(t, file1)
	if !strings.HasPrefix(content, "---\n") {
		t.Fatalf("File1 frontmatter lost after switching back, got:\n%s", content)
	}

	// 7-9. Move mode, move down, confirm
	// With filter-done active, we can only see Active task B1 and B2
	// Moving B1 down should swap with B2
	runPiped(t, file1, "mj\r")

	content = readTestFile(t, file1)
	if !strings.HasPrefix(content, "---\n") {
		t.Fatalf("File1 frontmatter lost after first move, got:\n%s", content)
	}
	if !strings.Contains(content, "filter-done: true") {
		t.Errorf("File1 should still have filter-done: true, got:\n%s", content)
	}

	// 10-12. Move mode again, move back up, confirm
	runPiped(t, file1, "mk\r")

	content = readTestFile(t, file1)
	if !strings.HasPrefix(content, "---\n") {
		t.Fatalf("File1 frontmatter lost after second move, got:\n%s", content)
	}

	// 13. Move mode and cancel
	runPiped(t, file1, "mj\x1b")

	// Final check - frontmatter should still be intact
	content = readTestFile(t, file1)
	if !strings.HasPrefix(content, "---\n") {
		t.Fatalf("File1 frontmatter lost after move cancel, got:\n%s", content)
	}

	// Verify all frontmatter settings are preserved
	expectedSettings := []string{
		"filter-done: true",
		"max-visible: 0",
		"show-headings: true",
	}
	for _, setting := range expectedSettings {
		if !strings.Contains(content, setting) {
			t.Errorf("File1 should have '%s', got:\n%s", setting, content)
		}
	}

	// Verify headings are preserved
	expectedHeadings := []string{
		"# Project Tasks",
		"## Section A",
		"## Section B",
		"## Section C",
	}
	for _, heading := range expectedHeadings {
		if !strings.Contains(content, heading) {
			t.Errorf("File1 should have heading '%s', got:\n%s", heading, content)
		}
	}
}

// TestFrontmatterBugRepro_AllInOneSession tries to reproduce the bug
// by doing all operations in a single runPiped session
func TestFrontmatterBugRepro_AllInOneSession(t *testing.T) {
	tmpDir := t.TempDir()

	config.SetConfigDirForTesting(tmpDir)
	defer config.ResetConfigDirForTesting()

	file1 := filepath.Join(tmpDir, "main.md")
	file2 := filepath.Join(tmpDir, "other.md")

	content1 := `---
filter-done: true
max-visible: 0
show-headings: true
---
# Main File

## Empty Section (all done)

- [x] Done 1
- [x] Done 2

## Active Section

- [ ] Active 1
- [ ] Active 2
`

	content2 := `---
word-wrap: false
---

- [ ] Other file task
`

	os.WriteFile(file1, []byte(content1), 0644)
	os.WriteFile(file2, []byte(content2), 0644)

	// Add file2 to recent
	runPiped(t, file2, "\x1b")

	// Do everything in one session from file1:
	// ? (help), ? (close help), r (recent), enter (switch),
	// then we're in file2...
	// This tests help -> recent files transition
	runPiped(t, file1, "??\x1b")

	content := readTestFile(t, file1)
	if !strings.HasPrefix(content, "---\n") {
		t.Fatalf("Frontmatter lost after help mode, got:\n%s", content)
	}

	// Now test: recent files -> switch -> recent files -> switch back -> move operations
	runPiped(t, file1, "r\r\x1b") // Switch to file2

	// Add file1 to recent from file2's perspective
	runPiped(t, file1, "\x1b")

	runPiped(t, file2, "r\r\x1b") // Switch back to file1

	// Multiple move operations
	runPiped(t, file1, "mj\rmk\rmj\x1b")

	// Verify frontmatter
	content = readTestFile(t, file1)
	if !strings.HasPrefix(content, "---\n") {
		t.Fatalf("Frontmatter lost after move sequence, got:\n%s", content)
	}

	for _, expected := range []string{"filter-done: true", "max-visible: 0", "show-headings: true"} {
		if !strings.Contains(content, expected) {
			t.Errorf("Should have '%s', got:\n%s", expected, content)
		}
	}
}

// TestFrontmatterBugRepro_MoveIntoFilteredHeading specifically tests
// moving into a heading that appears empty due to filter-done
func TestFrontmatterBugRepro_MoveIntoFilteredHeading(t *testing.T) {
	file := tempTestFile(t)

	// Create a file where Section A has only completed items (hidden by filter)
	// and Section B has active items
	content := `---
filter-done: true
show-headings: true
---
# Tasks

## Section A (appears empty)

- [x] Done A1
- [x] Done A2
- [x] Done A3

## Section B (has visible items)

- [ ] Active B1
- [ ] Active B2
`

	os.WriteFile(file, []byte(content), 0644)

	// Move Active B1 up - this might try to move it into the "empty" Section A
	// With filter-done, we should only see Active B1 and B2
	// Moving up from B1 should do nothing (it's at the top of visible items)
	runPiped(t, file, "mk\r")

	result := readTestFile(t, file)
	if !strings.HasPrefix(result, "---\n") {
		t.Fatalf("Frontmatter lost after moving into filtered heading, got:\n%s", result)
	}
	if !strings.Contains(result, "filter-done: true") {
		t.Errorf("Should have filter-done: true, got:\n%s", result)
	}
	if !strings.Contains(result, "show-headings: true") {
		t.Errorf("Should have show-headings: true, got:\n%s", result)
	}

	// Verify headings are preserved
	if !strings.Contains(result, "## Section A") {
		t.Errorf("Should have Section A heading, got:\n%s", result)
	}
	if !strings.Contains(result, "## Section B") {
		t.Errorf("Should have Section B heading, got:\n%s", result)
	}
}

// TestFrontmatterBugRepro_RepeatedMoveConfirmCancel tests multiple
// move-confirm and move-cancel cycles
func TestFrontmatterBugRepro_RepeatedMoveConfirmCancel(t *testing.T) {
	file := tempTestFile(t)

	content := `---
filter-done: true
show-headings: true
max-visible: 0
---
# Tasks

- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
`

	os.WriteFile(file, []byte(content), 0644)

	// Move down and confirm
	runPiped(t, file, "mj\r")
	assertFrontmatterExists(t, file, map[string]string{
		"filter-done":   "true",
		"show-headings": "true",
		"max-visible":   "0",
	})

	// Move down and cancel
	runPiped(t, file, "mj\x1b")
	assertFrontmatterExists(t, file, map[string]string{
		"filter-done":   "true",
		"show-headings": "true",
		"max-visible":   "0",
	})

	// Move up and confirm
	runPiped(t, file, "mk\r")
	assertFrontmatterExists(t, file, map[string]string{
		"filter-done":   "true",
		"show-headings": "true",
		"max-visible":   "0",
	})

	// Move and cancel multiple times
	runPiped(t, file, "mj\x1bmk\x1bmj\x1b")
	assertFrontmatterExists(t, file, map[string]string{
		"filter-done":   "true",
		"show-headings": "true",
		"max-visible":   "0",
	})

	// Final move confirm
	runPiped(t, file, "mj\r")
	assertFrontmatterExists(t, file, map[string]string{
		"filter-done":   "true",
		"show-headings": "true",
		"max-visible":   "0",
	})
}
