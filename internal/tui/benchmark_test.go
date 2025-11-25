package tui

import (
	"fmt"
	"testing"

	"github.com/niklas-heer/tdx/internal/markdown"
)

// generateTodos creates n todos for benchmarking
func generateTodos(n int) []markdown.Todo {
	todos := make([]markdown.Todo, n)
	for i := 0; i < n; i++ {
		todos[i] = markdown.Todo{
			Text:    fmt.Sprintf("Task %d: Do something important with @tag%d and more text here", i, i%5),
			Checked: i%3 == 0,
			Tags:    []string{fmt.Sprintf("@tag%d", i%5)},
		}
	}
	return todos
}

func BenchmarkUpdateSearchResults_10Todos(b *testing.B) {
	fm := &markdown.FileModel{Todos: generateTodos(10)}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.InputBuffer = "task"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.updateSearchResults()
	}
}

func BenchmarkUpdateSearchResults_100Todos(b *testing.B) {
	fm := &markdown.FileModel{Todos: generateTodos(100)}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.InputBuffer = "task"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.updateSearchResults()
	}
}

func BenchmarkUpdateSearchResults_500Todos(b *testing.B) {
	fm := &markdown.FileModel{Todos: generateTodos(500)}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.InputBuffer = "task"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.updateSearchResults()
	}
}

func BenchmarkUpdateSearchResults_EmptyQuery(b *testing.B) {
	fm := &markdown.FileModel{Todos: generateTodos(100)}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.InputBuffer = ""

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.updateSearchResults()
	}
}

func BenchmarkUpdateFilteredCommands(b *testing.B) {
	fm := &markdown.FileModel{Todos: generateTodos(10)}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.InputBuffer = "check"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.updateFilteredCommands()
	}
}

func BenchmarkGetHeadings_Cached(b *testing.B) {
	fm := &markdown.FileModel{Todos: generateTodos(100)}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	// Pre-populate cache
	m.GetHeadings()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.GetHeadings()
	}
}

func BenchmarkGetHeadings_Uncached(b *testing.B) {
	fm := &markdown.FileModel{Todos: generateTodos(100)}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.InvalidateHeadingsCache()
		m.GetHeadings()
	}
}

func BenchmarkGetVisibleTodos_NoFilter(b *testing.B) {
	fm := &markdown.FileModel{Todos: generateTodos(100)}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.getVisibleTodos()
	}
}

func BenchmarkGetVisibleTodos_FilterDone(b *testing.B) {
	fm := &markdown.FileModel{Todos: generateTodos(100)}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.FilterDone = true

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.getVisibleTodos()
	}
}

func BenchmarkFindNextVisibleTodo(b *testing.B) {
	fm := &markdown.FileModel{Todos: generateTodos(100)}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.FilterDone = true

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.findNextVisibleTodo(0)
	}
}

func BenchmarkFindPreviousVisibleTodo(b *testing.B) {
	fm := &markdown.FileModel{Todos: generateTodos(100)}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")
	m.FilterDone = true

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.findPreviousVisibleTodo(99)
	}
}

func BenchmarkProcessPipedInput_Navigation(b *testing.B) {
	fm := &markdown.FileModel{Todos: generateTodos(100)}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := New("/tmp/test.md", fm, true, false, -1, testConfig(), testStyles(), "")
		m.ProcessPipedInput([]byte("jjjjjkkkkk"))
	}
}

func BenchmarkProcessPipedInput_Search(b *testing.B) {
	fm := &markdown.FileModel{Todos: generateTodos(100)}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := New("/tmp/test.md", fm, true, false, -1, testConfig(), testStyles(), "")
		m.ProcessPipedInput([]byte("/task\r"))
	}
}
