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
	effectiveMaxVisible := configMaxVisible
	if m.InputMode && configMaxVisible > 0 {
		effectiveMaxVisible = configMaxVisible - 1
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

	// Show indicator for items above (always reserve space when max_visible is set)
	if configMaxVisible > 0 && totalCount > configMaxVisible {
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
		prefix := fmt.Sprintf("%s%s%s ", styles.Dim(indexStr), arrow, checkbox)
		prefixWidth := 3 + 3 + 3 + 1 // index(3) + arrow(3) + checkbox(3) + space(1)

		// Text with inline code rendering and tag colorization
		var text string
		var plainText string = todo.Text

		if m.SearchMode && m.InputBuffer != "" {
			// Highlight matches during search
			text = HighlightMatches(todo.Text, m.InputBuffer, styles.Green)
		} else {
			text = RenderInlineCode(todo.Text, todo.Checked, styles.Magenta, styles.Cyan, styles.Code)
			// Colorize tags
			text = ColorizeTags(text, styles.Yellow)
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
	if m.InputMode && !m.InsertAfterCursor {
		b.WriteString(m.renderInputLine(styles, config))
	}

	// Show indicator for items below (always reserve space when max_visible is set)
	if configMaxVisible > 0 && totalCount > configMaxVisible {
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
	} else if m.FilterMode {
		b.WriteString(ModeIndicator("🏷", "FILTER"))
		b.WriteString("  ")
		b.WriteString(styles.Dim("Select tags to filter todos"))
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

// renderFilterOverlay renders the tag filter overlay as a centered modal
func (m Model) renderFilterOverlay() string {
	var b strings.Builder
	styles := m.Styles()

	// Header
	b.WriteString(ModeIndicator("🏷", "FILTER"))
	b.WriteString("\n\n")
	b.WriteString(styles.Dim("Select tags to filter:"))
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

		b.WriteString(marker + checkbox + styles.Cyan("#"+tag))
		b.WriteString("\n")
	}

	// Footer with help
	b.WriteString("\n")
	b.WriteString(styles.Dim("  ↑/↓ navigate  space toggle  c clear all  esc done"))

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
	styles := m.Styles()

	// Header
	b.WriteString(ModeIndicator("⌘", "COMMAND"))
	b.WriteString("\n\n")

	// Command input with cursor
	before := m.InputBuffer[:m.CursorPos]
	after := m.InputBuffer[m.CursorPos:]
	cursor := lipgloss.NewStyle().Reverse(true).Render(" ")
	b.WriteString(styles.Cyan(":") + before + cursor + after)
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
			marker = styles.Cyan("→ ")
		} else {
			marker = "  "
		}

		// Command name and description
		cmdName := styles.Cyan(cmd.Name)
		cmdDesc := styles.Dim(" - " + cmd.Description)
		b.WriteString(marker + cmdName + cmdDesc)
		b.WriteString("\n")
	}

	// Footer with help
	b.WriteString("\n")
	b.WriteString(styles.Dim("  ↑/↓ navigate  tab complete  enter execute  esc cancel"))

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
