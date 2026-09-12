package usage

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/niklas-heer/tdx/internal/editor"
	"github.com/niklas-heer/tdx/internal/markdown"
)

func TestStructuralCampaign(t *testing.T) {
	for _, seed := range []uint64{100, 101, 102, 103, 4096} {
		t.Run(strconv.FormatUint(seed, 10), func(t *testing.T) {
			trace := Generate(seed, 180, "structural")
			encoded, _ := json.Marshal(trace)
			var decoded Trace
			if err := json.Unmarshal(encoded, &decoded); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(trace, decoded) || !reflect.DeepEqual(trace, Generate(seed, 180, "structural")) {
				t.Fatal("non-deterministic structural trace")
			}
			result, err := Replay(decoded, "")
			if err != nil {
				minimal, note := MinimizeFailure(trace, "", err, 32)
				data, _ := json.Marshal(minimal)
				t.Fatalf("%v\n%s\n%s", err, note, data)
			}
			if result.Actions != 180 || result.Counts["undo"] == 0 || result.Counts["reopen"] == 0 {
				t.Fatalf("incomplete campaign: %+v", result)
			}
		})
	}
}

func TestStructuralOracleRejectsOwnershipCorruption(t *testing.T) {
	source := "# Work\n\n<!-- protected -->\n\n- [ ] Parent\n\n  ```text\n  Parent body.\n  ```\n\n  - [ ] Child\n- [ ] Sibling\n\n  ```text\n  Sibling body.\n  ```\n\n# Later\n\n- [ ] Last\n"
	for name, corrupted := range map[string]string{
		"lost depth":       strings.Replace(source, "  - [ ] Child", "- [ ] Child", 1),
		"changed parent":   strings.Replace(strings.Replace(source, "  - [ ] Child\n", "", 1), "  ```text\n  Sibling body.\n  ```\n", "  ```text\n  Sibling body.\n  ```\n\n  - [ ] Child\n", 1),
		"swapped bodies":   strings.NewReplacer("Parent body.", "Sibling body.", "Sibling body.", "Parent body.").Replace(source),
		"body deleted":     strings.Replace(source, "  ```text\n  Parent body.\n  ```\n", "", 1),
		"protected region": strings.Replace(strings.Replace(source, "<!-- protected -->\n", "", 1), "# Later\n", "# Later\n\n<!-- protected -->\n", 1),
		"task section":     strings.Replace(strings.Replace(source, "- [ ] Last\n", "", 1), "# Later\n", "- [ ] Last\n\n# Later\n", 1),
	} {
		t.Run(name, func(t *testing.T) {
			action := editor.Action{Kind: editor.SetChecked, Index: 0, Checked: false}
			// These corruptions all pass the old multiset oracle.
			if err := checkTaskConservation(markdown.ParseMarkdown(source).Todos, markdown.ParseMarkdown(corrupted).Todos, action); err != nil {
				t.Fatalf("fixture must retain all task titles and completion states: %v", err)
			}
			if err := checkStructuralOwnership(source, corrupted, action, []string{"<!-- protected -->\n"}); err == nil {
				t.Fatal("ownership corruption escaped the structural oracle")
			}
		})
	}
}

