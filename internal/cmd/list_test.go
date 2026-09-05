package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestListJSON(t *testing.T) {
	path := writeTodoFile(t, "---\ntitle: Project\n---\n# Backend\n\n- [x] Parent #backend\n    - [ ] Käse #backend #urgent !p1 @due(2026-10-01)\n- [ ] Frontend #web\n")
	before, _ := os.ReadFile(path)
	var out bytes.Buffer
	if err := WriteList(&out, path, ListOptions{JSON: true, Status: "open", Tags: []string{"backend", "urgent"}}); err != nil {
		t.Fatal(err)
	}
	var tasks []Task
	if err := json.Unmarshal(out.Bytes(), &tasks); err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("got %s", out.String())
	}
	task := tasks[0]
	if task.Index != 2 || task.Checked || task.Depth != 1 || task.ParentIndex == nil || *task.ParentIndex != 1 || task.Priority != 1 || task.DueDate == nil || *task.DueDate != "2026-10-01" || !reflect.DeepEqual(task.Tags, []string{"backend", "urgent"}) {
		t.Fatalf("unexpected task: %s", out.String())
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("listing modified the document")
	}
}

func TestListFormatsAndFilters(t *testing.T) {
	setupCLIStyles(t)
	path := writeTodoFile(t, "- [x] Done #backend\n- [ ] Plain\n- [ ] Other #backend-extra\n")
	for _, tt := range []struct {
		opts ListOptions
		want string
	}{
		{ListOptions{Status: "done"}, "  1. [x] Done #backend\n"},
		{ListOptions{Status: "open", Tags: []string{"backend"}}, "No todos found\n"},
		{ListOptions{JSON: true, Tags: []string{"missing"}}, "[]\n"},
	} {
		var out bytes.Buffer
		if err := WriteList(&out, path, tt.opts); err != nil {
			t.Fatal(err)
		}
		if out.String() != tt.want {
			t.Fatalf("got %q, want %q", out.String(), tt.want)
		}
	}
	var out bytes.Buffer
	if err := WriteList(&out, path, ListOptions{JSON: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"tags": []`) || !strings.Contains(out.String(), `"parent_index": null`) || !strings.Contains(out.String(), `"due_date": null`) {
		t.Fatal(out.String())
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestListErrorsAndMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.md")
	var out bytes.Buffer
	if err := WriteList(&out, path, ListOptions{JSON: true}); err != nil {
		t.Fatal(err)
	}
	if out.String() != "[]\n" {
		t.Fatal(out.String())
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("listing created a file")
	}
	if err := WriteList(failingWriter{}, path, ListOptions{JSON: true}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("lost write error: %v", err)
	}
	if err := WriteList(io.Discard, t.TempDir(), ListOptions{}); err == nil {
		t.Fatal("accepted a directory")
	}
	if err := WriteList(io.Discard, path, ListOptions{Status: "typo"}); err == nil {
		t.Fatal("accepted invalid status")
	}
}

func TestCommandErrorsReturn(t *testing.T) {
	path := writeTodoFile(t, "- [ ] Keep me\n")
	before, _ := os.ReadFile(path)
	for _, tt := range []struct {
		command string
		args    []string
	}{
		{"add", nil}, {"edit", []string{"1"}}, {"toggle", []string{"abc"}}, {"delete", []string{"-1"}},
		{"toggle", []string{"1", "2"}}, {"list", []string{"extra"}}, {"unknown", nil}, {"delete", []string{"9"}},
	} {
		if err := HandleCommand(tt.command, tt.args, path); err == nil {
			t.Fatalf("accepted %s %q", tt.command, tt.args)
		}
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("invalid command modified file")
	}
}
