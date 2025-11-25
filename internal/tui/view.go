package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/niklas-heer/tdx/internal/markdown"
	"github.com/niklas-heer/tdx/internal/util"
	overlay "github.com/rmhubbert/bubbletea-overlay"
)

// Global config and styles - injected from main
type StyleFuncsType struct {
	Magenta func(string) string
	Cyan    func(string) string
	Dim     func(string) string
	Green   func(string) string
	Yellow  func(string) string
	Code    func(string) string
}

type ConfigType struct {
	Display struct {
		CheckSymbol  string
		SelectMarker string
		MaxVisible   int
	}
}

var (
	Config     *ConfigType
	StyleFuncs *StyleFuncsType
	Version    string
)

// View renders the TUI
func (m Model) View() string {
	if m.HelpMode {
		return RenderHelp(Version, StyleFuncs.Cyan, StyleFuncs.Dim)
	}

	// Render main content and status bar
	mainContent := m.renderMainContent()
	statusBar := m.renderStatusBar()

	// Combine main content and status bar
	background := mainContent + "\n" + statusBar

	// If there's an overlay active, composite it on top
	if m.FilterMode {
		// Ensure there's space for overlay positioning
		contentLines := strings.Count(mainContent, "\n")
		minLines := 10 // Minimum lines to ensure overlay positioning works well
		if contentLines < minLines {
			for i := contentLines; i < minLines; i++ {
				background += "\n"
			}
		}

		overlayContent := m.renderFilterOverlayCompact()
		// Position overlay just above status bar
		return overlay.Composite(overlayContent, background, overlay.Left, overlay.Bottom, 0, -1)
	}

	if m.CommandMode {
		// Ensure there's space for overlay positioning
		contentLines := strings.Count(mainContent, "\n")
		minLines := 10 // Minimum lines to ensure overlay positioning works well
		if contentLines < minLines {
			for i := contentLines; i < minLines; i++ {
				background += "\n"
			}
		}

		overlayContent := m.renderCommandOverlayCompact()
		// Position overlay just above status bar
		return overlay.Composite(overlayContent, background, overlay.Left, overlay.Bottom, 0, -1)
	}

	return background
}

