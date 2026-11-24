package util

import (
	"bytes"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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

// WrapText wraps styled text to maxWidth terminal cells and prepends indent to
// continuation lines, so they align with the first line's text. It breaks at
// spaces, so tokens such as @due(2026-12-01) stay whole, and splits only a word
// wider than a line. Escape sequences are never split; colours and links are
// closed at the end of a line and reopened on the next.
func WrapText(text string, maxWidth int, indent string) []string {
	// Below this, wrapping produces a column of fragments; the frame truncates.
	const minWrapWidth = 10
	if maxWidth <= minWrapWidth || ansi.StringWidth(text) <= maxWidth {
		return []string{text}
	}

	var wrapped, line strings.Builder
	lineWidth := 0
	newline := func() {
		wrapped.WriteString(line.String())
		wrapped.WriteByte('\n')
		line.Reset()
		lineWidth = 0
	}
	// Escape sequences contain no spaces, so each one stays inside a word.
	for _, word := range strings.Split(text, " ") {
		width := ansi.StringWidth(word)
		switch {
		case line.Len() == 0:
		case lineWidth+1+width <= maxWidth:
			line.WriteByte(' ')
			lineWidth++
		default:
			newline()
		}
		if width > maxWidth {
			pieces := strings.Split(ansi.Hardwrap(word, maxWidth, false), "\n")
			for _, piece := range pieces[:len(pieces)-1] {
				line.WriteString(piece)
				newline()
			}
			word = pieces[len(pieces)-1]
			width = ansi.StringWidth(word)
		}
		line.WriteString(word)
		lineWidth += width
	}
	wrapped.WriteString(line.String())

	var styled bytes.Buffer
	writer := lipgloss.NewWrapWriter(&styled)
	_, _ = writer.Write([]byte(wrapped.String()))
	_ = writer.Close()

	lines := strings.Split(styled.String(), "\n")
	for i := 1; i < len(lines); i++ {
		lines[i] = indent + lines[i]
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
