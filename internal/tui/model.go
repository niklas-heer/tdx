package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/niklas-heer/tdx/internal/config"
	"github.com/niklas-heer/tdx/internal/markdown"
)

// StyleFuncsType holds style functions for rendering
type StyleFuncsType struct {
	Magenta func(string) string
	Cyan    func(string) string
	Dim     func(string) string
	Green   func(string) string
	Yellow  func(string) string
	Code    func(string) string
}

// ConfigType holds display configuration
type ConfigType struct {
	Display struct {
		CheckSymbol  string
		SelectMarker string
		MaxVisible   int
	}
}

// Global variables for backward compatibility (deprecated - use Model methods instead)
var (
	Config     *ConfigType
	StyleFuncs *StyleFuncsType
	Version    string
)

// Model holds the TUI application state
type Model struct {
	FilePath            string
	FileModel           markdown.FileModel
	SelectedIndex       int
	SavedCursorIndex    int // Saved cursor position for move mode cancel
	InputMode           bool
	InsertAfterCursor   bool // true = insert after cursor (n), false = append to end (N)
	EditMode            bool
	MoveMode            bool
	HelpMode            bool
	SearchMode          bool
	CommandMode         bool
	RecentFilesMode     bool
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

	// Recent files state
	RecentFiles       []config.RecentFile // List of recent files
	RecentFilesCursor int                 // Cursor position in recent files list
	RecentFilesSearch string              // Search filter for recent files

	// Cached headings for performance (avoid re-extraction on every render)
	cachedHeadings []markdown.Heading
	headingsDirty  bool

	// Search debouncing
	searchPending bool // Whether a search update is pending

	// Vim-style multi-key sequence tracking
	gPressed bool // Whether 'g' was pressed (for gg sequence)

	// Document tree for predictable movement and deletion
	documentTree *DocumentTree
	treeDirty    bool // Whether the tree needs rebuilding

	// Injected dependencies (previously global)
	config     *ConfigType
	styles     *StyleFuncsType
	appVersion string
}

// ClearCopyFeedbackMsg is sent to clear copy feedback after a delay
type ClearCopyFeedbackMsg struct{}

// SearchDebounceMsg is sent after debounce delay to trigger search update
type SearchDebounceMsg struct{}

// CommandDebounceMsg is sent after debounce delay to trigger command filter update
type CommandDebounceMsg struct{}

// New creates a new TUI model with injected dependencies
func New(filePath string, fm *markdown.FileModel, readOnly bool, showHeadings bool, maxVisible int, config *ConfigType, styles *StyleFuncsType, version string) Model {
	// Extract all available tags from todos
	availableTags := markdown.GetAllTags(fm.Todos)

	m := Model{
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
		WordWrap:           true,  // Default to true for better UX
		headingsDirty:      true,  // Force initial cache population
		searchPending:      false, // No pending search on init
		treeDirty:          true,  // Force initial tree build
		config:             config,
		styles:             styles,
		appVersion:         version,
	}

	// Apply metadata settings (including FilterDone) from file
	if fm.Metadata != nil {
		if fm.Metadata.FilterDone != nil {
			m.FilterDone = *fm.Metadata.FilterDone
		}
		if fm.Metadata.WordWrap != nil {
			m.WordWrap = *fm.Metadata.WordWrap
		}
	}

	// Position cursor on first visible item if filters are active
	if m.hasActiveFilters() || m.ShowHeadings {
		tree := m.GetDocumentTree()
		// Find the first visible todo node
		for _, node := range tree.Flat {
			if node.Type == DocNodeTodo && node.Visible {
				if m.SelectedIndex != node.TodoIndex {
					m.SelectedIndex = node.TodoIndex
					// Invalidate tree since we changed SelectedIndex after building it
					// This ensures the tree's Selected index will be correct on next access
					m.InvalidateDocumentTree()
				}
				break
			}
		}
	}

	return m
}

// Config returns the model's configuration (for backward compatibility during transition)
func (m *Model) Config() *ConfigType {
	if m.config != nil {
		return m.config
	}
	return Config // Fall back to global for backward compatibility
}

// Styles returns the model's style functions (for backward compatibility during transition)
func (m *Model) Styles() *StyleFuncsType {
	if m.styles != nil {
		return m.styles
	}
	return StyleFuncs // Fall back to global for backward compatibility
}

// Version returns the app version string
func (m *Model) Version() string {
	if m.appVersion != "" {
		return m.appVersion
	}
	return Version // Fall back to global for backward compatibility
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

// GetHeadings returns cached headings, refreshing if dirty
func (m *Model) GetHeadings() []markdown.Heading {
	if m.headingsDirty {
		m.cachedHeadings = m.FileModel.GetHeadings()
		m.headingsDirty = false
	}
	return m.cachedHeadings
}

// InvalidateHeadingsCache marks the headings cache as needing refresh
func (m *Model) InvalidateHeadingsCache() {
	m.headingsDirty = true
	m.treeDirty = true // Headings affect visible tree
}

// GetDocumentTree returns the document tree, rebuilding if necessary
func (m *Model) GetDocumentTree() *DocumentTree {
	if m.treeDirty || m.documentTree == nil {
		m.documentTree = m.BuildDocumentTree()
		m.treeDirty = false
	}
	return m.documentTree
}

// InvalidateDocumentTree marks the document tree as needing rebuild
func (m *Model) InvalidateDocumentTree() {
	m.treeDirty = true
}

// searchDebounceCmd returns a command that triggers search update after a delay
func searchDebounceCmd() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		return SearchDebounceMsg{}
	})
}

// commandDebounceCmd returns a command that triggers command filter update after a delay
func commandDebounceCmd() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		return CommandDebounceMsg{}
	})
}
