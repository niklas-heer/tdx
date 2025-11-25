package util

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

// ANSI escape code regex (matches CSI sequences and OSC 8 hyperlinks)
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m|\x1b\]8;;[^\x1b]*\x1b\\`)

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

// StripANSI removes ANSI escape codes from text
func StripANSI(text string) string {
	return ansiRe.ReplaceAllString(text, "")
}

// VisibleWidth returns the visible width of text, ignoring ANSI escape codes
func VisibleWidth(text string) int {
	stripped := StripANSI(text)
	return runewidth.StringWidth(stripped)
}

// WrapText wraps text to fit within maxWidth, returning multiple lines
// indent is the string to prepend to continuation lines
// NOTE: This function is ANSI-aware - it calculates visual width correctly
// even when text contains escape codes (colors, hyperlinks, etc.)
func WrapText(text string, maxWidth int, indent string) []string {
	visibleWidth := VisibleWidth(text)
	if maxWidth <= 0 || visibleWidth <= maxWidth {
		return []string{text}
	}

	var lines []string
	indentWidth := VisibleWidth(indent)
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

		if VisibleWidth(remaining) <= lineWidth {
			if len(lines) > 0 {
				lines = append(lines, indent+remaining)
			} else {
				lines = append(lines, remaining)
			}
			break
		}

		// Find break point (prefer space)
		// We need to track both byte position and visible width
		breakAt := 0
		lastSpace := -1
		lastSpaceWidth := 0
		width := 0
		inEscape := false

		i := 0
		for i < len(remaining) {
			// Check for start of escape sequence
			if remaining[i] == '\x1b' {
				inEscape = true
				i++
				continue
			}

			// If we're in an escape sequence, find the end
			if inEscape {
				// CSI sequence ends with a letter
				if remaining[i] >= 'A' && remaining[i] <= 'z' {
					inEscape = false
					i++
					continue
				} else if remaining[i] == ']' {
					// OSC sequence - scan until we find the terminator
					for i < len(remaining)-1 {
						i++
						if remaining[i] == '\x1b' && i+1 < len(remaining) && remaining[i+1] == '\\' {
							i += 2
							inEscape = false
							break
						}
					}
					continue
				}
				i++
				continue
			}

			// Regular character - use rune iteration
			r, size := utf8.DecodeRuneInString(remaining[i:])
			w := runewidth.RuneWidth(r)

			if width+w > lineWidth {
				break
			}

			width += w
			i += size
			breakAt = i

			if r == ' ' {
				lastSpace = breakAt
				lastSpaceWidth = width
			}
		}

		// Prefer breaking at space if it's not too far back
		if lastSpace > 0 && lastSpaceWidth > lineWidth/2 {
			breakAt = lastSpace
		}

		if breakAt == 0 {
			// Can't fit even one character - force break
			breakAt = 1
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
