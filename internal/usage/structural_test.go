package usage

import (
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"strconv"
	"testing"

	"github.com/niklas-heer/tdx/internal/editor"
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
