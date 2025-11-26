package markdown

import (
	"strings"
	"testing"
)

func TestIndentTodo(t *testing.T) {
	input := `# Todos

- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
`
	doc, err := ParseAST(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Indent Task 2 under Task 1
	err = doc.IndentTodo(1)
	if err != nil {
		t.Fatalf("Failed to indent: %v", err)
	}

	// Re-extract todos and check depth
	todos := doc.ExtractTodos()
	if len(todos) != 3 {
		t.Fatalf("Expected 3 todos, got %d", len(todos))
	}

	if todos[0].Depth != 0 {
		t.Errorf("Task 1 depth: expected 0, got %d", todos[0].Depth)
	}
	if todos[1].Depth != 1 {
		t.Errorf("Task 2 depth: expected 1, got %d", todos[1].Depth)
	}
	if todos[2].Depth != 0 {
		t.Errorf("Task 3 depth: expected 0, got %d", todos[2].Depth)
	}

	// Serialize and verify output
	fm := &FileModel{ast: doc}
	output := SerializeMarkdown(fm)

	expectedParts := []string{
		"- [ ] Task 1",
		"  - [ ] Task 2",
		"- [ ] Task 3",
	}

	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Errorf("Expected output to contain %q, got:\n%s", part, output)
		}
	}
}

