package markdown

import (
	"regexp"
	"sort"
	"strings"
	"time"
)

// dueRegex matches due date markers like @due(2025-11-24)
// Format: @due(YYYY-MM-DD)
var dueRegex = regexp.MustCompile(`@due\((\d{4}-\d{2}-\d{2})\)`)

// ExtractDueDate extracts the due date from todo text.
// Returns nil if no due date is set or if the date is invalid.
// If multiple due dates exist, returns the earliest one.
func ExtractDueDate(text string) *time.Time {
	matches := dueRegex.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}

	var earliestDate *time.Time
	for _, match := range matches {
		if len(match) > 1 {
			dateStr := match[1]
			parsedDate, err := time.Parse("2006-01-02", dateStr)
			if err == nil {
				if earliestDate == nil || parsedDate.Before(*earliestDate) {
					earliestDate = &parsedDate
				}
			}
		}
	}

	return earliestDate
}

// HasDueDate checks if the text contains a due date marker
func HasDueDate(text string) bool {
	return dueRegex.MatchString(text)
}

// RemoveDueDate removes all due date markers from the text
// This is useful for display purposes if you want text without due date markers
func RemoveDueDate(text string) string {
	return strings.TrimSpace(dueRegex.ReplaceAllString(text, ""))
}

// GetDueDateMarker returns the first due date marker found in the text (e.g., "@due(2025-11-24)")
// Returns empty string if no due date is set
func GetDueDateMarker(text string) string {
	match := dueRegex.FindString(text)
	return match
}

// IsOverdue checks if the due date is before today (start of day)
func IsOverdue(dueDate *time.Time) bool {
	if dueDate == nil {
		return false
	}
	today := time.Now().Truncate(24 * time.Hour)
	return dueDate.Before(today)
}

// IsDueToday checks if the due date is today
func IsDueToday(dueDate *time.Time) bool {
	if dueDate == nil {
		return false
	}
	today := time.Now().Truncate(24 * time.Hour)
	dueDay := dueDate.Truncate(24 * time.Hour)
	return dueDay.Equal(today)
}

// IsDueSoon checks if the due date is within the next N days (not including today, not overdue)
func IsDueSoon(dueDate *time.Time, days int) bool {
	if dueDate == nil {
		return false
	}
	today := time.Now().Truncate(24 * time.Hour)
	dueDay := dueDate.Truncate(24 * time.Hour)

	// Not overdue and not today
	if dueDay.Before(today) || dueDay.Equal(today) {
		return false
	}

	// Within N days
	deadline := today.AddDate(0, 0, days)
	return dueDay.Before(deadline) || dueDay.Equal(deadline)
}

// GetAllDueDates returns all unique due dates from a list of todos, sorted chronologically
func GetAllDueDates(todos []Todo) []time.Time {
	dateSet := make(map[time.Time]bool)
	for _, todo := range todos {
		if todo.DueDate != nil {
			// Normalize to start of day for comparison
			normalized := todo.DueDate.Truncate(24 * time.Hour)
			dateSet[normalized] = true
		}
	}

	dates := make([]time.Time, 0, len(dateSet))
	for date := range dateSet {
		dates = append(dates, date)
	}

	// Sort chronologically (earliest first)
	sort.Slice(dates, func(i, j int) bool {
		return dates[i].Before(dates[j])
	})

	return dates
}

// Todo helper methods

// IsOverdue checks if this todo is overdue
func (t *Todo) IsOverdue() bool {
	return IsOverdue(t.DueDate)
}

// IsDueToday checks if this todo is due today
func (t *Todo) IsDueToday() bool {
	return IsDueToday(t.DueDate)
}

// IsDueSoon checks if this todo is due within the next N days
func (t *Todo) IsDueSoon(days int) bool {
	return IsDueSoon(t.DueDate, days)
}

// HasDueDateFilter checks if this todo matches the due date filter
// filterType can be: "overdue", "today", "week", "all", or empty (matches all)
func (t *Todo) HasDueDateFilter(filterType string) bool {
	switch filterType {
	case "":
		return true // No filter means match all
	case "all":
		return t.DueDate != nil // Has any due date
	case "overdue":
		return t.IsOverdue()
	case "today":
		return t.IsDueToday()
	case "week":
		return t.IsDueToday() || t.IsDueSoon(7)
	default:
		return true
	}
}

// HasAnyDueDateFilter checks if a todo matches any of the specified due date filters
func (t *Todo) HasAnyDueDateFilter(filters []string) bool {
	if len(filters) == 0 {
		return true // No filter means match all
	}
	for _, filter := range filters {
		if t.HasDueDateFilter(filter) {
			return true
		}
	}
	return false
}
