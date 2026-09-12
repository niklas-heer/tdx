package main

import (
	"charm.land/lipgloss/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/niklas-heer/tdx/internal/cmd"
	"github.com/niklas-heer/tdx/internal/config" // Still needed for recent files
	"github.com/niklas-heer/tdx/internal/markdown"
	"github.com/niklas-heer/tdx/internal/tui"
	"github.com/niklas-heer/tdx/internal/versioning"
)

// historyStore adapts a version store to the Markdown persistence boundary.
func historyStore(versions *versioning.Store) markdown.Store {
	capture := func(filePath, content string) error {
		return errors.Join(versions.SaveVersion(filePath, content), versions.Prune(filePath, versions.MaxVersions))
	}
	return markdown.Store{OnRead: capture, OnWrite: capture}
}

func wireVersioningTUI(cfg *tui.ConfigType, versions *versioning.Store) {
	cfg.ListVersionsFunc = func(filePath string) ([]tui.VersionInfo, error) {
		path, err := canonicalHistoryPath(filePath)
		if err != nil {
			return nil, err
		}
		list, err := versions.ListVersions(path)
		if err != nil {
			return nil, err
		}
		out := make([]tui.VersionInfo, len(list))
		for i, v := range list {
			out[i] = tui.VersionInfo{ID: v.ID, CreatedAt: v.CreatedAt}
		}
		return out, nil
	}
	cfg.ReadVersionFunc = func(filePath string, id int64) (string, error) {
		path, err := canonicalHistoryPath(filePath)
		if err != nil {
			return "", err
		}
		return versions.ReadVersion(path, id)
	}
}

// History lookup must use the same canonical identity as safe-save callbacks.
func canonicalHistoryPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err == nil {
		return resolved, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(abs))
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, filepath.Base(abs)), nil
}

func commandUsesVersioning(command string, args []string) bool {
	switch command {
	case "", "add", "toggle", "edit", "delete", "done", "undone", "last":
		return true
	case "recent":
		return len(args) > 0 && args[0] != "clear"
	default:
		return false
	}
}

func main() {
	os.Exit(run())
}

