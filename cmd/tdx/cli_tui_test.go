package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/niklas-heer/tdx/internal/cmd"
	"github.com/niklas-heer/tdx/internal/config"
	"github.com/niklas-heer/tdx/internal/tui"
)

var testBinary string

func TestMain(m *testing.M) {
	// Initialize config for unit tests
	appConfig := LoadConfig()
	styles := NewStyles(appConfig)

	// Inject config and styles into packages for testing
	cmd.GreenStyle = func(s string) string { return styles.Success.Render(s) }
	cmd.DimStyle = func(s string) string { return styles.Dim.Render(s) }
	cmd.CheckSymbol = appConfig.Display.CheckSymbol

	tui.Config = &tui.ConfigType{}
	tui.Config.Display.CheckSymbol = appConfig.Display.CheckSymbol
	tui.Config.Display.SelectMarker = appConfig.Display.SelectMarker
	tui.Config.Display.MaxVisible = appConfig.Display.MaxVisible

	tui.StyleFuncs = &tui.StyleFuncsType{
		Magenta:        func(s string) string { return styles.Important.Render(s) },
		Cyan:           func(s string) string { return styles.Accent.Render(s) },
		Dim:            func(s string) string { return styles.Dim.Render(s) },
		Green:          func(s string) string { return styles.Success.Render(s) },
		Yellow:         func(s string) string { return styles.Warning.Render(s) },
		Code:           func(s string) string { return styles.Code.Render(s) },
		Tag:            func(s string) string { return styles.Tag.Render(s) },
		PriorityHigh:   func(s string) string { return styles.PriorityHigh.Render(s) },
		PriorityMedium: func(s string) string { return styles.PriorityMedium.Render(s) },
		PriorityLow:    func(s string) string { return styles.PriorityLow.Render(s) },
		DueUrgent:      func(s string) string { return styles.DueUrgent.Render(s) },
		DueSoon:        func(s string) string { return styles.DueSoon.Render(s) },
		DueFuture:      func(s string) string { return styles.DueFuture.Render(s) },
	}
	tui.Version = Version

	// Setup theme picker globals for testing
	tui.AvailableThemes = GetBuiltinThemeNames()
	tui.CurrentThemeName = appConfig.Theme.Name
	tui.ThemeApplyFunc = func(themeName string) *tui.StyleFuncsType {
		colors, ok := GetBuiltinTheme(themeName)
		if !ok {
			return nil
		}
		tempConfig := &UserConfig{Colors: colors}
		newStyles := NewStyles(tempConfig)
		return &tui.StyleFuncsType{
			Magenta:        func(s string) string { return newStyles.Important.Render(s) },
			Cyan:           func(s string) string { return newStyles.Accent.Render(s) },
			Dim:            func(s string) string { return newStyles.Dim.Render(s) },
			Green:          func(s string) string { return newStyles.Success.Render(s) },
			Yellow:         func(s string) string { return newStyles.Warning.Render(s) },
			Code:           func(s string) string { return newStyles.Code.Render(s) },
			Tag:            func(s string) string { return newStyles.Tag.Render(s) },
			PriorityHigh:   func(s string) string { return newStyles.PriorityHigh.Render(s) },
			PriorityMedium: func(s string) string { return newStyles.PriorityMedium.Render(s) },
			PriorityLow:    func(s string) string { return newStyles.PriorityLow.Render(s) },
			DueUrgent:      func(s string) string { return newStyles.DueUrgent.Render(s) },
			DueSoon:        func(s string) string { return newStyles.DueSoon.Render(s) },
			DueFuture:      func(s string) string { return newStyles.DueFuture.Render(s) },
		}
	}
	tui.ThemeSaveFunc = func(themeName string) error {
		// For testing, don't actually save to disk
		return nil
	}

	// Build the binary for testing
	tmpDir, err := os.MkdirTemp("", "tdx-test")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Isolate config directory for tests to avoid race conditions with other packages
	configDir, err := os.MkdirTemp("", "tdx-test-config")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(configDir) }()
	config.SetConfigDirForTesting(configDir)

	testBinary = filepath.Join(tmpDir, "tdx")
	buildCmd := exec.Command("go", "build", "-o", testBinary, ".")
	if err := buildCmd.Run(); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

// Helper to run CLI command
func runCLI(t *testing.T, file string, args ...string) string {
	cmdArgs := append([]string{file}, args...)
	cmd := exec.Command(testBinary, cmdArgs...)
	out, _ := cmd.CombinedOutput() // Some commands may fail, that's okay
	return strings.TrimSpace(string(out))
}

