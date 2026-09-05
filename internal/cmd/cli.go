package cmd

import (
	"charm.land/lipgloss/v2"
	"os"
	"strconv"
	"strings"

	"github.com/niklas-heer/tdx/internal/markdown"
)

// CLI colors - will be initialized from main
var (
	GreenStyle  func(string) string
	DimStyle    func(string) string
	CheckSymbol string
)

// ListTodos lists all todos in a file
func ListTodos(filePath string) {
	fm, err := markdown.ReadFile(filePath)
	if err != nil {
		_, _ = lipgloss.Fprintf(os.Stdout, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(fm.Todos) == 0 {
		_, _ = lipgloss.Fprintln(os.Stdout, "No todos found")
		return
	}

	for _, todo := range fm.Todos {
		checkbox := "[ ]"
		if todo.Checked {
			checkbox = "[" + CheckSymbol + "]"
		}
		_, _ = lipgloss.Fprintf(os.Stdout, "  %d. %s %s\n", todo.Index, checkbox, todo.Text)
	}
}

// AddTodo adds a new todo to a file
func AddTodo(filePath string, text string) {
	// Remove surrounding quotes if present
	text = strings.Trim(text, "\"")

	fm, err := markdown.ReadFile(filePath)
	if err != nil {
		_, _ = lipgloss.Fprintf(os.Stdout, "Error: %v\n", err)
		os.Exit(1)
	}

	fm.AddTodoItem(text, false)

	if err := markdown.WriteFile(filePath, fm); err != nil {
		_, _ = lipgloss.Fprintf(os.Stdout, "Error: %v\n", err)
		os.Exit(1)
	}

	_, _ = lipgloss.Fprintf(os.Stdout, "%s Added: %s\n", GreenStyle("✓"), text)
}

// ToggleTodo toggles the completion status of a todo
func ToggleTodo(filePath string, index int) {
	fm, err := markdown.ReadFile(filePath)
	if err != nil {
		_, _ = lipgloss.Fprintf(os.Stdout, "Error: %v\n", err)
		os.Exit(1)
	}

	if index < 1 || index > len(fm.Todos) {
		_, _ = lipgloss.Fprintf(os.Stdout, "Error: invalid index %d\n", index)
		os.Exit(1)
	}

	todo := fm.Todos[index-1]
	if err := fm.UpdateTodoItem(index-1, todo.Text, !todo.Checked); err != nil {
		_, _ = lipgloss.Fprintf(os.Stdout, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := markdown.WriteFile(filePath, fm); err != nil {
		_, _ = lipgloss.Fprintf(os.Stdout, "Error: %v\n", err)
		os.Exit(1)
	}

	checkbox := "[ ]"
	if !todo.Checked {
		checkbox = "[" + CheckSymbol + "]"
	}
	_, _ = lipgloss.Fprintf(os.Stdout, "%s Toggled: %s %s\n", GreenStyle("✓"), checkbox, todo.Text)
}

// EditTodo edits the text of a todo
func EditTodo(filePath string, index int, text string) {
	text = strings.Trim(text, "\"")

	fm, err := markdown.ReadFile(filePath)
	if err != nil {
		_, _ = lipgloss.Fprintf(os.Stdout, "Error: %v\n", err)
		os.Exit(1)
	}

	if index < 1 || index > len(fm.Todos) {
		_, _ = lipgloss.Fprintf(os.Stdout, "Error: invalid index %d\n", index)
		os.Exit(1)
	}

	todo := fm.Todos[index-1]
	if err := fm.UpdateTodoItem(index-1, text, todo.Checked); err != nil {
		_, _ = lipgloss.Fprintf(os.Stdout, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := markdown.WriteFile(filePath, fm); err != nil {
		_, _ = lipgloss.Fprintf(os.Stdout, "Error: %v\n", err)
		os.Exit(1)
	}

	_, _ = lipgloss.Fprintf(os.Stdout, "%s Edited: %s\n", GreenStyle("✓"), text)
}

// DeleteTodo deletes a todo by index
func DeleteTodo(filePath string, index int) {
	fm, err := markdown.ReadFile(filePath)
	if err != nil {
		_, _ = lipgloss.Fprintf(os.Stdout, "Error: %v\n", err)
		os.Exit(1)
	}

	if index < 1 || index > len(fm.Todos) {
		_, _ = lipgloss.Fprintf(os.Stdout, "Error: invalid index %d\n", index)
		os.Exit(1)
	}

	todo := fm.Todos[index-1]

	if err := fm.DeleteTodoItem(index - 1); err != nil {
		_, _ = lipgloss.Fprintf(os.Stdout, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := markdown.WriteFile(filePath, fm); err != nil {
		_, _ = lipgloss.Fprintf(os.Stdout, "Error: %v\n", err)
		os.Exit(1)
	}

	_, _ = lipgloss.Fprintf(os.Stdout, "%s Deleted: %s\n", GreenStyle("✓"), todo.Text)
}

// HandleCommand parses and executes CLI commands
func HandleCommand(command string, cmdArgs []string, filePath string) {
	switch command {
	case "list":
		ListTodos(filePath)
	case "add":
		if len(cmdArgs) < 1 {
			_, _ = lipgloss.Fprintln(os.Stdout, "Error: add requires text argument")
			os.Exit(1)
		}
		AddTodo(filePath, strings.Join(cmdArgs, " "))
	case "toggle":
		if len(cmdArgs) < 1 {
			_, _ = lipgloss.Fprintln(os.Stdout, "Error: toggle requires index argument")
			os.Exit(1)
		}
		idx, err := strconv.Atoi(cmdArgs[0])
		if err != nil {
			_, _ = lipgloss.Fprintln(os.Stdout, "Error: invalid index")
			os.Exit(1)
		}
		ToggleTodo(filePath, idx)
	case "edit":
		if len(cmdArgs) < 2 {
			_, _ = lipgloss.Fprintln(os.Stdout, "Error: edit requires index and text arguments")
			os.Exit(1)
		}
		idx, err := strconv.Atoi(cmdArgs[0])
		if err != nil {
			_, _ = lipgloss.Fprintln(os.Stdout, "Error: invalid index")
			os.Exit(1)
		}
		EditTodo(filePath, idx, strings.Join(cmdArgs[1:], " "))
	case "delete":
		if len(cmdArgs) < 1 {
			_, _ = lipgloss.Fprintln(os.Stdout, "Error: delete requires index argument")
			os.Exit(1)
		}
		idx, err := strconv.Atoi(cmdArgs[0])
		if err != nil {
			_, _ = lipgloss.Fprintln(os.Stdout, "Error: invalid index")
			os.Exit(1)
		}
		DeleteTodo(filePath, idx)
	default:
		_, _ = lipgloss.Fprintf(os.Stdout, "Unknown command: %s\n", command)
		os.Exit(1)
	}
}
