package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"
)

// ListOptions applies equally to human-readable and JSON task queries.
type ListOptions struct {
	JSON   bool
	Status string
	Tags   []string
}

// Task is the scripting representation. Index and ParentIndex are one-based;
// a nil parent means a root task. Indexes refer to the unfiltered document.
type Task struct {
	Index       int      `json:"index"`
	Text        string   `json:"text"`
	Checked     bool     `json:"checked"`
	Depth       int      `json:"depth"`
	ParentIndex *int     `json:"parent_index"`
	Tags        []string `json:"tags"`
	Priority    int      `json:"priority"`
	DueDate     *string  `json:"due_date"`
}

// WriteList writes a query result without formatting JSON with terminal styles.
func (s Service) WriteList(out io.Writer, filePath string, opts ListOptions) error {
	if opts.Status != "" && opts.Status != "all" && opts.Status != "open" && opts.Status != "done" {
		return fmt.Errorf("status must be all, open, or done")
	}
	fm, err := s.Store.ReadFile(filePath)
	if err != nil {
		return err
	}
	tasks := make([]Task, 0, len(fm.Todos))
	for _, todo := range fm.Todos {
		if opts.Status == "open" && todo.Checked || opts.Status == "done" && !todo.Checked {
			continue
		}
		matches := true
		for _, tag := range opts.Tags {
			if !slices.Contains(todo.Tags, tag) {
				matches = false
				break
			}
		}
		if !matches {
			continue
		}
		task := Task{Index: todo.Index, Text: todo.Text, Checked: todo.Checked, Depth: todo.Depth, Tags: append([]string{}, todo.Tags...), Priority: todo.Priority}
		if todo.ParentIndex >= 0 {
			parent := todo.ParentIndex + 1
			task.ParentIndex = &parent
		}
		if todo.DueDate != nil {
			date := todo.DueDate.Format("2006-01-02")
			task.DueDate = &date
		}
		tasks = append(tasks, task)
	}
	if opts.JSON {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		encoder.SetEscapeHTML(false)
		return encoder.Encode(tasks)
	}
	if len(tasks) == 0 {
		_, err := fmt.Fprintln(out, "No todos found")
		return err
	}
	for _, task := range tasks {
		checkbox := "[ ]"
		if task.Checked {
			checkbox = "[" + s.checkSymbol() + "]"
		}
		if _, err := fmt.Fprintf(out, "  %d. %s %s\n", task.Index, checkbox, task.Text); err != nil {
			return err
		}
	}
	return nil
}

// WriteList is a hook-free query convenience for callers without a service.
func WriteList(out io.Writer, filePath string, opts ListOptions) error {
	return (Service{}).WriteList(out, filePath, opts)
}
