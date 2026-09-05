package markdown

import (
	"strings"
	"testing"
)

func TestSectionMutations(t *testing.T) {
	fm := ParseMarkdown("# Same\n\nParagraph with [link](https://example.com).\n\n# Same\n\n## Child\n\n- [ ] Existing\n")
	if err := fm.RenameHeading(1, "Different"); err != nil {
		t.Fatal(err)
	}
	if _, err := fm.AddTodoInSection(1, "New"); err != nil {
		t.Fatal(err)
	}
	got := SerializeMarkdown(fm)
	for _, want := range []string{"# Same", "# Different\n\n- [ ] New", "## Child", "[link](https://example.com)", "Existing"} {
		if !strings.Contains(got, want) {
			t.Fatalf("want %q: %s", want, got)
		}
	}
	clone := fm.Clone()
	if got := SerializeMarkdown(clone); !strings.Contains(got, "Different") || !strings.Contains(got, "New") {
		t.Fatalf("clone lost edits: %s", got)
	}
}

func TestSectionCreationHierarchy(t *testing.T) {
	fm := ParseMarkdown("# Work\n\n## Backend\n\n# Home\n")
	index, err := fm.CreateHeading(0, 2, "Planning")
	if err != nil {
		t.Fatal(err)
	}
	headings := fm.GetHeadings()
	if index != 2 || headings[2].Text != "Planning" || headings[3].Text != "Home" {
		t.Fatalf("wrong placement: %+v", headings)
	}
	if _, err := fm.CreateHeading(0, 7, "Invalid"); err == nil {
		t.Fatal("accepted level 7")
	}
	if err := fm.RenameHeading(0, "\n# Injected"); err == nil {
		t.Fatal("accepted multiline heading")
	}
}
