package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/niklas-heer/tdx/internal/markdown"
)

func TestUnfilteredMovePreservesNesting(t *testing.T) {
	for _, tc := range []struct {
		name     string
		source   string
		selected int
		key      rune
		want     string
		wantAt   int
	}{
		{
			name:     "first child cannot move above parent",
			source:   "- [ ] Parent\n  - [ ] Child\n    - [ ] Grandchild\n  - [ ] Sibling\n",
			selected: 1, key: 'k', wantAt: 1,
		},
		{
			name:     "last child cannot move below parent subtree",
			source:   "- [ ] Parent\n  - [ ] Child\n    - [ ] Grandchild\n- [ ] Next parent\n",
			selected: 1, key: 'j', wantAt: 1,
		},
		{
			name:     "child subtree moves down among siblings",
			source:   "- [ ] Parent\n  - [ ] Child\n    - [ ] Grandchild\n  - [ ] Sibling\n",
			selected: 1, key: 'j', wantAt: 2,
			want: "- [ ] Parent\n  - [ ] Sibling\n  - [ ] Child\n    - [ ] Grandchild\n",
		},
		{
			name:     "child subtree moves up among siblings",
			source:   "- [ ] Parent\n  - [ ] Sibling\n    - [ ] Sibling child\n  - [ ] Child\n    - [ ] Grandchild\n",
			selected: 3, key: 'k', wantAt: 1,
			want: "- [ ] Parent\n  - [ ] Child\n    - [ ] Grandchild\n  - [ ] Sibling\n    - [ ] Sibling child\n",
		},
		{
			name:     "root subtree moves past previous subtree",
			source:   "- [ ] Parent\n  - [ ] Child\n- [ ] Next parent\n  - [ ] Next child\n",
			selected: 2, key: 'k', wantAt: 0,
			want: "- [ ] Next parent\n  - [ ] Next child\n- [ ] Parent\n  - [ ] Child\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := persistentNestedModel(t, tc.source)
			m.SelectedIndex = tc.selected
			m.ShowHeadings = false
			m.FilterDone = false
			if m.hasActiveFilters() {
				t.Fatal("test must exercise unfiltered movement")
			}
			m.MoveMode = true
			updated, _ := m.handleMoveKey(tea.KeyPressMsg{Code: tc.key})
			m = updated.(Model)
			if m.Err != nil {
				t.Fatal(m.Err)
			}
			want := tc.want
			if want == "" {
				want = tc.source
			}
			if got := markdown.SerializeMarkdown(&m.FileModel); got != want {
				t.Fatalf("movement changed nesting:\n%s\nwant:\n%s", got, want)
			}
			if m.SelectedIndex != tc.wantAt {
				t.Fatalf("selected = %d, want %d", m.SelectedIndex, tc.wantAt)
			}
		})
	}
}