// renderMainContent renders the main todo list (without status bar)
func (m Model) renderMainContent() string {
	var b strings.Builder

	if len(m.FileModel.Todos) == 0 && !m.InputMode {
		b.WriteString(StyleFuncs.Dim("No todos. Press 'n' to create one."))
		b.WriteString("\n")
	}

	// Determine which todos to display
	var todosToShow []int
	if m.SearchMode {
		todosToShow = m.SearchResults
	} else {
		for i := range m.FileModel.Todos {
			todo := m.FileModel.Todos[i]

			// Apply filter-done if enabled
			if m.FilterDone && todo.Checked {
				continue
			}

			// Apply tag filtering if active
			if len(m.FilteredTags) > 0 && !todo.HasAnyTag(m.FilteredTags) {
				continue
			}

			todosToShow = append(todosToShow, i)
		}
	}

	// Apply max_visible limit with scrolling
	startIdx := 0
	totalCount := len(todosToShow)
	hasMoreAbove := false
	hasMoreBelow := false

	// When in input mode, reserve one slot for the new task input
	// Use maxVisibleOverride if set (>= 0), otherwise use config
	configMaxVisible := Config.Display.MaxVisible
	if m.MaxVisibleOverride >= 0 {
		configMaxVisible = m.MaxVisibleOverride
	}
	effectiveMaxVisible := configMaxVisible
	if m.InputMode && configMaxVisible > 0 {
		effectiveMaxVisible = configMaxVisible - 1
	}

	if effectiveMaxVisible > 0 && len(todosToShow) > effectiveMaxVisible {
		// Calculate visible window centered on selection
		var currentPos int
		if m.SearchMode {
			currentPos = m.SearchCursor
		} else if m.InputMode {
			// When adding new task, scroll to show last items before the input
			currentPos = totalCount - 1
		} else {
			// Find position of selectedIndex in todosToShow
			for i, idx := range todosToShow {
				if idx == m.SelectedIndex {
					currentPos = i
					break
				}
			}
		}

		// Center the window on current position
		halfWindow := effectiveMaxVisible / 2
		startIdx = currentPos - halfWindow
		if startIdx < 0 {
			startIdx = 0
		}
		endIdx := startIdx + effectiveMaxVisible
		if endIdx > totalCount {
			endIdx = totalCount
			startIdx = endIdx - effectiveMaxVisible
			if startIdx < 0 {
				startIdx = 0
			}
		}

		hasMoreAbove = startIdx > 0
		hasMoreBelow = endIdx < totalCount || m.InputMode
		todosToShow = todosToShow[startIdx:endIdx]
	} else if m.InputMode && effectiveMaxVisible > 0 && len(todosToShow) == effectiveMaxVisible {
		// Edge case: exactly at the limit when adding new task
		hasMoreBelow = true
	}

	// Show indicator for items above (always reserve space when max_visible is set)
	if configMaxVisible > 0 && totalCount > configMaxVisible {
		if hasMoreAbove {
			b.WriteString(fmt.Sprintf("      %s\n", StyleFuncs.Dim(fmt.Sprintf("▲ %d more", startIdx))))
		} else {
			b.WriteString("\n")
		}
	}

	// Find position of selected item in visible list for relative indexing
	selectedVisiblePos := 0
	for i, idx := range todosToShow {
		if idx == m.SelectedIndex {
			selectedVisiblePos = startIdx + i
			break
		}
	}

	// Get all headings if enabled
	var allHeadings []markdown.Heading
	if m.ShowHeadings {
		allHeadings = m.FileModel.GetHeadings()
	}

	// Track the last displayed todo index to show headings in between
	lastDisplayedTodoIdx := -1

	for displayIdx, todoIdx := range todosToShow {
		todo := m.FileModel.Todos[todoIdx]

		// Show headings that fall between last displayed todo and current todo
		if m.ShowHeadings {
			for _, heading := range allHeadings {
				// Show heading if it appears after the last displayed todo
				// and before or at the current todo
				if heading.BeforeTodoIndex > lastDisplayedTodoIdx && heading.BeforeTodoIndex <= todoIdx {
					// Render heading with appropriate formatting
					headingText := strings.Repeat("#", heading.Level) + " " + heading.Text
					b.WriteString(fmt.Sprintf("      %s\n", StyleFuncs.Cyan(headingText)))
				}
			}
		}

		lastDisplayedTodoIdx = todoIdx

		var isSelected bool
		var relIndex int

		if m.SearchMode {
			actualIdx := startIdx + displayIdx
			isSelected = actualIdx == m.SearchCursor
			relIndex = actualIdx - m.SearchCursor
		} else if m.InputMode {
			// When adding new task, all existing items are above (negative indices)
			isSelected = false
			relIndex = (startIdx + displayIdx) - totalCount
		} else {
			isSelected = todoIdx == m.SelectedIndex
			// Use position in visible list for relative index
			relIndex = (startIdx + displayIdx) - selectedVisiblePos
		}

		// Relative index
		var indexStr string
		if m.HideLineNumbers {
			indexStr = "   "
		} else {
			indexStr = fmt.Sprintf("%+3d", relIndex)
			if relIndex == 0 {
				indexStr = "  0"
			}
		}

		// Arrow - don't show on existing items when in input mode
		arrow := "   "
		if isSelected && !m.InputMode {
			arrow = StyleFuncs.Cyan(" " + Config.Display.SelectMarker + " ")
		}

		// Checkbox
		var checkbox string
		if todo.Checked {
			checkbox = StyleFuncs.Magenta("[" + Config.Display.CheckSymbol + "]")
		} else {
			checkbox = StyleFuncs.Dim("[ ]")
		}

		// Move indicator (check before building prefix)
		if m.MoveMode && isSelected {
			arrow = StyleFuncs.Yellow(" ≡ ")
		}

		// Build the line prefix (needed early for edit mode wrapping)
		prefix := fmt.Sprintf("%s%s%s ", StyleFuncs.Dim(indexStr), arrow, checkbox)
		prefixWidth := 3 + 3 + 3 + 1 // index(3) + arrow(3) + checkbox(3) + space(1)

		// Text with inline code rendering and tag colorization
		var text string
		var plainText string = todo.Text

		if m.SearchMode && m.InputBuffer != "" {
			// Highlight matches during search
			text = HighlightMatches(todo.Text, m.InputBuffer, StyleFuncs.Green)
		} else {
			text = RenderInlineCode(todo.Text, todo.Checked, StyleFuncs.Magenta, StyleFuncs.Cyan, StyleFuncs.Code)
			// Colorize tags
			text = ColorizeTags(text, StyleFuncs.Yellow)
		}

		// Show edit cursor if in edit mode on this item
		if m.EditMode && isSelected && !m.SearchMode {
			plainText = m.InputBuffer

			// If wrapping is enabled, insert cursor and wrap the text
			if m.WordWrap && m.TermWidth > 0 {
				before := m.InputBuffer[:m.CursorPos]
				after := m.InputBuffer[m.CursorPos:]
				cursor := lipgloss.NewStyle().Reverse(true).Render(" ")
				textWithCursor := before + cursor + after

				// Wrap text with cursor included
				availWidth := m.TermWidth - prefixWidth
				indent := strings.Repeat(" ", prefixWidth)
				wrappedLines := util.WrapText(textWithCursor, availWidth, indent)

				// Render wrapped lines
				for i, line := range wrappedLines {
					if i == 0 {
						b.WriteString(prefix + line + "\n")
					} else {
						b.WriteString(line + "\n")
					}
				}
				continue // Skip normal rendering
			} else {
				// No wrapping - simple cursor insertion
				before := m.InputBuffer[:m.CursorPos]
				after := m.InputBuffer[m.CursorPos:]
				text = before + lipgloss.NewStyle().Reverse(true).Render(" ") + after
			}
		}

		// Render the todo line
		b.WriteString(RenderTodoLine(
			prefix, text, plainText,
			m.SearchMode, m.InputBuffer, todo.Checked,
			m.WordWrap, m.TermWidth, prefixWidth,
			StyleFuncs.Magenta, StyleFuncs.Cyan, StyleFuncs.Code, StyleFuncs.Dim,
		))
	}

	// Input mode - show new task as part of the visible window
	if m.InputMode {
		arrow := StyleFuncs.Cyan(" " + Config.Display.SelectMarker + " ")
		checkbox := StyleFuncs.Dim("[ ]")
		before := m.InputBuffer[:m.CursorPos]
		after := m.InputBuffer[m.CursorPos:]
		cursor := lipgloss.NewStyle().Reverse(true).Render(" ")
		b.WriteString(fmt.Sprintf("%s%s%s %s%s%s\n", StyleFuncs.Dim("  0"), arrow, checkbox, before, cursor, after))
	}

	// Show indicator for items below (always reserve space when max_visible is set)
	if configMaxVisible > 0 && totalCount > configMaxVisible {
		if hasMoreBelow && !m.InputMode {
			b.WriteString(fmt.Sprintf("      %s\n", StyleFuncs.Dim(fmt.Sprintf("▼ %d more", totalCount-startIdx-len(todosToShow)))))
		} else {
			b.WriteString("\n")
		}
	}

	// Show message when search has no results
	if m.SearchMode && len(m.SearchResults) == 0 && m.InputBuffer != "" {
		b.WriteString(StyleFuncs.Dim("  No matches found"))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	return b.String()
}

// renderStatusBar renders the status bar at the bottom
func (m Model) renderStatusBar() string {
	var b strings.Builder

	// Show error if present
	if m.Err != nil {
		errorStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("#f7768e")).
			Foreground(lipgloss.Color("#1a1b26")).
			Bold(true).
			Padding(0, 1)
		b.WriteString(errorStyle.Render("⚠ " + m.Err.Error()))
		b.WriteString(" ")

		// Show helpful hints based on error type
		if m.Err.Error() == "file changed externally" {
			b.WriteString(StyleFuncs.Cyan(":reload") + StyleFuncs.Dim(" or ") + StyleFuncs.Cyan(":force-save"))
		} else {
			b.WriteString(StyleFuncs.Dim("any key to dismiss"))
		}
		b.WriteString("\n")
	}

	// Status bar
	if m.CommandMode {
		b.WriteString(ModeIndicator("⌘", "COMMAND"))
		b.WriteString("  ")
		b.WriteString(StyleFuncs.Dim("Type to filter commands"))
	} else if m.FilterMode {
		b.WriteString(ModeIndicator("🏷", "FILTER"))
		b.WriteString("  ")
		b.WriteString(StyleFuncs.Dim("Select tags to filter todos"))
	} else if m.SearchMode {
		b.WriteString(ModeIndicator("🔍", "SEARCH"))
		b.WriteString("  ")
		before := m.InputBuffer[:m.CursorPos]
		after := m.InputBuffer[m.CursorPos:]
		cursor := lipgloss.NewStyle().Reverse(true).Render(" ")
		b.WriteString(before + cursor + after)
		b.WriteString(StyleFuncs.Dim("  ↑/↓ navigate  enter select  esc cancel"))
	} else if m.MaxVisibleInputMode {
		b.WriteString(ModeIndicator("⊙", "SET MAX"))
		b.WriteString("  ")
		before := m.InputBuffer[:m.CursorPos]
		after := m.InputBuffer[m.CursorPos:]
		cursor := lipgloss.NewStyle().Reverse(true).Render(" ")
		b.WriteString(before + cursor + after)
		b.WriteString(StyleFuncs.Dim("  enter confirm  esc cancel"))
	} else if m.InputMode {
		b.WriteString(ModeIndicator("✎", "NEW"))
		b.WriteString("  ")
		b.WriteString(StyleFuncs.Dim("enter confirm  esc cancel"))
	} else if m.EditMode {
		b.WriteString(ModeIndicator("✎", "EDIT"))
		b.WriteString("  ")
		b.WriteString(StyleFuncs.Dim("enter confirm  esc cancel"))
	} else if m.MoveMode {
		b.WriteString(ModeIndicator("≡", "MOVE"))
		b.WriteString("  ")
		b.WriteString(StyleFuncs.Dim("j/k move  enter confirm  esc cancel"))
	} else if m.CopyFeedback {
		b.WriteString(StyleFuncs.Green("✓ Copied to clipboard!"))
	} else {
		// Normal status bar with mode indicators and help
		var indicators []string
		if m.ReadOnly {
			indicators = append(indicators, "🔒READ ONLY")
		}
		if m.FilterDone {
			indicators = append(indicators, "⊘ FILTERED")
		}
		if len(m.FilteredTags) > 0 {
			tagList := strings.Join(m.FilteredTags, " #")
			indicators = append(indicators, fmt.Sprintf("🏷  #%s", tagList))
		}
		if m.WordWrap {
			indicators = append(indicators, "↩ WRAP")
		}
		if m.ShowHeadings {
			indicators = append(indicators, "# HEADINGS")
		}
		if m.MaxVisibleOverride >= 0 {
			indicators = append(indicators, fmt.Sprintf("⊙ MAX:%d", m.MaxVisibleOverride))
		}

		// Show indicator block with background if any indicators are active
		if len(indicators) > 0 {
			indicatorText := strings.Join(indicators, " │ ")
			indicatorStyle := lipgloss.NewStyle().
				Background(lipgloss.Color("#3b4261")).
				Foreground(lipgloss.Color("#c0caf5")).
				Padding(0, 1)
			b.WriteString(indicatorStyle.Render(indicatorText))
			b.WriteString("  ")
		}

		// Help hints
		var helpParts []string
		helpParts = append(helpParts, StyleFuncs.Cyan("?")+StyleFuncs.Dim(" help"))
		helpParts = append(helpParts, StyleFuncs.Cyan(":")+StyleFuncs.Dim(" cmd"))
		helpParts = append(helpParts, StyleFuncs.Cyan("n")+StyleFuncs.Dim(" new"))
		helpParts = append(helpParts, StyleFuncs.Cyan("␣")+StyleFuncs.Dim(" toggle"))
		helpParts = append(helpParts, StyleFuncs.Cyan("esc")+StyleFuncs.Dim(" quit"))
		b.WriteString(strings.Join(helpParts, "  "))
	}

	return b.String()
}

