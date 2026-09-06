// Package savesim schedules the production save protocol under deterministic
// I/O faults. It models effect outcomes, not filesystem or SQLite internals.
package savesim

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/niklas-heer/tdx/internal/saveprotocol"
)

type Mutation string

const (
	NoMutation      Mutation = ""
	SkipValidation  Mutation = "skip-validation"
	SkipSync        Mutation = "skip-sync"
	SkipReplacement Mutation = "skip-replacement"
)

type Event struct {
	Tick    uint64 `json:"tick"`
	Writer  int    `json:"writer"`
	Effect  string `json:"effect"`
	Result  string `json:"result"`
	Visible string `json:"visible_sha256"`
	Durable string `json:"durable_sha256"`
}
type Report struct {
	Schema   int               `json:"schema"`
	Seed     uint64            `json:"seed"`
	Steps    int               `json:"steps"`
	Counts   map[string]uint64 `json:"counts"`
	Failures []string          `json:"failures"`
	Digest   string            `json:"trace_sha256"`
	Trace    []Event           `json:"trace"`
}

// splitMix64 specifies its arithmetic explicitly, so replay does not depend on
// changes to math/rand's implementation. Unsigned overflow is intentional.
type random uint64

func (r *random) next() uint64 {
	*r += 0x9e3779b97f4a7c15
	z := uint64(*r)
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}
func (r *random) chance(n uint64) bool { return r.next()%n == 0 }
func (r *random) latency() uint64 {
	if r.chance(17) {
		return 3000
	}
	return 1 + r.next()%50
}

type writer struct {
	save                                     *saveprotocol.Save
	baseline, proposed, captured             string
	prepared, capturedBefore, written, force bool
	ready                                    uint64
}
type world struct {
	visible, durable string
	lock             int
	known, history   map[string]bool
}

func digest(s string) string       { return fmt.Sprintf("%x", sha256.Sum256([]byte(s))) }
func (r *Report) count(key string) { r.Counts[key]++ }
func (r *Report) record(w *world, now uint64, id int, effect, result string) {
	r.Trace = append(r.Trace, Event{now, id, effect, result, digest(w.visible), digest(w.durable)})
}
func (w *world) writer(serial int, force bool, ready uint64) *writer {
	proposed := fmt.Sprintf("---\ncustom: retained\n---\n# Project\n\n<div>opaque</div>\n\n- [ ] café 🦀 revision %d\n\n| a | b |\n|---|---|\n| x | y |\n", serial)
	w.known[proposed] = true
	return &writer{save: saveprotocol.New(force, 2*time.Second), baseline: w.visible, proposed: proposed, force: force, ready: ready}
}
func (w *world) effect(a *writer, id int, phase saveprotocol.Phase, fault bool, mutation Mutation) error {
	if fault {
		switch phase {
		case saveprotocol.Prepare:
			return errors.New("torn temporary write / disk full")
		case saveprotocol.CaptureBefore, saveprotocol.CaptureAfter:
			return errors.New("history unavailable")
		case saveprotocol.SyncDirectory:
			return errors.New("storage sync failed")
		default:
			return errors.New("storage unavailable")
		}
	}
	switch phase {
	case saveprotocol.Prepare:
		a.prepared = true
	case saveprotocol.Lock:
		if w.lock != -1 {
			return saveprotocol.ErrBusy
		}
		w.lock = id
	case saveprotocol.Validate:
		if !a.force && a.baseline != w.visible && mutation != SkipValidation {
			return errors.New("file changed externally")
		}
	case saveprotocol.CaptureBefore:
		w.history[w.visible] = true
		a.captured = w.visible
		a.capturedBefore = true
	case saveprotocol.Replace:
		if mutation != SkipReplacement {
			w.visible = a.proposed
			a.written = true
		}
	case saveprotocol.SyncDirectory:
		if mutation != SkipSync {
			w.durable = w.visible
		}
	case saveprotocol.CaptureAfter:
		w.history[a.proposed] = true
	case saveprotocol.Unlock:
		w.lock = -1
	case saveprotocol.Done:
		return errors.New("completed writer scheduled")
	}
	return nil
}
func (r *Report) step(w *world, a *writer, id int, now uint64, fault bool, mutation Mutation) {
	phase, before := a.save.Phase(), w.visible
	// These checks use actual world state, independently of the protocol flags.
	if phase == saveprotocol.Replace && !fault {
		if w.lock != id || !a.prepared {
			r.Failures = append(r.Failures, "replacement without lock and prepared bytes")
		}
		if !a.force && a.baseline != before {
			r.Failures = append(r.Failures, "stale writer overwrote a newer revision")
		}
		if a.force && (!a.capturedBefore || a.captured != before || !w.history[before]) {
			r.Failures = append(r.Failures, "force-save omitted overwritten history")
		}
	}
	err := w.effect(a, id, phase, fault, mutation)
	if err != nil && w.visible != before {
		r.Failures = append(r.Failures, "failed effect changed target bytes")
	}
	if !w.known[w.visible] {
		r.Failures = append(r.Failures, "target contains partial or unknown bytes")
	}
	r.count("phase/" + string(phase))
	if fault {
		r.count("fault/" + string(phase))
	}
	result := "ok"
	if err != nil {
		result = err.Error()
	}
	if errors.Is(err, saveprotocol.ErrBusy) {
		r.count("lock_busy")
	}
	a.save.Advance(err, time.Duration(now)*time.Millisecond)
	r.record(w, now, id, string(phase), result)
	if a.save.Phase() == saveprotocol.Done {
		outcome := a.save.Outcome()
		if outcome.Committed != a.written {
			r.Failures = append(r.Failures, "reported commit disagrees with actual replacement")
		}
		if outcome.Err == nil && (!a.written || w.durable != a.proposed) {
			r.Failures = append(r.Failures, "acknowledged save is not durable")
		}
		if a.written {
			r.count("committed")
		} else {
			r.count("rejected")
		}
		// A simulated writer ends its process here. Native crash tests separately
		// check that OS process termination releases actual file locks.
		if w.lock == id {
			w.lock = -1
		}
	}
}

