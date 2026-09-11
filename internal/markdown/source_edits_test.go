package markdown

import (
	"strings"
	"testing"
)

func TestTextEditPreservesReferenceDefinitionsAndDocumentBytes(t *testing.T) {
	for _, source := range []string{
		"Title\n=====\n\nSee [guide][ref].\n\n- [ ] before\n\n[ref]: <https://example.com/a>\n  \"Keep title\"\n",
		"# Work\r\n\r\n9. [X] before\r\n   - [ ] child\r\n\r\n[ref]: /guide \"title\"",
		"> - [ ] before\n>   continuation\n\n[ref]: /guide\n",
	} {
		doc, _ := ParseAST(source)
		if err := doc.UpdateTodoText(0, "after **bold** [ref]"); err != nil {
			t.Fatal(err)
		}
		want := strings.Replace(source, "before", "after **bold** [ref]", 1)
		want = strings.Replace(want, "\n>   continuation", "", 1)
		if got := SerializeAST(doc); got != want {
			t.Fatalf("got %q; want %q", got, want)
		}
	}
}

func TestStructuralEditsPreserveRichDocument(t *testing.T) {
	prefix := "Title\n=====\n\n<!-- keep exact -->\n\n| A | B |\n| :--- | ---: |\n| café | `x` |\n\n~~~md\n- [ ] example\n~~~\n\nSee [guide][ref].\n\n"
	suffix := "\n[ref]: <https://example.com>\n  \"Original title\"\n\n<div>unchanged</div>"
	for _, tasks := range []string{
		"- [ ] Parent !p2\n  continuation with [guide][ref]\n\n  Another paragraph.\n\n  ```go\n  fmt.Println(\"keep\")\n  ```\n\n  - [ ] Child !p3\n  - [x] Done child !p1\n- [ ] Sibling !p1\n",
		"9. [ ] Parent !p2\n   continuation with [guide][ref]\n   1) [ ] Child !p3\n   2) [x] Done child !p1\n10. [ ] Sibling !p1\n",
	} {
		for name, op := range map[string]func(*FileModel) error{
			"add":           func(f *FileModel) error { return f.AddTodoItemChecked("Added", false) },
			"insert":        func(f *FileModel) error { _, err := f.InsertTodoItemAfterChecked(0, "Inserted", false); return err },
			"delete-parent": func(f *FileModel) error { return f.DeleteTodoItem(0) },
			"delete-child":  func(f *FileModel) error { return f.DeleteTodoItem(1) },
			"move-parent":   func(f *FileModel) error { return f.MoveTodoItem(0, 3) },
			"move-child":    func(f *FileModel) error { return f.MoveTodoItem(1, 3) },
			"indent":        func(f *FileModel) error { return f.IndentTodoItem(3) },
			"outdent":       func(f *FileModel) error { return f.OutdentTodoItem(1) },
			"sort":          func(f *FileModel) error { return f.SortTodos(func(a, b Todo) bool { return a.Priority < b.Priority }) },
			"rename":        func(f *FileModel) error { return f.RenameHeading(0, "Renamed") },
			"new-section":   func(f *FileModel) error { _, err := f.CreateHeading(-1, 2, "New"); return err },
			"section-task":  func(f *FileModel) error { _, err := f.AddTodoInSection(0, "New"); return err },
		} {
			t.Run(name, func(t *testing.T) {
				f := ParseMarkdown(prefix + tasks + suffix)
				if err := op(f); err != nil {
					t.Fatal(err)
				}
				got := SerializeMarkdown(f)
				if !strings.Contains(got, suffix) {
					t.Fatalf("lost suffix: %q", got)
				}
				if name != "rename" && name != "section-task" && !strings.HasPrefix(got, prefix) {
					t.Fatalf("changed prefix: %q", got)
				}
				if SerializeMarkdown(f.Clone()) != got {
					t.Fatal("clone changes source")
				}
			})
		}
	}
}

func TestUnsafeSourceOperationsAreAtomic(t *testing.T) {
	for _, tc := range []struct {
		source string
		op     func(*FileModel) error
	}{
		{"> - [ ] quoted\n\n[ref]: /x\n", func(f *FileModel) error { return f.DeleteTodoItem(0) }},
		{"- [ ] Parent\n  - [ ] Child\n", func(f *FileModel) error { return f.MoveTodoItem(0, 1) }},
		{"- [ ] task\n", func(f *FileModel) error { return f.UpdateTodoItem(0, "injected\n- [ ] extra", false) }},
	} {
		f := ParseMarkdown(tc.source)
		if err := tc.op(f); err == nil {
			t.Fatal("expected rejected source edit")
		}
		if SerializeMarkdown(f) != tc.source {
			t.Fatal("rejected operation changed bytes")
		}
	}
}