func run() (exitCode int) {
	// Load user config
	appConfig := LoadConfig()

	styles := NewStyles(appConfig)

	cli := cmd.Service{Out: os.Stdout, GreenStyle: func(s string) string { return styles.Success.Render(s) }, CheckSymbol: appConfig.Display.CheckSymbol}

	// Assemble this TUI runtime
	tuiCfg := &tui.ConfigType{}
	recentDir, _ := config.GetConfigDir()
	tuiCfg.Recent = config.RecentStore{Dir: recentDir, Limit: appConfig.Recent.MaxFiles}
	tuiCfg.Views = config.NewFileViewStore("")
	tuiCfg.ViewsRestore = appConfig.Views.Restore
	tuiCfg.Display.CheckSymbol = appConfig.Display.CheckSymbol
	tuiCfg.Display.SelectMarker = appConfig.Display.SelectMarker
	tuiCfg.Display.MaxVisible = appConfig.Defaults.MaxVisible
	tuiCfg.Defaults.WordWrap = appConfig.Defaults.WordWrap
	tuiCfg.Defaults.FilterDone = appConfig.Defaults.FilterDone
	tuiCfg.Defaults.ShowHeadings = appConfig.Defaults.ShowHeadings
	tuiCfg.Defaults.ReadOnly = appConfig.Defaults.ReadOnly

	tuiStyles := &tui.StyleFuncsType{
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

	// Setup theme picker support
	tuiCfg.AvailableThemes = GetBuiltinThemeNames()
	tuiCfg.CurrentThemeName = appConfig.Theme.Name
	tuiCfg.ThemeApplyFunc = func(themeName string) *tui.StyleFuncsType {
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
	tuiCfg.ThemeSaveFunc = SaveTheme

	runtime := tui.Runtime{Config: tuiCfg, Styles: tuiStyles, Version: Version}
	opts, err := parseArgs(os.Args[1:], appConfig.Defaults)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tdx: %v\n", err)
		return 1
	}
	filePath, command, cmdArgs := opts.File, opts.Command, opts.Args
	readOnly, showHeadings, maxVisible := opts.ReadOnly, opts.ShowHeadings, opts.MaxVisible
	switch command {
	case "list", "add", "toggle", "edit", "delete", "done", "undone":
		if err := cmd.ValidateCommand(command, cmdArgs); err != nil {
			fmt.Fprintf(os.Stderr, "tdx: %v\n", err)
			return 1
		}
		if readOnly && command != "list" {
			fmt.Fprintln(os.Stderr, "tdx: read-only mode: task editing is disabled")
			return 1
		}
	}

	if command == "revision" && len(cmdArgs) != 0 {
		fmt.Fprintln(os.Stderr, "tdx: revision takes no arguments")
		return 1
	}
	cli.ExpectedRevision = opts.ExpectedRevision

	// Resolve file path (expand ~ and make absolute)
	filePath = resolveFilePath(filePath)

	if commandUsesVersioning(command, cmdArgs) {
		versions, err := versioning.Open(appConfig.Versioning.MaxVersions)
		if err != nil {
			fmt.Fprintf(os.Stderr, "tdx: failed to open version store: %v\n", err)
			return 1
		}
		defer func() {
			if err := errors.Join(versions.PruneAll(versions.MaxVersions), versions.Close()); err != nil {
				fmt.Fprintf(os.Stderr, "tdx: close version store: %v\n", err)
				exitCode = 1
			}
		}()
		cli.Store = historyStore(versions)
		tuiCfg.Store = cli.Store
		wireVersioningTUI(tuiCfg, versions)
	}

	// Handle commands
	switch command {
	case "help", "--help", "-h":
		printHelp()
	case "--version", "-v":
		_, _ = lipgloss.Fprintf(os.Stdout, "tdx v%s\n", Version)
	case "--debug-config":
		_, _ = lipgloss.Fprintf(os.Stdout, "Theme: %s\n", appConfig.Theme.Name)
		_, _ = lipgloss.Fprintf(os.Stdout, "Colors.Accent: %s\n", appConfig.Colors.Accent)
		_, _ = lipgloss.Fprintf(os.Stdout, "Colors.Success: %s\n", appConfig.Colors.Success)
		_, _ = lipgloss.Fprintf(os.Stdout, "Display.CheckSymbol: %s\n", appConfig.Display.CheckSymbol)
		_, _ = lipgloss.Fprintf(os.Stdout, "Display.SelectMarker: %s\n", appConfig.Display.SelectMarker)
		_, _ = lipgloss.Fprintf(os.Stdout, "Defaults.File: %s\n", appConfig.Defaults.File)
		_, _ = lipgloss.Fprintf(os.Stdout, "Defaults.MaxVisible: %d\n", appConfig.Defaults.MaxVisible)
		_, _ = lipgloss.Fprintf(os.Stdout, "Defaults.WordWrap: %v\n", appConfig.Defaults.WordWrap)
		_, _ = lipgloss.Fprintf(os.Stdout, "Defaults.ShowHeadings: %v\n", appConfig.Defaults.ShowHeadings)
		_, _ = lipgloss.Fprintf(os.Stdout, "Defaults.ReadOnly: %v\n", appConfig.Defaults.ReadOnly)
		_, _ = lipgloss.Fprintf(os.Stdout, "Defaults.FilterDone: %v\n", appConfig.Defaults.FilterDone)
		_, _ = lipgloss.Fprintf(os.Stdout, "Recent.MaxFiles: %d\n", appConfig.Recent.MaxFiles)
	case "list":
		if err := cli.WriteList(os.Stdout, filePath, opts.List); err != nil {
			fmt.Fprintf(os.Stderr, "tdx: %v\n", err)
			return 1
		}
	case "revision":
		fm, err := (markdown.Store{}).ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "tdx: %v\n", err)
			return 1
		}
		token, err := fm.RevisionToken()
		if err != nil {
			fmt.Fprintf(os.Stderr, "tdx: %v\n", err)
			return 1
		}
		fmt.Println(token)
	case "completion":
		if len(cmdArgs) != 1 {
			fmt.Fprintln(os.Stderr, "tdx: completion requires bash, zsh, or fish")
			return 1
		}
		if err := cmd.WriteCompletion(os.Stdout, cmdArgs[0]); err != nil {
			fmt.Fprintf(os.Stderr, "tdx: %v\n", err)
			return 1
		}
	case "add", "toggle", "edit", "delete", "done", "undone":
		if err := cli.HandleCommand(command, cmdArgs, filePath); err != nil {
			fmt.Fprintf(os.Stderr, "tdx: %v\n", err)
			return 1
		}
	case "last":
		handleLastCommand(runtime, readOnly, showHeadings, maxVisible)
	case "recent":
		handleRecentCommand(runtime, cmdArgs, readOnly, showHeadings, maxVisible)
	case "":
		// Launch TUI
		runtime.Run(filePath, readOnly, showHeadings, maxVisible)
	default:
		fmt.Fprintf(os.Stderr, "tdx: unknown command: %s (see tdx help)\n", command)
		return 1
	}
	return 0
}

