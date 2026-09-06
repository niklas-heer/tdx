package tui

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/niklas-heer/tdx/internal/markdown"
)

func rzKey(m Model, key string) Model {
	var msg tea.KeyPressMsg
	switch key {
	case "enter":
		msg.Code = tea.KeyEnter
	case "esc":
		msg.Code = tea.KeyEsc
	case "space":
		msg.Code = ' '
	default:
		msg.Text = key
	}
	next, _ := m.Update(msg)
	return next.(Model)
}
func rzCommand(m Model, name string) Model {
	m = rzKey(m, ":")
	m = rzKey(m, name)
	return rzKey(m, "enter")
}
func rzDisk(t *testing.T, m Model) string {
	t.Helper()
	data, err := os.ReadFile(m.FilePath)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
func rzAll(m Model) Model {
	for m.rezero.Phase == "review" {
		m = rzKey(m, "space")
	}
	return m
}

func TestRezeroWholeReviewAndWorkOrder(t *testing.T) {
	source := "# One\n\n- [ ] older #a !p1\n  - [ ] child\n- [x] completed\n\n# Two\n\n- [ ] newest #b\n"
	m := persistentNestedModel(t, source)
	m.FilterDone = true
	m.FilteredTags = []string{"absent"}
	m.FilteredPriorities = []int{3}
	m.FilteredDueDate = "today"
	m.SectionFocus = 1
	m.FoldedSections = map[int]bool{0: true}
	m = rzCommand(m, "rezero")
	if m.rezero.Phase != "review" || m.SelectedIndex != 3 || !reflect.DeepEqual(m.rezero.Review, []int{3, 1, 0}) {
		t.Fatalf("bad review %+v: %v", m.rezero, m.Err)
	}
	view := ansi.Strip(m.View().Content)
	for _, title := range []string{"older", "child", "completed", "newest"} {
		if !strings.Contains(view, title) {
			t.Fatalf("hidden %s: %s", title, view)
		}
	}
	m = rzKey(m, "space")
	if m.SelectedIndex != 1 || m.rezero.Phase != "review" || rzDisk(t, m) != source {
		t.Fatal("selection completed task or skipped review")
	}
	m = rzKey(m, "enter")
	m = rzKey(m, "space")
	if m.rezero.Phase != "work" || m.SelectedIndex != 3 {
		t.Fatal("wrong work order")
	}
	m = rzKey(m, "space")
	if m.SelectedIndex != 0 || !m.FileModel.Todos[3].Checked {
		t.Fatal("work did not advance")
	}
	m = rzKey(m, "space")
	if m.rezero.Phase != "complete" {
		t.Fatal("round not finished")
	}
	m = rzKey(m, "enter")
	if !reflect.DeepEqual(m.rezero.Review, []int{1}) {
		t.Fatal("passed task omitted from next round")
	}
	m = rzKey(m, "esc")
	if m.rezero.Phase != "" || !m.FilterDone || m.FilteredTags[0] != "absent" || m.FilteredPriorities[0] != 3 || m.SectionFocus != 1 || !m.FoldedSections[0] || m.FilteredDueDate != "today" {
		t.Fatal("normal view not restored")
	}
}

func TestRezeroInlineContinueCancelSaveUndo(t *testing.T) {
	source := "- [ ] parent\n  notes\n  - [ ] child\n- [ ] last\n"
	m := persistentNestedModel(t, source)
	saves := 0
	m.config.Store.OnWrite = func(string, string) error { saves++; return nil }
	m = rzCommand(m, "rezero")
	m = rzAll(m)
	m = rzKey(m, "r")
	if !m.EditMode || m.InputBuffer != "last" {
		t.Fatal("continuation not inline")
	}
	m = rzKey(m, " changed")
	m = rzKey(m, "esc")
	if rzDisk(t, m) != source || saves != 0 || m.history.Len() != 0 {
		t.Fatal("cancel changed content/history")
	}
	m = rzKey(m, "r")
	m = rzKey(m, " café 🦀")
	m = rzKey(m, "enter")
	if m.Err != nil {
		t.Fatal(m.Err)
	}
	if saves != 1 || m.history.Len() != 1 || len(m.FileModel.Todos) != 4 || m.SelectedIndex != 1 || m.FileModel.Todos[3].Text != "last café 🦀" {
		t.Fatalf("wrong continuation saves=%d, model=%+v", saves, m.FileModel.Todos)
	}
	m = rzKey(m, "u")
	if rzDisk(t, m) != source || m.SelectedIndex != 2 || m.rezero.Phase != "work" || len(m.rezero.Work) != 3 || m.history.Len() != 0 {
		t.Fatal("undo did not restore document and round")
	}
	// Nested notes and child states remain in the continuation; originals retire together.
	m = rzKey(m, "p")
	m = rzKey(m, "p")
	m = rzKey(m, "r")
	m = rzKey(m, "enter")
	if m.Err != nil {
		t.Fatal(m.Err)
	}
	if !m.FileModel.Todos[0].Checked || !m.FileModel.Todos[1].Checked || !strings.Contains(rzDisk(t, m), "- [ ] parent\n  notes\n  - [ ] child") {
		t.Fatal("subtree continuation lost content")
	}
}

func TestRezeroNewTasksWaitAndUndoRetainsRound(t *testing.T) {
	m := persistentNestedModel(t, "- [ ] initial\n")
	m = rzCommand(m, "rezero")
	m = rzKey(m, "n")
	m = rzKey(m, "new task #new !p2")
	m = rzKey(m, "enter")
	if m.Err != nil {
		t.Fatal(m.Err)
	}
	if len(m.rezero.Review) != 1 || m.SelectedIndex != 0 {
		t.Fatal("new task entered ongoing review")
	}
	m = rzKey(m, "space")
	m = rzKey(m, "space")
	m = rzKey(m, "enter")
	if !reflect.DeepEqual(m.rezero.Review, []int{1}) {
		t.Fatal("new task omitted from next round")
	}
	m = rzKey(m, "u")
	if m.rezero.Phase != "work" || m.SelectedIndex != 0 || m.FileModel.Todos[0].Checked {
		t.Fatal("undo across rounds failed")
	}
	m = rzKey(m, "u")
	if len(m.FileModel.Todos) != 1 || m.rezero.Phase != "review" {
		t.Fatal("new task undo failed")
	}
}

func TestRezeroConflictRetainsCandidateAndDoesNotAdvance(t *testing.T) {
	for _, recovery := range []string{"reload", "force-save"} {
		t.Run(recovery, func(t *testing.T) {
			m := persistentNestedModel(t, "- [ ] original\n")
			m = rzCommand(m, "rezero")
			m = rzAll(m)
			m = rzKey(m, "r")
			m = rzKey(m, " next")
			external := "- [ ] external\n"
			if err := os.WriteFile(m.FilePath, []byte(external), 0600); err != nil {
				t.Fatal(err)
			}
			// A watcher event must not replace the baseline during input.
			next, _ := m.Update(FileChangedMsg{})
			m = next.(Model)
			m = rzKey(m, "enter")
			if !errors.Is(m.Err, markdown.ErrFileChanged) || !m.ConflictPending || m.rezero.Phase != "work" || !m.EditMode || m.history.Len() != 0 {
				t.Fatalf("bad conflict state: %v", m.Err)
			}
			candidate := m.ConflictLocalContent
			if !strings.Contains(candidate, "- [x] original") || !strings.Contains(candidate, "- [ ] original next") || rzDisk(t, m) != external {
				t.Fatal("candidate or external bytes lost")
			}
			m = rzKey(m, "esc") // close diff
			m = rzKey(m, "esc") // cancel inline field (candidate stays retained)
			m = rzCommand(m, recovery)
			if m.Err != nil || m.rezero.Phase != "" || m.ConflictPending {
				t.Fatalf("recovery failed: %v", m.Err)
			}
			want := external
			if recovery == "force-save" {
				want = candidate
			}
			if rzDisk(t, m) != want {
				t.Fatal("wrong recovery content")
			}
		})
	}
}

func TestRezeroReadOnlyAndCommandGuard(t *testing.T) {
	source := "- [ ] first\n- [ ] last\n"
	m := persistentNestedModel(t, source)
	m.ReadOnly = true
	m = rzCommand(m, "rezero")
	m = rzAll(m)
	for _, key := range []string{"space", "r", "n", "u"} {
		m = rzKey(m, key)
		if m.Err == nil {
			t.Fatalf("read-only %s accepted", key)
		}
		m = rzKey(m, "x") // dismiss error
	}
	m = rzCommand(m, "force-save")
	if m.Err == nil || rzDisk(t, m) != source {
		t.Fatal("read-only force-save allowed")
	}
	m = rzKey(m, "x")
	m = rzCommand(m, "sort-priority")
	if m.Err == nil || rzDisk(t, m) != source || m.rezero.Phase != "work" {
		t.Fatal("command changed round")
	}
}

func TestRezeroPostCommitWarningAdvancesAndCanUndo(t *testing.T) {
	m := persistentNestedModel(t, "- [ ] task\n")
	m.config.Store.OnWrite = func(string, string) error { return errors.New("history offline") }
	m = rzCommand(m, "rezero")
	m = rzAll(m)
	m = rzKey(m, "space")
	var post *markdown.PostCommitError
	if !errors.As(m.Err, &post) || m.rezero.Phase != "complete" || !m.FileModel.Todos[0].Checked {
		t.Fatalf("committed warning mishandled: %v", m.Err)
	}
	m = rzKey(m, "x")
	m.config.Store.OnWrite = nil
	m = rzKey(m, "u")
	if m.Err != nil || m.FileModel.Todos[0].Checked || m.rezero.Phase != "work" {
		t.Fatalf("undo after warning: %v", m.Err)
	}
}

func TestRezeroEmptyBackAndNarrowStatus(t *testing.T) {
	for _, source := range []string{"", "- [x] complete\n"} {
		m := persistentNestedModel(t, source)
		m = rzCommand(m, "rezero")
		if m.rezero.Phase != "complete" {
			t.Fatal("empty review not complete")
		}
		m = rzKey(m, "enter")
		m = rzKey(m, "esc")
	}
	m := persistentNestedModel(t, "- [ ] one\n- [ ] two\n")
	m = rzCommand(m, "rezero")
	m = rzKey(m, "space")
	m = rzKey(m, "b")
	if m.SelectedIndex != 1 || len(m.rezero.Ready) != 0 || m.rezero.Position != 0 {
		t.Fatal("back did not undo review decision")
	}
	m = rzKey(m, "enter")
	m = rzKey(m, "enter")
	if m.rezero.Phase != "complete" {
		t.Fatal("empty selection not complete")
	}
	for _, width := range []int{12, 24, 40, 80} {
		m.TermWidth = width
		for _, line := range strings.Split(m.rezeroStatus(), "\n") {
			if ansi.StringWidth(line) > width {
				t.Fatalf("status exceeds %d: %q", width, line)
			}
		}
	}
}

func TestRezeroReloadInvalidatesIndexes(t *testing.T) {
	m := persistentNestedModel(t, "- [ ] one\n- [ ] two\n")
	m = rzCommand(m, "rezero")
	m = rzKey(m, "space")
	if err := os.WriteFile(m.FilePath, []byte("- [ ] changed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := m.checkAndReloadFile()
	next, _ := m.Update(cmd())
	m = next.(Model)
	if m.rezero.Phase != "" || len(m.rezero.Ready) != 0 || m.FileModel.Todos[0].Text != "changed" {
		t.Fatal("reload retained stale task indexes")
	}
}

func TestRezeroFailedSaveCanRetryWithoutLosingInput(t *testing.T) {
	source := "- [ ] task\n"
	m := persistentNestedModel(t, source)
	m = rzCommand(m, "rezero")
	m = rzAll(m)
	m = rzKey(m, "r")
	m = rzKey(m, " next")
	path := m.FilePath
	m.FilePath = t.TempDir() // A directory is an unsupported save target.
	m = rzKey(m, "enter")
	if m.Err == nil || !m.EditMode || m.rezero.Phase != "work" || m.history.Len() != 0 || m.InputBuffer != "task next" {
		t.Fatal("failed save lost input or advanced round")
	}
	if markdown.SerializeMarkdown(&m.FileModel) != source {
		t.Fatal("failed save mutated current document")
	}
	m.FilePath = path
	m = rzKey(m, "x")
	m = rzKey(m, "enter")
	if m.Err != nil || m.EditMode || m.rezero.Phase != "complete" || m.history.Len() != 1 {
		t.Fatalf("retry failed: %v", m.Err)
	}
}

func TestRezeroUndoConflictDoesNotPopHistory(t *testing.T) {
	m := persistentNestedModel(t, "- [ ] task\n")
	m = rzCommand(m, "rezero")
	m = rzAll(m)
	m = rzKey(m, "space")
	external := "- [ ] external\n"
	if err := os.WriteFile(m.FilePath, []byte(external), 0600); err != nil {
		t.Fatal(err)
	}
	m = rzKey(m, "u")
	if !m.ConflictPending || m.history.Len() != 1 || len(m.rezeroUndo) != 1 || !m.FileModel.Todos[0].Checked || m.rezero.Phase != "complete" || rzDisk(t, m) != external {
		t.Fatal("failed undo changed committed state")
	}
}

func TestRezeroUndoStaysBoundedAndWorksAfterExit(t *testing.T) {
	m := persistentNestedModel(t, "- [ ] task\n")
	m = rzCommand(m, "rezero")
	m = rzAll(m)
	for i := 0; i < 105; i++ {
		m = rzKey(m, "r")
		m = rzKey(m, "enter")
		if m.Err != nil {
			t.Fatal(m.Err)
		}
		m = rzKey(m, "enter")
		m = rzAll(m)
	}
	if m.history.Len() != 100 || len(m.rezeroUndo) != 100 {
		t.Fatal("undo is unbounded")
	}
	m = rzKey(m, "esc")
	m = rzKey(m, "u")
	if len(m.FileModel.Todos) != 105 || m.FileModel.Todos[104].Checked {
		t.Fatal("normal undo after Rezero lost continuation")
	}
}

func TestRezeroInputTreatsTextAsText(t *testing.T) {
	m := persistentNestedModel(t, "- [ ] task\n")
	m = rzCommand(m, "rezero")
	m = rzAll(m)
	m = rzKey(m, "r")
	for _, text := range []string{"enter", "esc", "left"} {
		next, _ := m.Update(tea.KeyPressMsg{Text: text})
		m = next.(Model)
	}
	if !m.EditMode || m.InputBuffer != "taskenterescleft" || m.FileModel.Todos[0].Checked {
		t.Fatal("text executed an editing command")
	}
}
