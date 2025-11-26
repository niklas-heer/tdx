package markdown

import (
	"testing"
)

func TestExtractTodos_NestedDepth(t *testing.T) {
	content := `# Test

- [ ] Task 1
- [ ] Task 2
  - [x] Subtask a
  - [ ] Subtask b
- [ ] Task 3
`
	fm := ParseMarkdown(content)
	
	if len(fm.Todos) != 5 {
		t.Fatalf("Expected 5 todos, got %d", len(fm.Todos))
	}
	
	// Check Task 1
	if fm.Todos[0].Text != "Task 1" || fm.Todos[0].Depth != 0 || fm.Todos[0].ParentIndex != -1 {
		t.Errorf("Task 1: got text=%q depth=%d parent=%d", fm.Todos[0].Text, fm.Todos[0].Depth, fm.Todos[0].ParentIndex)
	}
	
	// Check Task 2
	if fm.Todos[1].Text != "Task 2" || fm.Todos[1].Depth != 0 || fm.Todos[1].ParentIndex != -1 {
		t.Errorf("Task 2: got text=%q depth=%d parent=%d", fm.Todos[1].Text, fm.Todos[1].Depth, fm.Todos[1].ParentIndex)
	}
	
	// Check Subtask a (child of Task 2)
	if fm.Todos[2].Text != "Subtask a" || fm.Todos[2].Depth != 1 || fm.Todos[2].ParentIndex != 1 {
		t.Errorf("Subtask a: got text=%q depth=%d parent=%d", fm.Todos[2].Text, fm.Todos[2].Depth, fm.Todos[2].ParentIndex)
	}
	
	// Check Subtask b (child of Task 2)
	if fm.Todos[3].Text != "Subtask b" || fm.Todos[3].Depth != 1 || fm.Todos[3].ParentIndex != 1 {
		t.Errorf("Subtask b: got text=%q depth=%d parent=%d", fm.Todos[3].Text, fm.Todos[3].Depth, fm.Todos[3].ParentIndex)
	}
	
	// Check Task 3
	if fm.Todos[4].Text != "Task 3" || fm.Todos[4].Depth != 0 || fm.Todos[4].ParentIndex != -1 {
		t.Errorf("Task 3: got text=%q depth=%d parent=%d", fm.Todos[4].Text, fm.Todos[4].Depth, fm.Todos[4].ParentIndex)
	}
	
	t.Logf("All todos extracted correctly with nesting info")
}