// renderFilterOverlay renders the tag filter overlay as a centered modal
func (m Model) renderFilterOverlay() string {
	var b strings.Builder

	// Header
	b.WriteString(ModeIndicator("🏷", "FILTER"))
	b.WriteString("\n\n")
	b.WriteString(StyleFuncs.Dim("Select tags to filter:"))
	b.WriteString("\n\n")

	// Display available tags
	for i, tag := range m.AvailableTags {
		isSelected := i == m.TagFilterCursor
		isActive := false
		for _, activeTag := range m.FilteredTags {
			if activeTag == tag {
				isActive = true
				break
			}
		}

		var marker string
		if isSelected {
			marker = StyleFuncs.Cyan("→ ")
		} else {
			marker = "  "
		}

		var checkbox string
		if isActive {
			checkbox = StyleFuncs.Green("[✓] ")
		} else {
			checkbox = StyleFuncs.Dim("[ ] ")
		}

		b.WriteString(marker + checkbox + StyleFuncs.Cyan("#"+tag))
		b.WriteString("\n")
	}

	// Footer with help
	b.WriteString("\n")
	b.WriteString(StyleFuncs.Dim("  ↑/↓ navigate  space toggle  c clear all  esc done"))

	// Style the overlay with a border and background (extension of status bar)
	content := b.String()
	overlayStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.Border{
			Top:         "─",
			Bottom:      "",
			Left:        "│",
			Right:       "│",
			TopLeft:     "┌",
			TopRight:    "┐",
			BottomLeft:  "",
			BottomRight: "",
		}).
		BorderForeground(lipgloss.Color("#7aa2f7")).
		Background(lipgloss.Color("#1a1b26")).
		Padding(1, 2).
		Width(50)

	return overlayStyle.Render(content)
}

