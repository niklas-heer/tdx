// Command go-parity exposes the production document actions to differential tests.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/niklas-heer/tdx/internal/editor"
	"github.com/niklas-heer/tdx/internal/markdown"
)

type action struct {
	Kind        string `json:"kind"`
	Index       int    `json:"index"`
	Target      int    `json:"target"`
	Text        string `json:"text"`
	Checked     bool   `json:"checked"`
	InsertAfter bool   `json:"insert_after"`
	Level       int    `json:"level"`
}
type request struct {
	Source  string   `json:"source"`
	Actions []action `json:"actions"`
}

func snapshot(doc *markdown.FileModel, content string, err error) map[string]any {
	tasks := []map[string]any{}
	for _, t := range doc.Todos {
		var parent *int
		if t.ParentIndex >= 0 {
			p := t.ParentIndex + 1
			parent = &p
		}
		var due *string
		if t.DueDate != nil {
			d := t.DueDate.Format("2006-01-02")
			due = &d
		}
		tasks = append(tasks, map[string]any{"index": t.Index, "text": t.Text, "checked": t.Checked, "depth": t.Depth, "parent_index": parent, "tags": append([]string{}, t.Tags...), "priority": t.Priority, "due_date": due})
	}
	headings := []map[string]any{}
	for _, h := range doc.GetHeadings() {
		headings = append(headings, map[string]any{"level": h.Level, "text": h.Text, "before_todo_index": h.BeforeTodoIndex})
	}
	out := map[string]any{"source": content, "tasks": tasks, "headings": headings}
	if err != nil {
		out["error"] = err.Error()
	}
	return out
}
func run() error {
	dir, err := os.MkdirTemp("", "tdx-go-parity-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	path := filepath.Join(dir, "tasks.md")
	scan := bufio.NewScanner(os.Stdin)
	scan.Buffer(make([]byte, 65536), 64*1024*1024)
	encoder := json.NewEncoder(os.Stdout)
	for scan.Scan() {
		var req request
		if err := json.Unmarshal(scan.Bytes(), &req); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(req.Source), 0600); err != nil {
			return err
		}
		doc, err := markdown.ReadFile(path)
		if err != nil {
			return err
		}
		steps := []map[string]any{snapshot(doc, req.Source, nil)}
		for _, a := range req.Actions {
			_, actionErr := editor.Apply(doc, editor.Action{Kind: editor.Kind(a.Kind), Index: a.Index, Target: a.Target, Text: a.Text, Checked: a.Checked, InsertAfter: a.InsertAfter, Level: a.Level})
			content := markdown.SerializeMarkdown(doc)
			if err := os.WriteFile(path, []byte(content), 0600); err != nil {
				return err
			}
			doc, err = markdown.ReadFile(path)
			if err != nil {
				return err
			}
			steps = append(steps, snapshot(doc, content, actionErr))
		}
		if err := encoder.Encode(steps); err != nil {
			return err
		}
	}
	return scan.Err()
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
