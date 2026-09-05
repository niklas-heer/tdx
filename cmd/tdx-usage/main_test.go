package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/niklas-heer/tdx/internal/usage"
)

func TestReplayCannotEscapeOutputDirectory(t *testing.T) {
	for _, driver := range []string{"../escaped", "nested/../../escaped", `..\escaped`, "unknown"} {
		t.Run(driver, func(t *testing.T) {
			dir := t.TempDir()
			output := filepath.Join(dir, "out")
			protected := filepath.Join(dir, "escaped-1.json")
			if err := os.WriteFile(protected, []byte("keep"), 0600); err != nil {
				t.Fatal(err)
			}
			trace := usage.Trace{Schema: usage.Schema, Seed: 1, Driver: driver}
			input := filepath.Join(dir, "input.json")
			if err := writeJSON(input, trace); err != nil {
				t.Fatal(err)
			}
			if err := run([]string{"-replay", input, "-output", output}); err == nil {
				t.Fatal("accepted unsupported driver")
			}
			got, err := os.ReadFile(protected)
			if err != nil || string(got) != "keep" {
				t.Fatal("replay overwrote a file outside output", err)
			}
			entries, err := os.ReadDir(output)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Fatal("rejected driver created output artifacts")
			}
		})
	}
}

func TestReplayWritesValidReport(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.json")
	output := filepath.Join(dir, "out")
	if err := writeJSON(input, usage.Generate(123, 3, "tui")); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"-replay", input, "-output", output}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(output, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var results []usage.Result
	if err = json.Unmarshal(data, &results); err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Actions != 3 || results[0].Error != "" {
		t.Fatalf("invalid replay report: %s", data)
	}
}

func TestHelpSucceeds(t *testing.T) {
	if err := run([]string{"-help"}); err != nil {
		t.Fatal(err)
	}
}
