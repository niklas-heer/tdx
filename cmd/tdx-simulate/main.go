// tdx-simulate is a developer tool. It never loads Markdown documents or opens
// the user's history database; all storage effects execute in a virtual world.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"runtime/debug"

	"github.com/niklas-heer/tdx/internal/savesim"
)

type evidence struct {
	Schema   int                       `json:"schema"`
	Seed     uint64                    `json:"seed"`
	Runs     int                       `json:"runs"`
	Steps    int                       `json:"steps"`
	Counts   map[string]uint64         `json:"counts"`
	Digests  []string                  `json:"trace_sha256"`
	Replay   bool                      `json:"identical_replay"`
	Controls map[string]savesim.Report `json:"negative_controls,omitempty"`
	Build    map[string]string         `json:"build"`
	Scope    string                    `json:"scope"`
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0600)
}
func run(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("tdx-simulate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	seed := flags.Uint64("seed", 0, "first replay seed")
	runs := flags.Int("runs", 1, "number of seeds (1..100000)")
	steps := flags.Int("steps", 200, "scheduled effects per seed (1..100000)")
	trace := flags.String("trace", "", "write one complete trace as JSON (requires one run)")
	output := flags.String("output", "", "write campaign evidence as JSON")
	mutation := flags.String("mutation", "", "negative control: skip-validation, skip-sync, skip-replacement")
	check := flags.Bool("check", false, "require fault coverage, exact replay and independent negative controls")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *runs < 1 || *runs > 100000 || *steps < 1 || *steps > 100000 || (*trace != "" && *runs != 1) || uint64(*runs-1) > math.MaxUint64-*seed {
		return errors.New("runs/steps must be 1..100000; trace requires one run; seed range must not overflow; no positional arguments")
	}
	m := savesim.Mutation(*mutation)
	if m != savesim.NoMutation && m != savesim.SkipValidation && m != savesim.SkipSync && m != savesim.SkipReplacement {
		return errors.New("unknown mutation")
	}
	if *check && (*runs < 1000 || *steps < 200 || m != savesim.NoMutation) {
		return errors.New("check requires at least 1000 seeds, 200 steps and no mutation")
	}
	report := evidence{Schema: 1, Seed: *seed, Runs: *runs, Steps: *steps, Counts: map[string]uint64{}, Digests: []string{}, Replay: *check, Build: map[string]string{}, Scope: "Production Go save protocol with simulated I/O outcomes and virtual time. Native filesystem/SQLite internals, arbitrary non-cooperating post-validation edits and hardware guarantees are outside the model. No network service exists."}
	if build, ok := debug.ReadBuildInfo(); ok {
		report.Build["go"] = build.GoVersion
		for _, setting := range build.Settings {
			if setting.Key == "vcs.revision" || setting.Key == "vcs.modified" {
				report.Build[setting.Key] = setting.Value
			}
		}
	}
	report.Build["binary_sha256"] = "unavailable"
	if executable, err := os.Executable(); err == nil {
		if file, err := os.Open(executable); err == nil {
			hash := sha256.New()
			_, hashErr := io.Copy(hash, file)
			closeErr := file.Close()
			if hashErr == nil && closeErr == nil {
				report.Build["binary_sha256"] = fmt.Sprintf("%x", hash.Sum(nil))
			}
		}
	}
	for offset := 0; offset < *runs; offset++ {
		r := savesim.Run(*seed+uint64(offset), *steps, m)
		if *trace != "" {
			if err := writeJSON(*trace, r); err != nil {
				return err
			}
		}
		if len(r.Failures) > 0 {
			if *output != "" {
				if err := writeJSON(*output, r); err != nil {
					return err
				}
			}
			if err := json.NewEncoder(out).Encode(r); err != nil {
				return err
			}
			return fmt.Errorf("invariant failure; replay --seed %d --runs 1 --steps %d --mutation=%s", r.Seed, *steps, m)
		}
		if *check && !reflect.DeepEqual(r, savesim.Run(r.Seed, *steps, m)) {
			return fmt.Errorf("non-deterministic seed %d", r.Seed)
		}
		for key, n := range r.Counts {
			report.Counts[key] += n
		}
		report.Digests = append(report.Digests, r.Digest)
	}
	if *check {
		for _, key := range []string{"process_crash", "power_loss", "lock_busy", "external_edit", "fault/Prepare", "fault/Lock", "fault/Validate", "fault/CaptureBefore", "fault/Replace", "fault/SyncDirectory", "fault/CaptureAfter", "fault/Unlock"} {
			if report.Counts[key] == 0 {
				return fmt.Errorf("unexercised fault: %s", key)
			}
		}
		if report.Counts["recovery_passed"] != uint64(*runs) {
			return errors.New("not all seeds recovered")
		}
		report.Controls = map[string]savesim.Report{}
		for _, mutation := range []savesim.Mutation{savesim.SkipValidation, savesim.SkipSync, savesim.SkipReplacement} {
			for seed := uint64(0); seed < 32; seed++ {
				r := savesim.Run(seed, *steps, mutation)
				if len(r.Failures) > 0 {
					report.Controls[string(mutation)] = r
					break
				}
			}
			if _, ok := report.Controls[string(mutation)]; !ok {
				return fmt.Errorf("undetected negative control: %s", mutation)
			}
		}
	}
	if *output != "" {
		if err := writeJSON(*output, report); err != nil {
			return err
		}
	}
	return json.NewEncoder(out).Encode(report)
}
func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
