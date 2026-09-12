package usage

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/niklas-heer/tdx/internal/editor"
	"github.com/niklas-heer/tdx/internal/markdown"
)

// GenerateStructural supplements the flat independent oracle with rich source
// preservation, complete task subtrees, rejected-edit atomicity and exact undo.
func GenerateStructural(seed uint64, count int) Trace {
	protected := []string{
		"<!-- preserve this comment exactly -->\n\n| A | B |\n| :--- | ---: |\n| café | `code` |\n\n~~~md\n- [ ] example only\n~~~\n\n",
		"[guide]: <https://example.com/guide>\n  \"Do not normalize\"\n",
	}
	source := "# Work\n\n" + protected[0] + "- [ ] Parent !p2 #work\n  continuation with [guide].\n\n  Body paragraph with **bold**.\n\n  ~~~text\n  Parent owned code.\n  ~~~\n\n  - [x] Child done !p1\n  - [ ] Child open !p3\n- [ ] Sibling !p1\n\n  ~~~text\n  Sibling owned code.\n  ~~~\n\n## Ordered\n\n9. [ ] Nine\n   1) [ ] Nested\n   2) [x] Nested done\n10. [ ] Ten\n\n" + protected[1]
	if seed%2 == 1 {
		source = strings.ReplaceAll(source, "\n", "\r\n")
		for i := range protected {
			protected[i] = strings.ReplaceAll(protected[i], "\n", "\r\n")
		}
	}
	trace := Trace{Schema: Schema, Seed: seed, Driver: "structural", Source: source, Protected: protected}
	r := rand.New(rand.NewPCG(seed, seed^0x737472756374))
	kinds := []editor.Kind{editor.Edit, editor.Toggle, editor.Insert, editor.Delete, editor.Move, editor.Indent, editor.Outdent, editor.SortDone, editor.SortPriority, editor.SortDue, editor.AddInSection, editor.RenameHeading, editor.CreateHeading, editor.ClearDone, editor.SetAllChecked, editor.Add, editor.Add}
	for i := 0; i < count; i++ {
		step := Step{Op: "action", Seconds: 10}
		switch i % 11 {
		case 9:
			step.Op = "undo"
		case 10:
			step.Op = "reopen"
		default:
			kind := kinds[r.IntN(len(kinds))]
			if i < len(kinds) {
				kind = kinds[i]
			}
			a := editor.Action{Kind: kind, Index: r.IntN(8), Target: r.IntN(8), Text: fmt.Sprintf("Task-%d-%d café !p%d #work @due(2027-01-12)", seed, i, 1+r.IntN(3)), Level: 2, Checked: r.IntN(2) == 0}
			if kind == editor.AddInSection || kind == editor.RenameHeading {
				a.Index = r.IntN(2)
			}
			if kind == editor.CreateHeading {
				a.Index = -1
			}
			step.Action = &a
		}
		trace.Steps = append(trace.Steps, step)
	}
	return trace
}