// Helper to run piped input to TUI
func runPiped(t *testing.T, file string, input string) string {
	output := tui.RunPiped(file, []byte(input), false)
	return strings.TrimSpace(output)
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
	_ = f.Close()
	t.Cleanup(func() { _ = os.Remove(f.Name()) })
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

func TestCLI_ToggleTodo(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "First todo")
	runCLI(t, file, "toggle", "1")

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Errorf("Expected 1 todo, got %d", len(todos))
	}
	if !strings.HasPrefix(todos[0], "- [x] ") {
		t.Errorf("Expected todo to be checked, got: %s", todos[0])
	}
}

func TestCLI_EditTodo(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "First todo")
	runCLI(t, file, "edit", "1", "Updated todo")

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Errorf("Expected 1 todo, got %d", len(todos))
	}
	if todos[0] != "- [ ] Updated todo" {
		t.Errorf("Expected '- [ ] Updated todo', got: %s", todos[0])
	}
}

func TestCLI_DeleteTodo(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "First todo")
	runCLI(t, file, "add", "Second todo")
	runCLI(t, file, "delete", "1")

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Errorf("Expected 1 todo after delete, got %d", len(todos))
	}
	if todos[0] != "- [ ] Second todo" {
		t.Errorf("Expected '- [ ] Second todo', got: %s", todos[0])
	}
}

func TestTUI_AddTodo(t *testing.T) {
	file := tempTestFile(t)

	// Simulate: n (new), type "Test", enter
	runPiped(t, file, "nTest\r")

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Errorf("Expected 1 todo, got %d", len(todos))
	}
	if todos[0] != "- [ ] Test" {
		t.Errorf("Expected '- [ ] Test', got: %s", todos[0])
	}
}

func TestTUI_ToggleTodo(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "First todo")

	// Simulate: space (toggle)
	runPiped(t, file, " ")

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Errorf("Expected 1 todo, got %d", len(todos))
	}
	if !strings.HasPrefix(todos[0], "- [x] ") {
		t.Errorf("Expected todo to be checked, got: %s", todos[0])
	}
}

func TestTUI_EditTodo(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "First todo")

	// Simulate: e (edit), clear existing, type "Edited", enter
	// Note: e enters edit mode with existing text, so we need to clear it first
	// Using backspace multiple times then typing new text
	runPiped(t, file, "e\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7fEdited\r")

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Errorf("Expected 1 todo, got %d", len(todos))
	}
	if todos[0] != "- [ ] Edited" {
		t.Errorf("Expected '- [ ] Edited', got: %s", todos[0])
	}
}

func TestTUI_EditTodoWithInlineCode(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Fix bug")

	// Simulate: e (edit), clear existing, type text with inline code, enter
	runPiped(t, file, "e\x7f\x7f\x7f\x7f\x7f\x7f\x7fUpdate `config.yaml`\r")

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Errorf("Expected 1 todo, got %d", len(todos))
	}
	expected := "- [ ] Update `config.yaml`"
	if todos[0] != expected {
		t.Errorf("Expected '%s', got: %s", expected, todos[0])
	}

	// Edit again to verify inline code is preserved
	runPiped(t, file, "e\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7fFix `main.go` error\r")

	todos = getTodos(t, file)
	expected = "- [ ] Fix `main.go` error"
	if todos[0] != expected {
		t.Errorf("Expected '%s', got: %s", expected, todos[0])
	}
}

func TestTUI_EditTodoWithMultipleCodeSpans(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Simple task")

	// Simulate: e (edit), clear existing, type text with multiple code spans, enter
	runPiped(t, file, "e\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7fUpdate `file1.go` and `file2.go`\r")

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Errorf("Expected 1 todo, got %d", len(todos))
	}
	expected := "- [ ] Update `file1.go` and `file2.go`"
	if todos[0] != expected {
		t.Errorf("Expected '%s', got: %s", expected, todos[0])
	}
}

func TestTUI_EditTodoWithSpecialChars(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "Task")

	// Simulate: e (edit), clear existing, type text with quotes and special chars, enter
	runPiped(t, file, "e\x7f\x7f\x7f\x7fAdd \"quotes\" test\r")

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Errorf("Expected 1 todo, got %d", len(todos))
	}
	expected := "- [ ] Add \"quotes\" test"
	if todos[0] != expected {
		t.Errorf("Expected '%s', got: %s", expected, todos[0])
	}
}

func TestTUI_EditTodoPreservesExistingInlineCode(t *testing.T) {
	file := tempTestFile(t)

	// Add a todo with inline code using CLI
	runCLI(t, file, "add", "Fix bug in `main.go` file")

	// Simulate: e (edit) to enter edit mode, then immediately confirm without changes
	// This tests that the InputBuffer correctly contains the backticks
	runPiped(t, file, "e\r")

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Errorf("Expected 1 todo, got %d", len(todos))
	}
	expected := "- [ ] Fix bug in `main.go` file"
	if todos[0] != expected {
		t.Errorf("Expected '%s', got: %s", expected, todos[0])
	}

	// Now edit it to change the filename
	runPiped(t, file, "e\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7fFix `utils.go` instead\r")

	todos = getTodos(t, file)
	expected = "- [ ] Fix `utils.go` instead"
	if todos[0] != expected {
		t.Errorf("Expected '%s', got: %s", expected, todos[0])
	}
}

