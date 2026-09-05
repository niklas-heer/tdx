package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/niklas-heer/tdx/internal/markdown"
)

func setupCLIStyles(t *testing.T) {
	t.Helper()

	originalGreenStyle := GreenStyle
	originalDimStyle := DimStyle
	originalCheckSymbol := CheckSymbol
	GreenStyle = func(value string) string { return value }
	DimStyle = func(value string) string { return value }
	CheckSymbol = "x"

	t.Cleanup(func() {
		GreenStyle = originalGreenStyle
		DimStyle = originalDimStyle
		CheckSymbol = originalCheckSymbol
	})
}

func writeTodoFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "todos.md")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write todo file: %v", err)
	}
	return path
}

func captureStdout(t *testing.T, run func()) string {
	t.Helper()

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = writer

	defer func() {
		os.Stdout = originalStdout
		_ = writer.Close()
		_ = reader.Close()
	}()

	type readResult struct {
		output []byte
		err    error
	}
	result := make(chan readResult, 1)
	go func() {
		output, err := io.ReadAll(reader)
		result <- readResult{output: output, err: err}
	}()

	run()
	os.Stdout = originalStdout
	if err := writer.Close(); err != nil {
		t.Fatalf("close stdout writer: %v", err)
	}

	captured := <-result
	if captured.err != nil {
		t.Fatalf("read stdout: %v", captured.err)
	}
	return string(captured.output)
}

func TestTodoCommandsRoundTrip(t *testing.T) {
	setupCLIStyles(t)
	path := writeTodoFile(t, "# Todos\n\n- [ ] First\n")

	output := captureStdout(t, func() {
		if err := ListTodos(path); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(output, "1. [ ] First") {
		t.Fatalf("list output = %q, want first todo", output)
	}

	output = captureStdout(t, func() {
		if err := AddTodo(path, "Second"); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(output, "Added: Second") {
		t.Fatalf("add output = %q, want added todo", output)
	}

	output = captureStdout(t, func() {
		if err := ToggleTodo(path, 1); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(output, "Toggled: [x] First") {
		t.Fatalf("toggle output = %q, want checked todo", output)
	}

	output = captureStdout(t, func() {
		if err := EditTodo(path, 2, "Changed"); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(output, "Edited: Changed") {
		t.Fatalf("edit output = %q, want edited todo", output)
	}

	output = captureStdout(t, func() {
		if err := DeleteTodo(path, 1); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(output, "Deleted: First") {
		t.Fatalf("delete output = %q, want deleted todo", output)
	}

	fm, err := markdown.ReadFile(path)
	if err != nil {
		t.Fatalf("read final todo file: %v", err)
	}
	if len(fm.Todos) != 1 || fm.Todos[0].Text != "Changed" || fm.Todos[0].Checked {
		t.Fatalf("final todos = %#v, want one unchecked Changed todo", fm.Todos)
	}
}

func TestListTodosEmpty(t *testing.T) {
	setupCLIStyles(t)
	path := writeTodoFile(t, "# Todos\n")

	output := captureStdout(t, func() {
		if err := ListTodos(path); err != nil {
			t.Fatal(err)
		}
	})
	if output != "No todos found\n" {
		t.Fatalf("list output = %q, want empty message", output)
	}
}

func TestHandleCommand(t *testing.T) {
	setupCLIStyles(t)

	tests := []struct {
		name    string
		command string
		args    []string
		check   func(*testing.T, *markdown.FileModel)
	}{
		{name: "list", command: "list"},
		{
			name:    "add joins arguments",
			command: "add",
			args:    []string{"Second", "todo"},
			check: func(t *testing.T, fm *markdown.FileModel) {
				if len(fm.Todos) != 2 || fm.Todos[1].Text != "Second todo" {
					t.Fatalf("todos = %#v, want joined added todo", fm.Todos)
				}
			},
		},
		{
			name:    "toggle parses index",
			command: "toggle",
			args:    []string{"1"},
			check: func(t *testing.T, fm *markdown.FileModel) {
				if !fm.Todos[0].Checked {
					t.Fatalf("todo = %#v, want checked", fm.Todos[0])
				}
			},
		},
		{
			name:    "edit joins arguments",
			command: "edit",
			args:    []string{"1", "Updated", "todo"},
			check: func(t *testing.T, fm *markdown.FileModel) {
				if fm.Todos[0].Text != "Updated todo" {
					t.Fatalf("todo = %#v, want updated text", fm.Todos[0])
				}
			},
		},
		{
			name:    "delete parses index",
			command: "delete",
			args:    []string{"1"},
			check: func(t *testing.T, fm *markdown.FileModel) {
				if len(fm.Todos) != 0 {
					t.Fatalf("todos = %#v, want no todos", fm.Todos)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := writeTodoFile(t, "# Todos\n\n- [ ] First\n")
			captureStdout(t, func() {
				if err := HandleCommand(test.command, test.args, path); err != nil {
					t.Fatal(err)
				}
			})
			if test.check == nil {
				return
			}

			fm, err := markdown.ReadFile(path)
			if err != nil {
				t.Fatalf("read todo file: %v", err)
			}
			test.check(t, fm)
		})
	}
}
