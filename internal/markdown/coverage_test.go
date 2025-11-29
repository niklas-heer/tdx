package markdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Tests for functions with 0% coverage

// ==================== tags.go tests ====================

func TestRemoveTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single tag at end",
			input:    "Fix bug #urgent",
			expected: "Fix bug",
		},
		{
			name:     "multiple tags",
			input:    "Fix bug #urgent #backend #security",
			expected: "Fix bug",
		},
		{
			name:     "tag in middle",
			input:    "Fix #urgent bug",
			expected: "Fix  bug",
		},
		{
			name:     "no tags",
			input:    "Fix bug",
			expected: "Fix bug",
		},
		{
			name:     "only tags",
			input:    "#urgent #backend",
			expected: "",
		},
		{
			name:     "tags with dashes and underscores",
			input:    "Task #high-priority #work_item",
			expected: "Task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoveTags(tt.input)
			if got != tt.expected {
				t.Errorf("RemoveTags(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestTodo_HasTag(t *testing.T) {
	todo := Todo{
		Text: "Fix bug #urgent #backend",
		Tags: []string{"urgent", "backend"},
	}

	tests := []struct {
		name     string
		tag      string
		expected bool
	}{
		{"has exact tag", "urgent", true},
		{"has exact tag 2", "backend", true},
		{"case insensitive", "URGENT", true},
		{"case insensitive mixed", "Urgent", true},
		{"missing tag", "frontend", false},
		{"partial match", "urge", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := todo.HasTag(tt.tag)
			if got != tt.expected {
				t.Errorf("HasTag(%q) = %v, want %v", tt.tag, got, tt.expected)
			}
		})
	}
}

func TestTodo_HasAnyTag(t *testing.T) {
	todo := Todo{
		Text: "Fix bug #urgent #backend",
		Tags: []string{"urgent", "backend"},
	}

	tests := []struct {
		name     string
		tags     []string
		expected bool
	}{
		{"empty filter matches all", []string{}, true},
		{"single matching tag", []string{"urgent"}, true},
		{"one matching in list", []string{"frontend", "urgent"}, true},
		{"no matching tags", []string{"frontend", "mobile"}, false},
		{"all matching", []string{"urgent", "backend"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := todo.HasAnyTag(tt.tags)
			if got != tt.expected {
				t.Errorf("HasAnyTag(%v) = %v, want %v", tt.tags, got, tt.expected)
			}
		})
	}
}

func TestGetAllTags(t *testing.T) {
	tests := []struct {
		name     string
		todos    []Todo
		expected []string
	}{
		{
			name: "multiple todos with tags",
			todos: []Todo{
				{Text: "Task 1 #urgent", Tags: []string{"urgent"}},
				{Text: "Task 2 #backend #urgent", Tags: []string{"backend", "urgent"}},
				{Text: "Task 3 #frontend", Tags: []string{"frontend"}},
			},
			expected: []string{"backend", "frontend", "urgent"}, // sorted
		},
		{
			name: "todos without tags",
			todos: []Todo{
				{Text: "Task 1", Tags: []string{}},
				{Text: "Task 2", Tags: []string{}},
			},
			expected: []string{},
		},
		{
			name:     "empty todos",
			todos:    []Todo{},
			expected: []string{},
		},
		{
			name: "single tag",
			todos: []Todo{
				{Text: "Task #work", Tags: []string{"work"}},
			},
			expected: []string{"work"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetAllTags(tt.todos)
			if len(got) != len(tt.expected) {
				t.Errorf("GetAllTags() = %v, want %v", got, tt.expected)
				return
			}
			for i, v := range got {
				if v != tt.expected[i] {
					t.Errorf("GetAllTags() = %v, want %v", got, tt.expected)
					return
				}
			}
		})
	}
}

// ==================== metadata.go tests ====================

// Helper to create bool pointer
func boolPtr(b bool) *bool {
	return &b
}

// Helper to create int pointer
func intPtr(i int) *int {
	return &i
}

func TestMetadata_GetBool(t *testing.T) {
	tests := []struct {
		name       string
		metadata   *Metadata
		key        string
		defaultVal bool
		expected   bool
	}{
		{
			name:       "empty metadata returns default",
			metadata:   &Metadata{},
			key:        "filter-done",
			defaultVal: true,
			expected:   true,
		},
		{
			name:       "unknown key returns default",
			metadata:   &Metadata{FilterDone: boolPtr(true)},
			key:        "unknown-key",
			defaultVal: false,
			expected:   false,
		},
		{
			name:       "filter-done true",
			metadata:   &Metadata{FilterDone: boolPtr(true)},
			key:        "filter-done",
			defaultVal: false,
			expected:   true,
		},
		{
			name:       "filter-done false",
			metadata:   &Metadata{FilterDone: boolPtr(false)},
			key:        "filter-done",
			defaultVal: true,
			expected:   false,
		},
		{
			name:       "show-headings true",
			metadata:   &Metadata{ShowHeadings: boolPtr(true)},
			key:        "show-headings",
			defaultVal: false,
			expected:   true,
		},
		{
			name:       "read-only true",
			metadata:   &Metadata{ReadOnly: boolPtr(true)},
			key:        "read-only",
			defaultVal: false,
			expected:   true,
		},
		{
			name:       "word-wrap true",
			metadata:   &Metadata{WordWrap: boolPtr(true)},
			key:        "word-wrap",
			defaultVal: false,
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.metadata.GetBool(tt.key, tt.defaultVal)
			if got != tt.expected {
				t.Errorf("GetBool(%q, %v) = %v, want %v", tt.key, tt.defaultVal, got, tt.expected)
			}
		})
	}
}

