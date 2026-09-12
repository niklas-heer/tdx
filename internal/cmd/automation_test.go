package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/niklas-heer/tdx/internal/markdown"
)

func TestQuerySnapshotAndCombinedFilters(t *testing.T) {
	path := writeTodoFile(t, "# Work\n\n- [ ] One !p2 @due(2026-10-01) #work\n## API\n- [ ] Two !p1 @due(2026-10-01) #work\n# Personal\n- [ ] Three !p1 @due(2026-10-01) #work\n# Work\n- [ ] Four !p1 @due(2026-10-01) #work\n")
	priority := 1
	var out bytes.Buffer
	if err := WriteList(&out, path, ListOptions{WithRevision: true, Status: "open", Tags: []string{"work"}, Priority: &priority, Due: "2026-10-01", Section: "Work"}); err != nil {
		t.Fatal(err)
	}
	var snapshot ListSnapshot
	if err := json.Unmarshal(out.Bytes(), &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.SchemaVersion != 1 || len(snapshot.Tasks) != 2 || snapshot.Tasks[0].Index != 2 || snapshot.Tasks[1].Index != 4 {
		t.Fatalf("unexpected snapshot: %s", out.String())
	}
	fm, err := markdown.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := fm.RequireRevision(snapshot.Revision); err != nil {
		t.Fatal(err)
	}
}

func TestSectionQueryUsesExposedTaskIndexes(t *testing.T) {
	path := writeTodoFile(t, "# Work\n\n- [ ] outer\n  > - [ ] quoted\n\n# Later\n\n- [ ] after\n\n# Tail\n")
	var out bytes.Buffer
	if err := WriteList(&out, path, ListOptions{JSON: true, Section: "Later"}); err != nil {
		t.Fatal(err)
	}
	var tasks []Task
	if err := json.Unmarshal(out.Bytes(), &tasks); err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].Text != "after" || tasks[0].Index != 2 {
		t.Fatalf("query selected a hidden or unrelated task: %s", out.String())
	}
}

func TestGuardedCompletionAndRetry(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path := writeTodoFile(t, "---\ntitle: Project\n---\n- [ ] First\n- [ ] Second\n")
	fm, err := markdown.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	token, _ := fm.RevisionToken()
	writes := 0
	s := Service{Out: io.Discard, ExpectedRevision: token, Store: markdown.Store{OnWrite: func(_, _ string) error { writes++; return nil }}}
	if err := s.SetTodoChecked(path, 2, true); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	if err := s.SetTodoChecked(path, 1, true); !errors.Is(err, markdown.ErrFileChanged) {
		t.Fatalf("stale index guard failed: %v", err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) || writes != 1 {
		t.Fatal("stale revision changed file")
	}
	s.ExpectedRevision = ""
	if err := s.SetTodoChecked(path, 2, true); err != nil || writes != 1 {
		t.Fatalf("retry rewrote an already complete task: %v, writes=%d", err, writes)
	}
	if err := s.SetTodoChecked(path, 2, false); err != nil || writes != 2 {
		t.Fatalf("undone failed: %v", err)
	}
}

func TestMissingAndEmptyRevisionsDiffer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.md")
	var out bytes.Buffer
	if err := WriteList(&out, path, ListOptions{WithRevision: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"revision": "missing"`) || !strings.Contains(out.String(), `"tasks": []`) {
		t.Fatal(out.String())
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("query created file")
	}
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	fm, err := markdown.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := fm.RequireRevision("missing"); !errors.Is(err, markdown.ErrFileChanged) {
		t.Fatalf("empty file accepted missing revision: %v", err)
	}
}

func TestMutationRejectsChangeAfterRevisionCheck(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path := writeTodoFile(t, "- [ ] Original\n")
	fm, _ := markdown.ReadFile(path)
	token, _ := fm.RevisionToken()
	external := "- [ ] External\n"
	s := Service{ExpectedRevision: token, Out: io.Discard, Store: markdown.Store{OnRead: func(_, _ string) error {
		return os.WriteFile(path, []byte(external), 0600)
	}}}
	if err := s.SetTodoChecked(path, 1, true); !errors.Is(err, markdown.ErrFileChanged) {
		t.Fatalf("accepted concurrent change: %v", err)
	}
	after, _ := os.ReadFile(path)
	if string(after) != external {
		t.Fatal("overwrote concurrent edit")
	}
}

func TestDueQueryVocabulary(t *testing.T) {
	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)
	week := today.AddDate(0, 0, 7)
	for _, tt := range []struct {
		date   *time.Time
		filter string
		want   bool
	}{
		{nil, "none", true}, {nil, "all", false}, {&today, "today", true},
		{&yesterday, "overdue", true}, {&week, "week", true}, {&yesterday, "week", false},
	} {
		if got := matchesDue(markdown.Todo{DueDate: tt.date}, tt.filter); got != tt.want {
			t.Errorf("filter %s = %v, want %v", tt.filter, got, tt.want)
		}
	}
	for _, due := range []string{"2026-02-30", "tomorow", "2026-1-01"} {
		if err := (ListOptions{Due: due}).Validate(); err == nil {
			t.Errorf("accepted invalid date %q", due)
		}
	}
}
