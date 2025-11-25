package tui

import (
	"testing"

	"github.com/niklas-heer/tdx/internal/markdown"
)

// TestNew_InitialCursorWithFilterDone tests that cursor starts on first visible item
// when FilterDone metadata is present
func TestNew_InitialCursorWithFilterDone(t *testing.T) {
	filterDone := true
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Done task 1", Checked: true},
			{Text: "Done task 2", Checked: true},
			{Text: "Task A1", Checked: false},
			{Text: "Task A2", Checked: false},
		},
		Metadata: &markdown.Metadata{
			FilterDone: &filterDone,
		},
	}

	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	// Cursor should be on index 2 (first unchecked task)
	if m.SelectedIndex != 2 {
		t.Errorf("SelectedIndex = %d, want 2 (first visible task)", m.SelectedIndex)
	}
}

// TestNew_InitialCursorWithFilterDoneMiddle tests cursor positioning when
// completed tasks are in the middle
func TestNew_InitialCursorWithFilterDoneMiddle(t *testing.T) {
	filterDone := true
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task A", Checked: false},
			{Text: "Done task 1", Checked: true},
			{Text: "Done task 2", Checked: true},
			{Text: "Task B", Checked: false},
		},
		Metadata: &markdown.Metadata{
			FilterDone: &filterDone,
		},
	}

	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	// Cursor should be on index 0 (first unchecked task)
	if m.SelectedIndex != 0 {
		t.Errorf("SelectedIndex = %d, want 0 (first visible task)", m.SelectedIndex)
	}
}

// TestNew_InitialCursorWithShowHeadings tests cursor starts on first todo
// (not heading) when show-headings is enabled
func TestNew_InitialCursorWithShowHeadings(t *testing.T) {
	showHeadings := true
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task A", Checked: false},
			{Text: "Task B", Checked: false},
		},
		Metadata: &markdown.Metadata{
			ShowHeadings: &showHeadings,
		},
	}

	m := New("/tmp/test.md", fm, false, true, -1, testConfig(), testStyles(), "")

	// Cursor should be on index 0 (first todo, not on heading)
	if m.SelectedIndex != 0 {
		t.Errorf("SelectedIndex = %d, want 0 (first todo)", m.SelectedIndex)
	}
}

// TestNew_InitialCursorNoFilters tests cursor starts at 0 when no filters
func TestNew_InitialCursorNoFilters(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Done task", Checked: true},
			{Text: "Task A", Checked: false},
		},
	}

	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	// Cursor should be on index 0 (default behavior)
	if m.SelectedIndex != 0 {
		t.Errorf("SelectedIndex = %d, want 0 (no filters)", m.SelectedIndex)
	}
}

// TestNew_InitialCursorAllCompleted tests behavior when all tasks are completed
func TestNew_InitialCursorAllCompleted(t *testing.T) {
	filterDone := true
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Done 1", Checked: true},
			{Text: "Done 2", Checked: true},
			{Text: "Done 3", Checked: true},
		},
		Metadata: &markdown.Metadata{
			FilterDone: &filterDone,
		},
	}

	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	// Cursor should remain at 0 (no visible items)
	if m.SelectedIndex != 0 {
		t.Errorf("SelectedIndex = %d, want 0 (no visible items)", m.SelectedIndex)
	}
}

// TestNew_MetadataAppliedInNew tests that metadata settings are applied during initialization
func TestNew_MetadataAppliedInNew(t *testing.T) {
	filterDone := true
	wordWrap := false
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task 1", Checked: false},
		},
		Metadata: &markdown.Metadata{
			FilterDone: &filterDone,
			WordWrap:   &wordWrap,
		},
	}

	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	// Verify metadata was applied
	if !m.FilterDone {
		t.Error("FilterDone should be true from metadata")
	}
	if m.WordWrap {
		t.Error("WordWrap should be false from metadata")
	}
}