func TestMetadata_GetInt(t *testing.T) {
	tests := []struct {
		name       string
		metadata   *Metadata
		key        string
		defaultVal int
		expected   int
	}{
		{
			name:       "empty metadata returns default",
			metadata:   &Metadata{},
			key:        "max-visible",
			defaultVal: 5,
			expected:   5,
		},
		{
			name:       "unknown key returns default",
			metadata:   &Metadata{MaxVisible: intPtr(100)},
			key:        "unknown-key",
			defaultVal: 20,
			expected:   20,
		},
		{
			name:       "max-visible set",
			metadata:   &Metadata{MaxVisible: intPtr(42)},
			key:        "max-visible",
			defaultVal: 10,
			expected:   42,
		},
		{
			name:       "max-visible zero",
			metadata:   &Metadata{MaxVisible: intPtr(0)},
			key:        "max-visible",
			defaultVal: 10,
			expected:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.metadata.GetInt(tt.key, tt.defaultVal)
			if got != tt.expected {
				t.Errorf("GetInt(%q, %v) = %v, want %v", tt.key, tt.defaultVal, got, tt.expected)
			}
		})
	}
}

// ==================== parser.go tests ====================

func TestFileModel_GetAST(t *testing.T) {
	content := `# Todos

- [ ] Task 1
- [x] Task 2
`
	fm := ParseMarkdown(content)

	ast := fm.GetAST()
	if ast == nil {
		t.Fatal("GetAST() returned nil")
	}
	if ast.AST == nil {
		t.Error("GetAST().AST is nil")
	}
	if len(ast.Source) == 0 {
		t.Error("GetAST().Source is empty")
	}
}

func TestCheckFileModified(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test in short mode")
	}

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.md")

	content := `# Todos

- [ ] Task 1
`
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Read the file
	fm, err := ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	// Check immediately - should not be modified
	modified, err := fm.CheckFileModified()
	if err != nil {
		t.Fatalf("CheckFileModified failed: %v", err)
	}
	if modified {
		t.Error("File should not be modified immediately after reading")
	}

	// Wait a bit and modify the file
	time.Sleep(1100 * time.Millisecond) // Wait more than 1 second for filesystem precision
	newContent := `# Todos

- [ ] Task 1
- [ ] Task 2
`
	err = os.WriteFile(filePath, []byte(newContent), 0644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Now it should be modified
	modified, err = fm.CheckFileModified()
	if err != nil {
		t.Fatalf("CheckFileModified failed after modification: %v", err)
	}
	if !modified {
		t.Error("File should be detected as modified")
	}
}

func TestCheckFileModified_NoFilePath(t *testing.T) {
	fm := ParseMarkdown("# Test\n- [ ] Task\n")
	// No file path set
	modified, err := fm.CheckFileModified()
	if err != nil {
		t.Errorf("CheckFileModified failed: %v", err)
	}
	if modified {
		t.Error("File with no path should not be detected as modified")
	}
}

func TestCheckFileModified_DeletedFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.md")

	content := "# Todos\n- [ ] Task\n"
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	fm, err := ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	// Delete the file
	_ = os.Remove(filePath)

	// Should not report modified (file doesn't exist)
	modified, err := fm.CheckFileModified()
	if err != nil {
		t.Errorf("CheckFileModified failed: %v", err)
	}
	if modified {
		t.Error("Deleted file should not be detected as modified")
	}
}

func TestWriteFile_ConflictDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test in short mode")
	}

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.md")

	content := "# Todos\n\n- [ ] Task 1\n"
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	fm, err := ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	// Modify the file externally
	time.Sleep(1100 * time.Millisecond)
	newContent := "# Todos\n\n- [ ] Task 1\n- [ ] External change\n"
	err = os.WriteFile(filePath, []byte(newContent), 0644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Try to write - should fail due to conflict
	err = WriteFile(filePath, fm)
	if err == nil {
		t.Error("WriteFile should have failed due to external modification")
	}
	if err.Error() != "file changed externally" {
		t.Errorf("Expected 'file changed externally' error, got: %v", err)
	}
}

func TestWriteFileUnchecked(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.md")

	content := "# Todos\n\n- [ ] Task 1\n"
	fm := ParseMarkdown(content)
	fm.FilePath = filePath

	// Write using unchecked version
	err := WriteFileUnchecked(filePath, fm)
	if err != nil {
		t.Fatalf("WriteFileUnchecked failed: %v", err)
	}

	// Verify file was written
	readContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}
	if len(readContent) == 0 {
		t.Error("Written file is empty")
	}
}

func TestFileModel_Clone(t *testing.T) {
	content := `# Todos

- [ ] Task 1
- [x] Task 2
`
	fm := ParseMarkdown(content)

	clone := fm.Clone()
	if clone == nil {
		t.Fatal("Clone() returned nil")
	}

	// Verify todos are copied
	if len(clone.Todos) != len(fm.Todos) {
		t.Errorf("Clone has %d todos, want %d", len(clone.Todos), len(fm.Todos))
	}

	// Verify it's a deep copy - modify original
	fm.Todos[0].Text = "Modified"
	if clone.Todos[0].Text == "Modified" {
		t.Error("Clone should be independent of original")
	}

	// Verify lines are copied
	if len(clone.Lines) != len(fm.Lines) {
		t.Errorf("Clone has %d lines, want %d", len(clone.Lines), len(fm.Lines))
	}
}

func TestFileModel_Clone_Nil(t *testing.T) {
	var fm *FileModel = nil
	clone := fm.Clone()
	if clone != nil {
		t.Error("Clone of nil should return nil")
	}
}

