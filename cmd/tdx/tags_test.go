package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/niklas-heer/tdx/internal/markdown"
)

func TestTagExtraction_NoTags(t *testing.T) {
	text := "This is a simple task"
	tags := markdown.ExtractTags(text)

	if len(tags) != 0 {
		t.Errorf("expected no tags, got %v", tags)
	}
}

func TestTagExtraction_SingleTag(t *testing.T) {
	text := "Fix bug #urgent"
	tags := markdown.ExtractTags(text)

	if len(tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(tags))
	}

	if tags[0] != "urgent" {
		t.Errorf("expected tag 'urgent', got '%s'", tags[0])
	}
}

func TestTagExtraction_MultipleTags(t *testing.T) {
	text := "Fix authentication bug #urgent #backend #security"
	tags := markdown.ExtractTags(text)

	if len(tags) != 3 {
		t.Fatalf("expected 3 tags, got %d", len(tags))
	}

	expected := []string{"urgent", "backend", "security"}
	for i, exp := range expected {
		if tags[i] != exp {
			t.Errorf("expected tag '%s', got '%s'", exp, tags[i])
		}
	}
}

func TestTagExtraction_DuplicateTags(t *testing.T) {
	text := "Task with duplicate tags #test #important #test"
	tags := markdown.ExtractTags(text)

	if len(tags) != 2 {
		t.Fatalf("expected 2 unique tags, got %d: %v", len(tags), tags)
	}
}

func TestTagExtraction_TagsWithDashesAndUnderscores(t *testing.T) {
	text := "Task #high-priority #work_item #bug_fix"
	tags := markdown.ExtractTags(text)

	if len(tags) != 3 {
		t.Fatalf("expected 3 tags, got %d", len(tags))
	}

	expected := []string{"high-priority", "work_item", "bug_fix"}
	for i, exp := range expected {
		if tags[i] != exp {
			t.Errorf("expected tag '%s', got '%s'", exp, tags[i])
		}
	}
}

func TestTagExtraction_MiddleOfText(t *testing.T) {
	text := "Fix the #urgent bug in authentication"
	tags := markdown.ExtractTags(text)

	if len(tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(tags))
	}

	if tags[0] != "urgent" {
		t.Errorf("expected tag 'urgent', got '%s'", tags[0])
	}
}

func TestRemoveTags(t *testing.T) {
	text := "Fix bug #urgent #backend"
	cleaned := markdown.RemoveTags(text)

	expected := "Fix bug"
	if cleaned != expected {
		t.Errorf("expected '%s', got '%s'", expected, cleaned)
	}
}

func TestTodo_HasTag(t *testing.T) {
	todo := markdown.Todo{
		Text: "Fix bug #urgent #backend",
		Tags: []string{"urgent", "backend"},
	}

	if !todo.HasTag("urgent") {
		t.Error("expected todo to have 'urgent' tag")
	}

	if !todo.HasTag("backend") {
		t.Error("expected todo to have 'backend' tag")
	}

	if todo.HasTag("frontend") {
		t.Error("expected todo not to have 'frontend' tag")
	}
}

func TestTodo_HasTag_CaseInsensitive(t *testing.T) {
	todo := markdown.Todo{
		Text: "Fix bug #urgent",
		Tags: []string{"urgent"},
	}

	if !todo.HasTag("URGENT") {
		t.Error("expected case-insensitive tag matching")
	}

	if !todo.HasTag("Urgent") {
		t.Error("expected case-insensitive tag matching")
	}
}

func TestTodo_HasAnyTag(t *testing.T) {
	todo := markdown.Todo{
		Text: "Fix bug #urgent #backend",
		Tags: []string{"urgent", "backend"},
	}

	// Should match if any tag matches
	if !todo.HasAnyTag([]string{"urgent"}) {
		t.Error("expected match with 'urgent'")
	}

	if !todo.HasAnyTag([]string{"frontend", "urgent"}) {
		t.Error("expected match with one of the tags")
	}

	if todo.HasAnyTag([]string{"frontend", "mobile"}) {
		t.Error("expected no match with different tags")
	}

	// Empty filter should match all
	if !todo.HasAnyTag([]string{}) {
		t.Error("expected empty filter to match all")
	}
}

