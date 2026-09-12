package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/niklas-heer/tdx/internal/markdown"
	"io"
	"slices"
	"time"
)

// ListOptions applies equally to human-readable and JSON task queries.
type ListOptions struct {
	JSON         bool
	Status       string
	Tags         []string
	Priority     *int
	Due          string
	Section      string
	WithRevision bool
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
	if err := opts.Validate(); err != nil {
		return err
	}
	fm, err := s.Store.ReadFile(filePath)
	if err != nil {
		return err
	}
	sectionMatches := matchingSections(fm, opts.Section)
	tasks := make([]Task, 0, len(fm.Todos))
	for _, todo := range fm.Todos {
		if opts.Priority != nil && todo.Priority != *opts.Priority {
			continue
		}
		if !matchesDue(todo, opts.Due) || (sectionMatches != nil && !sectionMatches[todo.Index-1]) {
			continue
		}
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
	if opts.JSON || opts.WithRevision {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		encoder.SetEscapeHTML(false)
		if opts.WithRevision {
			revision, err := fm.RevisionToken()
			if err != nil {
				return err
			}
			return encoder.Encode(ListSnapshot{SchemaVersion: 1, Revision: revision, Tasks: tasks})
		}
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

// ListSnapshot is the opt-in versioned envelope; ordinary --json stays an array.
type ListSnapshot struct {
	SchemaVersion int    `json:"schema_version"`
	Revision      string `json:"revision"`
	Tasks         []Task `json:"tasks"`
}

func (opts ListOptions) Validate() error {
	if opts.Status != "" && opts.Status != "all" && opts.Status != "open" && opts.Status != "done" {
		return fmt.Errorf("status must be all, open, or done")
	}
	if opts.Priority != nil && *opts.Priority < 0 {
		return fmt.Errorf("priority must be a non-negative integer (0 means unset)")
	}
	switch opts.Due {
	case "", "all", "none", "overdue", "today", "week":
	default:
		if _, err := time.Parse("2006-01-02", opts.Due); err != nil {
			return fmt.Errorf("due must be all, none, overdue, today, week, or YYYY-MM-DD")
		}
	}
	return nil
}

func matchesDue(todo markdown.Todo, due string) bool {
	switch due {
	case "none":
		return todo.DueDate == nil
	case "", "all", "overdue", "today", "week":
		return todo.HasDueDateFilter(due)
	default:
		return todo.DueDate != nil && todo.DueDate.Format("2006-01-02") == due
	}
}

// A section includes its descendants. Repeated titles select all matching
// sections, preserving document indexes even for overlapping parent sections.
func matchingSections(fm *markdown.FileModel, title string) []bool {
	if title == "" {
		return nil
	}
	matches := make([]bool, len(fm.Todos))
	headings := fm.GetHeadings()
	for i, h := range headings {
		if h.Text != title {
			continue
		}
		end := len(fm.Todos)
		for _, next := range headings[i+1:] {
			if next.Level <= h.Level {
				end = next.BeforeTodoIndex
				break
			}
		}
		for task := h.BeforeTodoIndex; task >= 0 && task < end; task++ {
			matches[task] = true
		}
	}
	return matches
}
