package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/niklas-heer/tdx/internal/cmd"
	"github.com/niklas-heer/tdx/internal/config"
	"github.com/niklas-heer/tdx/internal/tui"
)

const defaultFile = "todo.md"

func main() {
	// Load user config
	appConfig := LoadConfig()
	styles := NewStyles(appConfig)

	// Inject config and styles into packages
	cmd.GreenStyle = func(s string) string { return styles.Success.Render(s) }
	cmd.DimStyle = func(s string) string { return styles.Dim.Render(s) }
	cmd.CheckSymbol = appConfig.Display.CheckSymbol

	// Setup TUI package globals
	tui.Config = &tui.ConfigType{}
	tui.Config.Display.CheckSymbol = appConfig.Display.CheckSymbol
	tui.Config.Display.SelectMarker = appConfig.Display.SelectMarker
	tui.Config.Display.MaxVisible = appConfig.Display.MaxVisible

	tui.StyleFuncs = &tui.StyleFuncsType{
		Magenta:        func(s string) string { return styles.Important.Render(s) },
		Cyan:           func(s string) string { return styles.Accent.Render(s) },
		Dim:            func(s string) string { return styles.Dim.Render(s) },
		Green:          func(s string) string { return styles.Success.Render(s) },
		Yellow:         func(s string) string { return styles.Warning.Render(s) },
		Code:           func(s string) string { return styles.Code.Render(s) },
		Tag:            func(s string) string { return styles.Tag.Render(s) },
		PriorityHigh:   func(s string) string { return styles.PriorityHigh.Render(s) },
		PriorityMedium: func(s string) string { return styles.PriorityMedium.Render(s) },
		PriorityLow:    func(s string) string { return styles.PriorityLow.Render(s) },
		DueUrgent:      func(s string) string { return styles.DueUrgent.Render(s) },
		DueSoon:        func(s string) string { return styles.DueSoon.Render(s) },
		DueFuture:      func(s string) string { return styles.DueFuture.Render(s) },
	}
	tui.Version = Version

	// Setup theme picker support
	tui.AvailableThemes = GetBuiltinThemeNames()
	tui.CurrentThemeName = appConfig.Theme.Name
	tui.ThemeApplyFunc = func(themeName string) *tui.StyleFuncsType {
		colors, ok := GetBuiltinTheme(themeName)
		if !ok {
			return nil
		}
		// Create a temporary config with the new theme colors
		tempConfig := &UserConfig{Colors: colors}
		newStyles := NewStyles(tempConfig)
		return &tui.StyleFuncsType{
			Magenta:        func(s string) string { return newStyles.Important.Render(s) },
			Cyan:           func(s string) string { return newStyles.Accent.Render(s) },
			Dim:            func(s string) string { return newStyles.Dim.Render(s) },
			Green:          func(s string) string { return newStyles.Success.Render(s) },
			Yellow:         func(s string) string { return newStyles.Warning.Render(s) },
			Code:           func(s string) string { return newStyles.Code.Render(s) },
			Tag:            func(s string) string { return newStyles.Tag.Render(s) },
			PriorityHigh:   func(s string) string { return newStyles.PriorityHigh.Render(s) },
			PriorityMedium: func(s string) string { return newStyles.PriorityMedium.Render(s) },
			PriorityLow:    func(s string) string { return newStyles.PriorityLow.Render(s) },
			DueUrgent:      func(s string) string { return newStyles.DueUrgent.Render(s) },
			DueSoon:        func(s string) string { return newStyles.DueSoon.Render(s) },
			DueFuture:      func(s string) string { return newStyles.DueFuture.Render(s) },
		}
	}
	tui.ThemeSaveFunc = SaveTheme

	args := os.Args[1:]

	// Determine file path, flags, and command
	filePath := defaultFile
	var command string
	var cmdArgs []string
	readOnly := false
	showHeadings := false
	maxVisible := -1 // -1 means use config default

	// Process arguments
	var remainingArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--read-only", "-r":
			readOnly = true
		case "--show-headings":
			showHeadings = true
		case "--max-visible", "-m":
			// Get the next argument as the number
			if i+1 < len(args) {
				i++
				if num, err := strconv.Atoi(args[i]); err == nil && num >= 0 {
					maxVisible = num
				} else {
					fmt.Printf("Error: --max-visible requires a non-negative integer\n")
					os.Exit(1)
				}
			} else {
				fmt.Printf("Error: --max-visible requires a number argument\n")
				os.Exit(1)
			}
		default:
			remainingArgs = append(remainingArgs, arg)
		}
	}
	args = remainingArgs

	if len(args) > 0 {
		// Check if first arg is a .md file
		if strings.HasSuffix(args[0], ".md") {
			filePath = args[0]
			args = args[1:]
		}

		if len(args) > 0 {
			command = args[0]
			cmdArgs = args[1:]
		}
	}

	// Resolve to absolute path
	if !filepath.IsAbs(filePath) {
		cwd, _ := os.Getwd()
		filePath = filepath.Join(cwd, filePath)
	}

	// Handle commands
	switch command {
	case "help", "--help", "-h":
		printHelp()
	case "--version", "-v":
		fmt.Printf("tdx v%s\n", Version)
	case "--debug-config":
		fmt.Printf("Theme: %s\n", appConfig.Theme.Name)
		fmt.Printf("Accent: %s\n", appConfig.Colors.Accent)
		fmt.Printf("Success: %s\n", appConfig.Colors.Success)
		fmt.Printf("MaxVisible: %d\n", appConfig.Display.MaxVisible)
	case "list", "add", "toggle", "edit", "delete":
		cmd.HandleCommand(command, cmdArgs, filePath)
	case "recent":
		handleRecentCommand(cmdArgs, readOnly, showHeadings, maxVisible)
	case "":
		// Launch TUI
		tui.Run(filePath, readOnly, showHeadings, maxVisible)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	help := fmt.Sprintf(`tdx - %s

Usage:
  tdx [file.md] [command] [args]

Options:
  -r, --read-only         Don't save changes to disk (read-only mode)
      --show-headings     Display markdown headings between tasks
  -m, --max-visible <N>   Set max visible items (0 = unlimited)

Commands:
  (none)              Launch interactive TUI
  list                List all todos
  add "text"          Add a new todo
  toggle <index>      Toggle todo completion
  edit <index> "text" Edit todo text
  delete <index>      Delete a todo
  recent              List recently opened files
  recent <number>     Open a recent file by number
  recent clear        Clear recent files history
  help                Show this help

TUI Controls:
  j/k, ↑/↓            Navigate up/down
  Space, Enter        Toggle completion
  n                   New todo
  e                   Edit todo
  d                   Delete todo
  c                   Copy to clipboard
  m                   Move todo
  u                   Undo
  :                   Command palette
  ?                   Toggle help
  Esc                 Quit`, Description)
	fmt.Println(help)
}

