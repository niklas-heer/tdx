package editor

import (
	"testing"

	"github.com/niklas-heer/tdx/internal/markdown"
)

func TestMoveResultIndexIncludesSubtrees(t *testing.T) {
	for _, tc := range []struct {
		action Action
		want   int
	}{
		{Action{Kind: Move, Index: 0, Target: 3}, 1},
		{Action{Kind: MoveToPosition, Index: 3, Target: 0, InsertAfter: true}, 3},
		{Action{Kind: MoveToPosition, Index: 3, Target: 0}, 0},
		{Action{Kind: MoveToPosition, Index: 1, Target: 0, InsertAfter: true}, 2},
	} {
		f := markdown.ParseMarkdown("- [ ] duplicate\n  - [ ] child\n  - [ ] child\n- [ ] duplicate\n")
		index, err := Apply(f, tc.action)
		if err != nil {
			t.Fatal(err)
		}
		if index != tc.want {
			t.Fatalf("%+v: got index %d; want %d", tc.action, index, tc.want)
		}
	}
}

func TestRejectedClearDoneRestoresWholeDocument(t *testing.T) {
	source := "> - [x] quoted\n\n- [x] removable\n\n[ref]: /keep\n"
	f := markdown.ParseMarkdown(source)
	if _, err := Apply(f, Action{Kind: ClearDone}); err == nil {
		t.Fatal("expected unsupported quoted deletion")
	}
	if got := markdown.SerializeMarkdown(f); got != source {
		t.Fatalf("partial bulk edit: %q", got)
	}
}

func TestSourceUndoAcrossStructuralActions(t *testing.T) {
	source := "# Work\r\n\r\n- [ ] parent\r\n  body [guide]\r\n  - [x] child\r\n- [ ] sibling\r\n\r\n[guide]: /original\r\n"
	f := markdown.ParseMarkdown(source)
	for _, action := range []Action{{Kind: Edit, Text: "changed"}, {Kind: Delete}, {Kind: Move, Target: 2}, {Kind: SortDone}, {Kind: RenameHeading, Text: "Renamed"}} {
		var history History
		history.Push(f)
		if _, err := Apply(f, action); err != nil {
			t.Fatal(err)
		}
		if !history.Undo(f) {
			t.Fatal("undo unavailable")
		}
		if got := markdown.SerializeMarkdown(f); got != source {
			t.Fatalf("%s undo changed source: %q", action.Kind, got)
		}
	}
}
