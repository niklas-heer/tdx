package markdown

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// The shared rewrite-evaluation corpus remains a production Go regression.
//
//go:embed testdata/parser-corpus.json
var parserCorpus []byte

func TestParserCorpusAndExactCheckboxEdits(t *testing.T) {
	var fixtures []struct {
		Name, Source string
		Markers      []struct {
			Checked bool
			Depth   int
		}
	}
	if err := json.Unmarshal(parserCorpus, &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "tasks.md")
			if err := os.WriteFile(path, []byte(fixture.Source), 0600); err != nil {
				t.Fatal(err)
			}
			model, err := ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if len(model.Todos) != len(fixture.Markers) {
				t.Fatalf("got %d tasks, want %d", len(model.Todos), len(fixture.Markers))
			}
			for i, expected := range fixture.Markers {
				if actual := model.Todos[i]; actual.Checked != expected.Checked || actual.Depth != expected.Depth {
					t.Fatalf("task %d: %+v, expected %+v", i, actual, expected)
				}
				fresh, err := ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := fresh.UpdateTodoItem(i, fresh.Todos[i].Text, !expected.Checked); err != nil {
					t.Fatal(err)
				}
				updated := SerializeMarkdown(fresh)
				if len(updated) != len(fixture.Source) {
					t.Fatalf("task %d: checkbox edit changed source length", i)
				}
				differences := 0
				for j := range len(updated) {
					if updated[j] != fixture.Source[j] {
						differences++
					}
				}
				if differences != 1 {
					t.Fatalf("task %d: checkbox edit changed %d bytes", i, differences)
				}
			}
		})
	}
}
