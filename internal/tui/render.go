package tui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/niklas-heer/tdx/internal/util"
)

// Pre-compiled regexes for inline code rendering (performance optimization)
var (
	linkRe = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	codeRe = regexp.MustCompile("`([^`]+)`")
	tagRe  = regexp.MustCompile(`#([a-zA-Z0-9_-]+)`)
)

// RenderInlineCode renders text with backtick-enclosed code and markdown links highlighted
func RenderInlineCode(text string, isChecked bool, magentaStyle, cyanStyle, codeStyleFunc func(string) string) string {
	// Use unique markers to preserve links and code blocks during processing
	type segment struct {
		text   string
		isLink bool
		isCode bool
		url    string
	}

	var segments []segment
	remaining := text

	for len(remaining) > 0 {
		linkMatch := linkRe.FindStringSubmatchIndex(remaining)
		codeMatch := codeRe.FindStringSubmatchIndex(remaining)

		// Determine which comes first
		nextLink := -1
		nextCode := -1
		if linkMatch != nil {
			nextLink = linkMatch[0]
		}
		if codeMatch != nil {
			nextCode = codeMatch[0]
		}

		if nextLink == -1 && nextCode == -1 {
			// No more special elements
			segments = append(segments, segment{text: remaining})
			break
		}

		if nextLink != -1 && (nextCode == -1 || nextLink < nextCode) {
			// Link comes first
			if nextLink > 0 {
				segments = append(segments, segment{text: remaining[:nextLink]})
			}
			linkText := remaining[linkMatch[2]:linkMatch[3]]
			url := remaining[linkMatch[4]:linkMatch[5]]
			segments = append(segments, segment{text: linkText, isLink: true, url: url})
			remaining = remaining[linkMatch[1]:]
		} else {
			// Code comes first
			if nextCode > 0 {
				segments = append(segments, segment{text: remaining[:nextCode]})
			}
			code := remaining[codeMatch[2]:codeMatch[3]]
			segments = append(segments, segment{text: code, isCode: true})
			remaining = remaining[codeMatch[1]:]
		}
	}

	// Build result
	var result strings.Builder
	for _, seg := range segments {
		if seg.isLink {
			// OSC 8 hyperlink with cyan text
			result.WriteString(fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", seg.url, cyanStyle(seg.text)))
		} else if seg.isCode {
			result.WriteString(codeStyleFunc(" " + seg.text + " "))
		} else {
			// Regular text - apply magenta if checked
			if isChecked {
				result.WriteString(magentaStyle(seg.text))
			} else {
				result.WriteString(seg.text)
			}
		}
	}

	return result.String()
}

// RenderHelp renders the help screen
func RenderHelp(version string, cyanStyle, dimStyle func(string) string) string {
	var b strings.Builder

	title := cyanStyle("tdx") + " " + dimStyle("v"+version)
	b.WriteString("\n  " + title + "\n\n")

	// Define columns: header and entries (key, description)
	type entry struct {
		key  string
		desc string
	}
	type column struct {
		header  string
		entries []entry
	}

	columns := []column{
		{
			header: "NAVIGATION",
			entries: []entry{
				{"j", "Down"},
				{"k", "Up"},
				{"5j", "Jump 5 down"},
				{"/", "Search"},
				{"f", "Filter tags"},
			},
		},
		{
			header: "EDITING",
			entries: []entry{
				{"␣", "Toggle"},
				{"n", "New after"},
				{"N", "New at end"},
				{"e", "Edit"},
				{"d", "Delete"},
				{"c", "Copy"},
				{"m", "Move"},
			},
		},
		{
			header: "OTHER",
			entries: []entry{
				{"u", "Undo"},
				{"r", "Recent files"},
				{"?", "Help"},
				{"esc", "Quit"},
			},
		},
	}

	// Helper to get display width (proper unicode width)
	displayWidth := func(s string) int {
		return runewidth.StringWidth(s)
	}

	// Calculate max key width and max desc width per column
	keyWidths := make([]int, len(columns))
	descWidths := make([]int, len(columns))
	for i, col := range columns {
		for _, e := range col.entries {
			kw := displayWidth(e.key)
			dw := displayWidth(e.desc)
			if kw > keyWidths[i] {
				keyWidths[i] = kw
			}
			if dw > descWidths[i] {
				descWidths[i] = dw
			}
		}
	}

	// Calculate column widths: key + gap + desc + padding
	colWidths := make([]int, len(columns))
	for i, col := range columns {
		// Column width is max of: header width OR (keyWidth + 2 + descWidth)
		entryWidth := keyWidths[i] + 2 + descWidths[i]
		headerWidth := displayWidth(col.header)
		if entryWidth > headerWidth {
			colWidths[i] = entryWidth
		} else {
			colWidths[i] = headerWidth
		}
		// Add padding between columns
		colWidths[i] += 2
	}

	// Find max rows
	maxRows := 0
	for _, col := range columns {
		if len(col.entries) > maxRows {
			maxRows = len(col.entries)
		}
	}

	// Render header row with centered headers
	b.WriteString("  ")
	for i, col := range columns {
		header := col.header
		padding := colWidths[i] - displayWidth(header)
		leftPad := padding / 2
		rightPad := padding - leftPad
		b.WriteString(strings.Repeat(" ", leftPad) + cyanStyle(header) + strings.Repeat(" ", rightPad))
	}
	b.WriteString("\n")

	// Render entry rows
	for row := 0; row < maxRows; row++ {
		b.WriteString("  ")
		for i, col := range columns {
			if row < len(col.entries) {
				e := col.entries[row]
				// Pad key to max key width in this column
				keyPad := keyWidths[i] - displayWidth(e.key)
				content := cyanStyle(e.key) + strings.Repeat(" ", keyPad) + "  " + e.desc
				// Pad to column width
				visibleLen := keyWidths[i] + 2 + displayWidth(e.desc)
				padding := colWidths[i] - visibleLen
				b.WriteString(content + strings.Repeat(" ", padding))
			} else {
				b.WriteString(strings.Repeat(" ", colWidths[i]))
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle("  Press ") + cyanStyle("?") + dimStyle(" or ") + cyanStyle("esc") + dimStyle(" to close help"))

	return b.String()
}

// RenderTodoLine renders a single todo line with wrapping support
func RenderTodoLine(
	prefix string,
	text string,
	plainText string,
	isSearchMode bool,
	searchQuery string,
	isChecked bool,
	wordWrap bool,
	termWidth int,
	prefixWidth int,
	magentaStyle, cyanStyle, codeStyleFunc, dimStyle func(string) string,
) string {
	var b strings.Builder

	// Apply word wrap if enabled
	if wordWrap && termWidth > 0 {
		availWidth := termWidth - prefixWidth
		indent := strings.Repeat(" ", prefixWidth)

		// Wrap the STYLED text (which already has escape codes for links, code, tags)
		// This preserves rendering but may break across visual boundaries
		wrappedLines := util.WrapText(text, availWidth, indent)

		for i, line := range wrappedLines {
			if i == 0 {
				b.WriteString(prefix + line + "\n")
			} else {
				b.WriteString(line + "\n")
			}
		}
	} else {
		b.WriteString(prefix + text + "\n")
	}

	return b.String()
}

// ModeIndicator creates a styled mode indicator box
func ModeIndicator(icon, label string) string {
	// Use double space for wide emojis (like 🏷, 🔍) to align properly
	spacing := "  "
	return lipgloss.NewStyle().
		Background(lipgloss.Color("#3b4261")).
		Foreground(lipgloss.Color("#c0caf5")).
		Padding(0, 1).
		Render(icon + spacing + label)
}

// ColorizeTags highlights hashtags in the text with a specific color
func ColorizeTags(text string, tagStyle func(string) string) string {
	return tagRe.ReplaceAllStringFunc(text, func(match string) string {
		return tagStyle(match)
	})
}