func ReplayStructural(trace Trace) (result Result, err error) {
	started := time.Now()
	result = Result{Seed: trace.Seed, Driver: trace.Driver, Counts: map[string]int{}}
	encoded, _ := json.Marshal(trace)
	result.TraceSHA256 = fmt.Sprintf("%x", sha256.Sum256(encoded))
	var latency []float64
	var initialMem runtime.MemStats
	runtime.ReadMemStats(&initialMem)
	defer func() {
		var finalMem runtime.MemStats
		runtime.ReadMemStats(&finalMem)
		result.AllocatedBytes = finalMem.TotalAlloc - initialMem.TotalAlloc
		result.WallSeconds = time.Since(started).Seconds()
		slices.Sort(latency)
		if len(latency) > 0 {
			result.P50Millis = latency[len(latency)/2]
			result.P95Millis = latency[(len(latency)-1)*95/100]
			result.MaxMillis = latency[len(latency)-1]
		}
		if err != nil {
			result.Error = err.Error()
		}
	}()
	if trace.Schema != Schema || trace.Driver != "structural" {
		return result, fmt.Errorf("unsupported structural trace")
	}
	dir, err := os.MkdirTemp("", "tdx-structural-")
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(dir)) }()
	path := filepath.Join(dir, "tasks.md")
	if err = os.WriteFile(path, []byte(trace.Source), 0600); err != nil {
		return result, err
	}
	doc, err := markdown.ReadFile(path)
	if err != nil {
		return result, err
	}
	var history editor.History
	var past []string
	fail := func(index int, kind, detail string) error {
		return fmt.Errorf("seed %d step %d: %s: %s", trace.Seed, index, kind, detail)
	}
	for i, step := range trace.Steps {
		start := time.Now()
		if step.Seconds <= 0 || step.Seconds > 3600 {
			return result, fail(i, "invalid trace", "invalid think time")
		}
		before := markdown.SerializeMarkdown(doc)
		beforeTodos := slices.Clone(doc.Todos)
		label := step.Op
		switch step.Op {
		case "undo":
			if len(past) > 0 {
				want := past[len(past)-1]
				past = past[:len(past)-1]
				if !history.Undo(doc) || markdown.SerializeMarkdown(doc) != want {
					return result, fail(i, "undo fidelity", "snapshot did not restore original bytes")
				}
			}
		case "reopen":
			doc, err = markdown.ReadFile(path)
			if err != nil {
				return result, err
			}
			history.Clear()
			past = nil
			if markdown.SerializeMarkdown(doc) != before {
				return result, fail(i, "reopen fidelity", "reopened document changed bytes")
			}
		case "action":
			if step.Action == nil {
				return result, fail(i, "invalid trace", "missing action")
			}
			label = string(step.Action.Kind)
			history.Begin(doc)
			_, actionErr := editor.Apply(doc, *step.Action)
			if actionErr != nil {
				if markdown.SerializeMarkdown(doc) != before || !reflect.DeepEqual(doc.Todos, beforeTodos) {
					return result, fail(i, "rejection atomicity", actionErr.Error())
				}
				history.Cancel(doc)
				label += "-rejected"
			} else {
				if err = checkTaskConservation(beforeTodos, doc.Todos, *step.Action); err != nil {
					return result, fail(i, "task conservation", err.Error())
				}
				if err = checkStructuralOwnership(before, markdown.SerializeMarkdown(doc), *step.Action, trace.Protected); err != nil {
					return result, fail(i, "structural ownership", err.Error())
				}
				history.Commit()
				past = append(past, before)
				if len(past) > editor.HistoryLimit {
					past = past[1:]
				}
			}
		default:
			return result, fail(i, "invalid trace", "unknown operation")
		}
		source := markdown.SerializeMarkdown(doc)
		for _, protected := range trace.Protected {
			if strings.Count(source, protected) != strings.Count(trace.Source, protected) {
				return result, fail(i, "source preservation", "protected Markdown changed")
			}
		}
		if err = markdown.WriteFile(path, doc); err != nil {
			return result, err
		}
		data, e := os.ReadFile(path)
		if e != nil {
			return result, e
		}
		if string(data) != source {
			return result, fail(i, "save fidelity", "disk bytes differ from editor source")
		}
		parsed := markdown.ParseMarkdown(source)
		if !reflect.DeepEqual(parsed.Todos, doc.Todos) {
			return result, fail(i, "parse stability", "cached task structure differs after reopening")
		}
		result.Actions++
		result.SimulatedSeconds += step.Seconds
		result.Counts[label]++
		latency = append(latency, float64(time.Since(start).Microseconds())/1000)
	}
	return result, nil
}

// Check task identity independently of the source-patch implementation. Bodies
// and descendants must travel with their task through sorting and movement.
func checkTaskConservation(before, after []markdown.Todo, a editor.Action) error {
	key := func(t markdown.Todo) string { return fmt.Sprintf("%t:%s", t.Checked, t.Text) }
	keys := func(tasks []markdown.Todo) []string {
		out := make([]string, len(tasks))
		for i, t := range tasks {
			out[i] = key(t)
		}
		slices.Sort(out)
		return out
	}
	expected := slices.Clone(before)
	switch a.Kind {
	case editor.Toggle:
		expected[a.Index].Checked = !expected[a.Index].Checked
	case editor.SetChecked:
		expected[a.Index].Checked = a.Checked
	case editor.SetAllChecked:
		for i := range expected {
			expected[i].Checked = a.Checked
		}
	case editor.Delete:
		expected = slices.Delete(expected, a.Index, a.Index+1)
	case editor.ClearDone:
		expected = slices.DeleteFunc(expected, func(t markdown.Todo) bool { return t.Checked })
	case editor.Edit:
		if len(before) != len(after) {
			return fmt.Errorf("edit changed task count")
		}
		expected = slices.Delete(expected, a.Index, a.Index+1)
		after = slices.Delete(slices.Clone(after), a.Index, a.Index+1)
	case editor.Add, editor.Insert, editor.AddInSection:
		if len(after) != len(before)+1 {
			return fmt.Errorf("insert did not add exactly one task")
		}
		// Remove exactly one new task, leaving all previous bodies untouched.
		found := -1
		for i, t := range after {
			if t.Text == a.Text {
				found = i
				break
			}
		}
		if found < 0 {
			return fmt.Errorf("inserted task missing")
		}
		after = slices.Delete(slices.Clone(after), found, found+1)
	}
	if !slices.Equal(keys(expected), keys(after)) {
		return fmt.Errorf("unrelated task text or completion changed during %s", a.Kind)
	}
	return nil
}
