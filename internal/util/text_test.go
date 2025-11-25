package util

import "testing"

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
	if len(lines) <= 1 {
		t.Errorf("Long text should wrap, got %v", lines)
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