// renderCommandOverlayCompact renders a compact modal command palette
func (m Model) renderCommandOverlayCompact() string {
	var b strings.Builder

	// Command input with cursor
	before := m.InputBuffer[:m.CursorPos]
	after := m.InputBuffer[m.CursorPos:]
	cursor := lipgloss.NewStyle().Reverse(true).Render(" ")
	b.WriteString(StyleFuncs.Cyan(":") + before + cursor + after)
	b.WriteString("\n")

	// Show top 5 matching commands with scrolling
	maxCmds := 5
	totalCmds := len(m.FilteredCmds)

	if totalCmds == 0 {
		b.WriteString(StyleFuncs.Dim("  No matching commands"))
		b.WriteString("\n")
	} else {
		// Calculate scroll window to keep cursor visible
		startIdx := 0
		if m.CommandCursor >= maxCmds {
			startIdx = m.CommandCursor - maxCmds + 1
		}

		endIdx := startIdx + maxCmds
		if endIdx > totalCmds {
			endIdx = totalCmds
			startIdx = endIdx - maxCmds
			if startIdx < 0 {
				startIdx = 0
			}
		}

		// Display commands in the visible window
		for i := startIdx; i < endIdx; i++ {
			cmdIdx := m.FilteredCmds[i]
			cmd := m.Commands[cmdIdx]
			isSelected := i == m.CommandCursor

			var marker string
			if isSelected {
				marker = StyleFuncs.Cyan("→ ")
			} else {
				marker = "  "
			}

			cmdName := cmd.Name
			if isSelected {
				cmdName = StyleFuncs.Cyan(cmdName)
			}

			cmdDesc := StyleFuncs.Dim(" - " + cmd.Description)
			b.WriteString(marker + cmdName + cmdDesc)
			b.WriteString("\n")
		}

		// Show indicator for more commands
		if totalCmds > maxCmds {
			remaining := totalCmds - maxCmds
			b.WriteString(StyleFuncs.Dim(fmt.Sprintf("  ... %d more (type to filter)", remaining)))
			b.WriteString("\n")
		}
	}

	// Style as compact modal
	content := b.String()
	overlayStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.Border{
			Top:         "─",
			Bottom:      "─",
			Left:        "│",
			Right:       "│",
			TopLeft:     "┌",
			TopRight:    "┐",
			BottomLeft:  "└",
			BottomRight: "┘",
		}).
		BorderForeground(lipgloss.Color("#7aa2f7")).
		Padding(0, 1)

	return overlayStyle.Render(content)
}

