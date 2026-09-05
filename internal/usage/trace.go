// Package usage defines portable, deterministic user-session contracts.
package usage

import (
	"fmt"
	"math/rand/v2"
	"slices"
)

const Schema = 1

type Task struct {
	Text    string `json:"text"`
	Checked bool   `json:"checked"`
}
type Step struct {
	Op       string `json:"op"`
	Index    int    `json:"index"`
	Text     string `json:"text,omitempty"`
	Seconds  int    `json:"seconds"`
	Expected []Task `json:"expected"`
}
type Trace struct {
	Schema  int    `json:"schema"`
	Seed    uint64 `json:"seed"`
	Driver  string `json:"driver"`
	Initial []Task `json:"initial"`
	Steps   []Step `json:"steps"`
}

// Oracle deliberately uses only flat value slices, never the editor or Markdown parser.
type Oracle struct {
	Tasks []Task
	past  [][]Task
}

func (o *Oracle) Apply(s Step) error {
	switch s.Op {
	case "toggle", "edit", "delete", "move", "cancel", "navigate", "search", "cancel-move", "external", "conflict-reload":
		if s.Index < 0 || s.Index >= len(o.Tasks) {
			return fmt.Errorf("invalid index %d", s.Index)
		}
	case "add", "undo", "reopen", "cancel-add", "empty-add":
	default:
		return fmt.Errorf("unknown operation %q", s.Op)
	}
	switch s.Op {
	case "toggle", "edit", "delete", "move", "add":
		o.past = append(o.past, slices.Clone(o.Tasks))
		if len(o.past) > 100 {
			o.past = o.past[1:]
		}
	}
	switch s.Op {
	case "toggle":
		o.Tasks[s.Index].Checked = !o.Tasks[s.Index].Checked
	case "edit":
		o.Tasks[s.Index].Text = s.Text
	case "delete":
		o.Tasks = slices.Delete(o.Tasks, s.Index, s.Index+1)
	case "add":
		o.Tasks = append(o.Tasks, Task{Text: s.Text})
	case "move":
		if s.Index+1 < len(o.Tasks) {
			o.Tasks[s.Index], o.Tasks[s.Index+1] = o.Tasks[s.Index+1], o.Tasks[s.Index]
		}
	case "undo":
		if len(o.past) > 0 {
			o.Tasks = slices.Clone(o.past[len(o.past)-1])
			o.past = o.past[:len(o.past)-1]
		}
	case "external", "conflict-reload":
		o.Tasks[s.Index].Text = s.Text
		o.past = nil
	case "reopen":
		o.past = nil
	}
	return nil
}

func Generate(seed uint64, count int, driver string) Trace {
	r := rand.New(rand.NewPCG(seed, seed^0x746478))
	trace := Trace{Schema: Schema, Seed: seed, Driver: driver}
	for i := 0; i < 24; i++ {
		trace.Initial = append(trace.Initial, Task{Text: fmt.Sprintf("Task %d café 日本語 #work !p2 @due(2027-01-12)", i), Checked: i%4 == 0})
	}
	oracle := Oracle{Tasks: slices.Clone(trace.Initial)}
	ops := []string{"toggle", "toggle", "toggle", "toggle", "edit", "edit", "add", "add", "delete", "delete", "undo", "cancel", "move", "navigate", "search", "cancel-add", "cancel-move", "empty-add", "external", "conflict-reload", "reopen"}
	if driver == "cli" {
		ops = []string{"toggle", "toggle", "edit", "add", "delete", "reopen"}
	}
	for i := 0; i < count; i++ {
		op := ops[r.IntN(len(ops))]
		if len(oracle.Tasks) == 0 {
			op = "add"
		}
		if len(oracle.Tasks) > 64 && op == "add" {
			op = "delete"
		}
		s := Step{Op: op, Seconds: 10, Text: fmt.Sprintf("Task-%d-%d résumé 🦀 #next !p1", seed, i)}
		if len(oracle.Tasks) > 0 {
			s.Index = r.IntN(len(oracle.Tasks))
		}
		if err := oracle.Apply(s); err != nil {
			panic(err)
		}
		s.Expected = slices.Clone(oracle.Tasks)
		trace.Steps = append(trace.Steps, s)
	}
	return trace
}
