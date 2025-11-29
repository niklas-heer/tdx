package markdown

import (
	"testing"
	"time"
)

func TestExtractDueDate(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string // YYYY-MM-DD or empty for nil
	}{
		{
			name:     "due date at end",
			text:     "Submit report @due(2025-11-29)",
			expected: "2025-11-29",
		},
		{
			name:     "due date at start",
			text:     "@due(2025-12-01) Plan holiday party",
			expected: "2025-12-01",
		},
		{
			name:     "due date in middle",
			text:     "Fix bug @due(2025-11-30) urgently",
			expected: "2025-11-30",
		},
		{
			name:     "due date with tags and priority",
			text:     "Fix bug !p1 @due(2025-11-30) #urgent #backend",
			expected: "2025-11-30",
		},
		{
			name:     "no due date",
			text:     "Regular task without due date",
			expected: "",
		},
		{
			name:     "empty string",
			text:     "",
			expected: "",
		},
		{
			name:     "invalid date format - wrong separator",
			text:     "Task @due(2025/11/30)",
			expected: "",
		},
		{
			name:     "invalid date format - incomplete",
			text:     "Task @due(2025-11)",
			expected: "",
		},
		{
			name:     "due date with inline code",
			text:     "Fix `main.go` @due(2025-12-15)",
			expected: "2025-12-15",
		},
		{
			name:     "due date with link",
			text:     "Check [docs](http://example.com) @due(2025-12-20)",
			expected: "2025-12-20",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractDueDate(tt.text)
			if tt.expected == "" {
				if got != nil {
					t.Errorf("ExtractDueDate(%q) = %v, want nil", tt.text, got)
				}
			} else {
				if got == nil {
					t.Errorf("ExtractDueDate(%q) = nil, want %s", tt.text, tt.expected)
				} else {
					gotStr := got.Format("2006-01-02")
					if gotStr != tt.expected {
						t.Errorf("ExtractDueDate(%q) = %s, want %s", tt.text, gotStr, tt.expected)
					}
				}
			}
		})
	}
}

func TestExtractDueDate_MultipleDates(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string // Should return earliest date
	}{
		{
			name:     "two dates - first is earlier",
			text:     "Task @due(2025-11-20) @due(2025-11-25)",
			expected: "2025-11-20",
		},
		{
			name:     "two dates - second is earlier",
			text:     "Task @due(2025-12-01) @due(2025-11-15)",
			expected: "2025-11-15",
		},
		{
			name:     "three dates",
			text:     "Task @due(2025-12-25) @due(2025-11-01) @due(2025-12-01)",
			expected: "2025-11-01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractDueDate(tt.text)
			if got == nil {
				t.Errorf("ExtractDueDate(%q) = nil, want %s", tt.text, tt.expected)
			} else {
				gotStr := got.Format("2006-01-02")
				if gotStr != tt.expected {
					t.Errorf("ExtractDueDate(%q) = %s, want %s", tt.text, gotStr, tt.expected)
				}
			}
		})
	}
}

func TestHasDueDate(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name:     "has due date",
			text:     "Task @due(2025-11-29)",
			expected: true,
		},
		{
			name:     "no due date",
			text:     "Regular task",
			expected: false,
		},
		{
			name:     "empty string",
			text:     "",
			expected: false,
		},
		{
			name:     "similar but not due date - missing @",
			text:     "Task due(2025-11-29)",
			expected: false,
		},
		{
			name:     "similar but not due date - wrong format",
			text:     "Task @due(11-29-2025)",
			expected: false,
		},
		{
			name:     "at-mention not due date",
			text:     "Task @someone",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasDueDate(tt.text)
			if got != tt.expected {
				t.Errorf("HasDueDate(%q) = %v, want %v", tt.text, got, tt.expected)
			}
		})
	}
}

func TestRemoveDueDate(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "remove due date at end",
			text:     "Submit report @due(2025-11-29)",
			expected: "Submit report",
		},
		{
			name:     "remove due date at start",
			text:     "@due(2025-12-01) Plan party",
			expected: "Plan party",
		},
		{
			name:     "remove due date in middle",
			text:     "Fix bug @due(2025-11-30) urgently",
			expected: "Fix bug  urgently",
		},
		{
			name:     "remove multiple due dates",
			text:     "Task @due(2025-11-20) @due(2025-11-25)",
			expected: "Task",
		},
		{
			name:     "no due date to remove",
			text:     "Regular task",
			expected: "Regular task",
		},
		{
			name:     "preserve tags when removing due date",
			text:     "Fix bug @due(2025-11-30) #urgent",
			expected: "Fix bug  #urgent",
		},
		{
			name:     "preserve priority when removing due date",
			text:     "Fix bug !p1 @due(2025-11-30)",
			expected: "Fix bug !p1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoveDueDate(tt.text)
			if got != tt.expected {
				t.Errorf("RemoveDueDate(%q) = %q, want %q", tt.text, got, tt.expected)
			}
		})
	}
}

