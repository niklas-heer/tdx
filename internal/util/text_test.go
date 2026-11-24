package util

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestFuzzyScore_ExactMatch(t *testing.T) {
	score := FuzzyScore("task", "My task here")
	if score <= 0 {
		t.Errorf("Exact substring match should have positive score, got %d", score)
	}
}

func TestFuzzyScore_FuzzyMatch(t *testing.T) {
	score := FuzzyScore("tsk", "task")
	if score <= 0 {
		t.Errorf("Fuzzy match should have positive score, got %d", score)
	}
}

func TestFuzzyScore_NoMatch(t *testing.T) {
	score := FuzzyScore("xyz", "task")
	if score != 0 {
		t.Errorf("No match should return 0, got %d", score)
	}
}

func TestFuzzyScore_EmptyQuery(t *testing.T) {
	score := FuzzyScore("", "task")
	if score != 0 {
		t.Errorf("Empty query should return 0, got %d", score)
	}
}

func TestWrapText_NoWrapNeeded(t *testing.T) {
	lines := WrapText("short", 100, "  ")
	if len(lines) != 1 || lines[0] != "short" {
		t.Errorf("Short text should not wrap, got %v", lines)
	}
}

func TestWrapText_NeedsWrap(t *testing.T) {
	lines := WrapText("this is a long text that needs wrapping", 20, "  ")
	want := []string{"this is a long text", "  that needs wrapping"}
	if !reflect.DeepEqual(lines, want) {
		t.Errorf("got %q, want %q", lines, want)
	}
}

func TestWrapText_KeepsHyphenatedWordsWhole(t *testing.T) {
	lines := WrapText("ship it @due(2026-12-01) #work", 16, "")
	want := []string{"ship it", "@due(2026-12-01)", "#work"}
	if !reflect.DeepEqual(lines, want) {
		t.Errorf("got %q, want %q", lines, want)
	}
}

func TestWrapText_HardBreaksWordsWiderThanALine(t *testing.T) {
	lines := WrapText("go abcdefghijklmnopqrstuvwxyz", 12, " ")
	want := []string{"go", " abcdefghijkl", " mnopqrstuvwx", " yz"}
	if !reflect.DeepEqual(lines, want) {
		t.Errorf("got %q, want %q", lines, want)
	}
}

func TestWrapText_CarriesStylesAcrossLines(t *testing.T) {
	const dim = "\x1b[38;2;86;95;137m"
	lines := WrapText("plain words "+dim+"styled words that wrap\x1b[39m end", 16, "  ")
	for i, line := range lines {
		if strings.Count(line, "\x1b[") != strings.Count(line, "m") {
			t.Errorf("line %d splits an escape sequence: %q", i, line)
		}
		if w := ansi.StringWidth(line); w > 16+2 {
			t.Errorf("line %d is %d cells wide: %q", i, w, line)
		}
	}
	if !strings.HasPrefix(lines[1], "  "+dim) {
		t.Errorf("continuation line should reopen the colour after the indent: %q", lines[1])
	}
	if got := ansi.Strip(strings.Join(lines, "|")); got != "plain words|  styled words|  that wrap end" {
		t.Errorf("visible text %q", got)
	}
}

func TestMinMax(t *testing.T) {
	if Min(1, 2) != 1 {
		t.Error("Min(1, 2) should be 1")
	}
	if Max(1, 2) != 2 {
		t.Error("Max(1, 2) should be 2")
	}
	if Abs(-5) != 5 {
		t.Error("Abs(-5) should be 5")
	}
}

// Benchmarks for hot paths

func BenchmarkFuzzyScore_ExactMatch(b *testing.B) {
	text := "This is a sample todo item with some text @tag1 @tag2"
	query := "todo"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FuzzyScore(query, text)
	}
}

func BenchmarkFuzzyScore_FuzzyMatch(b *testing.B) {
	text := "This is a sample todo item with some text @tag1 @tag2"
	query := "smpt"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FuzzyScore(query, text)
	}
}

func BenchmarkFuzzyScore_NoMatch(b *testing.B) {
	text := "This is a sample todo item with some text @tag1 @tag2"
	query := "xyz"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FuzzyScore(query, text)
	}
}

func BenchmarkFuzzyScore_LongQuery(b *testing.B) {
	text := "This is a sample todo item with some text @tag1 @tag2"
	query := "sample todo item"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FuzzyScore(query, text)
	}
}

func BenchmarkWrapText_Short(b *testing.B) {
	text := "Short text"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		WrapText(text, 80, "  ")
	}
}

func BenchmarkWrapText_Long(b *testing.B) {
	text := "This is a much longer text that will need to be wrapped across multiple lines when displayed in the terminal interface with a limited width"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		WrapText(text, 40, "  ")
	}
}

func BenchmarkWrapText_VeryLong(b *testing.B) {
	text := "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat."
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		WrapText(text, 60, "    ")
	}
}

// Regression: a link destination may contain spaces, which must not count as
// word breaks; every row must link to the whole destination.
func TestWrapText_KeepsLinkDestinationsWhole(t *testing.T) {
	const url = "docs/a long file name.md"
	link := "\x1b]8;;" + url + "\x1b\\local documentation\x1b]8;;\x1b\\"
	lines := WrapText("read the "+link+" before you start", 13, "")
	if got := ansi.Strip(strings.Join(lines, "|")); got != "read the|local|documentation|before you|start" {
		t.Errorf("visible text %q", got)
	}
	for i, line := range lines {
		for _, dest := range regexp.MustCompile("\x1b]8;[^;]*;([^\x1b]*)\x1b\\\\").FindAllStringSubmatch(line, -1) {
			if dest[1] != "" && dest[1] != url {
				t.Errorf("line %d links to %q, want %q: %q", i, dest[1], url, line)
			}
		}
	}
}