func TestInsertTodoItemAfter(t *testing.T) {
	content := `# Todos

- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
`
	fm := ParseMarkdown(content)

	// Insert after first task
	newIdx := fm.InsertTodoItemAfter(0, "New task after 1", false)
	if newIdx != 1 {
		t.Errorf("InsertTodoItemAfter returned %d, want 1", newIdx)
	}

	if len(fm.Todos) != 4 {
		t.Fatalf("Expected 4 todos, got %d", len(fm.Todos))
	}

	if fm.Todos[1].Text != "New task after 1" {
		t.Errorf("Todo at index 1 = %q, want 'New task after 1'", fm.Todos[1].Text)
	}
}

func TestInsertTodoItemAfter_AtBeginning(t *testing.T) {
	content := `# Todos

- [ ] Task 1
- [ ] Task 2
`
	fm := ParseMarkdown(content)

	// Insert at beginning (afterIndex = -1)
	newIdx := fm.InsertTodoItemAfter(-1, "New first task", false)
	if newIdx != 0 {
		t.Errorf("InsertTodoItemAfter(-1, ...) returned %d, want 0", newIdx)
	}

	if len(fm.Todos) != 3 {
		t.Fatalf("Expected 3 todos, got %d", len(fm.Todos))
	}

	if fm.Todos[0].Text != "New first task" {
		t.Errorf("Todo at index 0 = %q, want 'New first task'", fm.Todos[0].Text)
	}
}

func TestMoveTodoItemToPosition(t *testing.T) {
	content := `# Todos

- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
`
	fm := ParseMarkdown(content)

	// Move task 3 to after task 1 (before task 2)
	err := fm.MoveTodoItemToPosition(2, 1, false)
	if err != nil {
		t.Fatalf("MoveTodoItemToPosition failed: %v", err)
	}

	if len(fm.Todos) != 3 {
		t.Fatalf("Expected 3 todos, got %d", len(fm.Todos))
	}

	// After move: Task 1, Task 3, Task 2
	if fm.Todos[1].Text != "Task 3" {
		t.Errorf("Todo at index 1 = %q, want 'Task 3'", fm.Todos[1].Text)
	}
}

func TestMoveTodoItemToPosition_SameIndex(t *testing.T) {
	content := `# Todos

- [ ] Task 1
- [ ] Task 2
`
	fm := ParseMarkdown(content)

	// Move to same position should be no-op
	err := fm.MoveTodoItemToPosition(1, 1, false)
	if err != nil {
		t.Errorf("MoveTodoItemToPosition(1, 1) should not fail: %v", err)
	}
}

func TestMoveTodoItemToPosition_InvalidIndex(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	fm := ParseMarkdown(content)

	err := fm.MoveTodoItemToPosition(-1, 0, false)
	if err == nil {
		t.Error("MoveTodoItemToPosition with negative index should fail")
	}

	err = fm.MoveTodoItemToPosition(0, 5, false)
	if err == nil {
		t.Error("MoveTodoItemToPosition with out of bounds index should fail")
	}
}

func TestSwapTodoItems(t *testing.T) {
	content := `# Todos

- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
`
	fm := ParseMarkdown(content)

	err := fm.SwapTodoItems(0, 2)
	if err != nil {
		t.Fatalf("SwapTodoItems failed: %v", err)
	}

	if fm.Todos[0].Text != "Task 3" {
		t.Errorf("Todo at index 0 = %q, want 'Task 3'", fm.Todos[0].Text)
	}
	if fm.Todos[2].Text != "Task 1" {
		t.Errorf("Todo at index 2 = %q, want 'Task 1'", fm.Todos[2].Text)
	}
}

func TestSwapTodoItems_InvalidIndex(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	fm := ParseMarkdown(content)

	err := fm.SwapTodoItems(-1, 0)
	if err == nil {
		t.Error("SwapTodoItems with negative index should fail")
	}

	err = fm.SwapTodoItems(0, 5)
	if err == nil {
		t.Error("SwapTodoItems with out of bounds index should fail")
	}
}

func TestIndentTodoItem(t *testing.T) {
	content := `# Todos

- [ ] Task 1
- [ ] Task 2
`
	fm := ParseMarkdown(content)

	// Indent task 2 (make it a child of task 1)
	err := fm.IndentTodoItem(1)
	if err != nil {
		t.Fatalf("IndentTodoItem failed: %v", err)
	}

	// After indent, task 2 should have depth 1
	if fm.Todos[1].Depth != 1 {
		t.Errorf("After indent, depth = %d, want 1", fm.Todos[1].Depth)
	}
}

func TestIndentTodoItem_FirstItem(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	fm := ParseMarkdown(content)

	// Can't indent first item (no previous sibling)
	err := fm.IndentTodoItem(0)
	if err == nil {
		t.Error("IndentTodoItem on first item should fail")
	}
}

func TestIndentTodoItem_InvalidIndex(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	fm := ParseMarkdown(content)

	err := fm.IndentTodoItem(-1)
	if err == nil {
		t.Error("IndentTodoItem with negative index should fail")
	}

	err = fm.IndentTodoItem(5)
	if err == nil {
		t.Error("IndentTodoItem with out of bounds index should fail")
	}
}

func TestOutdentTodoItem(t *testing.T) {
	content := `# Todos

- [ ] Task 1
  - [ ] Task 2
`
	fm := ParseMarkdown(content)

	// Task 2 should have depth 1
	if fm.Todos[1].Depth != 1 {
		t.Fatalf("Initial depth of task 2 = %d, want 1", fm.Todos[1].Depth)
	}

	// Outdent task 2
	err := fm.OutdentTodoItem(1)
	if err != nil {
		t.Fatalf("OutdentTodoItem failed: %v", err)
	}

	// After outdent, task 2 should have depth 0
	if fm.Todos[1].Depth != 0 {
		t.Errorf("After outdent, depth = %d, want 0", fm.Todos[1].Depth)
	}
}

