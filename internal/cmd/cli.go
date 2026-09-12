package cmd

import (
	"charm.land/lipgloss/v2"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/niklas-heer/tdx/internal/editor"
	"github.com/niklas-heer/tdx/internal/markdown"
)

// Service owns the output and persistence dependencies for a CLI instance.
// Its zero value uses stdout, plain styling, and a hook-free Markdown store.
type Service struct {
	Store            markdown.Store
	Out              io.Writer
	GreenStyle       func(string) string
	CheckSymbol      string
	ExpectedRevision string
}

func (s Service) output() io.Writer {
	if s.Out != nil {
		return s.Out
	}
	return os.Stdout
}
func (s Service) success(text string) string {
	if s.GreenStyle != nil {
		return s.GreenStyle(text)
	}
	return text
}
func (s Service) checkSymbol() string {
	if s.CheckSymbol != "" {
		return s.CheckSymbol
	}
	return "x"
}

// ListTodos lists all todos in a file
func (s Service) ListTodos(filePath string) error {
	return s.WriteList(s.output(), filePath, ListOptions{})
}

// AddTodo adds a new todo to a file
func (s Service) AddTodo(filePath string, text string) error {
	fm, err := s.readForMutation(filePath)
	if err != nil {
		return err
	}

	if _, err := editor.Apply(fm, editor.Action{Kind: editor.Add, Text: text}); err != nil {
		return err
	}

	if err := s.Store.WriteFile(filePath, fm); err != nil {
		return err
	}

	_, err = lipgloss.Fprintf(s.output(), "%s Added: %s\n", s.success("✓"), text)
	return err
}

// ToggleTodo toggles the completion status of a todo
func (s Service) ToggleTodo(filePath string, index int) error {
	fm, err := s.readForMutation(filePath)
	if err != nil {
		return err
	}

	if index < 1 || index > len(fm.Todos) {
		return fmt.Errorf("invalid index %d", index)
	}

	todo := fm.Todos[index-1]
	if _, err := editor.Apply(fm, editor.Action{Kind: editor.Toggle, Index: index - 1}); err != nil {
		return err
	}

	if err := s.Store.WriteFile(filePath, fm); err != nil {
		return err
	}

	checkbox := "[ ]"
	if !todo.Checked {
		checkbox = "[" + s.checkSymbol() + "]"
	}
	_, err = lipgloss.Fprintf(s.output(), "%s Toggled: %s %s\n", s.success("✓"), checkbox, todo.Text)
	return err
}

// EditTodo edits the text of a todo
func (s Service) EditTodo(filePath string, index int, text string) error {
	fm, err := s.readForMutation(filePath)
	if err != nil {
		return err
	}

	if index < 1 || index > len(fm.Todos) {
		return fmt.Errorf("invalid index %d", index)
	}

	if _, err := editor.Apply(fm, editor.Action{Kind: editor.Edit, Index: index - 1, Text: text}); err != nil {
		return err
	}

	if err := s.Store.WriteFile(filePath, fm); err != nil {
		return err
	}

	_, err = lipgloss.Fprintf(s.output(), "%s Edited: %s\n", s.success("✓"), text)
	return err
}

// DeleteTodo deletes a todo by index
func (s Service) DeleteTodo(filePath string, index int) error {
	fm, err := s.readForMutation(filePath)
	if err != nil {
		return err
	}

	if index < 1 || index > len(fm.Todos) {
		return fmt.Errorf("invalid index %d", index)
	}

	todo := fm.Todos[index-1]

	if _, err := editor.Apply(fm, editor.Action{Kind: editor.Delete, Index: index - 1}); err != nil {
		return err
	}

	if err := s.Store.WriteFile(filePath, fm); err != nil {
		return err
	}

	_, err = lipgloss.Fprintf(s.output(), "%s Deleted: %s\n", s.success("✓"), todo.Text)
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
	case "toggle", "delete", "done", "undone":
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
	if command == "toggle" || command == "delete" || command == "edit" || command == "done" || command == "undone" {
		index, err := strconv.Atoi(args[0])
		if err != nil || index < 1 {
			return fmt.Errorf("invalid index: use a positive integer")
		}
	}
	return nil
}

// HandleCommand returns errors to its caller so deferred cleanup can run.
func (s Service) HandleCommand(command string, cmdArgs []string, filePath string) error {
	if err := ValidateCommand(command, cmdArgs); err != nil {
		return err
	}
	switch command {
	case "list":
		return s.ListTodos(filePath)
	case "add":
		return s.AddTodo(filePath, strings.Join(cmdArgs, " "))
	case "toggle":
		idx, _ := strconv.Atoi(cmdArgs[0])
		return s.ToggleTodo(filePath, idx)
	case "edit":
		idx, _ := strconv.Atoi(cmdArgs[0])
		return s.EditTodo(filePath, idx, strings.Join(cmdArgs[1:], " "))
	case "done", "undone":
		idx, _ := strconv.Atoi(cmdArgs[0])
		return s.SetTodoChecked(filePath, idx, command == "done")
	case "delete":
		idx, _ := strconv.Atoi(cmdArgs[0])
		return s.DeleteTodo(filePath, idx)
	}
	return fmt.Errorf("unknown command: %s", command)
}

// readForMutation checks the supplied revision against the same snapshot that
// will be edited, avoiding a separate-read race before the guarded save.
func (s Service) readForMutation(path string) (*markdown.FileModel, error) {
	fm, err := s.Store.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if s.ExpectedRevision != "" {
		if err := fm.RequireRevision(s.ExpectedRevision); err != nil {
			return nil, err
		}
	}
	return fm, nil
}

// SetTodoChecked is idempotent: a retry does not toggle the task back or rewrite
// an already-correct file. Supplied revision preconditions still apply.
func (s Service) SetTodoChecked(path string, index int, checked bool) error {
	fm, err := s.readForMutation(path)
	if err != nil {
		return err
	}
	if index < 1 || index > len(fm.Todos) {
		return fmt.Errorf("invalid index %d", index)
	}
	if fm.Todos[index-1].Checked != checked {
		if _, err := editor.Apply(fm, editor.Action{Kind: editor.SetChecked, Index: index - 1, Checked: checked}); err != nil {
			return err
		}
		if err := s.Store.WriteFile(path, fm); err != nil {
			return err
		}
	}
	state := "open"
	if checked {
		state = "done"
	}
	_, err = fmt.Fprintf(s.output(), "%s: %s\n", state, fm.Todos[index-1].Text)
	return err
}
