package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/niklas-heer/tdx/internal/markdown"
)

// testModelWithMarkdown creates a test model by parsing markdown content.
// This ensures the FileModel has a proper AST for operations like delete.
func testModelWithMarkdown(content string) Model {
	fm := markdown.ParseMarkdown(content)
	return New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
}

// TestTagFilterRecognizesNewTags verifies that when a new todo with a new tag
// is added during a TUI session, the tag filter should recognize the new tag.
// This is a regression test for the bug where AvailableTags was only populated
// once during New() and never refreshed.
func TestTagFilterRecognizesNewTags(t *testing.T) {
	// Start with a todo that has no tags
	m := testModelWithMarkdown("- [ ] Task without tags\n")

	// Initially, there should be no available tags
	if len(m.AvailableTags) != 0 {
		t.Errorf("Initial AvailableTags = %v, want empty", m.AvailableTags)
	}

	// Simulate adding a new todo with a tag via input mode
	m.InputMode = true
	m.InputBuffer = "New task #newtag"
	m.addNewTodo()
	m.InputMode = false

	// The new todo should have the tag extracted
	if len(m.FileModel.Todos) != 2 {
		t.Fatalf("Expected 2 todos, got %d", len(m.FileModel.Todos))
	}

	newTodo := m.FileModel.Todos[1]
	if len(newTodo.Tags) == 0 || newTodo.Tags[0] != "newtag" {
		t.Errorf("New todo Tags = %v, want [newtag]", newTodo.Tags)
	}

	// BUG: AvailableTags should now include "newtag", but it doesn't get refreshed
	// This test will FAIL until the bug is fixed
	if len(m.AvailableTags) != 1 || m.AvailableTags[0] != "newtag" {
		t.Errorf("AvailableTags = %v, want [newtag] - AvailableTags not refreshed after adding new todo", m.AvailableTags)
	}

	// Enter tag filter mode - should show the new tag
	m.FilterMode = true
	m.TagFilterCursor = 0

	// Should be able to select the new tag
	if len(m.AvailableTags) == 0 {
		t.Error("Tag filter mode shows no tags, but we just added one")
	}
}

// TestTagFilterRecognizesTagsAddedViaEdit verifies that when a tag is added
// to an existing todo via edit mode, the tag filter recognizes the new tag.
func TestTagFilterRecognizesTagsAddedViaEdit(t *testing.T) {
	// Start with a todo that has one tag
	m := testModelWithMarkdown("- [ ] Task #existing\n")

	// Initially, only "existing" tag should be available
	if len(m.AvailableTags) != 1 || m.AvailableTags[0] != "existing" {
		t.Errorf("Initial AvailableTags = %v, want [existing]", m.AvailableTags)
	}

	// Simulate editing the todo to add another tag (as done in handleInputKey)
	m.SelectedIndex = 0
	_ = m.FileModel.UpdateTodoItem(0, "Task #existing #newedittag", false)
	m.InvalidateDocumentTree()
	m.RefreshAvailableTags() // This is called by handleInputKey after edit

	// Verify the todo now has both tags
	editedTodo := m.FileModel.Todos[0]
	if len(editedTodo.Tags) != 2 {
		t.Errorf("Edited todo Tags = %v, want 2 tags", editedTodo.Tags)
	}

	// AvailableTags should now include both tags
	if len(m.AvailableTags) != 2 {
		t.Errorf("AvailableTags = %v, want [existing newedittag] - AvailableTags not refreshed after editing todo", m.AvailableTags)
	}
}

// TestTagFilterRecognizesTagsRemovedViaEdit verifies that when a tag is removed
// from a todo (and no other todos have it), the tag filter no longer shows it.
func TestTagFilterRecognizesTagsRemovedViaEdit(t *testing.T) {
	// Start with todos that have different tags
	m := testModelWithMarkdown("- [ ] Task #unique #shared\n- [ ] Task #shared\n")

	// Initially, both tags should be available
	if len(m.AvailableTags) != 2 {
		t.Errorf("Initial AvailableTags = %v, want [shared unique]", m.AvailableTags)
	}

	// Edit first todo to remove the "unique" tag (as done in handleInputKey)
	_ = m.FileModel.UpdateTodoItem(0, "Task #shared", false)
	m.InvalidateDocumentTree()
	m.RefreshAvailableTags() // This is called by handleInputKey after edit

	// AvailableTags should now only have "shared" since "unique" is no longer used
	if len(m.AvailableTags) != 1 || m.AvailableTags[0] != "shared" {
		t.Errorf("AvailableTags = %v, want [shared] - AvailableTags not refreshed after removing tag", m.AvailableTags)
	}
}