func TestOutdentTodoItem_TopLevel(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	fm := ParseMarkdown(content)

	// Can't outdent top-level item
	err := fm.OutdentTodoItem(0)
	if err == nil {
		t.Error("OutdentTodoItem on top-level item should fail")
	}
}

func TestOutdentTodoItem_InvalidIndex(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	fm := ParseMarkdown(content)

	err := fm.OutdentTodoItem(-1)
	if err == nil {
		t.Error("OutdentTodoItem with negative index should fail")
	}

	err = fm.OutdentTodoItem(5)
	if err == nil {
		t.Error("OutdentTodoItem with out of bounds index should fail")
	}
}

func TestGetHeadings(t *testing.T) {
	content := `# Main Title

- [ ] Task 1

## Section A

- [ ] Task 2
- [ ] Task 3

## Section B

- [ ] Task 4
`
	fm := ParseMarkdown(content)

	headings := fm.GetHeadings()
	if len(headings) != 3 {
		t.Fatalf("Expected 3 headings, got %d", len(headings))
	}

	// First heading
	if headings[0].Level != 1 {
		t.Errorf("Heading 0 level = %d, want 1", headings[0].Level)
	}
	if headings[0].Text != "Main Title" {
		t.Errorf("Heading 0 text = %q, want 'Main Title'", headings[0].Text)
	}

	// Section A
	if headings[1].Level != 2 {
		t.Errorf("Heading 1 level = %d, want 2", headings[1].Level)
	}
	if headings[1].Text != "Section A" {
		t.Errorf("Heading 1 text = %q, want 'Section A'", headings[1].Text)
	}

	// Section B
	if headings[2].Level != 2 {
		t.Errorf("Heading 2 level = %d, want 2", headings[2].Level)
	}
	if headings[2].Text != "Section B" {
		t.Errorf("Heading 2 text = %q, want 'Section B'", headings[2].Text)
	}
}

func TestGetHeadings_NoAST(t *testing.T) {
	fm := &FileModel{ast: nil}
	headings := fm.GetHeadings()
	if len(headings) != 0 {
		t.Errorf("GetHeadings with no AST should return empty slice, got %d", len(headings))
	}
}

// ==================== ast.go tests ====================

func TestASTDocument_ToggleTodo(t *testing.T) {
	content := `# Todos

- [ ] Unchecked task
- [x] Checked task
`
	doc, err := ParseAST(content)
	if err != nil {
		t.Fatalf("ParseAST failed: %v", err)
	}

	// Toggle first (unchecked -> checked)
	err = doc.ToggleTodo(0)
	if err != nil {
		t.Fatalf("ToggleTodo(0) failed: %v", err)
	}

	todos := doc.ExtractTodos()
	if !todos[0].Checked {
		t.Error("Todo 0 should be checked after toggle")
	}

	// Toggle second (checked -> unchecked)
	err = doc.ToggleTodo(1)
	if err != nil {
		t.Fatalf("ToggleTodo(1) failed: %v", err)
	}

	todos = doc.ExtractTodos()
	if todos[1].Checked {
		t.Error("Todo 1 should be unchecked after toggle")
	}
}

func TestASTDocument_ToggleTodo_InvalidIndex(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	doc, _ := ParseAST(content)

	err := doc.ToggleTodo(-1)
	if err == nil {
		t.Error("ToggleTodo with negative index should fail")
	}

	err = doc.ToggleTodo(5)
	if err == nil {
		t.Error("ToggleTodo with out of bounds index should fail")
	}
}

func TestASTDocument_ExtractHeadings(t *testing.T) {
	content := `# Title

Some text

## Section 1

- [ ] Task 1

### Subsection

- [ ] Task 2

## Section 2

- [ ] Task 3
`
	doc, _ := ParseAST(content)
	headings := doc.ExtractHeadings()

	if len(headings) != 4 {
		t.Fatalf("Expected 4 headings, got %d", len(headings))
	}

	// Check levels
	expectedLevels := []int{1, 2, 3, 2}
	for i, h := range headings {
		if h.Level != expectedLevels[i] {
			t.Errorf("Heading %d level = %d, want %d", i, h.Level, expectedLevels[i])
		}
	}

	// Check texts
	expectedTexts := []string{"Title", "Section 1", "Subsection", "Section 2"}
	for i, h := range headings {
		if h.Text != expectedTexts[i] {
			t.Errorf("Heading %d text = %q, want %q", i, h.Text, expectedTexts[i])
		}
	}
}

func TestASTDocument_MoveTodoToPosition(t *testing.T) {
	content := `# Todos

- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
`
	doc, _ := ParseAST(content)

	// Move task 3 before task 2
	err := doc.MoveTodoToPosition(2, 1, false)
	if err != nil {
		t.Fatalf("MoveTodoToPosition failed: %v", err)
	}

	todos := doc.ExtractTodos()
	// Order should now be: Task 1, Task 3, Task 2
	if todos[1].Text != "Task 3" {
		t.Errorf("After move, todo 1 = %q, want 'Task 3'", todos[1].Text)
	}
	if todos[2].Text != "Task 2" {
		t.Errorf("After move, todo 2 = %q, want 'Task 2'", todos[2].Text)
	}
}

