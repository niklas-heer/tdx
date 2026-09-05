package tui

import (
	"github.com/niklas-heer/tdx/internal/markdown"
	"testing"
)

func TestExplicitFlagsOverrideFileMetadata(t *testing.T) {
	fm := markdown.ParseMarkdown("# Tasks\n\n- [ ] One\n")
	no, limit := false, 2
	fm.Metadata = &markdown.Metadata{ReadOnly: &no, ShowHeadings: &no, MaxVisible: &limit}
	cfg := testConfig()
	cfg.ReadOnlyFlag, cfg.ShowHeadingsFlag = true, true
	cfg.HasMaxVisibleFlag, cfg.MaxVisibleFlag = true, 7
	m := New("tasks.md", fm, true, true, 7, cfg, nil, "test")
	if !m.ReadOnly || !m.ShowHeadings || m.MaxVisibleOverride != 7 {
		t.Fatalf("file metadata overrode explicit flags: %+v", m)
	}
}

func TestPipedInputControlsAndOverlayEscape(t *testing.T) {
	fm := markdown.ParseMarkdown("# Tasks\n\n- [ ] Alpha !p1 @due(2028-01-01)\n")
	m := New("tasks.md", fm, true, false, -1, testConfig(), nil, "test")
	m.ProcessPipedInput([]byte("e\x01prefix \x05!\rp\x1bD\x1b \x1b"))
	if m.FileModel.Todos[0].Text != "prefix Alpha !p1 @due(2028-01-01)!" || !m.FileModel.Todos[0].Checked {
		t.Fatalf("script control decoding/mode close: %+v", m.FileModel.Todos)
	}
}

func TestInvalidIndentDoesNotConsumeUndo(t *testing.T) {
	fm := markdown.ParseMarkdown("# Tasks\n\n- [ ] First\n- [ ] Second\n")
	m := New("tasks.md", fm, true, false, -1, testConfig(), nil, "test")
	m.ProcessPipedInput([]byte(" "))
	m.ReadOnly = false
	m.ProcessPipedInput([]byte("\t")) // First item cannot be indented.
	m.ReadOnly = true
	m.ProcessPipedInput([]byte("u"))
	if m.FileModel.Todos[0].Checked {
		t.Fatal("invalid indent consumed the undo of the toggle")
	}
}
