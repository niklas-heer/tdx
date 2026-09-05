package editor

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/niklas-heer/tdx/internal/markdown"
)

func TestActionsAndUndo(t *testing.T) {
	doc := markdown.ParseMarkdown("# Work\n\n- [ ] First\n- [ ] Second\n")
	var history History
	actions := []Action{
		{Kind: Toggle, Index: 0}, {Kind: Edit, Index: 1, Text: "Käse #work"},
		{Kind: Insert, Index: 0, Text: "Inserted"}, {Kind: Move, Index: 2, Target: 1},
		{Kind: RenameHeading, Index: 0, Text: "Projects"}, {Kind: CreateHeading, Index: 0, Level: 2, Text: "Empty"},
		{Kind: AddInSection, Index: 1, Text: "New project task"}, {Kind: Delete, Index: 0},
	}
	for _, action := range actions {
		before := markdown.SerializeMarkdown(doc)
		history.Push(doc)
		if _, err := Apply(doc, action); err != nil {
			t.Fatalf("%s: %v", action.Kind, err)
		}
		if !history.Undo(doc) || markdown.SerializeMarkdown(doc) != before {
			t.Fatalf("%s did not undo", action.Kind)
		}
		if _, err := Apply(doc, action); err != nil {
			t.Fatal(err)
		}
	}
}

func TestInvalidActionsDoNotMutate(t *testing.T) {
	doc := markdown.ParseMarkdown("# Work\n\n- [ ] First\n- [x] Second\n")
	before := markdown.SerializeMarkdown(doc)
	for _, action := range []Action{
		{Kind: Toggle, Index: -1}, {Kind: Edit, Index: 20}, {Kind: Delete, Index: 20},
		{Kind: Move, Index: 0, Target: 20}, {Kind: MoveToPosition, Index: 0, Target: -1},
		{Kind: Indent, Index: 0}, {Kind: Outdent, Index: 0}, {Kind: Insert, Index: -1},
		{Kind: RenameHeading, Index: 0, Text: ""}, {Kind: CreateHeading, Index: 0, Level: 7, Text: "No"},
		{Kind: AddInSection, Index: 10, Text: "No"}, {Kind: "unknown"},
	} {
		if _, err := Apply(doc, action); err == nil {
			t.Errorf("accepted %+v", action)
		}
		if markdown.SerializeMarkdown(doc) != before {
			t.Fatalf("%+v changed document on error", action)
		}
	}
}

func TestHistoryRetainsCurrentDiskRevision(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.md")
	if err := os.WriteFile(path, []byte("- [ ] First\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	doc, err := markdown.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var history History
	history.Push(doc)
	if _, err := Apply(doc, Action{Kind: Toggle, Index: 0}); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(path, doc); err != nil {
		t.Fatal(err)
	}
	if !history.Undo(doc) {
		t.Fatal("no undo")
	}
	if err := markdown.WriteFile(path, doc); err != nil {
		t.Fatalf("undo restored stale revision: %v", err)
	}
}

func TestHistoryBoundAndMetadataIsolation(t *testing.T) {
	doc := markdown.ParseMarkdown("- [ ] First #work\n")
	value := true
	doc.Metadata.ReadOnly = &value
	var history History
	for i := 0; i < 150; i++ {
		history.Push(doc)
	}
	if history.Len() != HistoryLimit {
		t.Fatal(history.Len())
	}
	*doc.Metadata.ReadOnly = false
	doc.Todos[0].Tags[0] = "changed"
	if !history.Undo(doc) || !*doc.Metadata.ReadOnly || doc.Todos[0].Tags[0] != "work" {
		t.Fatal("snapshot aliases live document")
	}
	history.Clear()
	if history.Undo(doc) {
		t.Fatal("clear retained undo")
	}
}

func BenchmarkDocumentToggle(b *testing.B) {
	for _, count := range []int{100, 1000, 10000} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			source := "# Tasks\n\n" + strings.Repeat("- [ ] Task #backend !p2 @due(2026-10-01)\n", count)
			b.ReportAllocs()
			b.SetBytes(int64(len(source)))
			for b.Loop() {
				doc := markdown.ParseMarkdown(source)
				if _, err := Apply(doc, Action{Kind: Toggle, Index: count / 2}); err != nil {
					b.Fatal(err)
				}
				_ = markdown.SerializeMarkdown(doc)
			}
		})
	}
}

func TestParitySubtreesAndBlockOwnership(t *testing.T) {
	source := "# Work\n\n- [x] parent !p3\n  - [ ] child !p2\n\n  paragraph retained\n\n  1. [ ] other child\n- [ ] sibling !p1\n\n<div>retained</div>\n"
	t.Run("sort keeps children and paragraphs", func(t *testing.T) {
		doc := markdown.ParseMarkdown(source)
		if _, err := Apply(doc, Action{Kind: SortPriority}); err != nil {
			t.Fatal(err)
		}
		after := markdown.ParseMarkdown(markdown.SerializeMarkdown(doc))
		if len(after.Todos) != 4 || after.Todos[0].Text != "sibling !p1" || after.Todos[2].ParentIndex != 1 || after.Todos[3].ParentIndex != 1 {
			t.Fatalf("corrupted subtrees: %+v", after.Todos)
		}
		if !strings.Contains(markdown.SerializeMarkdown(doc), "\n  paragraph retained\n") {
			t.Fatal("paragraph moved or flattened")
		}
	})
	t.Run("delete promotes every child list", func(t *testing.T) {
		doc := markdown.ParseMarkdown(source)
		if _, err := Apply(doc, Action{Kind: Delete, Index: 0}); err != nil {
			t.Fatal(err)
		}
		if len(doc.Todos) != 3 || doc.Todos[0].Text != "child !p2" || doc.Todos[1].Text != "other child" || doc.Todos[1].Depth != 0 {
			t.Fatalf("lost children: %+v", doc.Todos)
		}
	})
	t.Run("moving into descendant rejects without mutation", func(t *testing.T) {
		for _, kind := range []Kind{Move, MoveToPosition} {
			doc := markdown.ParseMarkdown(source)
			if _, err := Apply(doc, Action{Kind: kind, Index: 0, Target: 1}); err == nil {
				t.Fatal("accepted cyclic move")
			}
			if got := markdown.SerializeMarkdown(doc); got != source {
				t.Fatalf("rejection changed source: %s", got)
			}
		}
	})
	t.Run("insert selects sibling after subtree", func(t *testing.T) {
		doc := markdown.ParseMarkdown(source)
		index, err := Apply(doc, Action{Kind: Insert, Index: 0, Text: "inserted"})
		if err != nil || index != 3 || doc.Todos[index].Text != "inserted" {
			t.Fatalf("index %d: %v %+v", index, err, doc.Todos)
		}
	})
	t.Run("add appends at root after trailing content", func(t *testing.T) {
		doc := markdown.ParseMarkdown(source)
		if _, err := Apply(doc, Action{Kind: Add, Text: "appended"}); err != nil {
			t.Fatal(err)
		}
		after := markdown.SerializeMarkdown(doc)
		if strings.Index(after, "appended") < strings.Index(after, "<div>retained</div>") || doc.Todos[len(doc.Todos)-1].Depth != 0 {
			t.Fatal(after)
		}
	})
}