func TestASTDocument_MoveTodoToPosition_InsertAfter(t *testing.T) {
	content := `# Todos

- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
`
	doc, _ := ParseAST(content)

	// Move task 1 after task 2
	err := doc.MoveTodoToPosition(0, 1, true)
	if err != nil {
		t.Fatalf("MoveTodoToPosition failed: %v", err)
	}

	todos := doc.ExtractTodos()
	// Order should now be: Task 2, Task 1, Task 3
	if todos[0].Text != "Task 2" {
		t.Errorf("After move, todo 0 = %q, want 'Task 2'", todos[0].Text)
	}
	if todos[1].Text != "Task 1" {
		t.Errorf("After move, todo 1 = %q, want 'Task 1'", todos[1].Text)
	}
}

func TestASTDocument_MoveTodoToPosition_SameIndex(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	doc, _ := ParseAST(content)

	// Move to same position - should be no-op
	err := doc.MoveTodoToPosition(0, 0, false)
	if err != nil {
		t.Errorf("MoveTodoToPosition(0, 0) should not fail: %v", err)
	}
}

func TestASTDocument_SwapTodos(t *testing.T) {
	content := `# Todos

- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
`
	doc, _ := ParseAST(content)

	err := doc.SwapTodos(0, 2)
	if err != nil {
		t.Fatalf("SwapTodos failed: %v", err)
	}

	todos := doc.ExtractTodos()
	if todos[0].Text != "Task 3" {
		t.Errorf("After swap, todo 0 = %q, want 'Task 3'", todos[0].Text)
	}
	if todos[2].Text != "Task 1" {
		t.Errorf("After swap, todo 2 = %q, want 'Task 1'", todos[2].Text)
	}
}

func TestASTDocument_SwapTodos_Adjacent(t *testing.T) {
	content := `# Todos

- [ ] Task 1
- [ ] Task 2
`
	doc, _ := ParseAST(content)

	err := doc.SwapTodos(0, 1)
	if err != nil {
		t.Fatalf("SwapTodos failed: %v", err)
	}

	todos := doc.ExtractTodos()
	if todos[0].Text != "Task 2" {
		t.Errorf("After swap, todo 0 = %q, want 'Task 2'", todos[0].Text)
	}
	if todos[1].Text != "Task 1" {
		t.Errorf("After swap, todo 1 = %q, want 'Task 1'", todos[1].Text)
	}
}

func TestASTDocument_IndentTodo(t *testing.T) {
	content := `# Todos

- [ ] Task 1
- [ ] Task 2
`
	doc, _ := ParseAST(content)

	err := doc.IndentTodo(1)
	if err != nil {
		t.Fatalf("IndentTodo failed: %v", err)
	}

	todos := doc.ExtractTodos()
	if todos[1].Depth != 1 {
		t.Errorf("After indent, depth = %d, want 1", todos[1].Depth)
	}
	if todos[1].ParentIndex != 0 {
		t.Errorf("After indent, parent index = %d, want 0", todos[1].ParentIndex)
	}
}

func TestASTDocument_IndentTodo_NoSibling(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	doc, _ := ParseAST(content)

	err := doc.IndentTodo(0)
	if err == nil {
		t.Error("IndentTodo on first item should fail")
	}
}

func TestASTDocument_OutdentTodo(t *testing.T) {
	content := `# Todos

- [ ] Task 1
  - [ ] Task 2
`
	doc, _ := ParseAST(content)

	todos := doc.ExtractTodos()
	if todos[1].Depth != 1 {
		t.Fatalf("Initial depth = %d, want 1", todos[1].Depth)
	}

	err := doc.OutdentTodo(1)
	if err != nil {
		t.Fatalf("OutdentTodo failed: %v", err)
	}

	todos = doc.ExtractTodos()
	if todos[1].Depth != 0 {
		t.Errorf("After outdent, depth = %d, want 0", todos[1].Depth)
	}
}

func TestASTDocument_OutdentTodo_TopLevel(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	doc, _ := ParseAST(content)

	err := doc.OutdentTodo(0)
	if err == nil {
		t.Error("OutdentTodo on top-level item should fail")
	}
}

// ==================== RebuildFileStructure test ====================

func TestRebuildFileStructure(t *testing.T) {
	content := `# Todos

- [ ] Task 1
- [ ] Task 2
`
	fm := ParseMarkdown(content)

	// Modify todos
	fm.Todos[0].Text = "Modified Task 1"
	fm.dirty = true

	// Rebuild
	RebuildFileStructure(fm)

	if !fm.dirty {
		t.Error("RebuildFileStructure should mark as dirty")
	}
}

// ==================== Additional parser.go tests ====================

func TestDeleteTodoItem(t *testing.T) {
	content := `# Todos

- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
`
	fm := ParseMarkdown(content)

	err := fm.DeleteTodoItem(1)
	if err != nil {
		t.Fatalf("DeleteTodoItem failed: %v", err)
	}

	if len(fm.Todos) != 2 {
		t.Errorf("Expected 2 todos after delete, got %d", len(fm.Todos))
	}

	// Task 2 should be gone
	for _, todo := range fm.Todos {
		if todo.Text == "Task 2" {
			t.Error("Task 2 should have been deleted")
		}
	}
}

func TestDeleteTodoItem_InvalidIndex(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	fm := ParseMarkdown(content)

	err := fm.DeleteTodoItem(-1)
	if err == nil {
		t.Error("DeleteTodoItem with negative index should fail")
	}

	err = fm.DeleteTodoItem(5)
	if err == nil {
		t.Error("DeleteTodoItem with out of bounds index should fail")
	}
}

