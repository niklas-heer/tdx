package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var testBinary string

func TestMain(m *testing.M) {
	// Build the binary for testing
	tmpDir, err := os.MkdirTemp("", "tdx-test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	testBinary = filepath.Join(tmpDir, "tdx")
	cmd := exec.Command("go", "build", "-o", testBinary, ".")
	if err := cmd.Run(); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

// Helper to run CLI command
func runCLI(t *testing.T, file string, args ...string) string {
	cmdArgs := append([]string{file}, args...)
	cmd := exec.Command(testBinary, cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Some commands may fail, that's okay
	}
	return strings.TrimSpace(string(out))
}

// Helper to run piped input to TUI
func runPiped(t *testing.T, file string, input string) string {
	cmd := exec.Command(testBinary, file)
	cmd.Stdin = strings.NewReader(input)
	out, _ := cmd.CombinedOutput()
	return strings.TrimSpace(string(out))
}

// Helper to read test file
func readTestFile(t *testing.T, file string) string {
	content, err := os.ReadFile(file)
	if err != nil {
		return ""
	}
	return string(content)
}

// Helper to get todos from file
func getTodos(t *testing.T, file string) []string {
	content := readTestFile(t, file)
	var todos []string
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "- [ ] ") || strings.HasPrefix(line, "- [x] ") {
			todos = append(todos, line)
		}
	}
	return todos
}

// Helper to create temp test file
func tempTestFile(t *testing.T) string {
	f, err := os.CreateTemp("", "tdx-test-*.md")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

func TestCLI_AddTodo(t *testing.T) {
	file := tempTestFile(t)

	output := runCLI(t, file, "add", "First todo")
	if !strings.Contains(output, "Added: First todo") {
		t.Errorf("Expected 'Added: First todo', got: %s", output)
	}

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Errorf("Expected 1 todo, got %d", len(todos))
	}
	if todos[0] != "- [ ] First todo" {
		t.Errorf("Expected '- [ ] First todo', got: %s", todos[0])
	}
}

func TestCLI_ListTodos(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Todo 1")
	runCLI(t, file, "add", "Todo 2")

	output := runCLI(t, file, "list")
	if !strings.Contains(output, "1.") || !strings.Contains(output, "Todo 1") {
		t.Errorf("Expected todo 1 in list, got: %s", output)
	}
	if !strings.Contains(output, "2.") || !strings.Contains(output, "Todo 2") {
		t.Errorf("Expected todo 2 in list, got: %s", output)
	}
}

func TestCLI_ToggleTodo(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Toggle me")

	todos := getTodos(t, file)
	if todos[0] != "- [ ] Toggle me" {
		t.Errorf("Expected unchecked todo, got: %s", todos[0])
	}

	runCLI(t, file, "toggle", "1")

	todos = getTodos(t, file)
	if todos[0] != "- [x] Toggle me" {
		t.Errorf("Expected checked todo, got: %s", todos[0])
	}

	// Toggle back
	runCLI(t, file, "toggle", "1")

	todos = getTodos(t, file)
	if todos[0] != "- [ ] Toggle me" {
		t.Errorf("Expected unchecked todo again, got: %s", todos[0])
	}
}

func TestCLI_EditTodo(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Original text")
	runCLI(t, file, "edit", "1", "Edited text")

	todos := getTodos(t, file)
	if todos[0] != "- [ ] Edited text" {
		t.Errorf("Expected '- [ ] Edited text', got: %s", todos[0])
	}
}

func TestCLI_DeleteTodo(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Keep me")
	runCLI(t, file, "add", "Delete me")
	runCLI(t, file, "add", "Keep me too")

	runCLI(t, file, "delete", "2")

	todos := getTodos(t, file)
	if len(todos) != 2 {
		t.Errorf("Expected 2 todos, got %d", len(todos))
	}
	if todos[0] != "- [ ] Keep me" {
		t.Errorf("Expected '- [ ] Keep me', got: %s", todos[0])
	}
	if todos[1] != "- [ ] Keep me too" {
		t.Errorf("Expected '- [ ] Keep me too', got: %s", todos[1])
	}
}

func TestCLI_Help(t *testing.T) {
	cmd := exec.Command(testBinary, "help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	output := string(out)
	if !strings.Contains(output, "tdx") {
		t.Errorf("Expected help to contain 'tdx', got: %s", output)
	}
	if !strings.Contains(output, "Usage:") {
		t.Errorf("Expected help to contain 'Usage:', got: %s", output)
	}
}

func TestTUI_CreateTodo(t *testing.T) {
	file := tempTestFile(t)

	runPiped(t, file, "nNew TUI todo\r")

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Errorf("Expected 1 todo, got %d", len(todos))
	}
	if todos[0] != "- [ ] New TUI todo" {
		t.Errorf("Expected '- [ ] New TUI todo', got: %s", todos[0])
	}
}

func TestTUI_CreateMultipleTodos(t *testing.T) {
	file := tempTestFile(t)

	runPiped(t, file, "nFirst\rnSecond\rnThird\r")

	todos := getTodos(t, file)
	if len(todos) != 3 {
		t.Errorf("Expected 3 todos, got %d", len(todos))
	}
	if todos[0] != "- [ ] First" {
		t.Errorf("Expected '- [ ] First', got: %s", todos[0])
	}
	if todos[1] != "- [ ] Second" {
		t.Errorf("Expected '- [ ] Second', got: %s", todos[1])
	}
	if todos[2] != "- [ ] Third" {
		t.Errorf("Expected '- [ ] Third', got: %s", todos[2])
	}
}

func TestTUI_ToggleWithSpace(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Toggle via space")
	runPiped(t, file, " ")

	todos := getTodos(t, file)
	if todos[0] != "- [x] Toggle via space" {
		t.Errorf("Expected checked todo, got: %s", todos[0])
	}
}

