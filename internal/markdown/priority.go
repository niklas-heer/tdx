package markdown

import (
	"regexp"
	"strconv"
	"strings"
)

// priorityRegex matches priority markers like !p1, !p2, etc.
// Format: !p followed by a number (e.g., !p1, !p2, !p10)
var priorityRegex = regexp.MustCompile(`!p(\d+)`)

// ExtractPriority extracts the priority level from todo text.
// Returns the priority number (1, 2, 3, etc.) or 0 if no priority is set.
// If multiple priorities exist, returns the highest (lowest number).
func ExtractPriority(text string) int {
	matches := priorityRegex.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return 0
	}

	// Find the highest priority (lowest number)
	highestPriority := 0
	for _, match := range matches {
		if len(match) > 1 {
			priority, err := strconv.Atoi(match[1])
			if err == nil && priority > 0 {
				if highestPriority == 0 || priority < highestPriority {
					highestPriority = priority
				}
			}
		}
	}

	return highestPriority
}

// HasPriority checks if the text contains a priority marker
func HasPriority(text string) bool {
	return priorityRegex.MatchString(text)
}

// RemovePriority removes all priority markers from the text
// This is useful for display purposes if you want text without priority markers
func RemovePriority(text string) string {
	return strings.TrimSpace(priorityRegex.ReplaceAllString(text, ""))
}

// GetPriorityMarker returns the first priority marker found in the text (e.g., "!p1")
// Returns empty string if no priority is set
func GetPriorityMarker(text string) string {
	match := priorityRegex.FindString(text)
	return match
}

// GetAllPriorities returns all unique priorities from a list of todos, sorted ascending
func GetAllPriorities(todos []Todo) []int {
	prioritySet := make(map[int]bool)
	for _, todo := range todos {
		if todo.Priority > 0 {
			prioritySet[todo.Priority] = true
		}
	}

	priorities := make([]int, 0, len(prioritySet))
	for priority := range prioritySet {
		priorities = append(priorities, priority)
	}

	// Sort priorities ascending (p1 first)
	for i := 0; i < len(priorities)-1; i++ {
		for j := i + 1; j < len(priorities); j++ {
			if priorities[j] < priorities[i] {
				priorities[i], priorities[j] = priorities[j], priorities[i]
			}
		}
	}

	return priorities
}

// HasAnyPriority checks if a todo has any of the specified priorities
func (t *Todo) HasAnyPriority(priorities []int) bool {
	if len(priorities) == 0 {
		return true // No filter means match all
	}
	for _, p := range priorities {
		if t.Priority == p {
			return true
		}
	}
	return false
}