func handleRecentCommand(args []string, readOnly bool, showHeadings bool, maxVisible int) {
	// Handle "clear" subcommand
	if len(args) > 0 && args[0] == "clear" {
		if err := config.ClearRecentFiles(); err != nil {
			fmt.Printf("Error clearing recent files: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Recent files cleared")
		return
	}

	// Load recent files
	recentFiles, err := config.LoadRecentFiles()
	if err != nil {
		fmt.Printf("Error loading recent files: %v\n", err)
		os.Exit(1)
	}

	if len(recentFiles.Files) == 0 {
		fmt.Println("No recent files")
		return
	}

	// Sort by score (recency * frequency)
	recentFiles.SortByScore()

	// If numeric argument, open that file
	if len(args) > 0 {
		index, err := strconv.Atoi(args[0])
		if err != nil || index < 1 || index > len(recentFiles.Files) {
			fmt.Printf("Error: invalid file number. Use 1-%d\n", len(recentFiles.Files))
			os.Exit(1)
		}

		// Open the selected file (1-indexed)
		filePath := recentFiles.Files[index-1].Path
		tui.Run(filePath, readOnly, showHeadings, maxVisible)
		return
	}

	// No args - list all recent files
	fmt.Println("Recent files:")
	for i, file := range recentFiles.Files {
		// Show relative path if in home directory
		displayPath := file.Path
		if home, err := os.UserHomeDir(); err == nil {
			if rel, err := filepath.Rel(home, file.Path); err == nil && !strings.HasPrefix(rel, "..") {
				displayPath = "~/" + rel
			}
		}

		fmt.Printf("  %d. %s (accessed %d times, last: %s)\n",
			i+1,
			displayPath,
			file.AccessCount,
			file.LastAccessed.Format("2006-01-02 15:04"))
	}
	fmt.Println("\nUse 'tdx recent <number>' to open a file")
}
