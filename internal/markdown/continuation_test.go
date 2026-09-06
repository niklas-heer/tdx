package markdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContinueSourceFidelity(t *testing.T) {
	for _, nl := range []string{"\n", "\r\n"} {
		source := strings.ReplaceAll("---\nowner: team # retain\n---\n# First\n\nParagraph\ncontinued.\n\n- [ ] Project **bold**\n  notes with [ref][id]\n\n  ```sh\n  echo keep\n  ```\n  - [x] finished child\n  - [ ] remaining child\n\n# Last\n\n- [ ] unrelated\n\n[id]: https://example.com\n", "\n", nl)
		path := filepath.Join(t.TempDir(), "todo.md")
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
		fm, err := ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		title, err := fm.ContinuationTitle(0)
		if err != nil || title != "Project **bold**" {
			t.Fatalf("title %q: %v", title, err)
		}
		old := fm.Clone()
		idx, err := fm.ContinueTodoSource(0, "Five minutes on **project**")
		if err != nil {
			t.Fatal(err)
		}
		if idx != 4 || len(fm.Todos) != 7 {
			t.Fatalf("index %d tasks %+v", idx, fm.Todos)
		}
		wantPrefix := strings.Replace(source, "[ ] Project", "[x] Project", 1)
		wantPrefix = strings.Replace(wantPrefix, "[ ] remaining child", "[x] remaining child", 1)
		got := SerializeMarkdown(fm)
		if !strings.HasPrefix(got, wantPrefix) {
			t.Fatalf("original bytes changed:\n%q", got)
		}
		copy := got[len(wantPrefix):]
		for _, part := range []string{"notes with [ref][id]", "```sh" + nl + "  echo keep" + nl + "  ```", "- [x] finished child", "- [ ] remaining child"} {
			if !strings.Contains(copy, part) {
				t.Fatalf("copy lost %q: %q", part, copy)
			}
		}
		if !fm.Todos[0].Checked || !fm.Todos[2].Checked || fm.Todos[4].Checked || fm.Todos[6].Checked {
			t.Fatal("incorrect continuation states")
		}
		if err = WriteFile(path, fm); err != nil {
			t.Fatal(err)
		}
		fm.RestoreContent(old)
		if err = WriteFile(path, fm); err != nil {
			t.Fatal(err)
		}
		disk, _ := os.ReadFile(path)
		if string(disk) != source {
			t.Fatal("undo changed original bytes")
		}
	}
}

func TestContinueNestedPromotesSubtree(t *testing.T) {
	source := "- [ ] Parent\n  - [ ] Child\n    notes\n    - [ ] Grandchild\n  - [ ] Sibling\n\nTail prose.\n"
	fm := ParseMarkdown(source)
	_, err := fm.ContinueTodoSource(1, "Child next step")
	if err != nil {
		t.Fatal(err)
	}
	if len(fm.Todos) != 6 || fm.Todos[4].Depth != 0 || fm.Todos[5].Depth != 1 || fm.Todos[5].ParentIndex != 4 {
		t.Fatalf("bad nesting %+v", fm.Todos)
	}
	if fm.Todos[0].Checked || fm.Todos[3].Checked || !fm.Todos[1].Checked || !fm.Todos[2].Checked {
		t.Fatal("retired wrong subtree")
	}
	if !strings.Contains(SerializeMarkdown(fm), "- [ ] Child next step\n  notes\n  - [ ] Grandchild") {
		t.Fatal(SerializeMarkdown(fm))
	}
}

func TestSourceActionsRejectWithoutMutation(t *testing.T) {
	for _, source := range []string{"- [ ] task\n\n```\nunclosed", "> - [ ] quoted\n", "- [ ] task\n\n<script>\nunclosed"} {
		fm := ParseMarkdown(source)
		if _, err := fm.ContinueTodoSource(0, "next"); err == nil {
			t.Fatalf("expected rejection: %q", source)
		}
		if SerializeMarkdown(fm) != source {
			t.Fatal("rejected continuation changed source")
		}
	}
	for _, title := range []string{"", "  ", "line\nbreak", "a\x1bb"} {
		fm := ParseMarkdown("- [ ] original")
		if _, err := fm.ContinueTodoSource(0, title); err == nil {
			t.Fatal("accepted invalid title")
		}
		if _, err := fm.AppendTodoSource(title); err == nil {
			t.Fatal("accepted invalid title")
		}
		if SerializeMarkdown(fm) != "- [ ] original" {
			t.Fatal("invalid input changed source")
		}
	}
}

func TestSourceAppendAndRepeatedContinuation(t *testing.T) {
	for _, source := range []string{"", "- [ ] original", "9. [ ] original\r\n", "# Title\n\n<!-- preserve -->\n\n|a|b|\n|-|-|\n|c|d|\n\n- [ ] original\n"} {
		fm := ParseMarkdown(source)
		idx, err := fm.AppendTodoSource("café 🦀 `code`")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(SerializeMarkdown(fm), source) {
			t.Fatal("append changed source")
		}
		for i := 0; i < 3; i++ {
			idx, err = fm.ContinueTodoSource(idx, "next café 🦀")
			if err != nil {
				t.Fatal(err)
			}
		}
		if fm.Todos[idx].Checked {
			t.Fatal("last continuation checked")
		}
	}
}

func FuzzContinuationPreservesOriginal(f *testing.F) {
	for _, source := range []string{
		"- [ ] simple", "- [ ]\n", "- [ ] **bold**\n  notes\n  - [ ] child\n", "> - [ ] quoted\n", "9) [ ] ordered\r\n    - [ ] nested\r\n", "- [ ] task\n\n  ```\n  fenced\n  ```\n", "- [ ] [reference][id]\n\n[id]: https://example.com\n",
	} {
		f.Add(source, "next step", uint8(0))
	}
	f.Fuzz(func(t *testing.T, source, title string, choice uint8) {
		if len(source) > 16000 || len(title) > 1000 {
			t.Skip()
		}
		fm := ParseMarkdown(source)
		if len(fm.Todos) == 0 {
			return
		}
		index := int(choice) % len(fm.Todos)
		baseline := SerializeMarkdown(fm)
		expected := fm.Clone()
		end := index + 1
		for end < len(fm.Todos) && fm.Todos[end].Depth > fm.Todos[index].Depth {
			end++
		}
		for i := index; i < end; i++ {
			if err := expected.UpdateTodoItem(i, expected.Todos[i].Text, true); err != nil {
				return
			}
		}
		prefix := SerializeMarkdown(expected)
		_, err := fm.ContinueTodoSource(index, title)
		got := SerializeMarkdown(fm)
		if err != nil {
			if got != baseline {
				t.Fatal("failed action mutated original")
			}
		} else if !strings.HasPrefix(got, prefix) || len(fm.Todos) != len(expected.Todos)+end-index {
			t.Fatalf("continuation changed original content: %q", got)
		}
	})
}
