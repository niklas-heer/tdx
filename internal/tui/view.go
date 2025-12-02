package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/niklas-heer/tdx/internal/config"
	"github.com/niklas-heer/tdx/internal/markdown"
	"github.com/niklas-heer/tdx/internal/util"
	overlay "github.com/rmhubbert/bubbletea-overlay"
)

// View renders the TUI
func (m Model) View() string {
	styles := m.Styles()
	if m.HelpMode {
		return RenderHelp(m.Version(), styles.Cyan, styles.Dim)
	}

	// Render main content and status bar
	mainContent := m.renderMainContent()
	statusBar := m.renderStatusBar()

	// Combine main content and status bar
	background := mainContent + "\n" + statusBar

	// If there's an overlay active, composite it on top
	if m.RecentFilesMode {
		// Ensure there's space for overlay positioning
		contentLines := strings.Count(mainContent, "\n")
		minLines := 10 // Minimum lines to ensure overlay positioning works well
		if contentLines < minLines {
			for i := contentLines; i < minLines; i++ {
				background += "\n"
			}
		}

		overlayContent := m.renderRecentFilesOverlay()
		// Position overlay just above status bar
		return overlay.Composite(overlayContent, background, overlay.Left, overlay.Bottom, 0, -1)
	}

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

	if m.PriorityFilterMode {
		// Ensure there's space for overlay positioning
		contentLines := strings.Count(mainContent, "\n")
		minLines := 10 // Minimum lines to ensure overlay positioning works well
		if contentLines < minLines {
			for i := contentLines; i < minLines; i++ {
				background += "\n"
			}
		}

		overlayContent := m.renderPriorityFilterOverlayCompact()
		// Position overlay just above status bar
		return overlay.Composite(overlayContent, background, overlay.Left, overlay.Bottom, 0, -1)
	}

	if m.DueFilterMode {
		// Ensure there's space for overlay positioning
		contentLines := strings.Count(mainContent, "\n")
		minLines := 10 // Minimum lines to ensure overlay positioning works well
		if contentLines < minLines {
			for i := contentLines; i < minLines; i++ {
				background += "\n"
			}
		}

		overlayContent := m.renderDueFilterOverlayCompact()
		// Position overlay just above status bar
		return overlay.Composite(overlayContent, background, overlay.Left, overlay.Bottom, 0, -1)
	}

	if m.ThemeMode {
		// Ensure there's space for overlay positioning
		contentLines := strings.Count(mainContent, "\n")
		minLines := 10 // Minimum lines to ensure overlay positioning works well
		if contentLines < minLines {
			for i := contentLines; i < minLines; i++ {
				background += "\n"
			}
		}

		overlayContent := m.renderThemeOverlayCompact()
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
	styles := m.Styles()
	config := m.Config()

	if len(m.FileModel.Todos) == 0 && !m.InputMode {
		b.WriteString(styles.Dim("No todos. Press 'n' to create one."))
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

			// Apply priority filtering if active
			if len(m.FilteredPriorities) > 0 && !todo.HasAnyPriority(m.FilteredPriorities) {
				continue
			}

			// Apply due date filtering if active
			if m.FilteredDueDate != "" && !todo.HasDueDateFilter(m.FilteredDueDate) {
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
	configMaxVisible := config.Display.MaxVisible
	if m.MaxVisibleOverride >= 0 {
		configMaxVisible = m.MaxVisibleOverride
	}

	// If max_visible is 0 (unlimited) but we have terminal height info,
	// auto-calculate a reasonable limit based on terminal size
	// Reserve lines for: status bar (1), empty line (1), potential headings, scroll indicators
	autoMaxVisible := 0
	if configMaxVisible == 0 && m.TermHeight > 0 {
		// Reserve ~4 lines for UI chrome (status bar, spacing, etc.)
		autoMaxVisible = m.TermHeight - 4
		if autoMaxVisible < 5 {
			autoMaxVisible = 5 // Minimum reasonable visible items
		}
	}

	effectiveMaxVisible := configMaxVisible
	if effectiveMaxVisible == 0 && autoMaxVisible > 0 {
		effectiveMaxVisible = autoMaxVisible
	}
	if m.InputMode && effectiveMaxVisible > 0 {
		effectiveMaxVisible = effectiveMaxVisible - 1
	}

	if effectiveMaxVisible > 0 && len(todosToShow) > effectiveMaxVisible {
		// Calculate visible window centered on selection
		var currentPos int
		if m.SearchMode {
			currentPos = m.SearchCursor
		} else if m.InputMode && !m.InsertAfterCursor {
			// When appending new task at end, scroll to show last items before the input
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

	// Show indicator for items above (when scrolling is active)
	if effectiveMaxVisible > 0 && totalCount > effectiveMaxVisible {
		if hasMoreAbove {
			b.WriteString(fmt.Sprintf("      %s\n", styles.Dim(fmt.Sprintf("▲ %d more", startIdx))))
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

	// Get all headings if enabled (uses cached headings for performance)
	var allHeadings []markdown.Heading
	if m.ShowHeadings {
		allHeadings = m.GetHeadings()
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
					b.WriteString(fmt.Sprintf("      %s\n", styles.Cyan(headingText)))
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
		} else if m.InputMode && !m.InsertAfterCursor {
			// When appending at end, all existing items are above (negative indices)
			isSelected = false
			relIndex = (startIdx + displayIdx) - totalCount
		} else if m.InputMode && m.InsertAfterCursor {
			// When inserting after cursor, show selection arrow on cursor item
			isSelected = todoIdx == m.SelectedIndex
			relIndex = (startIdx + displayIdx) - selectedVisiblePos
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

		// Arrow - don't show on existing items when in input mode (arrow goes on input line)
		arrow := "   "
		if isSelected && !m.InputMode {
			arrow = styles.Cyan(" " + config.Display.SelectMarker + " ")
		}
		// In insert-after-cursor mode, don't show arrow on the item we're inserting after
		if m.InputMode && m.InsertAfterCursor && isSelected {
			arrow = "   "
		}

		// Checkbox
		var checkbox string
		if todo.Checked {
			checkbox = styles.Magenta("[" + config.Display.CheckSymbol + "]")
		} else {
			checkbox = styles.Dim("[ ]")
		}

		// Move indicator (check before building prefix)
		if m.MoveMode && isSelected {
			arrow = styles.Yellow(" ≡ ")
		}

		// Build the line prefix (needed early for edit mode wrapping)
		// Add indentation based on nesting depth (2 spaces per level)
		indent := strings.Repeat("  ", todo.Depth)
		prefix := fmt.Sprintf("%s%s%s%s ", indent, styles.Dim(indexStr), arrow, checkbox)
		prefixWidth := (todo.Depth * 2) + 3 + 3 + 3 + 1 // indent + index(3) + arrow(3) + checkbox(3) + space(1)

		// Text with inline code rendering and tag colorization
		var text string
		plainText := todo.Text

		if m.SearchMode && m.InputBuffer != "" {
			// Highlight matches during search
			text = HighlightMatches(todo.Text, m.InputBuffer, styles.Green)
		} else {
			text = RenderInlineCode(todo.Text, todo.Checked, styles.Magenta, styles.Cyan, styles.Code)
			// Colorize tags, priorities, and due dates
			text = ColorizeTags(text, styles.Tag)
			text = ColorizePriorities(text, styles.PriorityHigh, styles.PriorityMedium, styles.PriorityLow)
			text = ColorizeDueDates(text, styles.DueUrgent, styles.DueSoon, styles.DueFuture)
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
			styles.Magenta, styles.Cyan, styles.Code, styles.Dim,
		))

		// If in input mode with insert-after-cursor, show input line after selected item
		if m.InputMode && m.InsertAfterCursor && isSelected {
			b.WriteString(m.renderInputLine(styles, config))
		}
	}

	// Input mode at end - show new task at end when not inserting after cursor
	// Also handles the case when inserting after cursor but there are no todos
	if m.InputMode && (!m.InsertAfterCursor || len(m.FileModel.Todos) == 0) {
		b.WriteString(m.renderInputLine(styles, config))
	}

	// Show indicator for items below (when scrolling is active)
	if effectiveMaxVisible > 0 && totalCount > effectiveMaxVisible {
		if hasMoreBelow && !m.InputMode {
			b.WriteString(fmt.Sprintf("      %s\n", styles.Dim(fmt.Sprintf("▼ %d more", totalCount-startIdx-len(todosToShow)))))
		} else {
			b.WriteString("\n")
		}
	}

	// Show message when search has no results
	if m.SearchMode && len(m.SearchResults) == 0 && m.InputBuffer != "" {
		b.WriteString(styles.Dim("  No matches found"))
		b.WriteString("\n")
	}

	// Show message when filters result in no visible todos
	if !m.SearchMode && !m.InputMode && len(m.FileModel.Todos) > 0 && len(todosToShow) == 0 {
		b.WriteString(styles.Dim("  No todos match current filters."))
		b.WriteString("\n")
		// Build hint about which filters are active
		var activeFilters []string
		if m.FilterDone {
			activeFilters = append(activeFilters, "completed hidden")
		}
		if len(m.FilteredTags) > 0 {
			activeFilters = append(activeFilters, fmt.Sprintf("tags: #%s", strings.Join(m.FilteredTags, " #")))
		}
		if len(m.FilteredPriorities) > 0 {
			var pStrs []string
			for _, p := range m.FilteredPriorities {
				pStrs = append(pStrs, fmt.Sprintf("p%d", p))
			}
			activeFilters = append(activeFilters, fmt.Sprintf("priorities: %s", strings.Join(pStrs, " ")))
		}
		if m.FilteredDueDate != "" {
			activeFilters = append(activeFilters, fmt.Sprintf("due: %s", m.FilteredDueDate))
		}
		if len(activeFilters) > 0 {
			b.WriteString(styles.Dim(fmt.Sprintf("  Active: %s", strings.Join(activeFilters, ", "))))
			b.WriteString("\n")
		}
		b.WriteString(styles.Dim("  Press 't' (tags), 'p' (priority), or ':filter-done' to adjust filters."))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	return b.String()
}

// renderInputLine renders the new task input line with word wrap support
func (m Model) renderInputLine(styles *StyleFuncsType, config *ConfigType) string {
	var b strings.Builder

	arrow := styles.Cyan(" " + config.Display.SelectMarker + " ")
	checkbox := styles.Dim("[ ]")
	indexStr := styles.Dim("  0")

	// Build prefix
	prefix := fmt.Sprintf("%s%s%s ", indexStr, arrow, checkbox)
	prefixWidth := 3 + 3 + 3 + 1 // index(3) + arrow(3) + checkbox(3) + space(1)

	before := m.InputBuffer[:m.CursorPos]
	after := m.InputBuffer[m.CursorPos:]
	cursor := lipgloss.NewStyle().Reverse(true).Render(" ")

	// Apply word wrap if enabled
	if m.WordWrap && m.TermWidth > 0 {
		textWithCursor := before + cursor + after
		availWidth := m.TermWidth - prefixWidth
		if availWidth > 10 {
			indent := strings.Repeat(" ", prefixWidth)
			wrappedLines := util.WrapText(textWithCursor, availWidth, indent)

			for i, line := range wrappedLines {
				if i == 0 {
					b.WriteString(prefix + line + "\n")
				} else {
					b.WriteString(line + "\n")
				}
			}
			return b.String()
		}
	}

	// No wrapping - simple output
	b.WriteString(fmt.Sprintf("%s%s%s%s\n", prefix, before, cursor, after))
	return b.String()
}

// renderStatusBar renders the status bar at the bottom
func (m Model) renderStatusBar() string {
	var b strings.Builder
	styles := m.Styles()

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
			b.WriteString(styles.Cyan(":reload") + styles.Dim(" or ") + styles.Cyan(":force-save"))
		} else {
			b.WriteString(styles.Dim("any key to dismiss"))
		}
		b.WriteString("\n")
	}

	// Status bar
	if m.CommandMode {
		b.WriteString(ModeIndicator("⌘", "COMMAND"))
		b.WriteString("  ")
		b.WriteString(styles.Dim("Type to filter commands"))
	} else if m.ThemeMode {
		b.WriteString(ModeIndicator("🎨", "THEME"))
		b.WriteString("  ")
		b.WriteString(styles.Dim("↑/↓ preview  enter apply  esc cancel"))
	} else if m.FilterMode {
		b.WriteString(ModeIndicator("🏷", "TAGS"))
		b.WriteString("  ")
		b.WriteString(styles.Dim("Select tags to filter todos"))
	} else if m.PriorityFilterMode {
		b.WriteString(ModeIndicator("⚡", "PRIORITY"))
		b.WriteString("  ")
		b.WriteString(styles.Dim("Select priorities to filter todos"))
	} else if m.DueFilterMode {
		b.WriteString(ModeIndicator("📅", "DUE DATE"))
		b.WriteString("  ")
		b.WriteString(styles.Dim("Select due date filter"))
	} else if m.SearchMode {
		b.WriteString(ModeIndicator("🔍", "SEARCH"))
		b.WriteString("  ")
		before := m.InputBuffer[:m.CursorPos]
		after := m.InputBuffer[m.CursorPos:]
		cursor := lipgloss.NewStyle().Reverse(true).Render(" ")
		b.WriteString(before + cursor + after)
		b.WriteString(styles.Dim("  ↑/↓ navigate  enter select  esc cancel"))
	} else if m.MaxVisibleInputMode {
		b.WriteString(ModeIndicator("⊙", "SET MAX"))
		b.WriteString("  ")
		before := m.InputBuffer[:m.CursorPos]
		after := m.InputBuffer[m.CursorPos:]
		cursor := lipgloss.NewStyle().Reverse(true).Render(" ")
		b.WriteString(before + cursor + after)
		b.WriteString(styles.Dim("  enter confirm  esc cancel"))
	} else if m.InputMode {
		b.WriteString(ModeIndicator("✎", "NEW"))
		b.WriteString("  ")
		b.WriteString(styles.Dim("enter confirm  esc cancel"))
	} else if m.EditMode {
		b.WriteString(ModeIndicator("✎", "EDIT"))
		b.WriteString("  ")
		b.WriteString(styles.Dim("enter confirm  esc cancel"))
	} else if m.MoveMode {
		b.WriteString(ModeIndicator("≡", "MOVE"))
		b.WriteString("  ")
		b.WriteString(styles.Dim("j/k move  enter confirm  esc cancel"))
	} else if m.CopyFeedback {
		b.WriteString(styles.Green("✓ Copied to clipboard!"))
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
		if len(m.FilteredPriorities) > 0 {
			var priorityStrs []string
			for _, p := range m.FilteredPriorities {
				priorityStrs = append(priorityStrs, fmt.Sprintf("p%d", p))
			}
			indicators = append(indicators, fmt.Sprintf("⚡ %s", strings.Join(priorityStrs, " ")))
		}
		if m.FilteredDueDate != "" {
			indicators = append(indicators, fmt.Sprintf("📅 %s", m.FilteredDueDate))
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
		helpParts = append(helpParts, styles.Cyan("?")+styles.Dim(" help"))
		helpParts = append(helpParts, styles.Cyan(":")+styles.Dim(" cmd"))
		helpParts = append(helpParts, styles.Cyan("n")+styles.Dim(" new"))
		helpParts = append(helpParts, styles.Cyan("␣")+styles.Dim(" toggle"))
		helpParts = append(helpParts, styles.Cyan("esc")+styles.Dim(" quit"))
		b.WriteString(strings.Join(helpParts, "  "))
	}

	return b.String()
}

// renderCommandOverlayCompact renders a compact modal command palette
func (m Model) renderCommandOverlayCompact() string {
	var b strings.Builder
	styles := m.Styles()

	// Command input with cursor
	before := m.InputBuffer[:m.CursorPos]
	after := m.InputBuffer[m.CursorPos:]
	cursor := lipgloss.NewStyle().Reverse(true).Render(" ")
	b.WriteString(styles.Cyan(":") + before + cursor + after)
	b.WriteString("\n")

	// Show top 5 matching commands with scrolling
	maxCmds := 5
	totalCmds := len(m.FilteredCmds)

	if totalCmds == 0 {
		b.WriteString(styles.Dim("  No matching commands"))
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
				marker = styles.Cyan("→ ")
			} else {
				marker = "  "
			}

			cmdName := cmd.Name
			if isSelected {
				cmdName = styles.Cyan(cmdName)
			}

			cmdDesc := styles.Dim(" - " + cmd.Description)
			b.WriteString(marker + cmdName + cmdDesc)
			b.WriteString("\n")
		}

		// Show indicator for more commands
		if totalCmds > maxCmds {
			remaining := totalCmds - maxCmds
			b.WriteString(styles.Dim(fmt.Sprintf("  ... %d more (type to filter)", remaining)))
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

// renderDueFilterOverlayCompact renders a compact modal due date filter selector
func (m Model) renderDueFilterOverlayCompact() string {
	var b strings.Builder
	styles := m.Styles()

	// Due date filter options with descriptions
	options := []struct {
		value string
		label string
		desc  string
	}{
		{"overdue", "Overdue", "Past due date"},
		{"today", "Today", "Due today"},
		{"week", "This Week", "Due within 7 days"},
		{"all", "Has Due Date", "Any due date set"},
	}

	for i, opt := range options {
		isSelected := i == m.DueFilterCursor
		isActive := m.FilteredDueDate == opt.value

		var marker string
		if isSelected {
			marker = styles.Cyan("→ ")
		} else {
			marker = "  "
		}

		var checkbox string
		if isActive {
			checkbox = styles.Green("[●] ")
		} else {
			checkbox = styles.Dim("[ ] ")
		}

		label := opt.label
		if isSelected {
			label = styles.Cyan(label)
		}

		b.WriteString(marker + checkbox + label + " " + styles.Dim(opt.desc))
		b.WriteString("\n")
	}

	// Add help text
	b.WriteString("\n")
	b.WriteString(styles.Dim("space select  c clear  esc done"))

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

// renderThemeOverlayCompact renders a compact modal theme picker
func (m Model) renderThemeOverlayCompact() string {
	var b strings.Builder
	styles := m.Styles()

	// Title
	b.WriteString(styles.Cyan("Select Theme"))
	b.WriteString("\n")

	// Check if there are any themes to display
	if len(m.AvailableThemes) == 0 {
		b.WriteString(styles.Dim("No themes available."))
		b.WriteString("\n\n")
		b.WriteString(styles.Dim("esc close"))
	} else {
		// Display themes with scrolling
		maxThemes := 10
		totalThemes := len(m.AvailableThemes)

		// Calculate scroll window to keep cursor visible
		startIdx := 0
		if m.ThemeCursor >= maxThemes {
			startIdx = m.ThemeCursor - maxThemes + 1
		}

		endIdx := startIdx + maxThemes
		if endIdx > totalThemes {
			endIdx = totalThemes
			startIdx = endIdx - maxThemes
			if startIdx < 0 {
				startIdx = 0
			}
		}

		// Show scroll indicator if there are items above
		if startIdx > 0 {
			b.WriteString(styles.Dim(fmt.Sprintf("  ▲ %d more", startIdx)))
			b.WriteString("\n")
		}

		for i := startIdx; i < endIdx; i++ {
			themeName := m.AvailableThemes[i]
			isSelected := i == m.ThemeCursor
			isCurrent := themeName == m.CurrentThemeName

			var marker string
			if isSelected {
				marker = styles.Cyan("→ ")
			} else {
				marker = "  "
			}

			// Show checkbox for current theme
			var checkbox string
			if isCurrent {
				checkbox = styles.Green("[●] ")
			} else {
				checkbox = styles.Dim("[ ] ")
			}

			// Theme name
			displayName := themeName
			if isSelected {
				displayName = styles.Cyan(themeName)
			}

			b.WriteString(marker + checkbox + displayName)
			b.WriteString("\n")
		}

		// Show scroll indicator if there are items below
		if endIdx < totalThemes {
			b.WriteString(styles.Dim(fmt.Sprintf("  ▼ %d more", totalThemes-endIdx)))
			b.WriteString("\n")
		}

		// Add help text
		b.WriteString("\n")
		b.WriteString(styles.Dim("↑/↓ preview  enter apply  esc cancel"))
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

// renderRecentFilesOverlay renders a compact modal for recent files
func (m Model) renderRecentFilesOverlay() string {
	var b strings.Builder
	styles := m.Styles()

	// Filter files based on search
	filteredFiles := []config.RecentFile{}
	for _, file := range m.RecentFiles {
		if m.RecentFilesSearch == "" || strings.Contains(strings.ToLower(file.Path), strings.ToLower(m.RecentFilesSearch)) {
			filteredFiles = append(filteredFiles, file)
		}
	}

	// Title with search input
	if m.RecentFilesSearch != "" {
		b.WriteString(styles.Cyan("Recent: ") + m.RecentFilesSearch)
	} else {
		b.WriteString(styles.Cyan("Recent Files"))
	}
	b.WriteString("\n")

	// Show top 8 files with scrolling
	maxFiles := 8
	totalFiles := len(filteredFiles)

	if totalFiles == 0 {
		if m.RecentFilesSearch != "" {
			b.WriteString(styles.Dim("  No matching files"))
		} else {
			b.WriteString(styles.Dim("  No recent files"))
		}
		b.WriteString("\n")
	} else {
		// Calculate scroll window to keep cursor visible
		startIdx := 0
		if m.RecentFilesCursor >= maxFiles {
			startIdx = m.RecentFilesCursor - maxFiles + 1
		}

		endIdx := startIdx + maxFiles
		if endIdx > totalFiles {
			endIdx = totalFiles
			startIdx = endIdx - maxFiles
			if startIdx < 0 {
				startIdx = 0
			}
		}

		// Render visible files
		for i := startIdx; i < endIdx; i++ {
			file := filteredFiles[i]
			isSelected := (i == m.RecentFilesCursor)

			// Get display path (show ~ for home directory)
			displayPath := file.Path
			if home, err := os.UserHomeDir(); err == nil {
				if rel, err := filepath.Rel(home, file.Path); err == nil && !strings.HasPrefix(rel, "..") {
					displayPath = "~/" + rel
				}
			}

			// Truncate long paths
			if len(displayPath) > 60 {
				displayPath = "..." + displayPath[len(displayPath)-57:]
			}

			// Format info
			info := fmt.Sprintf("×%d", file.AccessCount)

			var marker string
			if isSelected {
				marker = styles.Cyan("▸ ")
				displayPath = styles.Cyan(displayPath)
			} else {
				marker = "  "
			}

			b.WriteString(marker + displayPath + " " + styles.Dim(info))
			b.WriteString("\n")
		}

		// Show indicator for more files
		if totalFiles > maxFiles {
			remaining := totalFiles - endIdx
			if remaining > 0 {
				b.WriteString(styles.Dim(fmt.Sprintf("  ... %d more", remaining)))
				b.WriteString("\n")
			}
		}
	}

	// Help text
	b.WriteString(styles.Dim("Type to filter • ↑/↓ navigate • ↵ open • esc close"))

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
		Padding(0, 1).
		Width(70)

	return overlayStyle.Render(content)
}

// renderFilterOverlayCompact renders a compact modal filter selector
func (m Model) renderFilterOverlayCompact() string {
	var b strings.Builder
	styles := m.Styles()

	// Check if there are any tags to display
	if len(m.AvailableTags) == 0 {
		b.WriteString(styles.Dim("No tags available."))
		b.WriteString("\n\n")
		if m.FilterDone {
			b.WriteString(styles.Dim("Try disabling 'filter-done' to see"))
			b.WriteString("\n")
			b.WriteString(styles.Dim("tags from completed todos."))
			b.WriteString("\n\n")
		}
		b.WriteString(styles.Dim("esc close"))
	} else {
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
				marker = styles.Cyan("→ ")
			} else {
				marker = "  "
			}

			var checkbox string
			if isActive {
				checkbox = styles.Green("[✓] ")
			} else {
				checkbox = styles.Dim("[ ] ")
			}

			tagText := styles.Cyan("#" + tag)
			b.WriteString(marker + checkbox + tagText)
			b.WriteString("\n")
		}

		// Add help text
		b.WriteString("\n")
		b.WriteString(styles.Dim("space toggle  c clear  esc done"))
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

// renderPriorityFilterOverlayCompact renders a compact modal priority filter selector
func (m Model) renderPriorityFilterOverlayCompact() string {
	var b strings.Builder
	styles := m.Styles()

	// Check if there are any priorities to display
	if len(m.AvailablePriorities) == 0 {
		b.WriteString(styles.Dim("No priorities available."))
		b.WriteString("\n\n")
		if m.FilterDone {
			b.WriteString(styles.Dim("Try disabling 'filter-done' to see"))
			b.WriteString("\n")
			b.WriteString(styles.Dim("priorities from completed todos."))
			b.WriteString("\n\n")
		}
		b.WriteString(styles.Dim("esc close"))
	} else {
		// Display priorities vertically in compact modal
		maxPriorities := 8
		displayCount := len(m.AvailablePriorities)
		if displayCount > maxPriorities {
			displayCount = maxPriorities
		}

		for i := 0; i < displayCount; i++ {
			priority := m.AvailablePriorities[i]
			isSelected := i == m.PriorityFilterCursor
			isActive := false
			for _, activePriority := range m.FilteredPriorities {
				if activePriority == priority {
					isActive = true
					break
				}
			}

			var marker string
			if isSelected {
				marker = styles.Cyan("→ ")
			} else {
				marker = "  "
			}

			var checkbox string
			if isActive {
				checkbox = styles.Green("[✓] ")
			} else {
				checkbox = styles.Dim("[ ] ")
			}

			// Color the priority marker based on level
			priorityText := fmt.Sprintf("!p%d", priority)
			priorityText = ColorizePriorities(priorityText, styles.PriorityHigh, styles.PriorityMedium, styles.PriorityLow)
			b.WriteString(marker + checkbox + priorityText)
			b.WriteString("\n")
		}

		// Add help text
		b.WriteString("\n")
		b.WriteString(styles.Dim("space toggle  c clear  esc done"))
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