// Run executes a bounded campaign. I/O delay, failures and scheduling all come
// from the seed; there are no sleeps, goroutines, wall clocks or real files.
func Run(seed uint64, steps int, mutation Mutation) Report {
	initial := "# Project\n\n- [ ] original café\n"
	w := world{visible: initial, durable: initial, lock: -1, known: map[string]bool{initial: true}, history: map[string]bool{}}
	rng := random(seed)
	report := Report{Schema: 1, Seed: seed, Steps: steps, Counts: map[string]uint64{}, Failures: []string{}, Trace: []Event{}}
	actors := make([]*writer, 3)
	for id := range actors {
		actors[id] = w.writer(id, false, rng.latency())
	}
	var now uint64
	serial := 3
	for range steps {
		for id := range actors {
			if actors[id] == nil {
				force := rng.chance(7)
				actors[id] = w.writer(serial, force, now+rng.latency())
				serial++
			}
		}
		id := 0
		for i := 1; i < len(actors); i++ {
			if actors[i].ready < actors[id].ready {
				id = i
			}
		}
		now = actors[id].ready
		if rng.chance(101) {
			w.visible = w.durable
			w.lock = -1
			w.history = map[string]bool{}
			clear(actors)
			report.count("power_loss")
			report.record(&w, now, id, "PowerLoss", "discard unsynced state")
			continue
		}
		if rng.chance(29) {
			actors[id] = nil
			if w.lock == id {
				w.lock = -1
			}
			report.count("process_crash")
			report.record(&w, now, id, "ProcessCrash", "lock released")
			continue
		}
		// External writers obey no tdx lock in reality. This campaign injects their
		// changes before validation; post-validation external races are out of scope.
		if w.lock == -1 && rng.chance(19) {
			w.visible = fmt.Sprintf("# External editor\n\n- [ ] external %d\n", serial)
			serial++
			w.durable = w.visible
			w.known[w.visible] = true
			report.count("external_edit")
			report.record(&w, now, id, "ExternalEdit", "durable external revision")
		}
		actor := actors[id]
		report.step(&w, actor, id, now, rng.chance(13), mutation)
		actor.ready = now + rng.latency()
		if actor.save.Phase() == saveprotocol.Done {
			actors[id] = nil
		}
		if len(report.Failures) > 0 {
			break
		}
	}
	// Stop faults, terminate writers and reload to prove bounded recovery using
	// exactly the same protocol; a successful recovery must survive power loss.
	if len(report.Failures) == 0 {
		w.lock = -1
		recovered := w.writer(serial, false, now)
		for range 12 {
			now++
			report.step(&w, recovered, 0, now, false, mutation)
			if recovered.save.Phase() == saveprotocol.Done {
				break
			}
		}
		if recovered.save.Phase() != saveprotocol.Done || recovered.save.Outcome().Err != nil {
			report.Failures = append(report.Failures, "no progress after faults stopped")
		} else if len(report.Failures) == 0 {
			acknowledged := w.visible
			w.visible = w.durable
			if w.visible != acknowledged {
				report.Failures = append(report.Failures, "successful recovery lost on power failure")
			}
			report.count("recovery_passed")
			report.record(&w, now, 0, "Recovery", "save and power-loss check")
		}
	}
	data, err := json.Marshal(report.Trace)
	if err != nil {
		report.Failures = append(report.Failures, err.Error())
	} else {
		report.Digest = digest(string(data))
	}
	return report
}