func TestStructuralOracleAcceptsAuthorizedOwnershipChanges(t *testing.T) {
	source := "# Work\n\n- [x] Parent\n\n  ~~~text\n  Owned code.\n  ~~~\n\n  - [ ] Child\n  - [x] Other child\n- [ ] Sibling\n\n# Later\n\n- [ ] Last\n"
	for _, action := range []editor.Action{
		{Kind: editor.Move, Index: 0, Target: 4},
		{Kind: editor.MoveToPosition, Index: 0, Target: 4, InsertAfter: true},
		{Kind: editor.Move, Index: 0, Target: 0},
		{Kind: editor.Indent, Index: 3},
		{Kind: editor.Outdent, Index: 1},
		{Kind: editor.Delete, Index: 0},
		{Kind: editor.ClearDone},
		{Kind: editor.SortDone},
		{Kind: editor.Edit, Index: 3, Text: "Last"}, // Duplicate title remains a valid edit.
		{Kind: editor.CreateHeading, Index: 0, Level: 1, Text: "Work"},
	} {
		t.Run(fmt.Sprintf("%s-%d-%d", action.Kind, action.Index, action.Target), func(t *testing.T) {
			doc := markdown.ParseMarkdown(source)
			if _, err := editor.Apply(doc, action); err != nil {
				t.Fatal(err)
			}
			if err := checkStructuralOwnership(source, markdown.SerializeMarkdown(doc), action, nil); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestStructuralOracleRejectsImpossibleSuccessWithoutPanic(t *testing.T) {
	for _, action := range []editor.Action{
		{Kind: editor.Edit, Index: -1},
		{Kind: editor.Move, Index: 0, Target: 1},
		{Kind: editor.Outdent, Index: 0},
	} {
		if err := checkStructuralOwnership("- [ ] task\n", "- [ ] task\n", action, nil); err == nil {
			t.Fatalf("accepted impossible success: %+v", action)
		}
	}
}

func TestStructuralOracleRejectsIncorrectActionResults(t *testing.T) {
	source := "# A\n\n- [ ] one\n- [ ] two\n\n# B\n"
	for _, tc := range []struct {
		action editor.Action
		after  string
	}{
		{editor.Action{Kind: editor.Move, Index: 0, Target: 1}, source},
		{editor.Action{Kind: editor.Edit, Index: 0, Text: "changed"}, source},
		{editor.Action{Kind: editor.AddInSection, Index: 0, Text: "new"}, source + "\n- [ ] new\n"},
		{editor.Action{Kind: editor.Insert, Index: 0, Text: "new"}, strings.Replace(source, "- [ ] two\n", "- [ ] two\n- [ ] new\n", 1)},
	} {
		if err := checkTaskConservation(markdown.ParseMarkdown(source).Todos, markdown.ParseMarkdown(tc.after).Todos, tc.action); err != nil {
			t.Fatalf("fixture must pass the former multiset oracle: %v", err)
		}
		if err := checkStructuralOwnership(source, tc.after, tc.action, nil); err == nil {
			t.Fatalf("accepted incorrect action result: %+v", tc.action)
		}
	}
}

func TestMinimizedStructuralTraceReproducesFailure(t *testing.T) {
	trace := Trace{Schema: Schema, Seed: 42, Driver: "structural", Source: "- [ ] original\n", Protected: []string{"original"}, Steps: []Step{
		{Op: "reopen", Seconds: 10},
		{Op: "action", Seconds: 10, Action: &editor.Action{Kind: editor.Edit, Text: "changed"}},
		{Op: "undo", Seconds: 10},
	}}
	_, err := Replay(trace, "")
	if err == nil {
		t.Fatal("expected protected-source violation")
	}
	minimal, _ := MinimizeFailure(trace, "", err, 32)
	if len(minimal.Steps) != 1 {
		t.Fatalf("expected one-step reproducer, got %d", len(minimal.Steps))
	}
	_, replayErr := Replay(minimal, "")
	if failureClass(replayErr) != failureClass(err) {
		t.Fatalf("reduced failure differs: %v", replayErr)
	}
}

func TestHeadingCacheRegressionTrace(t *testing.T) {
	data, err := os.ReadFile("../../experiments/usage/heading-cache-regression.json")
	if err != nil {
		t.Fatal(err)
	}
	var trace Trace
	if err = json.Unmarshal(data, &trace); err != nil {
		t.Fatal(err)
	}
	if _, err = Replay(trace, ""); err != nil {
		t.Fatal(err)
	}
}

func TestTraceChunkReduction(t *testing.T) {
	steps := []Step{{Op: "noise"}, {Op: "trigger-a"}, {Op: "noise"}, {Op: "trigger-b"}, {Op: "noise"}}
	got := reduceSteps(steps, func(candidate []Step) bool {
		return slices.ContainsFunc(candidate, func(s Step) bool { return s.Op == "trigger-a" }) && slices.ContainsFunc(candidate, func(s Step) bool { return s.Op == "trigger-b" })
	})
	if len(got) != 2 || got[0].Op != "trigger-a" || got[1].Op != "trigger-b" {
		t.Fatalf("incorrect reduction: %+v", got)
	}
}
