package markdown

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckboxSourceFidelity(t *testing.T) {
	for _, source := range []string{
		"---\nowner: team\n---\n# Work\n\n<div>keep</div>\n\nParagraph one\ncontinued.\n\n| A | B |\n|---|---|\n| a | b |\n\n- [ ] task\n",
		"# Work\r\n\r\n9. [ ] task\r\n   - [X] child\r\n\r\n```md\r\n- [ ] example\r\n```",
		"- [ ] task",
	} {
		path := filepath.Join(t.TempDir(), "tasks.md")
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
		fm, err := ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		snapshot := fm.Clone()
		if err = fm.UpdateTodoItem(0, fm.Todos[0].Text, true); err != nil {
			t.Fatal(err)
		}
		if err = WriteFile(path, fm); err != nil {
			t.Fatal(err)
		}
		got, _ := os.ReadFile(path)
		want := strings.Replace(source, "[ ] task", "[x] task", 1)
		if string(got) != want {
			t.Errorf("toggle changed unrelated source:\ngot %q\nwant %q", got, want)
		}
		fm.RestoreContent(snapshot)
		if err = WriteFile(path, fm); err != nil {
			t.Fatal(err)
		}
		got, _ = os.ReadFile(path)
		if string(got) != source {
			t.Errorf("undo lost source:\ngot %q\nwant %q", got, source)
		}
	}
}

func BenchmarkLoadedCheckbox(b *testing.B) {
	for _, size := range []int{1000, 10000} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			var s strings.Builder
			s.WriteString("# Work\n\n")
			for i := 0; i < size; i++ {
				fmt.Fprintf(&s, "- [ ] Task %d #work !p2 @due(2027-01-12)\n", i)
			}
			fm := ParseMarkdown(s.String())
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				j := i % size
				if err := fm.UpdateTodoItem(j, fm.Todos[j].Text, !fm.Todos[j].Checked); err != nil {
					b.Fatal(err)
				}
				_ = SerializeMarkdown(fm)
			}
		})
	}
}

func TestStructuralEditRetainsOpaqueBlocks(t *testing.T) {
	source := "# Work\n\n<!--keep-->\n\n<div>HTML</div>\n\nFirst line\nsecond line\n\n| A | B |\n|:---|---:|\n| café | `code` |\n\n- [ ] task\n"
	fm := ParseMarkdown(source)
	fm.AddTodoItem("new task", false)
	saved := SerializeMarkdown(fm)
	for _, part := range []string{"<!--keep-->", "<div>HTML</div>", "First line\nsecond line", "| A | B |", "| café | `code` |"} {
		if !strings.Contains(saved, part) {
			t.Fatalf("lost %q in %q", part, saved)
		}
	}
	reopened := ParseMarkdown(saved)
	if len(reopened.Todos) != 2 {
		t.Fatalf("lost tasks: %q", saved)
	}
	clone := fm.Clone()
	if SerializeMarkdown(clone) != saved {
		t.Fatal("snapshot changed normalized structure")
	}
}

func TestCheckboxAfterStructuralEdits(t *testing.T) {
	fm := ParseMarkdown("# Work\n\n- [ ] original\n- [x] second\n")
	if err := fm.DeleteTodoItem(0); err != nil {
		t.Fatal(err)
	}
	fm.AddTodoItem("new", false)
	if err := fm.UpdateTodoItem(0, "second", false); err != nil {
		t.Fatal(err)
	}
	result := ParseMarkdown(SerializeMarkdown(fm))
	if len(result.Todos) != 2 || result.Todos[0].Checked || result.Todos[1].Checked || result.Todos[0].Text != "second" {
		t.Fatal("stale checkbox index used after mutation")
	}
}

func TestCloneRetainsPendingReorder(t *testing.T) {
	fm := ParseMarkdown("# Work\n\n- [x] first\n- [ ] second\n")
	fm.Todos[0], fm.Todos[1] = fm.Todos[1], fm.Todos[0]
	RebuildFileStructure(fm)
	clone := fm.Clone()
	result := ParseMarkdown(SerializeMarkdown(clone))
	if result.Todos[0].Text != "second" || result.Todos[0].Checked || result.Todos[1].Text != "first" || !result.Todos[1].Checked {
		t.Fatal("snapshot discarded an unsaved reorder")
	}
	if string(fm.GetAST().Source) != "# Work\n\n- [x] first\n- [ ] second\n" {
		t.Fatal("snapshot changed original AST")
	}
}

func TestInvalidStructuralEditDoesNotChangeSource(t *testing.T) {
	source := "- [ ] task\r\n<!-- keep -->"
	for _, edit := range []func(*ASTDocument) error{
		func(doc *ASTDocument) error { return doc.InsertTodoAfter(50, "invalid", false) },
		func(doc *ASTDocument) error { return doc.IndentTodo(0) },
		func(doc *ASTDocument) error { return doc.OutdentTodo(0) },
		func(doc *ASTDocument) error { return doc.DeleteTodo(50) },
	} {
		doc, _ := ParseAST(source)
		if err := edit(doc); err == nil {
			t.Fatal("expected rejected edit")
		}
		if SerializeAST(doc) != source {
			t.Fatal("rejected edit changed source")
		}
	}
}