// renderFilterOverlayCompact renders a compact modal filter selector
func (m Model) renderFilterOverlayCompact() string {
	var b strings.Builder

	// Display tags vertically in compact modal
	maxTags := 8
	displayCount := len(m.AvailableTags)
	if displayCount > maxTags {
		displayCount = maxTags
	}

	for i := 0; i < displayCount; i++ {
		tag := m.AvailableTags[i]
		isSelected := i == m.TagFilterCursor
		isActive := false
		for _, activeTag := range m.FilteredTags {
			if activeTag == tag {
				isActive = true
				break
			}
		}

		var marker string
		if isSelected {
			marker = StyleFuncs.Cyan("→ ")
		} else {
			marker = "  "
		}

		var checkbox string
		if isActive {
			checkbox = StyleFuncs.Green("[✓] ")
		} else {
			checkbox = StyleFuncs.Dim("[ ] ")
		}

		tagText := StyleFuncs.Cyan("#" + tag)
		b.WriteString(marker + checkbox + tagText)
		b.WriteString("\n")
	}

	// Add help text
	b.WriteString("\n")
	b.WriteString(StyleFuncs.Dim("space toggle  c clear  esc done"))

	// Style as compact modal
	content := b.String()
	overlayStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.Border{
			Top:         "─",
			Bottom:      "─",
			Left:        "│",
			Right:       "│",
			TopLeft:     "┌",
			TopRight:    "┐",
			BottomLeft:  "└",
			BottomRight: "┘",
		}).
		BorderForeground(lipgloss.Color("#7aa2f7")).
		Padding(0, 1)

	return overlayStyle.Render(content)
}

