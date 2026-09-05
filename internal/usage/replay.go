package usage

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/niklas-heer/tdx/internal/markdown"
	"github.com/niklas-heer/tdx/internal/tui"
)

type Result struct {
	TraceSHA256      string         `json:"trace_sha256"`
	AllocatedBytes   uint64         `json:"allocated_bytes"`
	Seed             uint64         `json:"seed"`
	Driver           string         `json:"driver"`
	Actions          int            `json:"actions"`
	SimulatedSeconds int            `json:"simulated_seconds"`
	WallSeconds      float64        `json:"wall_seconds"`
	P50Millis        float64        `json:"p50_ms"`
	P95Millis        float64        `json:"p95_ms"`
	MaxMillis        float64        `json:"max_ms"`
	Counts           map[string]int `json:"counts"`
	Error            string         `json:"error,omitempty"`
}

func initialSource(tasks []Task) string {
	var b strings.Builder
	b.WriteString("# Usage replay\n\n")
	for _, task := range tasks {
		mark := " "
		if task.Checked {
			mark = "x"
		}
		fmt.Fprintf(&b, "- [%s] %s\n", mark, task.Text)
	}
	return b.String()
}

// Replay exercises real TUI Update and disk saves, or an independent executable's CLI.
// Think time is accounted for, not slept. Timers and terminal I/O have a separate PTY suite.
func Replay(trace Trace, binary string) (result Result, err error) {
	start := time.Now()
	result = Result{Seed: trace.Seed, Driver: trace.Driver, Counts: map[string]int{}}
	encoded, marshalErr := json.Marshal(trace)
	if marshalErr != nil {
		return result, marshalErr
	}
	result.TraceSHA256 = fmt.Sprintf("%x", sha256.Sum256(encoded))
	var initialMem runtime.MemStats
	runtime.ReadMemStats(&initialMem)
	var latencies []float64
	defer func() {
		result.WallSeconds = time.Since(start).Seconds()
		var finalMem runtime.MemStats
		runtime.ReadMemStats(&finalMem)
		result.AllocatedBytes = finalMem.TotalAlloc - initialMem.TotalAlloc
		slices.Sort(latencies)
		if len(latencies) > 0 {
			result.P50Millis = latencies[len(latencies)/2]
			result.P95Millis = latencies[(len(latencies)-1)*95/100]
			result.MaxMillis = latencies[len(latencies)-1]
		}
		if err != nil {
			result.Error = err.Error()
		}
	}()
	if trace.Schema != Schema {
		return result, fmt.Errorf("unsupported trace schema %d", trace.Schema)
	}
	if trace.Driver != "tui" && trace.Driver != "cli" {
		return result, fmt.Errorf("unsupported driver %q", trace.Driver)
	}
	if trace.Driver == "cli" && binary == "" {
		return result, fmt.Errorf("CLI replay requires -binary")
	}
	dir, err := os.MkdirTemp("", "tdx-usage-")
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(dir)) }()
	path := filepath.Join(dir, "tasks.md")
	if err = os.WriteFile(path, []byte(initialSource(trace.Initial)), 0600); err != nil {
		return result, err
	}
	var model tui.Model
	reopen := func() error {
		fm, e := markdown.ReadFile(path)
		if e != nil {
			return e
		}
		model = tui.New(path, fm, false, false, -1, nil, nil, "usage")
		return nil
	}
	if trace.Driver == "tui" {
		if err = reopen(); err != nil {
			return result, err
		}
	}
	oracle := Oracle{Tasks: slices.Clone(trace.Initial)}
	for i, s := range trace.Steps {
		stepStart := time.Now()
		if s.Seconds <= 0 || s.Seconds > 3600 {
			return result, fmt.Errorf("step %d: invalid think time", i)
		}
		if err = oracle.Apply(s); err != nil {
			return result, fmt.Errorf("step %d: %w", i, err)
		}
		if !slices.Equal(oracle.Tasks, s.Expected) {
			return result, fmt.Errorf("step %d: trace expectation disagrees with oracle", i)
		}
		if trace.Driver == "cli" {
			err = runCLI(binary, dir, path, s)
		} else if s.Op == "external" || s.Op == "conflict-reload" {
			err = externalChange(&model, path, s)
		} else if s.Op == "reopen" {
			err = reopen()
		} else {
			err = driveTUI(&model, s)
		}
		if err != nil {
			return result, fmt.Errorf("seed %d step %d (%s): %w", trace.Seed, i, s.Op, err)
		}
		if trace.Driver == "tui" {
			got := make([]Task, len(model.FileModel.Todos))
			for j, t := range model.FileModel.Todos {
				got[j] = Task{Text: t.Text, Checked: t.Checked}
			}
			if !slices.Equal(got, s.Expected) {
				return result, fmt.Errorf("seed %d step %d (%s): TUI state mismatch: got %v; want %v", trace.Seed, i, s.Op, got, s.Expected)
			}
			// Derived UI state must agree with the task oracle after undo and reopen too.
			tagSet := map[string]bool{}
			for _, task := range s.Expected {
				for _, field := range strings.Fields(task.Text) {
					if strings.HasPrefix(field, "#") {
						tagSet[field[1:]] = true
					}
				}
			}
			tags := make([]string, 0, len(tagSet))
			for tag := range tagSet {
				tags = append(tags, tag)
			}
			slices.Sort(tags)
			prioritySet := map[int]bool{}
			for _, task := range s.Expected {
				for _, field := range strings.Fields(task.Text) {
					if strings.HasPrefix(field, "!p") {
						value, e := strconv.Atoi(field[2:])
						if e == nil && value > 0 {
							prioritySet[value] = true
						}
					}
				}
			}
			priorities := make([]int, 0, len(prioritySet))
			for value := range prioritySet {
				priorities = append(priorities, value)
			}
			slices.Sort(priorities)
			actualPriorities := slices.Clone(model.AvailablePriorities)
			slices.Sort(actualPriorities)
			if !slices.Equal(priorities, actualPriorities) {
				return result, fmt.Errorf("seed %d step %d (%s): stale available priorities: got %v; want %v", trace.Seed, i, s.Op, actualPriorities, priorities)
			}
			actualTags := slices.Clone(model.AvailableTags)
			slices.Sort(actualTags)
			if !slices.Equal(tags, actualTags) {
				return result, fmt.Errorf("seed %d step %d (%s): stale available tags: got %v; want %v", trace.Seed, i, s.Op, actualTags, tags)
			}
			if len(got) > 0 && (model.SelectedIndex < 0 || model.SelectedIndex >= len(got)) {
				return result, fmt.Errorf("step %d: selection outside document", i)
			}
			// Render after every action to exercise invalidation and selection, at varied sizes.
			next, _ := model.Update(tea.WindowSizeMsg{Width: []int{24, 80, 120}[i%3], Height: []int{8, 24, 40}[i%3]})
			model = next.(tui.Model)
			_ = model.View()
		}
		data, e := os.ReadFile(path)
		if e != nil {
			return result, e
		}
		// The generated fixture grammar is intentionally small: compare each raw task
		// line, without using production parsing/serialization as the persistence oracle.
		var disk []Task
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "- [ ] ") || strings.HasPrefix(line, "- [x] ") {
				disk = append(disk, Task{Text: line[6:], Checked: line[3] == 'x'})
			}
		}
		if !slices.Equal(disk, s.Expected) {
			return result, fmt.Errorf("seed %d step %d (%s): disk state mismatch", trace.Seed, i, s.Op)
		}
		result.Actions++
		result.SimulatedSeconds += s.Seconds
		result.Counts[s.Op]++
		latencies = append(latencies, float64(time.Since(stepStart).Microseconds())/1000)
	}
	return result, nil
}

