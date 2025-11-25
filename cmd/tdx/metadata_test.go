package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/niklas-heer/tdx/internal/markdown"
)

func TestMetadataParsing_EmptyFrontmatter(t *testing.T) {
	content := `# Todos

- [ ] Task one
- [x] Task two
`
	metadata, cleanContent, err := markdown.ParseMetadata(content)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !metadata.IsEmpty() {
		t.Errorf("expected empty metadata")
	}

	if cleanContent != content {
		t.Errorf("expected content to be unchanged")
	}
}

func TestMetadataParsing_WithFrontmatter(t *testing.T) {
	content := `---
read-only: false
max-visible: 10
show-headings: true
---
# Todos

- [ ] Task one
`
	metadata, cleanContent, err := markdown.ParseMetadata(content)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if metadata.ReadOnly == nil || *metadata.ReadOnly != false {
		t.Errorf("expected read-only: false")
	}

	if metadata.MaxVisible == nil || *metadata.MaxVisible != 10 {
		t.Errorf("expected max-visible: 10")
	}

	if metadata.ShowHeadings == nil || *metadata.ShowHeadings != true {
		t.Errorf("expected show-headings: true")
	}

	expectedClean := `# Todos

- [ ] Task one
`
	if cleanContent != expectedClean {
		t.Errorf("expected frontmatter to be stripped\ngot:\n%s\nwant:\n%s", cleanContent, expectedClean)
	}
}

func TestMetadataParsing_AllFields(t *testing.T) {
	content := `---
filter-done: true
max-visible: 20
show-headings: false
read-only: true
word-wrap: false
---
# Todos
`
	metadata, _, err := markdown.ParseMetadata(content)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if metadata.FilterDone == nil || *metadata.FilterDone != true {
		t.Errorf("expected filter-done: true")
	}

	if metadata.MaxVisible == nil || *metadata.MaxVisible != 20 {
		t.Errorf("expected max-visible: 20")
	}

	if metadata.ShowHeadings == nil || *metadata.ShowHeadings != false {
		t.Errorf("expected show-headings: false")
	}

	if metadata.ReadOnly == nil || *metadata.ReadOnly != true {
		t.Errorf("expected read-only: true")
	}

	if metadata.WordWrap == nil || *metadata.WordWrap != false {
		t.Errorf("expected word-wrap: false")
	}
}

func TestMetadataParsing_InvalidYAML(t *testing.T) {
	content := `---
this is not: valid: yaml
---
# Todos
`
	metadata, cleanContent, err := markdown.ParseMetadata(content)
	if err == nil {
		t.Fatalf("expected error for invalid YAML")
	}

	// Should still strip frontmatter even if invalid
	if metadata == nil {
		t.Errorf("expected metadata to be non-nil")
	}

	expectedClean := `# Todos
`
	if cleanContent != expectedClean {
		t.Errorf("expected frontmatter to be stripped even on error")
	}
}

func TestMetadataParsing_UnknownField(t *testing.T) {
	content := `---
read-only: false
unknown-field: value
---
# Todos
`
	_, _, err := markdown.ParseMetadata(content)
	if err == nil {
		t.Errorf("expected error for unknown field")
	}
}

func TestMetadataSerialization_Empty(t *testing.T) {
	metadata := &markdown.Metadata{}
	content := "# Todos\n\n- [ ] Task one\n"

	result := markdown.SerializeMetadata(metadata, content)
	if result != content {
		t.Errorf("expected no frontmatter for empty metadata\ngot:\n%s", result)
	}
}

func TestMetadataSerialization_WithFields(t *testing.T) {
	readOnlyFalse := false
	maxVis := 10
	metadata := &markdown.Metadata{
		ReadOnly:   &readOnlyFalse,
		MaxVisible: &maxVis,
	}
	content := "# Todos\n"

	result := markdown.SerializeMetadata(metadata, content)

	// Should contain frontmatter
	if result == content {
		t.Errorf("expected frontmatter to be added")
	}

	// Re-parse to verify
	parsed, cleanContent, err := markdown.ParseMetadata(result)
	if err != nil {
		t.Fatalf("failed to re-parse serialized metadata: %v", err)
	}

	if parsed.ReadOnly == nil || *parsed.ReadOnly != false {
		t.Errorf("expected read-only: false after round-trip")
	}

	if parsed.MaxVisible == nil || *parsed.MaxVisible != 10 {
		t.Errorf("expected max-visible: 10 after round-trip")
	}

	if cleanContent != content {
		t.Errorf("expected content to be preserved after round-trip")
	}
}

func TestMetadataIntegration_ReadWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.md")

	// Create file with metadata
	content := `---
filter-done: false
max-visible: 15
---
# Test Todos

- [ ] First task
- [x] Second task
`
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Read file
	fm, err := markdown.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	// Verify metadata was parsed
	if fm.Metadata == nil {
		t.Fatalf("expected metadata to be parsed")
	}

	if fm.Metadata.FilterDone == nil || *fm.Metadata.FilterDone != false {
		t.Errorf("expected filter-done: false from file")
	}

	if fm.Metadata.MaxVisible == nil || *fm.Metadata.MaxVisible != 15 {
		t.Errorf("expected max-visible: 15 from file")
	}

	// Verify todos were parsed correctly
	if len(fm.Todos) != 2 {
		t.Fatalf("expected 2 todos, got %d", len(fm.Todos))
	}

	if fm.Todos[0].Text != "First task" || fm.Todos[0].Checked {
		t.Errorf("first todo incorrect")
	}

	if fm.Todos[1].Text != "Second task" || !fm.Todos[1].Checked {
		t.Errorf("second todo incorrect")
	}

	// Modify a todo
	err = fm.UpdateTodoItem(0, "First task", true)
	if err != nil {
		t.Fatalf("failed to update todo: %v", err)
	}

	// Write back
	err = markdown.WriteFileUnchecked(filePath, fm)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Read again
	fm2, err := markdown.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to re-read file: %v", err)
	}

	// Verify metadata was preserved
	if fm2.Metadata.FilterDone == nil || *fm2.Metadata.FilterDone != false {
		t.Errorf("metadata not preserved after write")
	}

	// Verify todo was updated
	if !fm2.Todos[0].Checked {
		t.Errorf("todo update not preserved")
	}
}

func TestMetadataGetters(t *testing.T) {
	trueVal := true
	falseVal := false
	maxVis := 25

	metadata := &markdown.Metadata{
		FilterDone:   &falseVal,
		ShowHeadings: &trueVal,
		MaxVisible:   &maxVis,
	}

	// Test GetBool with set value
	if metadata.GetBool("filter-done", true) != false {
		t.Errorf("expected filter-done to return false")
	}

	// Test GetBool with default
	if metadata.GetBool("read-only", true) != true {
		t.Errorf("expected read-only to return default true")
	}

	if metadata.GetBool("read-only", false) != false {
		t.Errorf("expected read-only to return default false")
	}

	// Test GetInt with set value
	if metadata.GetInt("max-visible", 10) != 25 {
		t.Errorf("expected max-visible to return 25")
	}

	// Test GetInt with default
	if metadata.GetInt("unknown", 99) != 99 {
		t.Errorf("expected unknown field to return default 99")
	}
}

func TestMetadataIsEmpty(t *testing.T) {
	// Test completely empty
	metadata := &markdown.Metadata{}
	if !metadata.IsEmpty() {
		t.Errorf("expected empty metadata to report as empty")
	}

	// Test with one field set
	trueVal := true
	metadata.FilterDone = &trueVal
	if metadata.IsEmpty() {
		t.Errorf("expected metadata with filter-done field to not be empty")
	}
}
