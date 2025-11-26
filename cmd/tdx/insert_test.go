package main

import (
	"os"
	"strings"
	"testing"
)

// TestTUI_InsertAfterCursor tests that 'n' inserts after cursor position
func TestTUI_InsertAfterCursor(t *testing.T) {
	file := tempTestFile(t)

	// Add initial todos
	runCLI(t, file, "add", "First")
	runCLI(t, file, "add", "Second")
	runCLI(t, file, "add", "Third")

	// Cursor starts at position 0 (First)
	// Press 'n' to insert after cursor, type "Inserted", press enter
	runPiped(t, file, "nInserted\r")

	todos := getTodos(t, file)
	if len(todos) != 4 {
		t.Fatalf("Expected 4 todos, got %d", len(todos))
	}

	// Should be: First, Inserted, Second, Third
	expected := []string{"First", "Inserted", "Second", "Third"}
	for i, exp := range expected {
		if !strings.Contains(todos[i], exp) {
			t.Errorf("Todo %d: expected to contain %q, got %q", i, exp, todos[i])
		}
	}
}

// TestTUI_InsertAfterCursor_Middle tests inserting in the middle of the list
func TestTUI_InsertAfterCursor_Middle(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "First")
	runCLI(t, file, "add", "Second")
	runCLI(t, file, "add", "Third")

	// Move to Second (index 1), then insert after it
	runPiped(t, file, "jnMiddle\r")

	todos := getTodos(t, file)
	if len(todos) != 4 {
		t.Fatalf("Expected 4 todos, got %d", len(todos))
	}

	// Should be: First, Second, Middle, Third
	expected := []string{"First", "Second", "Middle", "Third"}
	for i, exp := range expected {
		if !strings.Contains(todos[i], exp) {
			t.Errorf("Todo %d: expected to contain %q, got %q", i, exp, todos[i])
		}
	}
}

// TestTUI_InsertAfterCursor_End tests inserting after the last item
func TestTUI_InsertAfterCursor_End(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "First")
	runCLI(t, file, "add", "Second")

	// Move to last item (Second at index 1), then insert after it
	runPiped(t, file, "jnThird\r")

	todos := getTodos(t, file)
	if len(todos) != 3 {
		t.Fatalf("Expected 3 todos, got %d", len(todos))
	}

	// Should be: First, Second, Third
	expected := []string{"First", "Second", "Third"}
	for i, exp := range expected {
		if !strings.Contains(todos[i], exp) {
			t.Errorf("Todo %d: expected to contain %q, got %q", i, exp, todos[i])
		}
	}
}

// TestTUI_AppendToEnd tests that 'N' appends to end of file
func TestTUI_AppendToEnd(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "First")
	runCLI(t, file, "add", "Second")
	runCLI(t, file, "add", "Third")

	// Cursor at position 0 (First)
	// Press 'N' to append to end, type "Last", press enter
	runPiped(t, file, "NLast\r")

	todos := getTodos(t, file)
	if len(todos) != 4 {
		t.Fatalf("Expected 4 todos, got %d", len(todos))
	}

	// Should be: First, Second, Third, Last
	expected := []string{"First", "Second", "Third", "Last"}
	for i, exp := range expected {
		if !strings.Contains(todos[i], exp) {
			t.Errorf("Todo %d: expected to contain %q, got %q", i, exp, todos[i])
		}
	}
}

// TestTUI_AppendToEnd_FromMiddle tests that 'N' appends to end regardless of cursor
func TestTUI_AppendToEnd_FromMiddle(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "First")
	runCLI(t, file, "add", "Second")
	runCLI(t, file, "add", "Third")

	// Move to middle (Second at index 1), then append to end with N
	runPiped(t, file, "jNLast\r")

	todos := getTodos(t, file)
	if len(todos) != 4 {
		t.Fatalf("Expected 4 todos, got %d", len(todos))
	}

	// Should be: First, Second, Third, Last (appended at end, not after cursor)
	expected := []string{"First", "Second", "Third", "Last"}
	for i, exp := range expected {
		if !strings.Contains(todos[i], exp) {
			t.Errorf("Todo %d: expected to contain %q, got %q", i, exp, todos[i])
		}
	}
}

// TestTUI_InsertEmptyFile tests that 'n' works on empty file
func TestTUI_InsertEmptyFile(t *testing.T) {
	file := tempTestFile(t)

	// File is empty, press 'n' to add first todo
	runPiped(t, file, "nFirst\r")

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Fatalf("Expected 1 todo, got %d", len(todos))
	}

	if !strings.Contains(todos[0], "First") {
		t.Errorf("Expected todo to contain 'First', got %q", todos[0])
	}
}

// TestTUI_AppendEmptyFile tests that 'N' works on empty file
func TestTUI_AppendEmptyFile(t *testing.T) {
	file := tempTestFile(t)

	// File is empty, press 'N' to add first todo
	runPiped(t, file, "NFirst\r")

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Fatalf("Expected 1 todo, got %d", len(todos))
	}

	if !strings.Contains(todos[0], "First") {
		t.Errorf("Expected todo to contain 'First', got %q", todos[0])
	}
}

// TestTUI_CursorAfterInsert tests that cursor moves to inserted item
func TestTUI_CursorAfterInsert(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "First")
	runCLI(t, file, "add", "Second")
	runCLI(t, file, "add", "Third")

	// Insert after First, then immediately edit (should edit the inserted item)
	// n -> insert "New" -> enter -> e -> change to "Edited" -> enter
	runPiped(t, file, "nNew\reEdited\r")

	todos := getTodos(t, file)
	if len(todos) != 4 {
		t.Fatalf("Expected 4 todos, got %d", len(todos))
	}

	// Should be: First, Edited (was "New"), Second, Third
	if !strings.Contains(todos[1], "Edited") {
		t.Errorf("Expected second todo to be 'Edited', got %q", todos[1])
	}
}

// TestInsertTodoAfter_AST tests the AST-level InsertTodoAfter function
func TestInsertTodoAfter_AST(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Todos

- [ ] First
- [ ] Second
- [ ] Third
`
	_ = os.WriteFile(file, []byte(initial), 0644)

	// Use CLI to verify markdown structure after insert
	runPiped(t, file, "nInserted\r")

	content, _ := os.ReadFile(file)
	result := string(content)

	// Verify structure: should have "Inserted" after "First"
	firstIdx := strings.Index(result, "First")
	insertedIdx := strings.Index(result, "Inserted")
	secondIdx := strings.Index(result, "Second")

	if firstIdx == -1 || insertedIdx == -1 || secondIdx == -1 {
		t.Fatalf("Missing expected content in result:\n%s", result)
	}

	if firstIdx >= insertedIdx || insertedIdx >= secondIdx {
		t.Errorf("Wrong order: First(%d) should come before Inserted(%d) before Second(%d)\nContent:\n%s",
			firstIdx, insertedIdx, secondIdx, result)
	}
}