func TestGetDueDateMarker(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "get due date marker",
			text:     "Submit report @due(2025-11-29)",
			expected: "@due(2025-11-29)",
		},
		{
			name:     "no marker",
			text:     "Regular task",
			expected: "",
		},
		{
			name:     "multiple - returns first",
			text:     "Task @due(2025-11-20) @due(2025-11-25)",
			expected: "@due(2025-11-20)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetDueDateMarker(tt.text)
			if got != tt.expected {
				t.Errorf("GetDueDateMarker(%q) = %q, want %q", tt.text, got, tt.expected)
			}
		})
	}
}

func TestIsOverdue(t *testing.T) {
	today := time.Now().Truncate(24 * time.Hour)
	yesterday := today.AddDate(0, 0, -1)
	tomorrow := today.AddDate(0, 0, 1)

	tests := []struct {
		name     string
		dueDate  *time.Time
		expected bool
	}{
		{
			name:     "nil due date",
			dueDate:  nil,
			expected: false,
		},
		{
			name:     "yesterday is overdue",
			dueDate:  &yesterday,
			expected: true,
		},
		{
			name:     "today is not overdue",
			dueDate:  &today,
			expected: false,
		},
		{
			name:     "tomorrow is not overdue",
			dueDate:  &tomorrow,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsOverdue(tt.dueDate)
			if got != tt.expected {
				t.Errorf("IsOverdue(%v) = %v, want %v", tt.dueDate, got, tt.expected)
			}
		})
	}
}

func TestIsDueToday(t *testing.T) {
	today := time.Now().Truncate(24 * time.Hour)
	yesterday := today.AddDate(0, 0, -1)
	tomorrow := today.AddDate(0, 0, 1)

	tests := []struct {
		name     string
		dueDate  *time.Time
		expected bool
	}{
		{
			name:     "nil due date",
			dueDate:  nil,
			expected: false,
		},
		{
			name:     "yesterday is not today",
			dueDate:  &yesterday,
			expected: false,
		},
		{
			name:     "today is today",
			dueDate:  &today,
			expected: true,
		},
		{
			name:     "tomorrow is not today",
			dueDate:  &tomorrow,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDueToday(tt.dueDate)
			if got != tt.expected {
				t.Errorf("IsDueToday(%v) = %v, want %v", tt.dueDate, got, tt.expected)
			}
		})
	}
}

func TestIsDueSoon(t *testing.T) {
	today := time.Now().Truncate(24 * time.Hour)
	yesterday := today.AddDate(0, 0, -1)
	tomorrow := today.AddDate(0, 0, 1)
	in3Days := today.AddDate(0, 0, 3)
	in7Days := today.AddDate(0, 0, 7)
	in8Days := today.AddDate(0, 0, 8)

	tests := []struct {
		name     string
		dueDate  *time.Time
		days     int
		expected bool
	}{
		{
			name:     "nil due date",
			dueDate:  nil,
			days:     7,
			expected: false,
		},
		{
			name:     "yesterday is not soon",
			dueDate:  &yesterday,
			days:     7,
			expected: false,
		},
		{
			name:     "today is not soon (it's today)",
			dueDate:  &today,
			days:     7,
			expected: false,
		},
		{
			name:     "tomorrow is soon within 7 days",
			dueDate:  &tomorrow,
			days:     7,
			expected: true,
		},
		{
			name:     "in 3 days is soon within 7 days",
			dueDate:  &in3Days,
			days:     7,
			expected: true,
		},
		{
			name:     "in 7 days is soon within 7 days (edge)",
			dueDate:  &in7Days,
			days:     7,
			expected: true,
		},
		{
			name:     "in 8 days is not soon within 7 days",
			dueDate:  &in8Days,
			days:     7,
			expected: false,
		},
		{
			name:     "in 3 days is soon within 3 days",
			dueDate:  &in3Days,
			days:     3,
			expected: true,
		},
		{
			name:     "tomorrow is not soon within 0 days",
			dueDate:  &tomorrow,
			days:     0,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDueSoon(tt.dueDate, tt.days)
			if got != tt.expected {
				t.Errorf("IsDueSoon(%v, %d) = %v, want %v", tt.dueDate, tt.days, got, tt.expected)
			}
		})
	}
}

