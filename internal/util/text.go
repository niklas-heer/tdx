package util

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// FuzzyScore returns a score for how well query matches text
// Higher score = better match
func FuzzyScore(query, text string) int {
	if query == "" {
		return 0
	}

	// Exact substring match gets highest score
	if strings.Contains(text, query) {
		return 1000 + len(query)
	}

	// Fuzzy match: check if all query chars appear in order
	score := 0
	queryIdx := 0
	lastMatchIdx := -1

	for i := 0; i < len(text) && queryIdx < len(query); i++ {
		if text[i] == query[queryIdx] {
			score += 10
			// Bonus for consecutive matches
			if lastMatchIdx == i-1 {
				score += 5
			}
			// Bonus for matching at word start
			if i == 0 || text[i-1] == ' ' {
				score += 3
			}
			lastMatchIdx = i
			queryIdx++
		}
	}

	// Only count as match if all query chars were found
	if queryIdx == len(query) {
		return score
	}

	return 0
}

// WrapText wraps text to fit within maxWidth, returning multiple lines
// indent is the string to prepend to continuation lines
func WrapText(text string, maxWidth int, indent string) []string {
	if maxWidth <= 0 || len(text) <= maxWidth {
		return []string{text}
	}

	var lines []string
	indentWidth := runewidth.StringWidth(indent)
	firstLineWidth := maxWidth
	contLineWidth := maxWidth - indentWidth

	if contLineWidth <= 10 {
		// Too narrow to wrap meaningfully
		return []string{text}
	}

	remaining := text
	isFirst := true

	for len(remaining) > 0 {
		lineWidth := contLineWidth
		if isFirst {
			lineWidth = firstLineWidth
			isFirst = false
		}

		if runewidth.StringWidth(remaining) <= lineWidth {
			if len(lines) > 0 {
				lines = append(lines, indent+remaining)
			} else {
				lines = append(lines, remaining)
			}
			break
		}

		// Find break point (prefer space)
		breakAt := 0
		lastSpace := -1
		width := 0
		for i, r := range remaining {
			w := runewidth.RuneWidth(r)
			if width+w > lineWidth {
				break
			}
			width += w
			breakAt = i + len(string(r))
			if r == ' ' {
				lastSpace = breakAt
			}
		}

		// Prefer breaking at space
		if lastSpace > 0 && lastSpace > breakAt/2 {
			breakAt = lastSpace
		}

		line := remaining[:breakAt]
		if len(lines) > 0 {
			lines = append(lines, indent+line)
		} else {
			lines = append(lines, line)
		}
		remaining = strings.TrimLeft(remaining[breakAt:], " ")
	}

	return lines
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