// renderCommandOverlay renders the command palette overlay as a bottom-left modal
func (m Model) renderCommandOverlay() string {
	var b strings.Builder

	// Header
	b.WriteString(ModeIndicator("⌘", "COMMAND"))
	b.WriteString("\n\n")

	// Command input with cursor
	before := m.InputBuffer[:m.CursorPos]
	after := m.InputBuffer[m.CursorPos:]
	cursor := lipgloss.NewStyle().Reverse(true).Render(" ")
	b.WriteString(StyleFuncs.Cyan(":") + before + cursor + after)
	b.WriteString("\n\n")

	// Display filtered commands
	for i, cmdIdx := range m.FilteredCmds {
		if i >= 5 { // Limit display to first 5 commands
			break
		}

		cmd := m.Commands[cmdIdx]
		isSelected := i == m.CommandCursor

		var marker string
		if isSelected {
			marker = StyleFuncs.Cyan("→ ")
		} else {
			marker = "  "
		}

		// Command name and description
		cmdName := StyleFuncs.Cyan(cmd.Name)
		cmdDesc := StyleFuncs.Dim(" - " + cmd.Description)
		b.WriteString(marker + cmdName + cmdDesc)
		b.WriteString("\n")
	}

	// Footer with help
	b.WriteString("\n")
	b.WriteString(StyleFuncs.Dim("  ↑/↓ navigate  tab complete  enter execute  esc cancel"))

	// Style the overlay with a border and background (extension of status bar)
	content := b.String()
	overlayStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.Border{
			Top:         "─",
			Bottom:      "",
			Left:        "│",
			Right:       "│",
			TopLeft:     "┌",
			TopRight:    "┐",
			BottomLeft:  "",
			BottomRight: "",
		}).
		BorderForeground(lipgloss.Color("#7aa2f7")).
		Background(lipgloss.Color("#1a1b26")).
		Padding(1, 2).
		Width(60)

	return overlayStyle.Render(content)
}