func TestGetAllTags(t *testing.T) {
	todos := []markdown.Todo{
		{Text: "Task 1 #urgent", Tags: []string{"urgent"}},
		{Text: "Task 2 #backend #urgent", Tags: []string{"backend", "urgent"}},
		{Text: "Task 3 #frontend", Tags: []string{"frontend"}},
		{Text: "Task 4", Tags: []string{}},
	}

	allTags := markdown.GetAllTags(todos)

	// Should have 3 unique tags
	if len(allTags) != 3 {
		t.Fatalf("expected 3 unique tags, got %d: %v", len(allTags), allTags)
	}

	// Check that all expected tags are present
	expectedTags := map[string]bool{
		"urgent":   false,
		"backend":  false,
		"frontend": false,
	}

	for _, tag := range allTags {
		if _, exists := expectedTags[tag]; exists {
			expectedTags[tag] = true
		} else {
			t.Errorf("unexpected tag: %s", tag)
		}
	}

	for tag, found := range expectedTags {
		if !found {
			t.Errorf("expected tag '%s' not found", tag)
		}
	}
}

func TestTagsIntegration_ParseAndExtract(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.md")

	content := `# Test Todos

- [ ] Fix authentication bug #urgent #backend
- [x] Update documentation #docs
- [ ] Add new feature #frontend #enhancement
`

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	fm, err := markdown.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if len(fm.Todos) != 3 {
		t.Fatalf("expected 3 todos, got %d", len(fm.Todos))
	}

	// Check first todo
	if len(fm.Todos[0].Tags) != 2 {
		t.Errorf("expected 2 tags for first todo, got %d", len(fm.Todos[0].Tags))
	}
	if !fm.Todos[0].HasTag("urgent") || !fm.Todos[0].HasTag("backend") {
		t.Errorf("first todo should have 'urgent' and 'backend' tags")
	}

	// Check second todo
	if len(fm.Todos[1].Tags) != 1 {
		t.Errorf("expected 1 tag for second todo, got %d", len(fm.Todos[1].Tags))
	}
	if !fm.Todos[1].HasTag("docs") {
		t.Errorf("second todo should have 'docs' tag")
	}

	// Check third todo
	if len(fm.Todos[2].Tags) != 2 {
		t.Errorf("expected 2 tags for third todo, got %d", len(fm.Todos[2].Tags))
	}
	if !fm.Todos[2].HasTag("frontend") || !fm.Todos[2].HasTag("enhancement") {
		t.Errorf("third todo should have 'frontend' and 'enhancement' tags")
	}

	// Check GetAllTags
	allTags := markdown.GetAllTags(fm.Todos)
	if len(allTags) != 5 {
		t.Errorf("expected 5 unique tags total, got %d: %v", len(allTags), allTags)
	}
}

func TestTagsIntegration_PreserveAfterWrite(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.md")

	content := `# Test Todos

- [ ] Task with tags #urgent #backend
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

	// Toggle the todo
	err = fm.UpdateTodoItem(0, fm.Todos[0].Text, true)
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

	// Check tags are preserved
	if len(fm2.Todos[0].Tags) != 2 {
		t.Errorf("expected tags to be preserved, got %d tags", len(fm2.Todos[0].Tags))
	}

	if !fm2.Todos[0].HasTag("urgent") || !fm2.Todos[0].HasTag("backend") {
		t.Errorf("tags 'urgent' and 'backend' should be preserved")
	}

	// Check the text still contains the tags
	if !strings.Contains(fm2.Todos[0].Text, "#urgent") || !strings.Contains(fm2.Todos[0].Text, "#backend") {
		t.Errorf("text should still contain tag markers: %s", fm2.Todos[0].Text)
	}
}