func TestDeleteTodoItem_FirstItem(t *testing.T) {
	content := `# Todos

- [ ] Task 1
- [ ] Task 2
`
	fm := ParseMarkdown(content)

	err := fm.DeleteTodoItem(0)
	if err != nil {
		t.Fatalf("DeleteTodoItem(0) failed: %v", err)
	}

	if len(fm.Todos) != 1 {
		t.Errorf("Expected 1 todo after delete, got %d", len(fm.Todos))
	}

	if fm.Todos[0].Text != "Task 2" {
		t.Errorf("Remaining todo = %q, want 'Task 2'", fm.Todos[0].Text)
	}
}

func TestDeleteTodoItem_LastItem(t *testing.T) {
	content := `# Todos

- [ ] Task 1
- [ ] Task 2
`
	fm := ParseMarkdown(content)

	err := fm.DeleteTodoItem(1)
	if err != nil {
		t.Fatalf("DeleteTodoItem(1) failed: %v", err)
	}

	if len(fm.Todos) != 1 {
		t.Errorf("Expected 1 todo after delete, got %d", len(fm.Todos))
	}

	if fm.Todos[0].Text != "Task 1" {
		t.Errorf("Remaining todo = %q, want 'Task 1'", fm.Todos[0].Text)
	}
}

func TestMoveTodoItem(t *testing.T) {
	content := `# Todos

- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
`
	fm := ParseMarkdown(content)

	// Move task 1 to position 2
	err := fm.MoveTodoItem(0, 2)
	if err != nil {
		t.Fatalf("MoveTodoItem failed: %v", err)
	}

	if len(fm.Todos) != 3 {
		t.Fatalf("Expected 3 todos, got %d", len(fm.Todos))
	}

	// Order should now be: Task 2, Task 3, Task 1
	if fm.Todos[0].Text != "Task 2" {
		t.Errorf("Todo 0 = %q, want 'Task 2'", fm.Todos[0].Text)
	}
	if fm.Todos[2].Text != "Task 1" {
		t.Errorf("Todo 2 = %q, want 'Task 1'", fm.Todos[2].Text)
	}
}

func TestMoveTodoItem_SameIndex(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	fm := ParseMarkdown(content)

	err := fm.MoveTodoItem(0, 0)
	if err != nil {
		t.Errorf("MoveTodoItem to same index should be no-op: %v", err)
	}
}

func TestMoveTodoItem_InvalidIndex(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	fm := ParseMarkdown(content)

	err := fm.MoveTodoItem(-1, 0)
	if err == nil {
		t.Error("MoveTodoItem with negative fromIndex should fail")
	}

	err = fm.MoveTodoItem(0, 5)
	if err == nil {
		t.Error("MoveTodoItem with out of bounds toIndex should fail")
	}
}

// ==================== Additional tags.go tests ====================

func TestExtractTags(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "single tag",
			text:     "Task #urgent",
			expected: []string{"urgent"},
		},
		{
			name:     "multiple tags",
			text:     "Fix #backend #security #high-priority",
			expected: []string{"backend", "security", "high-priority"},
		},
		{
			name:     "tag with underscore",
			text:     "Work on #work_item",
			expected: []string{"work_item"},
		},
		{
			name:     "no tags",
			text:     "Simple task without tags",
			expected: []string{},
		},
		{
			name:     "tag at beginning",
			text:     "#important Buy groceries",
			expected: []string{"important"},
		},
		{
			name:     "duplicate tags",
			text:     "#urgent Task #urgent again",
			expected: []string{"urgent"},
		},
		{
			name:     "empty string",
			text:     "",
			expected: []string{},
		},
		{
			name:     "tag with numbers",
			text:     "Version #v2 update",
			expected: []string{"v2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTags(tt.text)
			if len(got) != len(tt.expected) {
				t.Errorf("ExtractTags(%q) = %v, want %v", tt.text, got, tt.expected)
				return
			}
			for i, tag := range got {
				if tag != tt.expected[i] {
					t.Errorf("ExtractTags(%q)[%d] = %q, want %q", tt.text, i, tag, tt.expected[i])
				}
			}
		})
	}
}

// ==================== Additional AST tests ====================

func TestASTDocument_DeleteTodo(t *testing.T) {
	content := `# Todos

- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
`
	doc, _ := ParseAST(content)

	err := doc.DeleteTodo(1)
	if err != nil {
		t.Fatalf("DeleteTodo failed: %v", err)
	}

	todos := doc.ExtractTodos()
	if len(todos) != 2 {
		t.Errorf("Expected 2 todos after delete, got %d", len(todos))
	}

	// Should have Task 1 and Task 3
	if todos[0].Text != "Task 1" {
		t.Errorf("Todo 0 = %q, want 'Task 1'", todos[0].Text)
	}
	if todos[1].Text != "Task 3" {
		t.Errorf("Todo 1 = %q, want 'Task 3'", todos[1].Text)
	}
}

func TestASTDocument_DeleteTodo_InvalidIndex(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	doc, _ := ParseAST(content)

	err := doc.DeleteTodo(-1)
	if err == nil {
		t.Error("DeleteTodo with negative index should fail")
	}

	err = doc.DeleteTodo(5)
	if err == nil {
		t.Error("DeleteTodo with out of bounds index should fail")
	}
}

func TestASTDocument_AddTodo(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	doc, _ := ParseAST(content)

	err := doc.AddTodo("New task", false)
	if err != nil {
		t.Fatalf("AddTodo failed: %v", err)
	}

	todos := doc.ExtractTodos()
	if len(todos) != 2 {
		t.Errorf("Expected 2 todos after add, got %d", len(todos))
	}

	// New task should be unchecked
	found := false
	for _, todo := range todos {
		if todo.Text == "New task" {
			found = true
			if todo.Checked {
				t.Error("New task should be unchecked")
			}
		}
	}
	if !found {
		t.Error("New task not found after AddTodo")
	}
}

