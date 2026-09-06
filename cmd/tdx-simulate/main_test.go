package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReplayTraceAndArgumentBounds(t *testing.T) {
	for _, args := range [][]string{{"--runs", "0"}, {"--seed", "18446744073709551615", "--runs", "2"}, {"--steps", "100001"}, {"--mutation", "unknown"}, {"--check"}, {"tasks.md"}, {"--runs", "2", "--trace", "no.json"}} {
		if err := run(args, &bytes.Buffer{}); err == nil {
			t.Fatalf("accepted invalid arguments: %v", args)
		}
	}
	path := filepath.Join(t.TempDir(), "seed.json")
	var first, second bytes.Buffer
	if err := run([]string{"--seed", "3", "--trace", path}, &first); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"--seed", "3"}, &second); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatal("different replay evidence")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var trace struct {
		Digest string `json:"trace_sha256"`
	}
	if err := json.Unmarshal(data, &trace); err != nil {
		t.Fatal(err)
	}
	var summary evidence
	if err := json.Unmarshal(first.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if trace.Digest != summary.Digests[0] {
		t.Fatal("trace and summary differ")
	}
}
func TestMutationReturnsReplayableFailure(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"--mutation", "skip-sync"}, &out); err == nil {
		t.Fatal("accepted broken durability")
	}
	var report struct {
		Seed     uint64
		Failures []string
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Seed != 0 || len(report.Failures) == 0 {
		t.Fatal("missing replay failure")
	}
}
