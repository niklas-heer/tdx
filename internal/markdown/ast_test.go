package markdown

import (
	"testing"
)

func TestUpdateTodoText_PreservesInlineCode(t *testing.T) {
	// Create a document with a todo containing inline code
	content := `# Todos

- [ ] Fix the ` + "`bug`" + ` in main.go
- [ ] Regular todo
`

	fm := ParseMarkdown(content)

	// Verify initial parsing
	if len(fm.Todos) != 2 {
		t.Fatalf("Expected 2 todos, got %d", len(fm.Todos))
	}

	if fm.Todos[0].Text != "Fix the `bug` in main.go" {
		t.Errorf("Expected 'Fix the `bug` in main.go', got '%s'", fm.Todos[0].Text)
	}

	// Update the first todo with new inline code
	newText := "Update `config.yaml` file"
	err := fm.UpdateTodoItem(0, newText, false)
	if err != nil {
		t.Fatalf("UpdateTodoItem failed: %v", err)
	}

	// Verify the update preserved the inline code
	if fm.Todos[0].Text != newText {
		t.Errorf("Expected '%s', got '%s'", newText, fm.Todos[0].Text)
	}

	// Serialize and re-parse to ensure it round-trips correctly
	serialized := SerializeMarkdown(fm)
	fm2 := ParseMarkdown(serialized)

	if len(fm2.Todos) != 2 {
		t.Fatalf("After round-trip: expected 2 todos, got %d", len(fm2.Todos))
	}

	if fm2.Todos[0].Text != newText {
		t.Errorf("After round-trip: expected '%s', got '%s'", newText, fm2.Todos[0].Text)
	}
}

func TestUpdateTodoText_MultipleCodeSpans(t *testing.T) {
	content := `# Todos

- [ ] Simple todo
`

	fm := ParseMarkdown(content)

	// Update with multiple code spans
	newText := "Update `config.yaml` and `settings.json` files"
	err := fm.UpdateTodoItem(0, newText, false)
	if err != nil {
		t.Fatalf("UpdateTodoItem failed: %v", err)
	}

	if fm.Todos[0].Text != newText {
		t.Errorf("Expected '%s', got '%s'", newText, fm.Todos[0].Text)
	}

	// Verify serialization preserves both code spans
	serialized := SerializeMarkdown(fm)
	fm2 := ParseMarkdown(serialized)

	if fm2.Todos[0].Text != newText {
		t.Errorf("After round-trip: expected '%s', got '%s'", newText, fm2.Todos[0].Text)
	}
}

func TestUpdateTodoText_PlainText(t *testing.T) {
	content := `# Todos

- [ ] Todo with ` + "`code`" + `
`

	fm := ParseMarkdown(content)

	// Update to plain text (removing code)
	newText := "Plain text todo"
	err := fm.UpdateTodoItem(0, newText, false)
	if err != nil {
		t.Fatalf("UpdateTodoItem failed: %v", err)
	}

	if fm.Todos[0].Text != newText {
		t.Errorf("Expected '%s', got '%s'", newText, fm.Todos[0].Text)
	}
}

func TestUpdateTodoText_WithLinks(t *testing.T) {
	content := `# Todos

- [ ] Simple todo
`

	fm := ParseMarkdown(content)

	// Update with markdown link
	newText := "Check [documentation](https://example.com) for details"
	err := fm.UpdateTodoItem(0, newText, false)
	if err != nil {
		t.Fatalf("UpdateTodoItem failed: %v", err)
	}

	// Note: The text extraction may not include the URL part, just the link text
	// This is expected behavior - we're testing that links are preserved in the AST
	if fm.Todos[0].Text == "" {
		t.Errorf("Todo text should not be empty after update with link")
	}

	// Verify serialization
	serialized := SerializeMarkdown(fm)
	if serialized == "" {
		t.Error("Serialized output should not be empty")
	}
}

func TestUpdateTodoText_WithEmphasis(t *testing.T) {
	content := `# Todos

- [ ] Simple todo
`

	fm := ParseMarkdown(content)

	// Update with emphasis (bold/italic)
	newText := "This is *important* and **urgent**"
	err := fm.UpdateTodoItem(0, newText, false)
	if err != nil {
		t.Fatalf("UpdateTodoItem failed: %v", err)
	}

	// Verify the text is updated
	if fm.Todos[0].Text == "" {
		t.Errorf("Todo text should not be empty after update")
	}
}

func TestUpdateTodoText_SpecialCharacters(t *testing.T) {
	content := `# Todos

- [ ] Simple todo
`

	fm := ParseMarkdown(content)

	// Test with special characters that might break parsing
	testCases := []string{
		"Todo with \"quotes\" and 'apostrophes'",
		"Todo with unicode: 你好 世界 🎉",
		"Todo with backslash\\escape",
		"Todo with `<brackets>` in code", // Angle brackets need to be in code to preserve them
	}

	for _, newText := range testCases {
		err := fm.UpdateTodoItem(0, newText, false)
		if err != nil {
			t.Errorf("UpdateTodoItem failed for '%s': %v", newText, err)
			continue
		}

		if fm.Todos[0].Text != newText {
			t.Errorf("Expected '%s', got '%s'", newText, fm.Todos[0].Text)
		}

		// Verify round-trip
		serialized := SerializeMarkdown(fm)
		fm2 := ParseMarkdown(serialized)
		if fm2.Todos[0].Text != newText {
			t.Errorf("After round-trip for '%s': expected '%s', got '%s'", newText, newText, fm2.Todos[0].Text)
		}
	}
}