func TestIndentTodo_FirstItemFails(t *testing.T) {
	input := `# Todos

- [ ] Task 1
- [ ] Task 2
`
	doc, err := ParseAST(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Try to indent first item (should fail)
	err = doc.IndentTodo(0)
	if err == nil {
		t.Error("Expected error when indenting first item, got nil")
	}
}

func TestOutdentTodo(t *testing.T) {
	input := `# Todos

- [ ] Task 1
  - [ ] Task 2
- [ ] Task 3
`
	doc, err := ParseAST(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Verify initial state
	todos := doc.ExtractTodos()
	if todos[1].Depth != 1 {
		t.Fatalf("Task 2 initial depth: expected 1, got %d", todos[1].Depth)
	}

	// Outdent Task 2
	err = doc.OutdentTodo(1)
	if err != nil {
		t.Fatalf("Failed to outdent: %v", err)
	}

	// Re-extract todos and check depth
	todos = doc.ExtractTodos()
	if len(todos) != 3 {
		t.Fatalf("Expected 3 todos, got %d", len(todos))
	}

	// All tasks should now be at depth 0
	for i, todo := range todos {
		if todo.Depth != 0 {
			t.Errorf("Task %d depth: expected 0, got %d", i+1, todo.Depth)
		}
	}
}

func TestOutdentTodo_TopLevelFails(t *testing.T) {
	input := `# Todos

- [ ] Task 1
- [ ] Task 2
`
	doc, err := ParseAST(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Try to outdent top-level item (should fail)
	err = doc.OutdentTodo(0)
	if err == nil {
		t.Error("Expected error when outdenting top-level item, got nil")
	}
}

func TestIndentOutdentRoundTrip(t *testing.T) {
	input := `# Todos

- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
`
	doc, err := ParseAST(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Indent Task 2
	err = doc.IndentTodo(1)
	if err != nil {
		t.Fatalf("Failed to indent: %v", err)
	}

	// Verify Task 2 is nested
	todos := doc.ExtractTodos()
	if todos[1].Depth != 1 {
		t.Fatalf("Task 2 depth after indent: expected 1, got %d", todos[1].Depth)
	}

	// Outdent Task 2
	err = doc.OutdentTodo(1)
	if err != nil {
		t.Fatalf("Failed to outdent: %v", err)
	}

	// Verify Task 2 is back at top level
	todos = doc.ExtractTodos()
	if todos[1].Depth != 0 {
		t.Errorf("Task 2 depth after outdent: expected 0, got %d", todos[1].Depth)
	}
}

func TestDeleteTodoPromotesChildren(t *testing.T) {
	input := `# Todos

- [ ] Parent
  - [ ] Child 1
  - [ ] Child 2
- [ ] Sibling
`
	doc, err := ParseAST(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Verify initial state
	todos := doc.ExtractTodos()
	if len(todos) != 4 {
		t.Fatalf("Expected 4 todos, got %d", len(todos))
	}
	if todos[1].Depth != 1 || todos[2].Depth != 1 {
		t.Fatalf("Children should be at depth 1")
	}

	// Delete Parent - children should be promoted
	err = doc.DeleteTodo(0)
	if err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	// Re-extract todos
	todos = doc.ExtractTodos()
	if len(todos) != 3 {
		t.Fatalf("Expected 3 todos after delete, got %d", len(todos))
	}

	// All remaining todos should be at depth 0
	for i, todo := range todos {
		if todo.Depth != 0 {
			t.Errorf("Todo %d (%s) depth: expected 0, got %d", i, todo.Text, todo.Depth)
		}
	}

	// Verify order: Child 1, Child 2, Sibling
	expectedTexts := []string{"Child 1", "Child 2", "Sibling"}
	for i, expected := range expectedTexts {
		if todos[i].Text != expected {
			t.Errorf("Todo %d: expected %q, got %q", i, expected, todos[i].Text)
		}
	}
}

func TestDeleteTodoNoChildren(t *testing.T) {
	input := `# Todos

- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
`
	doc, err := ParseAST(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Delete Task 2
	err = doc.DeleteTodo(1)
	if err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	// Re-extract todos
	todos := doc.ExtractTodos()
	if len(todos) != 2 {
		t.Fatalf("Expected 2 todos after delete, got %d", len(todos))
	}

	// Verify remaining: Task 1, Task 3
	if todos[0].Text != "Task 1" || todos[1].Text != "Task 3" {
		t.Errorf("Unexpected remaining todos: %v", todos)
	}
}

func TestInsertTodoAfterNestedItem(t *testing.T) {
	input := `# Todos

- [ ] Parent
  - [ ] Child 1
  - [ ] Child 2
- [ ] Sibling
`
	doc, err := ParseAST(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Insert after Child 1 (index 1) - should be at same depth
	err = doc.InsertTodoAfter(1, "New Child", false)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Re-extract todos
	todos := doc.ExtractTodos()
	if len(todos) != 5 {
		t.Fatalf("Expected 5 todos, got %d", len(todos))
	}

	// Verify order and depths
	expected := []struct {
		text  string
		depth int
	}{
		{"Parent", 0},
		{"Child 1", 1},
		{"New Child", 1},
		{"Child 2", 1},
		{"Sibling", 0},
	}

	for i, exp := range expected {
		if todos[i].Text != exp.text {
			t.Errorf("Todo %d: expected text %q, got %q", i, exp.text, todos[i].Text)
		}
		if todos[i].Depth != exp.depth {
			t.Errorf("Todo %d (%s): expected depth %d, got %d", i, exp.text, exp.depth, todos[i].Depth)
		}
	}
}

func TestInsertTodoAfterTopLevel(t *testing.T) {
	input := `# Todos

- [ ] Task 1
  - [ ] Child A
- [ ] Task 2
`
	doc, err := ParseAST(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Insert after Task 1 (index 0) - should be at top level, after Task 1's children
	err = doc.InsertTodoAfter(0, "New Task", false)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Re-extract todos
	todos := doc.ExtractTodos()
	if len(todos) != 4 {
		t.Fatalf("Expected 4 todos, got %d", len(todos))
	}

	// Verify order and depths
	// Note: New Task comes after Task 1's entire block (including children)
	// because Task 1's list item contains the nested list
	expected := []struct {
		text  string
		depth int
	}{
		{"Task 1", 0},
		{"Child A", 1},
		{"New Task", 0},
		{"Task 2", 0},
	}

	for i, exp := range expected {
		if todos[i].Text != exp.text {
			t.Errorf("Todo %d: expected text %q, got %q", i, exp.text, todos[i].Text)
		}
		if todos[i].Depth != exp.depth {
			t.Errorf("Todo %d (%s): expected depth %d, got %d", i, exp.text, exp.depth, todos[i].Depth)
		}
	}
}

func TestIndentIntoExistingNestedList(t *testing.T) {
	input := `# Todos

- [ ] Task 1
  - [ ] Subtask A
- [ ] Task 2
`
	doc, err := ParseAST(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Indent Task 2 into Task 1's nested list
	err = doc.IndentTodo(2)
	if err != nil {
		t.Fatalf("Failed to indent: %v", err)
	}

	// Re-extract todos and check
	todos := doc.ExtractTodos()
	if len(todos) != 3 {
		t.Fatalf("Expected 3 todos, got %d", len(todos))
	}

	// Task 2 should now be at depth 1
	if todos[2].Depth != 1 {
		t.Errorf("Task 2 depth: expected 1, got %d", todos[2].Depth)
	}

	// Serialize and verify structure
	fm := &FileModel{ast: doc}
	output := SerializeMarkdown(fm)

	// Task 2 should be indented
	if !strings.Contains(output, "  - [ ] Task 2") {
		t.Errorf("Expected Task 2 to be indented, got:\n%s", output)
	}
}
