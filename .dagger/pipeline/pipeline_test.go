package pipeline

import (
	"reflect"
	"testing"
)

func TestParseMetadata(t *testing.T) {
	metadata, err := ParseMetadata("version = \"0.13.0\"\ndescription = \"your todos, in markdown, done fast\"\n")
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Version != "0.13.0" || metadata.Description != "your todos, in markdown, done fast" {
		t.Fatalf("unexpected metadata: %#v", metadata)
	}
}

func TestParseMetadataRequiresFields(t *testing.T) {
	if _, err := ParseMetadata("version = \"0.13.0\"\n"); err == nil {
		t.Fatal("expected missing description to fail")
	}
}

func TestCoverageColor(t *testing.T) {
	tests := []struct {
		coverage float64
		want     string
	}{
		{coverage: 39.99, want: "red"},
		{coverage: 40, want: "yellow"},
		{coverage: 60, want: "green"},
		{coverage: 80, want: "brightgreen"},
	}
	for _, test := range tests {
		if got := CoverageColor(test.coverage); got != test.want {
			t.Errorf("CoverageColor(%v) = %q, want %q", test.coverage, got, test.want)
		}
	}
}

func TestParseTotalCoverage(t *testing.T) {
	report := "github.com/niklas-heer/tdx/cmd/tdx/main.go:10:\tmain\t75.0%\ntotal:\t(statements)\t68.4%\n"
	percent, err := ParseTotalCoverage(report)
	if err != nil {
		t.Fatal(err)
	}
	if percent != "68.4" {
		t.Fatalf("ParseTotalCoverage() = %q, want %q", percent, "68.4")
	}
}

func TestParseTotalCoverageRequiresTotal(t *testing.T) {
	if _, err := ParseTotalCoverage("no total here"); err == nil {
		t.Fatal("expected missing total to fail")
	}
}

func TestCoverageBadgeJSON(t *testing.T) {
	want := "{\"schemaVersion\":1,\"label\":\"coverage\",\"message\":\"68.4%\",\"color\":\"green\"}\n"
	got, err := CoverageBadgeJSON("68.4")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("CoverageBadgeJSON() = %q, want %q", got, want)
	}
}

func TestCoverageBadgeJSONRejectsInvalidCoverage(t *testing.T) {
	if _, err := CoverageBadgeJSON("invalid"); err == nil {
		t.Fatal("expected invalid coverage to fail")
	}
}

func TestReleaseTargets(t *testing.T) {
	want := []Target{
		{Name: "tdx-darwin-amd64", OS: "darwin", Arch: "amd64"},
		{Name: "tdx-darwin-arm64", OS: "darwin", Arch: "arm64"},
		{Name: "tdx-linux-amd64", OS: "linux", Arch: "amd64"},
		{Name: "tdx-linux-arm64", OS: "linux", Arch: "arm64"},
		{Name: "tdx-windows-amd64.exe", OS: "windows", Arch: "amd64"},
	}
	if got := ReleaseTargets(); !reflect.DeepEqual(got, want) {
		t.Fatalf("ReleaseTargets() = %#v, want %#v", got, want)
	}
}

func TestReleaseTargetsReturnsCopy(t *testing.T) {
	targets := ReleaseTargets()
	targets[0].Name = "changed"
	if ReleaseTargets()[0].Name == "changed" {
		t.Fatal("ReleaseTargets returned mutable shared state")
	}
}