// TestTagFilterRecognizesTagsAfterDelete verifies that when a todo with a unique
// tag is deleted, the tag filter no longer shows that tag.
func TestTagFilterRecognizesTagsAfterDelete(t *testing.T) {
	// Start with todos that have different tags
	m := testModelWithMarkdown("- [ ] Task #unique\n- [ ] Task #common\n")

	// Initially, both tags should be available
	if len(m.AvailableTags) != 2 {
		t.Errorf("Initial AvailableTags = %v, want [common unique]", m.AvailableTags)
	}

	// Delete the first todo (which has the unique tag) - as done in deleteTodo
	m.SelectedIndex = 0
	_ = m.FileModel.DeleteTodoItem(0)
	m.InvalidateDocumentTree()
	m.RefreshAvailableTags() // This is called by deleteTodo after deletion

	// AvailableTags should now only have "common" since "unique" was deleted
	if len(m.AvailableTags) != 1 || m.AvailableTags[0] != "common" {
		t.Errorf("AvailableTags = %v, want [common] - AvailableTags not refreshed after deleting todo", m.AvailableTags)
	}
}

// TestFilteredTagsStillWorkAfterSourceDeleted verifies that if a user filters
// by a tag, and then the only todo with that tag is deleted, the filter state
// is cleaned up properly.
func TestFilteredTagsStillWorkAfterSourceDeleted(t *testing.T) {
	// Start with todos that have different tags
	m := testModelWithMarkdown("- [ ] Task #unique\n- [ ] Task #common\n")

	// Filter by the "unique" tag
	m.FilteredTags = []string{"unique"}
	m.InvalidateDocumentTree()

	// Verify filter is active
	if len(m.FilteredTags) != 1 {
		t.Fatalf("FilteredTags = %v, want [unique]", m.FilteredTags)
	}

	// Delete the todo with the "unique" tag - as done in deleteTodo
	m.SelectedIndex = 0
	_ = m.FileModel.DeleteTodoItem(0)
	m.InvalidateDocumentTree()
	m.RefreshAvailableTags() // This is called by deleteTodo and also cleans up FilteredTags

	// The filter should be cleaned up since the tag no longer exists
	// After RefreshAvailableTags, FilteredTags should only contain valid tags
	for _, tag := range m.FilteredTags {
		found := false
		for _, availTag := range markdown.GetAllTags(m.FileModel.Todos) {
			if availTag == tag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("FilteredTags contains %q which no longer exists in any todo", tag)
		}
	}
}

// TestEnterTagFilterModeRefreshesAvailableTags verifies that entering tag filter
// mode refreshes the available tags list. This is one way to fix the bug.
func TestEnterTagFilterModeRefreshesAvailableTags(t *testing.T) {
	// Start with no tags
	m := testModelWithMarkdown("- [ ] Task without tags\n")

	// Add a todo with a tag directly to FileModel (simulating what happens after addNewTodo)
	m.FileModel.Todos = append(m.FileModel.Todos, markdown.Todo{
		Text: "Task #newtag",
		Tags: []string{"newtag"},
	})

	// Simulate pressing 't' to enter tag filter mode
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
	result, _ := m.handleKey(msg)
	m = result.(Model)

	// BUG: After entering tag filter mode, AvailableTags should be refreshed
	// This test will FAIL until the bug is fixed
	if !m.FilterMode {
		t.Fatal("Should be in FilterMode after pressing 't'")
	}
	if len(m.AvailableTags) != 1 || m.AvailableTags[0] != "newtag" {
		t.Errorf("AvailableTags = %v, want [newtag] - entering filter mode should refresh available tags", m.AvailableTags)
	}
}