func TestASTDocument_AddTodo_Checked(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	doc, _ := ParseAST(content)

	err := doc.AddTodo("Done task", true)
	if err != nil {
		t.Fatalf("AddTodo failed: %v", err)
	}

	todos := doc.ExtractTodos()
	found := false
	for _, todo := range todos {
		if todo.Text == "Done task" {
			found = true
			if !todo.Checked {
				t.Error("Done task should be checked")
			}
		}
	}
	if !found {
		t.Error("Done task not found after AddTodo")
	}
}

func TestASTDocument_UpdateTodoText(t *testing.T) {
	content := `# Todos

- [ ] Original task
`
	doc, _ := ParseAST(content)

	err := doc.UpdateTodoText(0, "Updated task")
	if err != nil {
		t.Fatalf("UpdateTodoText failed: %v", err)
	}

	todos := doc.ExtractTodos()
	if todos[0].Text != "Updated task" {
		t.Errorf("Todo text = %q, want 'Updated task'", todos[0].Text)
	}
}

func TestASTDocument_UpdateTodoText_InvalidIndex(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	doc, _ := ParseAST(content)

	err := doc.UpdateTodoText(-1, "Test")
	if err == nil {
		t.Error("UpdateTodoText with negative index should fail")
	}

	err = doc.UpdateTodoText(5, "Test")
	if err == nil {
		t.Error("UpdateTodoText with out of bounds index should fail")
	}
}

func TestASTDocument_MoveTodo(t *testing.T) {
	content := `# Todos

- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
`
	doc, _ := ParseAST(content)

	err := doc.MoveTodo(0, 2)
	if err != nil {
		t.Fatalf("MoveTodo failed: %v", err)
	}

	todos := doc.ExtractTodos()
	// After moving 0 to 2: Task 2, Task 3, Task 1
	if todos[0].Text != "Task 2" {
		t.Errorf("Todo 0 = %q, want 'Task 2'", todos[0].Text)
	}
	if todos[2].Text != "Task 1" {
		t.Errorf("Todo 2 = %q, want 'Task 1'", todos[2].Text)
	}
}

func TestASTDocument_MoveTodo_InvalidIndex(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	doc, _ := ParseAST(content)

	err := doc.MoveTodo(-1, 0)
	if err == nil {
		t.Error("MoveTodo with negative fromIndex should fail")
	}

	err = doc.MoveTodo(0, 5)
	if err == nil {
		t.Error("MoveTodo with out of bounds toIndex should fail")
	}
}

func TestASTDocument_InsertTodoAfter(t *testing.T) {
	content := `# Todos

- [ ] Task 1
- [ ] Task 3
`
	doc, _ := ParseAST(content)

	// Insert after position 0 (after Task 1)
	err := doc.InsertTodoAfter(0, "Task 2", false)
	if err != nil {
		t.Fatalf("InsertTodoAfter failed: %v", err)
	}

	todos := doc.ExtractTodos()
	if len(todos) != 3 {
		t.Errorf("Expected 3 todos, got %d", len(todos))
	}

	if todos[1].Text != "Task 2" {
		t.Errorf("Todo 1 = %q, want 'Task 2'", todos[1].Text)
	}
}

func TestASTDocument_InsertTodoAfter_Beginning(t *testing.T) {
	content := `# Todos

- [ ] Task 2
`
	doc, _ := ParseAST(content)

	// Insert after -1 means at beginning
	err := doc.InsertTodoAfter(-1, "Task 1", false)
	if err != nil {
		t.Fatalf("InsertTodoAfter(-1, ...) failed: %v", err)
	}

	todos := doc.ExtractTodos()
	if todos[0].Text != "Task 1" {
		t.Errorf("Todo 0 = %q, want 'Task 1'", todos[0].Text)
	}
}

func TestASTDocument_InsertTodoAfter_End(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	doc, _ := ParseAST(content)

	// Insert after last todo
	err := doc.InsertTodoAfter(0, "Task 2", false)
	if err != nil {
		t.Fatalf("InsertTodoAfter(0, ...) failed: %v", err)
	}

	todos := doc.ExtractTodos()
	if len(todos) != 2 {
		t.Errorf("Expected 2 todos, got %d", len(todos))
	}
	if todos[1].Text != "Task 2" {
		t.Errorf("Todo 1 = %q, want 'Task 2'", todos[1].Text)
	}
}

// ==================== AddTodoItem tests ====================

func TestAddTodoItem(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	fm := ParseMarkdown(content)

	fm.AddTodoItem("Task 2", false)

	if len(fm.Todos) != 2 {
		t.Errorf("Expected 2 todos, got %d", len(fm.Todos))
	}

	found := false
	for _, todo := range fm.Todos {
		if todo.Text == "Task 2" {
			found = true
			if todo.Checked {
				t.Error("New task should be unchecked")
			}
		}
	}
	if !found {
		t.Error("Task 2 not found")
	}
}

func TestAddTodoItem_Checked(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	fm := ParseMarkdown(content)

	fm.AddTodoItem("Done task", true)

	found := false
	for _, todo := range fm.Todos {
		if todo.Text == "Done task" {
			found = true
			if !todo.Checked {
				t.Error("Done task should be checked")
			}
		}
	}
	if !found {
		t.Error("Done task not found")
	}
}

// ==================== UpdateTodoItem tests ====================

