// tdx-usage is a developer-only deterministic sustained-use runner.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/niklas-heer/tdx/internal/usage"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(args []string) error {
	flags := flag.NewFlagSet("tdx-usage", flag.ContinueOnError)
	sessions := flags.Int("sessions", 20, "number of independent sessions")
	steps := flags.Int("steps", 1800, "actions per session; each represents 10 simulated seconds")
	seed := flags.Uint64("seed", 100, "first deterministic seed")
	driver := flags.String("driver", "tui", "tui, cli or structural")
	binary := flags.String("binary", "", "CLI executable under test (Go or alternative implementation)")
	replay := flags.String("replay", "", "replay a saved JSON trace instead of generating")
	output := flags.String("output", "dist/usage", "trace and report directory")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if *sessions < 1 || *steps < 1 {
		return fmt.Errorf("sessions and steps must be positive")
	}
	if *binary != "" {
		var err error
		*binary, err = filepath.Abs(*binary)
		if err != nil {
			return err
		}
	}
	if err := os.MkdirAll(*output, 0755); err != nil {
		return err
	}
	var results []usage.Result
	for n := 0; n < *sessions; n++ {
		var trace usage.Trace
		if *replay != "" {
			data, err := os.ReadFile(*replay)
			if err != nil {
				return err
			}
			if err = json.Unmarshal(data, &trace); err != nil {
				return err
			}
		} else {
			trace = usage.Generate(*seed+uint64(n), *steps, *driver)
		}
		if trace.Driver != "tui" && trace.Driver != "cli" && trace.Driver != "structural" {
			return fmt.Errorf("unsupported driver %q", trace.Driver)
		}
		name := fmt.Sprintf("%s-%d", trace.Driver, trace.Seed)
		if err := writeJSON(filepath.Join(*output, name+".json"), trace); err != nil {
			return err
		}
		result, runErr := usage.Replay(trace, *binary)
		results = append(results, result)
		if err := writeJSON(filepath.Join(*output, "report.json"), results); err != nil {
			return err
		}
		fmt.Printf("%s: %d actions, %.2f simulated hours, %.2fs elapsed, p95 %.2fms\n", name, result.Actions, float64(result.SimulatedSeconds)/3600, result.WallSeconds, result.P95Millis)
		if runErr != nil {
			trace.Steps = trace.Steps[:min(len(trace.Steps), result.Actions+1)]
			if err := writeJSON(filepath.Join(*output, name+"-failure.json"), trace); err != nil {
				return err
			}
			minimal, diagnostic := usage.MinimizeFailure(trace, *binary, runErr, 64)
			if err := writeJSON(filepath.Join(*output, name+"-minimized.json"), minimal); err != nil {
				return err
			}
			if err := writeJSON(filepath.Join(*output, name+"-reduction.json"), map[string]string{"diagnostic": diagnostic}); err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, diagnostic)
			return runErr
		}
		if *replay != "" {
			break
		}
	}
	return nil
}
func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}
