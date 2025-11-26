package markdown

import (
	"os"
	"testing"
)

func TestMetadataIntegration_FilterDone(t *testing.T) {
	// Create a temporary file with filter-done metadata
	tmpfile, err := os.CreateTemp("", "test_filter_done_*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	content := `---
filter-done: true
---
# Tasks

- [ ] Active task
- [x] Completed task
`

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	_ = tmpfile.Close()

	// Read the file
	fm, err := ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	// Verify metadata was parsed
	if fm.Metadata == nil {
		t.Fatal("Metadata should not be nil")
	}

	if fm.Metadata.FilterDone == nil {
		t.Fatal("FilterDone should not be nil")
	}

	if !*fm.Metadata.FilterDone {
		t.Error("FilterDone should be true")
	}

	// Verify todos were parsed
	if len(fm.Todos) != 2 {
		t.Errorf("Expected 2 todos, got %d", len(fm.Todos))
	}
}

func TestMetadataIntegration_WordWrap(t *testing.T) {
	// Create a temporary file with word-wrap metadata
	tmpfile, err := os.CreateTemp("", "test_word_wrap_*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	content := `---
word-wrap: true
---
# Tasks

- [ ] This is a very long task that should wrap
`

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	_ = tmpfile.Close()

	// Read the file
	fm, err := ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	// Verify metadata was parsed
	if fm.Metadata == nil {
		t.Fatal("Metadata should not be nil")
	}

	if fm.Metadata.WordWrap == nil {
		t.Fatal("WordWrap should not be nil")
	}

	if !*fm.Metadata.WordWrap {
		t.Error("WordWrap should be true")
	}
}

func TestMetadataIntegration_AllFields(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test_all_fields_*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	content := `---
filter-done: true
max-visible: 10
show-headings: true
read-only: false
word-wrap: true
---
# Tasks

- [ ] Task 1
`

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	_ = tmpfile.Close()

	// Read the file
	fm, err := ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	// Verify all metadata fields
	if fm.Metadata == nil {
		t.Fatal("Metadata should not be nil")
	}

	if fm.Metadata.FilterDone == nil || *fm.Metadata.FilterDone != true {
		t.Error("FilterDone should be true")
	}

	if fm.Metadata.MaxVisible == nil || *fm.Metadata.MaxVisible != 10 {
		t.Error("MaxVisible should be 10")
	}

	if fm.Metadata.ShowHeadings == nil || *fm.Metadata.ShowHeadings != true {
		t.Error("ShowHeadings should be true")
	}

	if fm.Metadata.ReadOnly == nil || *fm.Metadata.ReadOnly != false {
		t.Error("ReadOnly should be false")
	}

	if fm.Metadata.WordWrap == nil || *fm.Metadata.WordWrap != true {
		t.Error("WordWrap should be true")
	}
}