func TestTUI_DeleteTodo(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "First todo")
	runCLI(t, file, "add", "Second todo")

	// Simulate: d (delete first)
	runPiped(t, file, "d")

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Errorf("Expected 1 todo after delete, got %d", len(todos))
	}
	if todos[0] != "- [ ] Second todo" {
		t.Errorf("Expected '- [ ] Second todo', got: %s", todos[0])
	}
}

func TestTUI_Navigation(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "First todo")
	runCLI(t, file, "add", "Second todo")
	runCLI(t, file, "add", "Third todo")

	// Simulate: j (down), j (down), space (toggle third)
	runPiped(t, file, "jj ")

	todos := getTodos(t, file)
	if len(todos) != 3 {
		t.Errorf("Expected 3 todos, got %d", len(todos))
	}
	// Third todo should be checked
	if !strings.HasPrefix(todos[2], "- [x] ") {
		t.Errorf("Expected third todo to be checked, got: %s", todos[2])
	}
}

func TestTUI_Undo(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "First todo")

	// Simulate: space (toggle), u (undo)
	runPiped(t, file, " u")

	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Errorf("Expected 1 todo, got %d", len(todos))
	}
	// Should be unchecked after undo
	if !strings.HasPrefix(todos[0], "- [ ] ") {
		t.Errorf("Expected todo to be unchecked after undo, got: %s", todos[0])
	}
}

func TestTUI_Move(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "First todo")
	runCLI(t, file, "add", "Second todo")

	// Simulate: m (move mode), j (move down), enter (confirm)
	runPiped(t, file, "mj\r")

	todos := getTodos(t, file)
	if len(todos) != 2 {
		t.Errorf("Expected 2 todos, got %d", len(todos))
	}
	// First and second should be swapped
	if todos[0] != "- [ ] Second todo" {
		t.Errorf("Expected 'Second todo' first, got: %s", todos[0])
	}
	if todos[1] != "- [ ] First todo" {
		t.Errorf("Expected 'First todo' second, got: %s", todos[1])
	}
}

func TestTUI_ReadOnlyMode(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "First todo")

	// Simulate: :read-only command, then space to toggle
	// The toggle should not persist because read-only mode is on
	output := tui.RunPiped(file, []byte(":read-only\r "), false)

	// Check that output shows READ ONLY indicator
	if !strings.Contains(output, "READ ONLY") {
		t.Errorf("Expected READ ONLY indicator in output")
	}

	// File should still have unchecked todo (change not persisted)
	todos := getTodos(t, file)
	if len(todos) != 1 {
		t.Errorf("Expected 1 todo, got %d", len(todos))
	}
	// Expected todo to remain unchecked in read-only mode
	if !strings.HasPrefix(todos[0], "- [ ] ") {
		t.Errorf("Expected todo to remain unchecked in read-only mode, got: %s", todos[0])
	}
}

func TestTUI_MultipleJumps(t *testing.T) {
	file := tempTestFile(t)

	// Add 10 todos
	for i := 1; i <= 10; i++ {
		runCLI(t, file, "add", "Todo "+string(rune('0'+i)))
	}

	// Simulate: 5j (jump 5 down), space (toggle 6th item)
	runPiped(t, file, "5j ")

	todos := getTodos(t, file)
	// 6th todo (index 5) should be checked
	if !strings.HasPrefix(todos[5], "- [x] ") {
		t.Errorf("Expected 6th todo to be checked after 5j, got: %s", todos[5])
	}
}

func TestTUI_EmptyInput(t *testing.T) {
	file := tempTestFile(t)

	// Simulate: n (new), enter without typing (should not create empty todo)
	runPiped(t, file, "n\r")

	todos := getTodos(t, file)
	if len(todos) != 0 {
		t.Errorf("Expected 0 todos after empty input, got %d", len(todos))
	}
}

func TestTUI_CommandPalette(t *testing.T) {
	file := tempTestFile(t)

	runCLI(t, file, "add", "First todo")
	runCLI(t, file, "add", "Second todo")

	// Simulate: : (command mode), type "check-all", enter
	runPiped(t, file, ":check-all\r")

	todos := getTodos(t, file)
	// Both todos should be checked
	for i, todo := range todos {
		if !strings.HasPrefix(todo, "- [x] ") {
			t.Errorf("Expected todo %d to be checked, got: %s", i+1, todo)
		}
	}
}