func driveTUI(m *tui.Model, s Step) error {
	send := func(msg tea.Msg) { next, _ := m.Update(msg); *m = next.(tui.Model) }
	key := func(code rune) { send(tea.KeyPressMsg{Code: code}) }
	// Navigate through real key events; headings/filters are covered separately.
	for m.SelectedIndex < s.Index {
		old := m.SelectedIndex
		key('j')
		if old == m.SelectedIndex {
			return fmt.Errorf("navigation stuck at %d", old)
		}
	}
	for m.SelectedIndex > s.Index {
		old := m.SelectedIndex
		key('k')
		if old == m.SelectedIndex {
			return fmt.Errorf("navigation stuck at %d", old)
		}
	}
	switch s.Op {
	case "toggle":
		key(tea.KeySpace)
	case "edit":
		key('e')
		key(tea.KeyEnd)
		for range utf8.RuneCountInString(m.InputBuffer) {
			key(tea.KeyBackspace)
		}
		send(tea.PasteMsg{Content: s.Text})
		key(tea.KeyEnter)
	case "add":
		key('N')
		send(tea.PasteMsg{Content: s.Text})
		key(tea.KeyEnter)
	case "delete":
		key('d')
	case "undo":
		key('u')
	case "cancel":
		key('e')
		send(tea.PasteMsg{Content: "discard 日本語"})
		key(tea.KeyEscape)
	case "cancel-add":
		key('N')
		send(tea.PasteMsg{Content: "discard"})
		key(tea.KeyEscape)
	case "empty-add":
		key('N')
		key(tea.KeyEnter)
	case "cancel-move":
		key('m')
		key('j')
		key(tea.KeyEscape)
	case "move":
		key('m')
		key('j')
		key(tea.KeyEnter)
	case "search":
		key('/')
		send(tea.PasteMsg{Content: m.FileModel.Todos[s.Index].Text})
		key(tea.KeyEnter)
		if m.SelectedIndex != s.Index {
			return fmt.Errorf("search selected %d, want %d", m.SelectedIndex, s.Index)
		}
	case "navigate":
	default:
		return fmt.Errorf("unsupported TUI action %q", s.Op)
	}
	if m.Err != nil {
		return m.Err
	}
	return nil
}

