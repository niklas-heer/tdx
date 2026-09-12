package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Leave one row for the inline renderer's cursor. Oversized frames scroll
// previous content out of its redraw area and leave permanent terminal residue.
func (m Model) frameHeight() int {
	if m.TermHeight <= 0 {
		return 0
	}
	return max(1, m.TermHeight-1)
}

// fitFrame prevents terminal soft wrapping and bounds every mode, including
// overlays. Keep the final footer when a modal is taller than the terminal.
func fitFrame(content string, width, height int) string {
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if height > 0 && len(lines) > height {
		lines = append(lines[:height-1], lines[len(lines)-1])
	}
	if width > 0 {
		for i, line := range lines {
			lines[i] = ansi.Truncate(line, width, "…")
		}
	}
	return strings.Join(lines, "\n")
}

// taskViewport scrolls rendered rows, rather than treating a wrapped task or a
// heading as a single task slot. The selected task/editor cursor is the anchor.
func taskViewport(content string, anchor, height int) string {
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if len(lines) <= height {
		return strings.Join(lines, "\n")
	}
	anchor = min(max(0, anchor), len(lines)-1)
	rows := height
	if height >= 3 {
		rows -= 2
	}
	start := min(max(0, anchor-rows/2), len(lines)-rows)
	end := start + rows
	visible := append([]string(nil), lines[start:end]...)
	if height >= 3 {
		above, below := "", ""
		if start > 0 {
			above = "      ▲ more"
		}
		if end < len(lines) {
			below = "      ▼ more"
		}
		visible = append([]string{above}, visible...)
		visible = append(visible, below)
	}
	return strings.Join(visible, "\n")
}

// renderEditorLine keeps the cursor visible both in wrapped text and when
// wrapping is disabled. Widths use terminal cells, including custom markers.
func (m Model) renderEditorLine(prefix string) (string, int) {
	before, after := m.InputBuffer[:m.CursorPos], m.InputBuffer[m.CursorPos:]
	cursor := lipgloss.NewStyle().Reverse(true).Render(" ")
	text := before + cursor + after
	width := m.TermWidth - lipgloss.Width(prefix)
	if m.TermWidth <= 0 {
		return prefix + text + "\n", 0
	}
	width = max(1, width)
	if !m.WordWrap {
		left := max(0, ansi.StringWidth(before)+1-width)
		visible := ansi.Cut(text, left, left+width)
		// A cut through a double-width grapheme can include its whole cell.
		// Remove that leading cluster rather than truncate the cursor's tail.
		if excess := ansi.StringWidth(visible) - width; excess > 0 {
			visible = ansi.TruncateLeft(visible, excess+1, "")
		}
		return prefix + visible + "\n", 0
	}
	// Hard wrapping preserves spaces and the exact cursor position during input.
	lines := strings.Split(ansi.Hardwrap(text, width, true), "\n")
	cursorRow := strings.Count(ansi.Hardwrap(before+cursor, width, true), "\n")
	indent := strings.Repeat(" ", lipgloss.Width(prefix))
	return prefix + strings.Join(lines, "\n"+indent) + "\n", cursorRow
}
