package savesim

import (
	"reflect"
	"testing"
)

func TestDeterministicCampaign(t *testing.T) {
	counts := map[string]uint64{}
	for seed := uint64(0); seed < 1000; seed++ {
		first, second := Run(seed, 200, NoMutation), Run(seed, 200, NoMutation)
		if len(first.Failures) > 0 {
			t.Fatalf("replay: go run ./cmd/tdx-simulate --seed %d --runs 1 --trace failure.json: %v", seed, first.Failures)
		}
		if !reflect.DeepEqual(first, second) {
			t.Fatalf("non-deterministic seed %d", seed)
		}
		if first.Counts["recovery_passed"] != 1 {
			t.Fatalf("no recovery for seed %d", seed)
		}
		for key, value := range first.Counts {
			counts[key] += value
		}
	}
	for _, key := range []string{"process_crash", "power_loss", "lock_busy", "external_edit", "fault/Prepare", "fault/Lock", "fault/Validate", "fault/CaptureBefore", "fault/Replace", "fault/SyncDirectory", "fault/CaptureAfter", "fault/Unlock"} {
		if counts[key] == 0 {
			t.Errorf("unexercised fault: %s", key)
		}
	}
	t.Logf("1000 seeds, 200 steps, exact replay and recovery; fault coverage: %v", counts)
}
func TestIndependentOracleDetectsBrokenGuarantees(t *testing.T) {
	for _, mutation := range []Mutation{SkipValidation, SkipSync, SkipReplacement} {
		found := false
		for seed := uint64(0); seed < 32; seed++ {
			report := Run(seed, 200, mutation)
			if len(report.Failures) > 0 {
				found = true
				t.Logf("%s detected at seed %d: %v", mutation, seed, report.Failures)
				break
			}
		}
		if !found {
			t.Fatalf("undetected mutation: %s", mutation)
		}
	}
}