func runCLI(binary, dir, path string, s Step) error {
	args := []string{"--file", path}
	switch s.Op {
	case "add":
		args = append(args, "add", s.Text)
	case "edit":
		args = append(args, "edit", strconv.Itoa(s.Index+1), s.Text)
	case "toggle", "delete":
		args = append(args, s.Op, strconv.Itoa(s.Index+1))
	case "reopen":
	default:
		return fmt.Errorf("unsupported CLI action %q (use a CLI trace)", s.Op)
	}
	run := func(args []string) ([]byte, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+filepath.Join(dir, "config"))
		out, e := cmd.CombinedOutput()
		if e != nil {
			return nil, fmt.Errorf("%v: %w: %s", args, e, out)
		}
		return out, nil
	}
	if s.Op != "reopen" {
		if _, err := run(args); err != nil {
			return err
		}
	}
	out, err := run([]string{"--file", path, "list", "--json"})
	if err != nil {
		return err
	}
	var tasks []cliTask
	if err = json.Unmarshal(out, &tasks); err != nil {
		return fmt.Errorf("list JSON: %w: %s", err, out)
	}
	if len(tasks) != len(s.Expected) {
		return fmt.Errorf("CLI JSON count mismatch: %s", out)
	}
	for i, task := range tasks {
		expected := expectedCLI(s.Expected[i], i+1)
		if !reflect.DeepEqual(task, expected) {
			return fmt.Errorf("CLI JSON task %d mismatch: got %+v; want %+v", i+1, task, expected)
		}
	}
	return nil
}

// externalChange controls the ordering around a real external filesystem write.
// The save conflict is checked before choosing the ordinary :reload action.
func externalChange(m *tui.Model, path string, s Step) error {
	if err := driveTUI(m, Step{Op: "navigate", Index: s.Index}); err != nil {
		return err
	}
	external := initialSource(s.Expected)
	if err := os.WriteFile(path, []byte(external), 0600); err != nil {
		return err
	}
	send := func(msg tea.Msg) { next, _ := m.Update(msg); *m = next.(tui.Model) }
	if s.Op == "conflict-reload" {
		send(tea.KeyPressMsg{Code: tea.KeySpace})
		if !m.ConflictPending || !m.ConflictDiffMode || m.Err == nil {
			return fmt.Errorf("external edit did not produce save conflict")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if string(data) != external {
			return fmt.Errorf("conflicting save overwrote external data")
		}
		send(tea.KeyPressMsg{Code: tea.KeyEscape})
		send(tea.KeyPressMsg{Code: ':'})
		send(tea.PasteMsg{Content: "reload"})
		send(tea.KeyPressMsg{Code: tea.KeyEnter})
		if m.ConflictPending || m.Err != nil {
			return fmt.Errorf("reload did not resolve conflict: %v", m.Err)
		}
	} else {
		next, cmd := m.Update(tui.FileChangedMsg{})
		*m = next.(tui.Model)
		if cmd == nil {
			return fmt.Errorf("external change produced no reload")
		}
		// Execute only the one known reload command, never the recursive watch timer.
		send(cmd())
	}
	return nil
}

// This is the public JSON contract, independently described here for candidate
// executables. Generated texts use whitespace-delimited metadata tokens.
type cliTask struct {
	Index       int      `json:"index"`
	Text        string   `json:"text"`
	Checked     bool     `json:"checked"`
	Depth       int      `json:"depth"`
	ParentIndex *int     `json:"parent_index"`
	Tags        []string `json:"tags"`
	Priority    int      `json:"priority"`
	DueDate     *string  `json:"due_date"`
}

func expectedCLI(task Task, index int) cliTask {
	expected := cliTask{Index: index, Text: task.Text, Checked: task.Checked, Tags: []string{}}
	for _, field := range strings.Fields(task.Text) {
		switch {
		case strings.HasPrefix(field, "#"):
			tag := field[1:]
			if !slices.Contains(expected.Tags, tag) {
				expected.Tags = append(expected.Tags, tag)
			}
		case strings.HasPrefix(field, "!p"):
			priority, err := strconv.Atoi(field[2:])
			if err == nil && priority > 0 && (expected.Priority == 0 || priority < expected.Priority) {
				expected.Priority = priority
			}
		case strings.HasPrefix(field, "@due(") && strings.HasSuffix(field, ")"):
			date := field[5 : len(field)-1]
			if _, err := time.Parse("2006-01-02", date); err == nil && (expected.DueDate == nil || date < *expected.DueDate) {
				expected.DueDate = &date
			}
		}
	}
	return expected
}
