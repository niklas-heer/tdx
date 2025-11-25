package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/niklas-heer/tdx/internal/markdown"
)

// Model holds the TUI application state
type Model struct {
	FilePath            string
	FileModel           markdown.FileModel
	SelectedIndex       int
	InputMode           bool
	EditMode            bool
	MoveMode            bool
	HelpMode            bool
	SearchMode          bool
	CommandMode         bool
	MaxVisibleInputMode bool
	SearchResults       []int
	SearchCursor        int
	InputBuffer         string
	CursorPos           int
	NumberBuffer        string
	History             *markdown.FileModel

	CopyFeedback bool
	Err          error

	// Command palette state
	Commands           []Command
	FilteredCmds       []int
	CommandCursor      int
	ReadOnly           bool
	FilterDone         bool
	WordWrap           bool
	TermWidth          int
	HideLineNumbers    bool
	MaxVisibleOverride int
	ShowHeadings       bool

	// Track which todos we've locally modified (by text) since last sync
	LocallyModified map[string]bool // todo text -> true if we toggled it

	// Tag filtering state
	FilterMode      bool     // Whether we're in tag filter mode
	FilteredTags    []string // Currently active tag filters
	AvailableTags   []string // All unique tags from todos
	TagFilterCursor int      // Cursor position in tag filter list
}

// ClearCopyFeedbackMsg is sent to clear copy feedback after a delay
type ClearCopyFeedbackMsg struct{}

// New creates a new TUI model
func New(filePath string, fm *markdown.FileModel, readOnly bool, showHeadings bool, maxVisible int) Model {
	// Extract all available tags from todos
	availableTags := markdown.GetAllTags(fm.Todos)

	return Model{
		FilePath:           filePath,
		FileModel:          *fm,
		SelectedIndex:      0,
		Commands:           InitCommands(),
		ReadOnly:           readOnly,
		ShowHeadings:       showHeadings,
		MaxVisibleOverride: maxVisible,
		LocallyModified:    make(map[string]bool),
		AvailableTags:      availableTags,
		FilteredTags:       []string{},
		WordWrap:           true, // Default to true for better UX
	}
}

// Init initializes the TUI
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnableBracketedPaste,
		watchFileChanges(), // Start watching for file changes
	)
}

// watchFileChanges returns a command that checks for file changes periodically
func watchFileChanges() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return FileChangedMsg{}
	})
}