func TestUpdateTodoItem(t *testing.T) {
	content := `# Todos

- [ ] Original text
`
	fm := ParseMarkdown(content)

	err := fm.UpdateTodoItem(0, "Updated text", false)
	if err != nil {
		t.Fatalf("UpdateTodoItem failed: %v", err)
	}

	if fm.Todos[0].Text != "Updated text" {
		t.Errorf("Todo text = %q, want 'Updated text'", fm.Todos[0].Text)
	}
}

func TestUpdateTodoItem_InvalidIndex(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	fm := ParseMarkdown(content)

	err := fm.UpdateTodoItem(-1, "Test", false)
	if err == nil {
		t.Error("UpdateTodoItem with negative index should fail")
	}

	err = fm.UpdateTodoItem(5, "Test", false)
	if err == nil {
		t.Error("UpdateTodoItem with out of bounds index should fail")
	}
}

// ==================== SerializeMarkdown tests ====================

func TestSerializeMarkdown(t *testing.T) {
	content := `# Todos

- [ ] Task 1
- [x] Task 2
`
	fm := ParseMarkdown(content)

	output := SerializeMarkdown(fm)
	if len(output) == 0 {
		t.Error("SerializeMarkdown returned empty string")
	}

	// Should contain our tasks
	if !strings.Contains(output, "Task 1") {
		t.Error("Output should contain 'Task 1'")
	}
	if !strings.Contains(output, "Task 2") {
		t.Error("Output should contain 'Task 2'")
	}

	// Should preserve checkbox state
	if !strings.Contains(output, "[ ]") {
		t.Error("Output should contain unchecked checkbox")
	}
	if !strings.Contains(output, "[x]") {
		t.Error("Output should contain checked checkbox")
	}
}

func TestSerializeMarkdown_WithHeadings(t *testing.T) {
	content := `# Main Title

- [ ] Task 1

## Section

- [ ] Task 2
`
	fm := ParseMarkdown(content)

	output := SerializeMarkdown(fm)

	if !strings.Contains(output, "# Main Title") {
		t.Error("Output should preserve main heading")
	}
	if !strings.Contains(output, "## Section") {
		t.Error("Output should preserve section heading")
	}
}

// ==================== Metadata parsing tests ====================

func TestParseMetadata_WithFrontmatter(t *testing.T) {
	content := `---
filter-done: true
max-visible: 25
---

# Todos

- [ ] Task 1
`
	metadata, contentWithoutMeta, err := ParseMetadata(content)
	if err != nil {
		t.Fatalf("ParseMetadata failed: %v", err)
	}

	if metadata == nil {
		t.Fatal("Metadata should not be nil")
	}

	filterDone := metadata.GetBool("filter-done", false)
	if !filterDone {
		t.Error("filter-done should be true")
	}

	maxVisible := metadata.GetInt("max-visible", 10)
	if maxVisible != 25 {
		t.Errorf("max-visible = %d, want 25", maxVisible)
	}

	// Content without meta should not have frontmatter
	if strings.Contains(contentWithoutMeta, "---") {
		t.Error("Content should not contain frontmatter delimiters")
	}
	if !strings.Contains(contentWithoutMeta, "# Todos") {
		t.Error("Content should still contain the heading")
	}
}

func TestParseMetadata_NoFrontmatter(t *testing.T) {
	content := `# Todos

- [ ] Task 1
`
	metadata, contentWithoutMeta, err := ParseMetadata(content)
	if err != nil {
		t.Fatalf("ParseMetadata failed: %v", err)
	}

	// With no frontmatter, metadata should be nil or empty
	if metadata != nil {
		// If metadata exists, check defaults work
		filterDone := metadata.GetBool("filter-done", false)
		if filterDone {
			t.Error("filter-done should default to false")
		}
	}

	// Content should be unchanged
	if contentWithoutMeta != content {
		t.Error("Content without frontmatter should be unchanged")
	}
}

func TestParseMetadata_AllOptions(t *testing.T) {
	content := `---
filter-done: true
show-headings: false
read-only: true
word-wrap: true
max-visible: 50
---

# Todos
`
	metadata, _, err := ParseMetadata(content)
	if err != nil {
		t.Fatalf("ParseMetadata failed: %v", err)
	}

	if !metadata.GetBool("filter-done", false) {
		t.Error("filter-done should be true")
	}
	if metadata.GetBool("show-headings", true) {
		t.Error("show-headings should be false")
	}
	if !metadata.GetBool("read-only", false) {
		t.Error("read-only should be true")
	}
	if !metadata.GetBool("word-wrap", false) {
		t.Error("word-wrap should be true")
	}
	if metadata.GetInt("max-visible", 10) != 50 {
		t.Errorf("max-visible = %d, want 50", metadata.GetInt("max-visible", 10))
	}
}

// ==================== UpdateTodoItem with checked state tests ====================

func TestUpdateTodoItem_ToggleChecked(t *testing.T) {
	content := `# Todos

- [ ] Unchecked task
- [x] Checked task
`
	fm := ParseMarkdown(content)

	// Update and check
	err := fm.UpdateTodoItem(0, "Unchecked task", true)
	if err != nil {
		t.Fatalf("UpdateTodoItem failed: %v", err)
	}

	if !fm.Todos[0].Checked {
		t.Error("Todo 0 should be checked after update")
	}

	// Update and uncheck
	err = fm.UpdateTodoItem(1, "Checked task", false)
	if err != nil {
		t.Fatalf("UpdateTodoItem failed: %v", err)
	}

	if fm.Todos[1].Checked {
		t.Error("Todo 1 should be unchecked after update")
	}
}
