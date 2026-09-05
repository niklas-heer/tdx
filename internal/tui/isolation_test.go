package tui

import (
	"path/filepath"
	"testing"

	"github.com/niklas-heer/tdx/internal/editor"
	"github.com/niklas-heer/tdx/internal/markdown"
)

func TestModelsKeepConfigurationAndHistorySeparate(t *testing.T) {
	cfg, styles := testConfig(), testStyles()
	cfg.AvailableThemes = []string{"original"}
	cfg.Display.CheckSymbol = "a"
	pathA, pathB := filepath.Join(t.TempDir(), "a.md"), filepath.Join(t.TempDir(), "b.md")
	fmA, err := markdown.ReadFile(pathA)
	if err != nil {
		t.Fatal(err)
	}
	fmB, err := markdown.ReadFile(pathB)
	if err != nil {
		t.Fatal(err)
	}
	writesA, writesB := 0, 0
	cfg.Store.OnWrite = func(string, string) error { writesA++; return nil }
	a := New(pathA, fmA, false, false, -1, cfg, styles, "a")
	cfg.Store.OnWrite = func(string, string) error { writesB++; return nil }
	cfg.Display.CheckSymbol = "b"
	cfg.AvailableThemes[0] = "changed"
	styles.Cyan = func(string) string { return "changed" }
	b := New(pathB, fmB, false, false, -1, cfg, styles, "b")
	if a.Config().Display.CheckSymbol != "a" || a.AvailableThemes[0] != "original" || a.Styles().Cyan("plain") == "changed" {
		t.Fatal("caller mutation changed an existing model")
	}
	if err := a.applyAction(editor.Action{Kind: editor.Add, Text: "first"}); err != nil {
		t.Fatal(err)
	}
	if err := b.applyAction(editor.Action{Kind: editor.Add, Text: "second"}); err != nil {
		t.Fatal(err)
	}
	a.writeIfPersist()
	b.writeIfPersist()
	if a.Err != nil || b.Err != nil || writesA != 1 || writesB != 1 {
		t.Fatalf("history isolation: %v %v %d %d", a.Err, b.Err, writesA, writesB)
	}
	a.saveHistory()
	if err := a.applyAction(editor.Action{Kind: editor.Toggle, Index: 0}); err != nil {
		t.Fatal(err)
	}
	if !a.history.Undo(&a.FileModel) || a.FileModel.Todos[0].Checked || b.history.Len() != 0 {
		t.Fatal("undo crossed models")
	}
}
