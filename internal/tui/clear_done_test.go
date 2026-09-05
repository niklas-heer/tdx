package tui

import (
	"github.com/niklas-heer/tdx/internal/markdown"
	"testing"
)

func TestClearDoneRefreshesHeadingAndTaskCaches(t *testing.T) {
	doc := markdown.ParseMarkdown("# One\n\n- [x] Gone\n\n# Two\n\n- [ ] Keep\n")
	m := New("", doc, true, true, -1, testConfig(), testStyles(), "test")
	m.GetHeadings()
	m.GetDocumentTree()
	for _, command := range m.Commands {
		if command.Name == "clear-done" {
			command.Handler(&m)
			break
		}
	}
	if len(m.FileModel.Todos) != 1 || m.FileModel.Todos[0].Text != "Keep" {
		t.Fatal(m.FileModel.Todos)
	}
	headings := m.GetHeadings()
	if len(headings) != 2 || headings[1].BeforeTodoIndex != 0 {
		t.Fatalf("stale headings: %+v", headings)
	}
	tasks := 0
	for _, node := range m.GetDocumentTree().Flat {
		if node.Type == DocNodeTodo {
			tasks++
			if node.Text != "Keep" || node.TodoIndex != 0 {
				t.Fatalf("stale task node: %+v", node)
			}
		}
	}
	if tasks != 1 {
		t.Fatalf("rendered %d tasks", tasks)
	}
}
