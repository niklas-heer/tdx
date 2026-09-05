package main

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestSharedFixtures(t *testing.T) {
	data, err := os.ReadFile("../fixtures.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct {
		Name, Source string
		Markers      []marker
	}
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			assertMarkers := func(source string, want []marker) {
				t.Helper()
				got := scan(source)
				for i := range got {
					got[i].offset = 0
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("markers = %+v; want %+v\n%s", got, want, source)
				}
			}
			assertMarkers(fixture.Source, fixture.Markers)
			for index := range fixture.Markers {
				want := append([]marker(nil), fixture.Markers...)
				want[index].Checked = !want[index].Checked
				patched, err := patch(fixture.Source, index)
				if err != nil {
					t.Fatal(err)
				}
				assertMarkers(patched, want)
				changes := 0
				if len(patched) != len(fixture.Source) {
					t.Fatal("patch changed source length")
				}
				for i := range patched {
					if patched[i] != fixture.Source[i] {
						changes++
					}
				}
				if changes != 1 {
					t.Fatalf("patch changed %d bytes", changes)
				}
				edited, err := production(fixture.Source, index)
				if err != nil {
					t.Fatal(err)
				}
				assertMarkers(edited, want)
			}
			if _, err := patch(fixture.Source, len(fixture.Markers)); err == nil {
				t.Fatal("accepted invalid index")
			}
		})
	}
}