func TestGetAllDueDates(t *testing.T) {
	date1, _ := time.Parse("2006-01-02", "2025-11-20")
	date2, _ := time.Parse("2006-01-02", "2025-11-25")
	date3, _ := time.Parse("2006-01-02", "2025-12-01")

	tests := []struct {
		name     string
		todos    []Todo
		expected []string // YYYY-MM-DD format
	}{
		{
			name: "single due date",
			todos: []Todo{
				{Text: "Task 1 @due(2025-11-20)", DueDate: &date1},
				{Text: "Task 2", DueDate: nil},
			},
			expected: []string{"2025-11-20"},
		},
		{
			name: "multiple due dates sorted",
			todos: []Todo{
				{Text: "Task 1 @due(2025-12-01)", DueDate: &date3},
				{Text: "Task 2 @due(2025-11-20)", DueDate: &date1},
				{Text: "Task 3 @due(2025-11-25)", DueDate: &date2},
			},
			expected: []string{"2025-11-20", "2025-11-25", "2025-12-01"},
		},
		{
			name: "duplicate due dates",
			todos: []Todo{
				{Text: "Task 1 @due(2025-11-20)", DueDate: &date1},
				{Text: "Task 2 @due(2025-11-20)", DueDate: &date1},
				{Text: "Task 3 @due(2025-11-25)", DueDate: &date2},
			},
			expected: []string{"2025-11-20", "2025-11-25"},
		},
		{
			name: "no due dates",
			todos: []Todo{
				{Text: "Task 1", DueDate: nil},
				{Text: "Task 2", DueDate: nil},
			},
			expected: []string{},
		},
		{
			name:     "empty todos",
			todos:    []Todo{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetAllDueDates(tt.todos)
			if len(got) != len(tt.expected) {
				t.Errorf("GetAllDueDates() = %d items, want %d", len(got), len(tt.expected))
				return
			}
			for i, v := range got {
				gotStr := v.Format("2006-01-02")
				if gotStr != tt.expected[i] {
					t.Errorf("GetAllDueDates()[%d] = %s, want %s", i, gotStr, tt.expected[i])
				}
			}
		})
	}
}

func TestTodo_DueDateMethods(t *testing.T) {
	today := time.Now().Truncate(24 * time.Hour)
	yesterday := today.AddDate(0, 0, -1)
	tomorrow := today.AddDate(0, 0, 1)
	in3Days := today.AddDate(0, 0, 3)

	tests := []struct {
		name       string
		todo       Todo
		isOverdue  bool
		isDueToday bool
		isDueSoon3 bool
		isDueSoon7 bool
	}{
		{
			name:       "no due date",
			todo:       Todo{Text: "Task", DueDate: nil},
			isOverdue:  false,
			isDueToday: false,
			isDueSoon3: false,
			isDueSoon7: false,
		},
		{
			name:       "overdue task",
			todo:       Todo{Text: "Task @due(yesterday)", DueDate: &yesterday},
			isOverdue:  true,
			isDueToday: false,
			isDueSoon3: false,
			isDueSoon7: false,
		},
		{
			name:       "due today",
			todo:       Todo{Text: "Task @due(today)", DueDate: &today},
			isOverdue:  false,
			isDueToday: true,
			isDueSoon3: false,
			isDueSoon7: false,
		},
		{
			name:       "due tomorrow",
			todo:       Todo{Text: "Task @due(tomorrow)", DueDate: &tomorrow},
			isOverdue:  false,
			isDueToday: false,
			isDueSoon3: true,
			isDueSoon7: true,
		},
		{
			name:       "due in 3 days",
			todo:       Todo{Text: "Task @due(in3days)", DueDate: &in3Days},
			isOverdue:  false,
			isDueToday: false,
			isDueSoon3: true,
			isDueSoon7: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.todo.IsOverdue(); got != tt.isOverdue {
				t.Errorf("Todo.IsOverdue() = %v, want %v", got, tt.isOverdue)
			}
			if got := tt.todo.IsDueToday(); got != tt.isDueToday {
				t.Errorf("Todo.IsDueToday() = %v, want %v", got, tt.isDueToday)
			}
			if got := tt.todo.IsDueSoon(3); got != tt.isDueSoon3 {
				t.Errorf("Todo.IsDueSoon(3) = %v, want %v", got, tt.isDueSoon3)
			}
			if got := tt.todo.IsDueSoon(7); got != tt.isDueSoon7 {
				t.Errorf("Todo.IsDueSoon(7) = %v, want %v", got, tt.isDueSoon7)
			}
		})
	}
}

