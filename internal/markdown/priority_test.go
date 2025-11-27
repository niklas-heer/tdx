package markdown

import "testing"

func TestExtractPriority(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "p1 priority",
			text:     "Fix critical bug !p1",
			expected: 1,
		},
		{
			name:     "p2 priority",
			text:     "Update dependencies !p2",
			expected: 2,
		},
		{
			name:     "p3 priority",
			text:     "Write documentation !p3",
			expected: 3,
		},
		{
			name:     "p10 priority (double digit)",
			text:     "Low priority task !p10",
			expected: 10,
		},
		{
			name:     "priority at start",
			text:     "!p1 Fix critical bug",
			expected: 1,
		},
		{
			name:     "priority in middle",
			text:     "Fix !p2 critical bug",
			expected: 2,
		},
		{
			name:     "no priority",
			text:     "Regular task without priority",
			expected: 0,
		},
		{
			name:     "empty string",
			text:     "",
			expected: 0,
		},
		{
			name:     "priority with tags",
			text:     "Fix bug !p1 #urgent #backend",
			expected: 1,
		},
		{
			name:     "priority with inline code",
			text:     "Fix `main.go` !p2",
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractPriority(tt.text)
			if got != tt.expected {
				t.Errorf("ExtractPriority(%q) = %d, want %d", tt.text, got, tt.expected)
			}
		})
	}
}

func TestExtractPriority_MultiplePriorities(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "p1 and p2 - takes highest (p1)",
			text:     "Task !p2 !p1",
			expected: 1,
		},
		{
			name:     "p3 and p1 - takes highest (p1)",
			text:     "Task !p3 !p1",
			expected: 1,
		},
		{
			name:     "p2 and p3 - takes highest (p2)",
			text:     "Task !p2 !p3",
			expected: 2,
		},
		{
			name:     "three priorities",
			text:     "Task !p5 !p2 !p3",
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractPriority(tt.text)
			if got != tt.expected {
				t.Errorf("ExtractPriority(%q) = %d, want %d", tt.text, got, tt.expected)
			}
		})
	}
}

func TestHasPriority(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name:     "has p1",
			text:     "Task !p1",
			expected: true,
		},
		{
			name:     "has p2",
			text:     "Task !p2",
			expected: true,
		},
		{
			name:     "no priority",
			text:     "Regular task",
			expected: false,
		},
		{
			name:     "empty string",
			text:     "",
			expected: false,
		},
		{
			name:     "similar but not priority",
			text:     "Task p1", // missing !
			expected: false,
		},
		{
			name:     "exclamation but not priority",
			text:     "Task !important",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasPriority(tt.text)
			if got != tt.expected {
				t.Errorf("HasPriority(%q) = %v, want %v", tt.text, got, tt.expected)
			}
		})
	}
}

func TestRemovePriority(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "remove p1 at end",
			text:     "Fix critical bug !p1",
			expected: "Fix critical bug",
		},
		{
			name:     "remove p1 at start",
			text:     "!p1 Fix critical bug",
			expected: "Fix critical bug",
		},
		{
			name:     "remove p2 in middle",
			text:     "Fix !p2 critical bug",
			expected: "Fix  critical bug",
		},
		{
			name:     "remove multiple priorities",
			text:     "Task !p1 !p2 !p3",
			expected: "Task",
		},
		{
			name:     "no priority to remove",
			text:     "Regular task",
			expected: "Regular task",
		},
		{
			name:     "preserve tags when removing priority",
			text:     "Fix bug !p1 #urgent",
			expected: "Fix bug  #urgent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemovePriority(tt.text)
			if got != tt.expected {
				t.Errorf("RemovePriority(%q) = %q, want %q", tt.text, got, tt.expected)
			}
		})
	}
}

func TestGetPriorityMarker(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "get p1 marker",
			text:     "Fix critical bug !p1",
			expected: "!p1",
		},
		{
			name:     "get p2 marker",
			text:     "Update deps !p2",
			expected: "!p2",
		},
		{
			name:     "get p10 marker",
			text:     "Low priority !p10",
			expected: "!p10",
		},
		{
			name:     "no marker",
			text:     "Regular task",
			expected: "",
		},
		{
			name:     "multiple - returns first",
			text:     "Task !p2 !p1",
			expected: "!p2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetPriorityMarker(tt.text)
			if got != tt.expected {
				t.Errorf("GetPriorityMarker(%q) = %q, want %q", tt.text, got, tt.expected)
			}
		})
	}
}

func TestGetAllPriorities(t *testing.T) {
	tests := []struct {
		name     string
		todos    []Todo
		expected []int
	}{
		{
			name: "single priority",
			todos: []Todo{
				{Text: "Task 1 !p1", Priority: 1},
				{Text: "Task 2", Priority: 0},
			},
			expected: []int{1},
		},
		{
			name: "multiple priorities sorted",
			todos: []Todo{
				{Text: "Task 1 !p3", Priority: 3},
				{Text: "Task 2 !p1", Priority: 1},
				{Text: "Task 3 !p2", Priority: 2},
			},
			expected: []int{1, 2, 3},
		},
		{
			name: "duplicate priorities",
			todos: []Todo{
				{Text: "Task 1 !p1", Priority: 1},
				{Text: "Task 2 !p1", Priority: 1},
				{Text: "Task 3 !p2", Priority: 2},
			},
			expected: []int{1, 2},
		},
		{
			name: "no priorities",
			todos: []Todo{
				{Text: "Task 1", Priority: 0},
				{Text: "Task 2", Priority: 0},
			},
			expected: []int{},
		},
		{
			name:     "empty todos",
			todos:    []Todo{},
			expected: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetAllPriorities(tt.todos)
			if len(got) != len(tt.expected) {
				t.Errorf("GetAllPriorities() = %v, want %v", got, tt.expected)
				return
			}
			for i, v := range got {
				if v != tt.expected[i] {
					t.Errorf("GetAllPriorities() = %v, want %v", got, tt.expected)
					return
				}
			}
		})
	}
}

func TestHasAnyPriority(t *testing.T) {
	tests := []struct {
		name       string
		todo       Todo
		priorities []int
		expected   bool
	}{
		{
			name:       "has matching priority",
			todo:       Todo{Text: "Task !p1", Priority: 1},
			priorities: []int{1, 2, 3},
			expected:   true,
		},
		{
			name:       "no matching priority",
			todo:       Todo{Text: "Task !p4", Priority: 4},
			priorities: []int{1, 2, 3},
			expected:   false,
		},
		{
			name:       "empty filter matches all",
			todo:       Todo{Text: "Task !p1", Priority: 1},
			priorities: []int{},
			expected:   true,
		},
		{
			name:       "no priority with filter",
			todo:       Todo{Text: "Task", Priority: 0},
			priorities: []int{1, 2},
			expected:   false,
		},
		{
			name:       "single priority match",
			todo:       Todo{Text: "Task !p2", Priority: 2},
			priorities: []int{2},
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.todo.HasAnyPriority(tt.priorities)
			if got != tt.expected {
				t.Errorf("Todo.HasAnyPriority(%v) = %v, want %v", tt.priorities, got, tt.expected)
			}
		})
	}
}

// Benchmark tests
func BenchmarkExtractPriority(b *testing.B) {
	text := "Fix critical bug !p1 #urgent #backend"
	for i := 0; i < b.N; i++ {
		ExtractPriority(text)
	}
}

func BenchmarkHasPriority(b *testing.B) {
	text := "Fix critical bug !p1 #urgent #backend"
	for i := 0; i < b.N; i++ {
		HasPriority(text)
	}
}
