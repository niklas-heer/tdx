package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveAndLoadRecentFiles(t *testing.T) {
	// Create a temp directory for test
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")

	// Create test file
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Override config dir for testing
	oldGetConfigDir := getConfigDir
	getConfigDir = func() (string, error) {
		return tmpDir, nil
	}
	defer func() { getConfigDir = oldGetConfigDir }()

	// Save a recent file
	if err := SaveRecentFile(testFile, 5); err != nil {
		t.Fatalf("SaveRecentFile failed: %v", err)
	}

	// Load recent files
	recentFiles, err := LoadRecentFiles()
	if err != nil {
		t.Fatalf("LoadRecentFiles failed: %v", err)
	}

	// Verify the file was saved
	if len(recentFiles.Files) != 1 {
		t.Fatalf("Expected 1 recent file, got %d", len(recentFiles.Files))
	}

	file := recentFiles.Files[0]
	if file.Path != testFile {
		t.Errorf("Expected path %s, got %s", testFile, file.Path)
	}
	if file.AccessCount != 1 {
		t.Errorf("Expected access count 1, got %d", file.AccessCount)
	}
	if file.LastCursorPos != 5 {
		t.Errorf("Expected cursor position 5, got %d", file.LastCursorPos)
	}
}

func TestRecentFilesAccessCount(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")

	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	oldGetConfigDir := getConfigDir
	getConfigDir = func() (string, error) {
		return tmpDir, nil
	}
	defer func() { getConfigDir = oldGetConfigDir }()

	// Access the file multiple times
	for i := 0; i < 3; i++ {
		if err := SaveRecentFile(testFile, i); err != nil {
			t.Fatalf("SaveRecentFile failed: %v", err)
		}
	}

	recentFiles, err := LoadRecentFiles()
	if err != nil {
		t.Fatalf("LoadRecentFiles failed: %v", err)
	}

	if len(recentFiles.Files) != 1 {
		t.Fatalf("Expected 1 recent file, got %d", len(recentFiles.Files))
	}

	file := recentFiles.Files[0]
	if file.AccessCount != 3 {
		t.Errorf("Expected access count 3, got %d", file.AccessCount)
	}
	if file.LastCursorPos != 2 {
		t.Errorf("Expected last cursor position 2, got %d", file.LastCursorPos)
	}
}

func TestRecentFilesSortByScore(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	file1 := filepath.Join(tmpDir, "file1.md")
	file2 := filepath.Join(tmpDir, "file2.md")
	file3 := filepath.Join(tmpDir, "file3.md")

	for _, f := range []string{file1, file2, file3} {
		if err := os.WriteFile(f, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create recent files with different access patterns
	now := time.Now()
	rf := &RecentFiles{
		Files: []RecentFile{
			{
				Path:         file1,
				LastAccessed: now.Add(-24 * time.Hour), // 1 day ago
				AccessCount:  1,
			},
			{
				Path:         file2,
				LastAccessed: now, // Just now
				AccessCount:  1,
			},
			{
				Path:         file3,
				LastAccessed: now.Add(-1 * time.Hour), // 1 hour ago
				AccessCount:  5,                       // High frequency
			},
		},
	}

	rf.SortByScore()

	// file3 should be first (high frequency, recent)
	// file2 should be second (low frequency, very recent)
	// file1 should be last (low frequency, old)
	if rf.Files[0].Path != file3 {
		t.Errorf("Expected file3 to be first, got %s", rf.Files[0].Path)
	}
	if rf.Files[len(rf.Files)-1].Path != file1 {
		t.Errorf("Expected file1 to be last, got %s", rf.Files[len(rf.Files)-1].Path)
	}
}

func TestGetCursorPosition(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")

	// Create test file with specific content
	content := []byte("original content")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	oldGetConfigDir := getConfigDir
	getConfigDir = func() (string, error) {
		return tmpDir, nil
	}
	defer func() { getConfigDir = oldGetConfigDir }()

	// Save with cursor position
	if err := SaveRecentFile(testFile, 10); err != nil {
		t.Fatalf("SaveRecentFile failed: %v", err)
	}

	recentFiles, err := LoadRecentFiles()
	if err != nil {
		t.Fatalf("LoadRecentFiles failed: %v", err)
	}

	// Should return cursor position if file hasn't changed
	cursorPos := recentFiles.GetCursorPosition(testFile)
	if cursorPos != 10 {
		t.Errorf("Expected cursor position 10, got %d", cursorPos)
	}

	// Modify the file
	if err := os.WriteFile(testFile, []byte("modified content"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Should return -1 if file has changed
	cursorPos = recentFiles.GetCursorPosition(testFile)
	if cursorPos != -1 {
		t.Errorf("Expected cursor position -1 for modified file, got %d", cursorPos)
	}
}

func TestMaxRecentFiles(t *testing.T) {
	tmpDir := t.TempDir()

	oldGetConfigDir := getConfigDir
	getConfigDir = func() (string, error) {
		return tmpDir, nil
	}
	defer func() { getConfigDir = oldGetConfigDir }()

	// Create more files than the default max
	maxFiles := 5
	for i := 0; i < maxFiles+3; i++ {
		testFile := filepath.Join(tmpDir, "test"+string(rune('0'+i))+".md")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Override max for this test
		rf, _ := LoadRecentFiles()
		rf.MaxRecent = maxFiles
		if err := rf.Save(); err != nil {
			t.Fatalf("Save failed: %v", err)
		}

		if err := SaveRecentFile(testFile, 0); err != nil {
			t.Fatalf("SaveRecentFile failed: %v", err)
		}
	}

	recentFiles, err := LoadRecentFiles()
	if err != nil {
		t.Fatalf("LoadRecentFiles failed: %v", err)
	}

	// Should not exceed max
	if len(recentFiles.Files) > maxFiles {
		t.Errorf("Expected at most %d recent files, got %d", maxFiles, len(recentFiles.Files))
	}
}

func TestClearRecentFiles(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")

	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	oldGetConfigDir := getConfigDir
	getConfigDir = func() (string, error) {
		return tmpDir, nil
	}
	defer func() { getConfigDir = oldGetConfigDir }()

	// Add a file
	if err := SaveRecentFile(testFile, 0); err != nil {
		t.Fatalf("SaveRecentFile failed: %v", err)
	}

	// Verify it exists
	rf, _ := LoadRecentFiles()
	if len(rf.Files) != 1 {
		t.Fatalf("Expected 1 file before clear, got %d", len(rf.Files))
	}

	// Clear
	if err := ClearRecentFiles(); err != nil {
		t.Fatalf("ClearRecentFiles failed: %v", err)
	}

	// Verify it's empty
	rf, _ = LoadRecentFiles()
	if len(rf.Files) != 0 {
		t.Errorf("Expected 0 files after clear, got %d", len(rf.Files))
	}
}
