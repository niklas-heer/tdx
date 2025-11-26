package markdown

import (
	"strings"
	"testing"
)

func TestSerializeAST_NestedLists(t *testing.T) {
	content := `# Test

- [ ] Task 1
- [ ] Task 2
  - [x] Subtask a
  - [ ] Subtask b
- [ ] Task 3
`
	doc, err := ParseAST(content)
	if err != nil {
		t.Fatalf("ParseAST failed: %v", err)
	}
	
	output := SerializeAST(doc)
	
	// Check that subtasks are indented with 2 spaces
	if !strings.Contains(output, "  - [x] Subtask a") {
		t.Errorf("Expected indented '  - [x] Subtask a', got:\n%s", output)
	}
	if !strings.Contains(output, "  - [ ] Subtask b") {
		t.Errorf("Expected indented '  - [ ] Subtask b', got:\n%s", output)
	}
	
	// Check that top-level tasks are NOT indented
	if strings.Contains(output, "  - [ ] Task 1") {
		t.Errorf("Task 1 should not be indented, got:\n%s", output)
	}
	if strings.Contains(output, "  - [ ] Task 3") {
		t.Errorf("Task 3 should not be indented, got:\n%s", output)
	}
	
	t.Logf("Serialized output:\n%s", output)
}