func TestSortPreservesTaskSubtreesAndBodies(t *testing.T) {
	source := "# Work\n\n- [x] Parent\n  body **bold**\n  - [x] Child done\n  - [ ] Child open\n- [ ] Open\n\n[ref]: /kept\n"
	f := ParseMarkdown(source)
	if err := f.SortTodos(func(a, b Todo) bool { return !a.Checked && b.Checked }); err != nil {
		t.Fatal(err)
	}
	want := "# Work\n\n- [ ] Open\n- [x] Parent\n  body **bold**\n  - [ ] Child open\n  - [x] Child done\n\n[ref]: /kept\n"
	if got := SerializeMarkdown(f); got != want {
		t.Fatalf("got %q; want %q", got, want)
	}
}

func TestStructuralBoundaryPreservesIndentedOutsideParagraph(t *testing.T) {
	for _, source := range []string{
		"9. [ ] task\n\n  This is outside the ordered list.\n",
		"- [ ] task\n\n [ref]: /outside\n",
	} {
		f := ParseMarkdown(source)
		if err := f.DeleteTodoItem(0); err != nil {
			t.Fatal(err)
		}
		want := source[strings.Index(source, "\n")+1:]
		if !strings.HasSuffix(SerializeMarkdown(f), want) {
			t.Fatalf("deleted unrelated indented source: %q", SerializeMarkdown(f))
		}
	}
}

func TestOpaqueQuotedCheckboxDoesNotShiftTaskIndexes(t *testing.T) {
	source := "# Work\n\n- [ ] outer\n  > - [ ] quoted example\n\n# Later\n\n- [ ] after\n\n# Tail\n"
	for _, first := range []string{"headings", "todos", "edit"} {
		t.Run(first, func(t *testing.T) {
			doc, _ := ParseAST(source)
			if first == "headings" {
				headings := doc.ExtractHeadings()
				if len(headings) != 3 || headings[1].BeforeTodoIndex != 1 || headings[2].BeforeTodoIndex != 2 {
					t.Fatalf("heading task boundaries: %+v", headings)
				}
			}
			if first == "todos" {
				if len(doc.ExtractTodos()) != 2 {
					t.Fatal("opaque example became an indexed task")
				}
			}
			if err := doc.UpdateTodoText(1, "changed"); err != nil {
				t.Fatal(err)
			}
			if got := SerializeAST(doc); got != strings.Replace(source, "[ ] after", "[ ] changed", 1) {
				t.Fatalf("edit selected the wrong checkbox: %q", got)
			}
			if err := doc.ToggleTodo(1); err != nil {
				t.Fatal(err)
			}
			if got := SerializeAST(doc); got != strings.Replace(source, "[ ] after", "[x] changed", 1) {
				t.Fatalf("toggle selected the wrong checkbox: %q", got)
			}
			if err := doc.InsertTodoAfter(1, "new", false); err != nil {
				t.Fatal(err)
			}
			if err := doc.DeleteTodo(1); err != nil {
				t.Fatal(err)
			}
			if todos := doc.ExtractTodos(); len(todos) != 2 || todos[1].Text != "new" {
				t.Fatalf("stale checkbox cache: %+v", todos)
			}
			if !strings.Contains(SerializeAST(doc), "> - [ ] quoted example") {
				t.Fatal("changed opaque quoted example")
			}
		})
	}
}

func TestEmptyHeadingOperationsRejectWithoutMutation(t *testing.T) {
	source := "# Work\n\n- [ ] task\n\n#\n"
	for _, op := range []func(*FileModel) error{
		func(f *FileModel) error { _, err := f.AddTodoInSection(1, "new"); return err },
		func(f *FileModel) error { _, err := f.CreateHeading(0, 1, "new"); return err },
		func(f *FileModel) error { _, err := f.CreateHeading(1, 1, "new"); return err },
		func(f *FileModel) error { return f.RenameHeading(1, "new") },
	} {
		f := ParseMarkdown(source)
		if err := op(f); err == nil {
			t.Fatal("expected missing source location error")
		}
		if got := SerializeMarkdown(f); got != source {
			t.Fatalf("changed source: %q", got)
		}
	}
}