func TestTodo_HasDueDateFilter(t *testing.T) {
	today := time.Now().Truncate(24 * time.Hour)
	yesterday := today.AddDate(0, 0, -1)
	tomorrow := today.AddDate(0, 0, 1)
	in5Days := today.AddDate(0, 0, 5)
	in10Days := today.AddDate(0, 0, 10)

	tests := []struct {
		name       string
		todo       Todo
		filterType string
		expected   bool
	}{
		{
			name:       "empty filter matches all",
			todo:       Todo{Text: "Task", DueDate: nil},
			filterType: "",
			expected:   true,
		},
		{
			name:       "all filter - has due date",
			todo:       Todo{Text: "Task", DueDate: &tomorrow},
			filterType: "all",
			expected:   true,
		},
		{
			name:       "all filter - no due date",
			todo:       Todo{Text: "Task", DueDate: nil},
			filterType: "all",
			expected:   false,
		},
		{
			name:       "overdue filter - overdue task",
			todo:       Todo{Text: "Task", DueDate: &yesterday},
			filterType: "overdue",
			expected:   true,
		},
		{
			name:       "overdue filter - future task",
			todo:       Todo{Text: "Task", DueDate: &tomorrow},
			filterType: "overdue",
			expected:   false,
		},
		{
			name:       "today filter - today task",
			todo:       Todo{Text: "Task", DueDate: &today},
			filterType: "today",
			expected:   true,
		},
		{
			name:       "today filter - tomorrow task",
			todo:       Todo{Text: "Task", DueDate: &tomorrow},
			filterType: "today",
			expected:   false,
		},
		{
			name:       "week filter - today",
			todo:       Todo{Text: "Task", DueDate: &today},
			filterType: "week",
			expected:   true,
		},
		{
			name:       "week filter - in 5 days",
			todo:       Todo{Text: "Task", DueDate: &in5Days},
			filterType: "week",
			expected:   true,
		},
		{
			name:       "week filter - in 10 days",
			todo:       Todo{Text: "Task", DueDate: &in10Days},
			filterType: "week",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.todo.HasDueDateFilter(tt.filterType)
			if got != tt.expected {
				t.Errorf("Todo.HasDueDateFilter(%q) = %v, want %v", tt.filterType, got, tt.expected)
			}
		})
	}
}

func TestTodo_HasAnyDueDateFilter(t *testing.T) {
	today := time.Now().Truncate(24 * time.Hour)
	yesterday := today.AddDate(0, 0, -1)
	tomorrow := today.AddDate(0, 0, 1)

	tests := []struct {
		name     string
		todo     Todo
		filters  []string
		expected bool
	}{
		{
			name:     "empty filter matches all",
			todo:     Todo{Text: "Task", DueDate: nil},
			filters:  []string{},
			expected: true,
		},
		{
			name:     "matches one of multiple filters - overdue",
			todo:     Todo{Text: "Task", DueDate: &yesterday},
			filters:  []string{"overdue", "today"},
			expected: true,
		},
		{
			name:     "matches one of multiple filters - today",
			todo:     Todo{Text: "Task", DueDate: &today},
			filters:  []string{"overdue", "today"},
			expected: true,
		},
		{
			name:     "matches none of filters",
			todo:     Todo{Text: "Task", DueDate: &tomorrow},
			filters:  []string{"overdue", "today"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.todo.HasAnyDueDateFilter(tt.filters)
			if got != tt.expected {
				t.Errorf("Todo.HasAnyDueDateFilter(%v) = %v, want %v", tt.filters, got, tt.expected)
			}
		})
	}
}

// Benchmark tests
func BenchmarkExtractDueDate(b *testing.B) {
	text := "Fix critical bug !p1 @due(2025-11-30) #urgent #backend"
	for i := 0; i < b.N; i++ {
		ExtractDueDate(text)
	}
}

func BenchmarkHasDueDate(b *testing.B) {
	text := "Fix critical bug !p1 @due(2025-11-30) #urgent #backend"
	for i := 0; i < b.N; i++ {
		HasDueDate(text)
	}
}

func BenchmarkIsOverdue(b *testing.B) {
	yesterday := time.Now().AddDate(0, 0, -1)
	for i := 0; i < b.N; i++ {
		IsOverdue(&yesterday)
	}
}
