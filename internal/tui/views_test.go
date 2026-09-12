package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/niklas-heer/tdx/internal/config"
	"github.com/niklas-heer/tdx/internal/markdown"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func viewModel(t *testing.T) Model {
	t.Helper()
	m := testModelWithMarkdown("# Work\n## Backend\n- [ ] Ship #work !p1 @due(2030-01-01)\n## Frontend\n- [x] Polish #design\n")
	m.FilePath = filepath.Join(t.TempDir(), "todo.md")
	m.ReadOnly = true
	m.Config().Views = config.NewFileViewStore(t.TempDir())
	return m
}
func viewKey(m Model, key rune) Model {
	result, _ := m.Update(tea.KeyPressMsg{Code: key})
	return result.(Model)
}
func TestSavedViewWorkflow(t *testing.T) {
	m := viewModel(t)
	before := markdown.SerializeMarkdown(&m.FileModel)
	m.FilteredTags = []string{"work"}
	m.FilteredPriorities = []int{1}
	m.FilteredDueDate = "all"
	m.FilterDone = true
	m.SectionFocus = 2
	m.FoldedSections = map[int]bool{2: true}
	m.ShowHeadings = true
	want := m.captureView()
	m.openViews("save")
	m.InputBuffer = "Today"
	m.CursorPos = 5
	m = viewKey(m, tea.KeyEnter)
	if m.ActiveView != "Today" || m.ViewMode != "" {
		t.Fatalf("view not saved: %+v", m)
	}
	m.FilterDone = false
	if !strings.Contains(m.activeViewLabel(), "modified") {
		t.Fatal("modified view not identified")
	}
	m.openViews("load")
	m = viewKey(m, tea.KeyEnter)
	if !m.FilterDone || m.SectionFocus != 2 || !m.FoldedSections[2] || m.FilteredDueDate != want.Due {
		t.Fatal("view not applied")
	}
	if markdown.SerializeMarkdown(&m.FileModel) != before {
		t.Fatal("saving a view edited document")
	}
	m.openViews("delete")
	m = viewKey(m, tea.KeyEnter)
	m = viewKey(m, tea.KeyEsc)
	state, _ := m.Config().Views.Load(m.FilePath)
	if len(state.Views) != 1 {
		t.Fatal("cancel deleted view")
	}
	m = viewKey(m, tea.KeyEnter)
	m = viewKey(m, 'y')
	state, _ = m.Config().Views.Load(m.FilePath)
	if len(state.Views) != 0 || m.ActiveView != "" {
		t.Fatal("delete failed")
	}
}
func TestSavedViewCancelAndOverwrite(t *testing.T) {
	m := viewModel(t)
	m.openViews("save")
	m.InputBuffer = "Keep"
	m.CursorPos = 4
	m = viewKey(m, tea.KeyEnter)
	m.FilterDone = true
	m.openViews("save")
	m.InputBuffer = "Keep"
	m.CursorPos = 4
	m = viewKey(m, tea.KeyEnter)
	if !m.ViewConfirm {
		t.Fatal("overwrite should confirm")
	}
	m = viewKey(m, tea.KeyEsc)
	m = viewKey(m, tea.KeyEsc)
	state, _ := m.Config().Views.Load(m.FilePath)
	if state.Views["Keep"].FilterDone {
		t.Fatal("cancel overwrote saved view")
	}
	if !m.FilterDone {
		t.Fatal("cancel changed current filters")
	}
}
func TestSavedViewRestoreOptInAndHeadingChanges(t *testing.T) {
	m := viewModel(t)
	m.SectionFocus = 2
	m.openViews("save")
	m.InputBuffer = "Backend"
	m.CursorPos = 7
	m = viewKey(m, tea.KeyEnter)
	fm := markdown.ParseMarkdown("# Intro\n- [ ] Intro\n# Work\n## Backend\n- [ ] Ship\n")
	off := New(m.FilePath, fm, true, false, -1, m.Config(), testStyles(), "test")
	if off.SectionFocus != 0 {
		t.Fatal("restore must be opt-in")
	}
	m.Config().ViewsRestore = true
	on := New(m.FilePath, fm, true, false, -1, m.Config(), testStyles(), "test")
	if on.SectionFocus != 3 {
		t.Fatalf("heading context was not remapped: %d", on.SectionFocus)
	}
	removed := New(m.FilePath, markdown.ParseMarkdown("# Other\n- [ ] Item\n"), true, false, -1, m.Config(), testStyles(), "test")
	if removed.SectionFocus != 0 {
		t.Fatal("missing heading should not focus unrelated section")
	}
}
func TestClearSavedViewAndStorageError(t *testing.T) {
	m := viewModel(t)
	m.FilterDone = true
	m.openViews("save")
	m.InputBuffer = "Done"
	m.CursorPos = 4
	m = viewKey(m, tea.KeyEnter)
	m.clearSavedView()
	state, _ := m.Config().Views.Load(m.FilePath)
	if state.Active != "" || m.FilterDone || m.ActiveView != "" {
		t.Fatal("clear failed")
	}
	bad := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(bad, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	m.Config().Views = config.NewFileViewStore(bad)
	m.openViews("save")
	if m.Err == nil || m.ViewMode != "" {
		t.Fatal("storage error should be visible and preserve mode")
	}
}
func TestManualSaveAlias(t *testing.T) {
	m := viewModel(t)
	m.ReadOnly = false
	for _, command := range m.Commands {
		if command.Name == "manual-save" {
			command.Handler(&m)
		}
	}
	if !m.ReadOnly {
		t.Fatal("manual-save alias did not enable temporary edits")
	}
	if !strings.Contains(m.renderStatusBar(), "MANUAL SAVE") {
		t.Fatal("manual-save label missing")
	}
	m = viewKey(m, tea.KeyEnter)
	if !m.FileModel.Todos[0].Checked {
		t.Fatal("temporary toggle unavailable")
	}
	if _, err := os.Stat(m.FilePath); !os.IsNotExist(err) {
		t.Fatal("temporary edit wrote file")
	}
}

func TestSavedViewPaletteAndPipedCancel(t *testing.T) {
	m := viewModel(t)
	m.ProcessPipedInput([]byte(":save-view\nToday\n"))
	if m.ActiveView != "Today" {
		t.Fatalf("palette save failed: %q %v", m.ActiveView, m.Err)
	}
	m.ProcessPipedInput([]byte(":save-view\nCanceled\x1b:views\n"))
	if m.ViewMode != "load" {
		t.Fatalf("escape quit instead of canceling name prompt: %q", m.ViewMode)
	}
	state, _ := m.Config().Views.Load(m.FilePath)
	if len(state.Views) != 1 {
		t.Fatal("cancel stored a view")
	}
}

func TestSavedViewNarrowUnicodeInput(t *testing.T) {
	m := viewModel(t)
	m.TermWidth = 24
	m.TermHeight = 8
	m.openViews("save")
	m.InputBuffer = strings.Repeat("世界é", 40)
	m.CursorPos = len(m.InputBuffer)
	rendered := m.renderViews()
	if !utf8.ValidString(rendered) {
		t.Fatal("narrow render corrupted UTF-8")
	}
	for _, line := range strings.Split(strings.TrimSuffix(rendered, "\n"), "\n") {
		if lipgloss.Width(line) > 24 {
			t.Fatalf("overflow: %d %q", lipgloss.Width(line), line)
		}
	}
	if len(strings.Split(strings.TrimSuffix(rendered, "\n"), "\n")) > 8 {
		t.Fatal("height overflow")
	}
	m = viewKey(m, tea.KeyEnter)
	if m.ActiveView == "" {
		t.Fatal("long Unicode name not saved")
	}
}

func TestMoveSubtreeKeepsSelectionWithDuplicateText(t *testing.T) {
	m := testModelWithMarkdown("- [ ] Same\n  - [ ] Child A\n- [ ] Same\n  - [ ] Child B\n")
	m.ReadOnly = true
	m = viewKey(m, 'm')
	m = viewKey(m, 'j')
	if m.Err != nil {
		t.Fatal(m.Err)
	}
	if m.SelectedIndex != 2 || m.FileModel.Todos[1].Text != "Child B" || m.FileModel.Todos[3].Text != "Child A" {
		t.Fatalf("down subtree selection/content: %d %+v", m.SelectedIndex, m.FileModel.Todos)
	}
	m = viewKey(m, 'k')
	if m.Err != nil {
		t.Fatal(m.Err)
	}
	if m.SelectedIndex != 0 || m.FileModel.Todos[1].Text != "Child A" {
		t.Fatalf("up subtree selection/content: %d %+v", m.SelectedIndex, m.FileModel.Todos)
	}
}

func TestRecentSwitchRestoresCursorUsingDestinationView(t *testing.T) {
	for _, tc := range []struct {
		name        string
		metadata    bool
		savedView   bool
		savedCursor int
		want        int
	}{
		{name: "outgoing filters do not reject saved cursor", savedCursor: 2, want: 2},
		{name: "metadata hides saved cursor", metadata: true, savedCursor: 0, want: 1},
		{name: "saved view hides saved cursor", savedView: true, savedCursor: 0, want: 1},
		{name: "saved view accepts later cursor", savedView: true, savedCursor: 2, want: 2},
		{name: "missing cursor retains visible fallback", metadata: true, savedCursor: -1, want: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			source := filepath.Join(dir, "source.md")
			destination := filepath.Join(dir, "destination.md")
			sourceText := "# Old section\n- [ ] Source #old !p1 @due(2030-01-01)\n"
			destinationText := "# Destination\n- [x] Hidden #new !p2\n- [ ] First visible #new !p2\n- [ ] Saved cursor #new !p2\n"
			if tc.metadata {
				destinationText = "---\nfilter-done: true\n---\n" + destinationText
			}
			for path, content := range map[string]string{source: sourceText, destination: destinationText} {
				if err := os.WriteFile(path, []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			cfg := testConfig()
			cfg.Recent = config.RecentStore{Dir: t.TempDir()}
			cfg.Views = config.NewFileViewStore(t.TempDir())
			cfg.ViewsRestore = tc.savedView
			if tc.savedCursor >= 0 {
				if err := cfg.Recent.SaveFile(destination, tc.savedCursor); err != nil {
					t.Fatal(err)
				}
			}
			if tc.savedView {
				if err := cfg.Views.Save(destination, &config.SavedViews{Active: "Open", Views: map[string]config.SavedView{"Open": {FilterDone: true, Tags: []string{"new"}}}}); err != nil {
					t.Fatal(err)
				}
			}
			m := New(source, markdown.ParseMarkdown(sourceText), true, true, -1, cfg, testStyles(), "test")
			m.FilteredTags = []string{"old"}
			m.FilteredPriorities = []int{1}
			m.FilteredDueDate = "all"
			m.SectionFocus = 1
			m.RecentFiles = []config.RecentFile{{Path: destination}}
			m.RecentFilesMode = true
			m = viewKey(m, tea.KeyEnter)
			if m.Err != nil {
				t.Fatal(m.Err)
			}
			if m.SelectedIndex != tc.want || !m.isTodoVisible(m.SelectedIndex) {
				t.Fatalf("selected %d, want visible %d", m.SelectedIndex, tc.want)
			}
			tree := m.GetDocumentTree()
			if selected := tree.GetSelectedNode(); selected == nil || selected.TodoIndex != tc.want {
				t.Fatalf("tree selection stale: %+v", selected)
			}
			actual, err := os.ReadFile(destination)
			if err != nil || string(actual) != destinationText {
				t.Fatal("switch modified destination", err)
			}
		})
	}
}

func TestSavedViewsPreserveEmptyHeadingLabels(t *testing.T) {
	for _, tc := range []struct {
		name, content string
		focus         int
		fold          int
	}{
		{"blank focus", "#\n- [ ] Task\n", 1, -1},
		{"blank folded heading", "#\n- [ ] Task\n", 0, 0},
		{"blank ancestor", "#\n## Child\n- [ ] Task\n", 2, 0},
		{"repeated blank headings", "#\n- [ ] First\n#\n- [ ] Second\n", 2, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := New(filepath.Join(t.TempDir(), "todo.md"), markdown.ParseMarkdown(tc.content), true, true, -1, testConfig(), testStyles(), "test")
			m.Config().Views = config.NewFileViewStore(t.TempDir())
			m.Config().ViewsRestore = true
			m.SectionFocus = tc.focus
			if tc.fold >= 0 {
				m.FoldedSections = map[int]bool{tc.fold: true}
			}
			expected := m.captureView()
			m.openViews("save")
			m.InputBuffer = "Blank heading view"
			m.CursorPos = len(m.InputBuffer)
			m = viewKey(m, tea.KeyEnter)
			if m.Err != nil {
				t.Fatal(m.Err)
			}
			restored := New(m.FilePath, markdown.ParseMarkdown(tc.content), true, true, -1, m.Config(), testStyles(), "test")
			if restored.Err != nil {
				t.Fatal(restored.Err)
			}
			if restored.SectionFocus != tc.focus || (tc.fold >= 0 && !restored.FoldedSections[tc.fold]) {
				t.Fatalf("heading state not restored: focus=%d folds=%v expected=%+v", restored.SectionFocus, restored.FoldedSections, expected)
			}
			if restored.ActiveView != "Blank heading view" {
				t.Fatal("saved view not restored")
			}
		})
	}
}
