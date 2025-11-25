package markdown

import (
	"strings"
	"testing"
)

func TestFrontmatterPreservation_AddTodo(t *testing.T) {
	content := `---
read-only: true
filter-done: false
max-visible: 10
---
# Todos

- [ ] Existing task
`

	metadata, contentWithout, _ := ParseMetadata(content)
	fm := ParseMarkdown(contentWithout)
	fm.Metadata = metadata

	// Verify metadata is not empty
	if fm.Metadata.IsEmpty() {
		t.Fatal("Metadata should not be empty after parsing")
	}

	// Add a new todo
	fm.AddTodoItem("New task", false)

	// Serialize and check frontmatter is preserved
	serialized := SerializeMarkdown(fm)

	if !strings.HasPrefix(serialized, "---\n") {
		t.Errorf("Frontmatter should be preserved after adding todo. Got:\n%s", serialized)
	}

	if !strings.Contains(serialized, "read-only: true") {
		t.Error("read-only setting should be preserved")
	}

	if !strings.Contains(serialized, "max-visible: 10") {
		t.Error("max-visible setting should be preserved")
	}
}

func TestFrontmatterPreservation_UpdateTodo(t *testing.T) {
	content := `---
read-only: false
word-wrap: true
---
# Todos

- [ ] Task to update
`

	metadata, contentWithout, _ := ParseMetadata(content)
	fm := ParseMarkdown(contentWithout)
	fm.Metadata = metadata

	// Update the todo
	err := fm.UpdateTodoItem(0, "Updated task", false)
	if err != nil {
		t.Fatalf("UpdateTodoItem failed: %v", err)
	}

	// Serialize and check frontmatter is preserved
	serialized := SerializeMarkdown(fm)

	if !strings.HasPrefix(serialized, "---\n") {
		t.Error("Frontmatter should be preserved after updating todo")
	}

	if !strings.Contains(serialized, "word-wrap: true") {
		t.Error("word-wrap setting should be preserved")
	}
}

func TestFrontmatterPreservation_DeleteTodo(t *testing.T) {
	content := `---
word-wrap: false
show-headings: true
---
# Todos

- [ ] Task to delete
- [ ] Task to keep
`

	metadata, contentWithout, _ := ParseMetadata(content)
	fm := ParseMarkdown(contentWithout)
	fm.Metadata = metadata

	// Delete the first todo
	err := fm.DeleteTodoItem(0)
	if err != nil {
		t.Fatalf("DeleteTodoItem failed: %v", err)
	}

	// Serialize and check frontmatter is preserved
	serialized := SerializeMarkdown(fm)

	if !strings.HasPrefix(serialized, "---\n") {
		t.Error("Frontmatter should be preserved after deleting todo")
	}

	if !strings.Contains(serialized, "word-wrap: false") {
		t.Error("word-wrap setting should be preserved")
	}

	if !strings.Contains(serialized, "show-headings: true") {
		t.Error("show-headings setting should be preserved")
	}
}

func TestFrontmatterPreservation_MoveTodo(t *testing.T) {
	content := `---
max-visible: 5
---
# Todos

- [ ] First
- [ ] Second
`

	metadata, contentWithout, _ := ParseMetadata(content)
	fm := ParseMarkdown(contentWithout)
	fm.Metadata = metadata

	// Move todos
	err := fm.MoveTodoItem(0, 1)
	if err != nil {
		t.Fatalf("MoveTodoItem failed: %v", err)
	}

	// Serialize and check frontmatter is preserved
	serialized := SerializeMarkdown(fm)

	if !strings.HasPrefix(serialized, "---\n") {
		t.Error("Frontmatter should be preserved after moving todo")
	}

	if !strings.Contains(serialized, "max-visible: 5") {
		t.Error("max-visible setting should be preserved")
	}
}

func TestFrontmatterPreservation_RoundTrip(t *testing.T) {
	content := `---
filter-done: false
max-visible: 10
show-headings: true
read-only: false
word-wrap: true
---
# Todos

- [ ] Task one
- [x] Task two
`

	// Parse
	metadata, contentWithout, err := ParseMetadata(content)
	if err != nil {
		t.Fatalf("ParseMetadata failed: %v", err)
	}

	fm := ParseMarkdown(contentWithout)
	fm.Metadata = metadata

	// Serialize
	serialized := SerializeMarkdown(fm)

	// Parse again
	metadata2, contentWithout2, err := ParseMetadata(serialized)
	if err != nil {
		t.Fatalf("Second ParseMetadata failed: %v", err)
	}

	// Verify all metadata fields preserved
	if metadata2.FilterDone == nil || *metadata2.FilterDone != false {
		t.Error("filter-done should be false")
	}

	if metadata2.MaxVisible == nil || *metadata2.MaxVisible != 10 {
		t.Error("max-visible should be 10")
	}

	if metadata2.ShowHeadings == nil || *metadata2.ShowHeadings != true {
		t.Error("show-headings should be true")
	}

	if metadata2.ReadOnly == nil || *metadata2.ReadOnly != false {
		t.Error("read-only should be false")
	}

	if metadata2.WordWrap == nil || *metadata2.WordWrap != true {
		t.Error("word-wrap should be true")
	}

	// Verify content is preserved
	fm2 := ParseMarkdown(contentWithout2)
	if len(fm2.Todos) != 2 {
		t.Errorf("Expected 2 todos after round-trip, got %d", len(fm2.Todos))
	}
}

func TestFrontmatterPreservation_EmptyMetadata(t *testing.T) {
	content := `# Todos

- [ ] Task without frontmatter
`

	fm := ParseMarkdown(content)

	// Serialize - should not add empty frontmatter
	serialized := SerializeMarkdown(fm)

	if strings.HasPrefix(serialized, "---\n") {
		t.Error("Empty frontmatter should not be added")
	}
}

func TestFrontmatterPreservation_WithInlineCode(t *testing.T) {
	content := `---
filter-done: true
---
# Todos

- [ ] Fix bug in ` + "`main.go`" + `
`

	metadata, contentWithout, _ := ParseMetadata(content)
	fm := ParseMarkdown(contentWithout)
	fm.Metadata = metadata

	// Update with more inline code
	err := fm.UpdateTodoItem(0, "Fix `main.go` and `utils.go`", false)
	if err != nil {
		t.Fatalf("UpdateTodoItem failed: %v", err)
	}

	// Serialize and check
	serialized := SerializeMarkdown(fm)

	if !strings.Contains(serialized, "filter-done: true") {
		t.Error("filter-done setting should be preserved with inline code")
	}

	if !strings.Contains(serialized, "`main.go`") {
		t.Error("Inline code should be preserved")
	}

	if !strings.Contains(serialized, "`utils.go`") {
		t.Error("New inline code should be preserved")
	}
}
