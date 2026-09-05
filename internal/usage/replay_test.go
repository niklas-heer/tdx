package usage

import (
	"encoding/json"
	"reflect"
	"strconv"
	"testing"
)

func TestSeededSessions(t *testing.T) {
	for _, seed := range []uint64{100, 101, 102} {
		t.Run(strconv.FormatUint(seed, 10), func(t *testing.T) {
			trace := Generate(seed, 120, "tui")
			if result, err := Replay(trace, ""); err != nil {
				t.Fatalf("%+v: %v", result, err)
			}
		})
	}
}
func TestPortableTrace(t *testing.T) {
	trace := Generate(73, 200, "tui")
	data, err := json.Marshal(trace)
	if err != nil {
		t.Fatal(err)
	}
	var copy Trace
	if err = json.Unmarshal(data, &copy); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(trace, copy) || !reflect.DeepEqual(trace, Generate(73, 200, "tui")) {
		t.Fatal("trace is not reproducible")
	}
	trace.Steps[0].Expected[0].Text = "incorrect oracle"
	if _, err = Replay(trace, ""); err == nil {
		t.Fatal("accepted invalid expectation")
	}
}

func TestExternalReloadRefreshesMetadata(t *testing.T) {
	for _, op := range []string{"external", "conflict-reload"} {
		t.Run(op, func(t *testing.T) {
			trace := Trace{Schema: Schema, Driver: "tui", Initial: []Task{{Text: "original #old !p2"}}, Steps: []Step{{Op: op, Seconds: 10, Text: "external #new !p1", Expected: []Task{{Text: "external #new !p1"}}}}}
			if _, err := Replay(trace, ""); err != nil {
				t.Fatal(err)
			}
		})
	}
}