func printHelp() {
	help := fmt.Sprintf(`tdx - %s

Usage:
  tdx [file.md] [command] [args]
  tdx --file <path> [command] [args]

Options:
  -f, --file <path>       Select any file path (including paths without .md)
  -r, --read-only         Manual save in TUI; reject CLI mutations
      --manual-save     Alias for --read-only
      --if-revision <R> Reject a mutation unless the loaded revision matches
      --show-headings    Display markdown headings between tasks
  -m, --max-visible <N>   Set max visible items (0 = unlimited)
      --                 Treat remaining arguments as literal text

List options:
      --json             Emit a JSON array for scripts and editor integrations
      --status <value>   Filter by all (default), open, or done
      --tag <tag>        Filter by exact tag; repeat to require every tag
      --priority <N>     Filter by priority (0 means unset)
      --due <value>      all, none, overdue, today, week, or YYYY-MM-DD
      --section <title>  Exact heading title, including its subsections
      --with-revision    JSON envelope with schema_version, revision, tasks

Commands:
  (none)              Launch interactive TUI
  list                List todos with composable filters
  add "text"          Add a new todo
  toggle <index>      Toggle todo completion
  done <index>        Mark complete (safe to retry)
  undone <index>      Mark open (safe to retry)
  revision            Print current document revision
  completion <shell>  Generate bash, zsh, or fish completion
  edit <index> "text" Edit todo text
  delete <index>      Delete a todo
  last                Open the most recently used file
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
  s / S               Browse sections / show all sections
  ?                   Toggle help
  Esc                 Quit

Examples:
  tdx --file tasks.md list --json --status open --tag backend
  tdx add -- --read-only
  tdx --read-only tasks.md list

JSON indexes are one-based positions in the full file, not stable IDs.
Errors go to stderr and return a nonzero exit code.`, Description)
	_, _ = lipgloss.Fprintln(os.Stdout, help)
}

func handleLastCommand(runtime tui.Runtime, readOnly bool, showHeadings bool, maxVisible int) {
	// Load recent files
	recentFiles, err := runtime.Config.Recent.Load()
	if err != nil {
		_, _ = lipgloss.Fprintf(os.Stdout, "Error loading recent files: %v\n", err)
		os.Exit(1)
	}

	if len(recentFiles.Files) == 0 {
		_, _ = lipgloss.Fprintln(os.Stdout, "No recent files. Open a file first with 'tdx <file.md>'")
		os.Exit(1)
	}

	// Sort by score and open the most recent
	recentFiles.SortByScore()
	filePath := recentFiles.Files[0].Path
	runtime.Run(filePath, readOnly, showHeadings, maxVisible)
}

func handleRecentCommand(runtime tui.Runtime, args []string, readOnly bool, showHeadings bool, maxVisible int) {
	// Handle "clear" subcommand
	if len(args) > 0 && args[0] == "clear" {
		if err := runtime.Config.Recent.Clear(); err != nil {
			_, _ = lipgloss.Fprintf(os.Stdout, "Error clearing recent files: %v\n", err)
			os.Exit(1)
		}
		_, _ = lipgloss.Fprintln(os.Stdout, "Recent files cleared")
		return
	}

	// Load recent files
	recentFiles, err := runtime.Config.Recent.Load()
	if err != nil {
		_, _ = lipgloss.Fprintf(os.Stdout, "Error loading recent files: %v\n", err)
		os.Exit(1)
	}

	if len(recentFiles.Files) == 0 {
		_, _ = lipgloss.Fprintln(os.Stdout, "No recent files")
		return
	}

	// Sort by score (recency * frequency)
	recentFiles.SortByScore()

	// If numeric argument, open that file
	if len(args) > 0 {
		index, err := strconv.Atoi(args[0])
		if err != nil || index < 1 || index > len(recentFiles.Files) {
			_, _ = lipgloss.Fprintf(os.Stdout, "Error: invalid file number. Use 1-%d\n", len(recentFiles.Files))
			os.Exit(1)
		}

		// Open the selected file (1-indexed)
		filePath := recentFiles.Files[index-1].Path
		runtime.Run(filePath, readOnly, showHeadings, maxVisible)
		return
	}

	// No args - list all recent files
	_, _ = lipgloss.Fprintln(os.Stdout, "Recent files:")
	for i, file := range recentFiles.Files {
		// Show relative path if in home directory
		displayPath := file.Path
		if home, err := os.UserHomeDir(); err == nil {
			if rel, err := filepath.Rel(home, file.Path); err == nil && !strings.HasPrefix(rel, "..") {
				displayPath = "~/" + rel
			}
		}

		_, _ = lipgloss.Fprintf(os.Stdout, "  %d. %s (accessed %d times, last: %s)\n",
			i+1,
			displayPath,
			file.AccessCount,
			file.LastAccessed.Format("2006-01-02 15:04"))
	}
	_, _ = lipgloss.Fprintln(os.Stdout, "\nUse 'tdx recent <number>' to open a file")
}

// resolveFilePath expands ~ to home directory and resolves relative paths to absolute
func resolveFilePath(filePath string) string {
	// Expand ~ to home directory
	if strings.HasPrefix(filePath, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			filePath = filepath.Join(home, filePath[2:])
		}
	}

	// Resolve to absolute path (for relative paths like "todo.md")
	if !filepath.IsAbs(filePath) {
		if cwd, err := os.Getwd(); err == nil {
			filePath = filepath.Join(cwd, filePath)
		}
	}

	return filePath
}