func TestTUI_ToggleWithEnter(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Toggle via enter")
	runPiped(t, file, "\r")

	todos := getTodos(t, file)
	if todos[0] != "- [x] Toggle via enter" {
		t.Errorf("Expected checked todo, got: %s", todos[0])
	}
}

func TestTUI_Delete(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Delete me")
	runCLI(t, file, "add", "Keep me")

	runPiped(t, file, "d")

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Errorf("Expected 1 todo, got %d", len(todos))
	}
	if todos[0] != "- [ ] Keep me" {
		t.Errorf("Expected '- [ ] Keep me', got: %s", todos[0])
	}
}

func TestTUI_NavigateAndToggle(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "First")
	runCLI(t, file, "add", "Second")
	runCLI(t, file, "add", "Third")

	// Navigate down once and toggle
	runPiped(t, file, "j ")

	todos := getTodos(t, file)
	if todos[0] != "- [ ] First" {
		t.Errorf("Expected first unchecked, got: %s", todos[0])
	}
	if todos[1] != "- [x] Second" {
		t.Errorf("Expected second checked, got: %s", todos[1])
	}
	if todos[2] != "- [ ] Third" {
		t.Errorf("Expected third unchecked, got: %s", todos[2])
	}
}

func TestTUI_VimJump(t *testing.T) {
	file := tempTestFile(t)

	// Add 10 todos
	for i := 1; i <= 10; i++ {
		runCLI(t, file, "add", "Todo "+string(rune('0'+i)))
	}

	// Navigate down 5 and toggle
	runPiped(t, file, "5j ")

	todos := getTodos(t, file)
	// Index 5 (6th todo) should be toggled
	if !strings.Contains(todos[5], "[x]") {
		t.Errorf("Expected 6th todo to be checked, got: %s", todos[5])
	}
}

func TestTUI_EditWithBackspace(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Original")

	// Press 'e' to edit, backspaces to clear, type new text
	backspaces := strings.Repeat("\x7f", 8) // 8 backspaces
	runPiped(t, file, "e"+backspaces+"Edited\r")

	todos := getTodos(t, file)
	if todos[0] != "- [ ] Edited" {
		t.Errorf("Expected '- [ ] Edited', got: %s", todos[0])
	}
}

func TestTUI_Undo(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "First")
	runCLI(t, file, "add", "Second")

	// Delete first, then undo
	runPiped(t, file, "du")

	todos := getTodos(t, file)
	if len(todos) != 2 {
		t.Errorf("Expected 2 todos after undo, got %d", len(todos))
	}
	if todos[0] != "- [ ] First" {
		t.Errorf("Expected '- [ ] First', got: %s", todos[0])
	}
}

func TestTUI_EmptyInput(t *testing.T) {
	file := tempTestFile(t)

	// Just pressing enter with no text should not create todo
	runPiped(t, file, "n\r\r")

	todos := getTodos(t, file)
	if len(todos) != 0 {
		t.Errorf("Expected 0 todos for empty input, got %d", len(todos))
	}
}

func TestTUI_HelpMenu(t *testing.T) {
	file := tempTestFile(t)
	runCLI(t, file, "add", "Test")

	output := runPiped(t, file, "?")
	if !strings.Contains(output, "NAVIGATION") {
		t.Errorf("Expected help to contain 'NAVIGATION', got: %s", output)
	}
	if !strings.Contains(output, "EDITING") {
		t.Errorf("Expected help to contain 'EDITING', got: %s", output)
	}
}

func TestEdgeCase_SpecialCharacters(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Todo with special chars: @#$%^&*()")

	todos := getTodos(t, file)
	if !strings.Contains(todos[0], "special chars") {
		t.Errorf("Expected special chars to be preserved, got: %s", todos[0])
	}
}

func TestEdgeCase_LongText(t *testing.T) {
	file := tempTestFile(t)

	longText := strings.Repeat("A", 200)
	runCLI(t, file, "add", longText)

	todos := getTodos(t, file)
	if !strings.Contains(todos[0], longText) {
		t.Errorf("Expected long text to be preserved")
	}
}

func TestEdgeCase_MultipleToggles(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Toggle multiple times")

	// Toggle 3 times
	runPiped(t, file, "   ")

	todos := getTodos(t, file)
	// Should end up checked (odd number of toggles)
	if todos[0] != "- [x] Toggle multiple times" {
		t.Errorf("Expected checked after 3 toggles, got: %s", todos[0])
	}
}

func TestEdgeCase_PreserveFileHeader(t *testing.T) {
	// Use a non-existent file so it gets created fresh with header
	file := filepath.Join(os.TempDir(), "tdx-header-test.md")
	os.Remove(file) // Ensure it doesn't exist
	t.Cleanup(func() { os.Remove(file) })

	runCLI(t, file, "add", "Test")

	content := readTestFile(t, file)
	if !strings.HasPrefix(content, "# Todos") {
		t.Errorf("Expected file to start with '# Todos', got: %s", content)
	}
}

func TestEdgeCase_DeleteOnEmptyList(t *testing.T) {
	file := tempTestFile(t)

	// Should not crash
	runPiped(t, file, "d")

	todos := getTodos(t, file)
	if len(todos) != 0 {
		t.Errorf("Expected 0 todos, got %d", len(todos))
	}
}

func TestEdgeCase_NavigationOnEmptyList(t *testing.T) {
	file := tempTestFile(t)

	// Should not crash
	runPiped(t, file, "jjkk")

	todos := getTodos(t, file)
	if len(todos) != 0 {
		t.Errorf("Expected 0 todos, got %d", len(todos))
	}
}
