package markdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnrelatedTogglePreservesInlineMarkdownOnDisk(t *testing.T) {
	isolateSaveLocks(t)
	path := filepath.Join(t.TempDir(), "todo.md")
	content := strings.Join([]string{
		"# Todos",
		"",
		"- [ ] toggle me",
		"- [ ] bare https://example.com/path?q=1#fragment",
		"- [ ] www www.example.com/path",
		"- [ ] email user@example.com",
		"- [ ] angle <https://example.com/path>",
		"- [ ] angle email <user@example.com>",
		`- [ ] html <span data-x="1">value</span>`,
		"- [ ] image ![alt](https://example.com/image.png)",
		"- [ ] strike ~~removed~~",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	fm, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := fm.UpdateTodoItem(0, fm.Todos[0].Text, true); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(path, fm); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(content, "- [ ] toggle me", "- [x] toggle me", 1)
	if string(got) != want {
		t.Fatalf("saved markdown changed unrelated inline content:\n--- got ---\n%s--- want ---\n%s", got, want)
	}
}

func TestExtractTodosPreservesAutolinksAndRawHTML(t *testing.T) {
	content := strings.Join([]string{
		"- [ ] angle <https://example.com/path>",
		"- [ ] email <user@example.com>",
		`- [ ] html <kbd data-key="x">X</kbd>`,
		"",
	}, "\n")
	fm := ParseMarkdown(content)
	want := []string{
		"angle <https://example.com/path>",
		"email <user@example.com>",
		`html <kbd data-key="x">X</kbd>`,
	}
	if len(fm.Todos) != len(want) {
		t.Fatalf("todo count = %d, want %d", len(fm.Todos), len(want))
	}
	for i, expected := range want {
		if fm.Todos[i].Text != expected {
			t.Errorf("todo %d text = %q, want %q", i, fm.Todos[i].Text, expected)
		}
	}
}

func TestUpdateTodoTextPreservesLinkLikeText(t *testing.T) {
	tests := []string{
		"bare http://example.com/path",
		"secure https://example.com/path?q=1#fragment",
		"ftp ftp://example.com/file.txt",
		"www www.example.com/path",
		"email user@example.com",
		"angle <https://example.com/path>",
		"angle email <user@example.com>",
		`html <span class="value">text</span>`,
	}
	for _, text := range tests {
		t.Run(text, func(t *testing.T) {
			fm := ParseMarkdown("# Todos\n\n- [ ] placeholder\n")
			if err := fm.UpdateTodoItem(0, text, false); err != nil {
				t.Fatal(err)
			}
			if fm.Todos[0].Text != text {
				t.Fatalf("updated text = %q, want %q", fm.Todos[0].Text, text)
			}
			roundTripped := ParseMarkdown(SerializeMarkdown(fm))
			if roundTripped.Todos[0].Text != text {
				t.Fatalf("round-tripped text = %q, want %q", roundTripped.Todos[0].Text, text)
			}
		})
	}
}
