package usage

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

var failureLocation = regexp.MustCompile(`^seed [0-9]+ step [0-9]+(?: \([^)]*\))?: `)

func failureClass(err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	if !failureLocation.MatchString(message) {
		return ""
	}
	message = failureLocation.ReplaceAllString(message, "")
	class, _, _ := strings.Cut(message, ":")
	return class
}

// MinimizeFailure removes action chunks while reproducing the same failure
// category. Flat traces regenerate expectations through their independent oracle;
// candidates with invalid positional actions are discarded. The bounded budget
// prevents a failing CLI or platform dependency from stalling a CI campaign.
func MinimizeFailure(trace Trace, binary string, original error, budget int) (Trace, string) {
	class := failureClass(original)
	if class == "" {
		return trace, "reduction unavailable: failure is not a reproducible step invariant"
	}
	attempts := 0
	steps := reduceSteps(trace.Steps, func(candidate []Step) bool {
		if attempts >= budget {
			return false
		}
		attempts++
		next := trace
		next.Steps = slices.Clone(candidate)
		if trace.Driver != "structural" {
			oracle := Oracle{Tasks: slices.Clone(trace.Initial)}
			for i := range next.Steps {
				if err := oracle.Apply(next.Steps[i]); err != nil {
					return false
				}
				next.Steps[i].Expected = slices.Clone(oracle.Tasks)
			}
		}
		_, err := Replay(next, binary)
		return failureClass(err) == class
	})
	trace.Steps = slices.Clone(steps)
	if trace.Driver != "structural" {
		oracle := Oracle{Tasks: slices.Clone(trace.Initial)}
		for i := range trace.Steps {
			_ = oracle.Apply(trace.Steps[i])
			trace.Steps[i].Expected = slices.Clone(oracle.Tasks)
		}
	}
	return trace, fmt.Sprintf("reduced to %d steps after %d/%d replay attempts; failure category: %s", len(steps), attempts, budget, class)
}

func reduceSteps(steps []Step, fails func([]Step) bool) []Step {
	current := slices.Clone(steps)
	for chunk := max(1, len(current)/2); chunk >= 1; chunk /= 2 {
		for start := 0; start < len(current); {
			end := min(len(current), start+chunk)
			candidate := slices.Concat(current[:start], current[end:])
			if fails(candidate) {
				current = candidate
			} else {
				start = end
			}
		}
	}
	return current
}
