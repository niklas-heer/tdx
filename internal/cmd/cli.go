package cmd

import (
	"charm.land/lipgloss/v2"
	"fmt"
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
func ListTodos(filePath string) error {
	return WriteList(os.Stdout, filePath, ListOptions{})
}

// AddTodo adds a new todo to a file
func AddTodo(filePath string, text string) error {
	fm, err := markdown.ReadFile(filePath)
	if err != nil {
		return err
	}

	fm.AddTodoItem(text, false)

	if err := markdown.WriteFile(filePath, fm); err != nil {
		return err
	}

	_, err = lipgloss.Fprintf(os.Stdout, "%s Added: %s\n", GreenStyle("✓"), text)
	return err
}

// ToggleTodo toggles the completion status of a todo
func ToggleTodo(filePath string, index int) error {
	fm, err := markdown.ReadFile(filePath)
	if err != nil {
		return err
	}

	if index < 1 || index > len(fm.Todos) {
		return fmt.Errorf("invalid index %d", index)
	}

	todo := fm.Todos[index-1]
	if err := fm.UpdateTodoItem(index-1, todo.Text, !todo.Checked); err != nil {
		return err
	}

	if err := markdown.WriteFile(filePath, fm); err != nil {
		return err
	}

	checkbox := "[ ]"
	if !todo.Checked {
		checkbox = "[" + CheckSymbol + "]"
	}
	_, err = lipgloss.Fprintf(os.Stdout, "%s Toggled: %s %s\n", GreenStyle("✓"), checkbox, todo.Text)
	return err
}

// EditTodo edits the text of a todo
func EditTodo(filePath string, index int, text string) error {
	fm, err := markdown.ReadFile(filePath)
	if err != nil {
		return err
	}

	if index < 1 || index > len(fm.Todos) {
		return fmt.Errorf("invalid index %d", index)
	}

	todo := fm.Todos[index-1]
	if err := fm.UpdateTodoItem(index-1, text, todo.Checked); err != nil {
		return err
	}

	if err := markdown.WriteFile(filePath, fm); err != nil {
		return err
	}

	_, err = lipgloss.Fprintf(os.Stdout, "%s Edited: %s\n", GreenStyle("✓"), text)
	return err
}

// DeleteTodo deletes a todo by index
func DeleteTodo(filePath string, index int) error {
	fm, err := markdown.ReadFile(filePath)
	if err != nil {
		return err
	}

	if index < 1 || index > len(fm.Todos) {
		return fmt.Errorf("invalid index %d", index)
	}

	todo := fm.Todos[index-1]

	if err := fm.DeleteTodoItem(index - 1); err != nil {
		return err
	}

	if err := markdown.WriteFile(filePath, fm); err != nil {
		return err
	}

	_, err = lipgloss.Fprintf(os.Stdout, "%s Deleted: %s\n", GreenStyle("✓"), todo.Text)
	return err
}

// ValidateCommand rejects malformed invocations before opening files or history.
func ValidateCommand(command string, args []string) error {
	switch command {
	case "list":
		if len(args) != 0 {
			return fmt.Errorf("list does not take positional arguments")
		}
	case "add":
		if len(args) < 1 {
			return fmt.Errorf("add requires text argument")
		}
	case "toggle", "delete":
		if len(args) != 1 {
			return fmt.Errorf("%s requires exactly one index argument", command)
		}
	case "edit":
		if len(args) < 2 {
			return fmt.Errorf("edit requires index and text arguments")
		}
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
	if command == "toggle" || command == "delete" || command == "edit" {
		index, err := strconv.Atoi(args[0])
		if err != nil || index < 1 {
			return fmt.Errorf("invalid index: use a positive integer")
		}
	}
	return nil
}

// HandleCommand returns errors to its caller so deferred cleanup can run.
func HandleCommand(command string, cmdArgs []string, filePath string) error {
	if err := ValidateCommand(command, cmdArgs); err != nil {
		return err
	}
	switch command {
	case "list":
		return ListTodos(filePath)
	case "add":
		return AddTodo(filePath, strings.Join(cmdArgs, " "))
	case "toggle":
		idx, _ := strconv.Atoi(cmdArgs[0])
		return ToggleTodo(filePath, idx)
	case "edit":
		idx, _ := strconv.Atoi(cmdArgs[0])
		return EditTodo(filePath, idx, strings.Join(cmdArgs[1:], " "))
	case "delete":
		idx, _ := strconv.Atoi(cmdArgs[0])
		return DeleteTodo(filePath, idx)
	}
	return fmt.Errorf("unknown command: %s", command)
}
