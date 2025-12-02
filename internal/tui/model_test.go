package tui

import (
	"testing"

	"github.com/niklas-heer/tdx/internal/markdown"
)

// testConfig creates a test configuration
func testConfig() *ConfigType {
	cfg := &ConfigType{}
	cfg.Display.CheckSymbol = "[x]"
	cfg.Display.SelectMarker = ">"
	cfg.Display.MaxVisible = 10
	return cfg
}

// testStyles creates test style functions that return plain text
func testStyles() *StyleFuncsType {
	identity := func(s string) string { return s }
	return &StyleFuncsType{
		Magenta:        identity,
		Cyan:           identity,
		Dim:            identity,
		Green:          identity,
		Yellow:         identity,
		Code:           identity,
		Tag:            identity,
		PriorityHigh:   identity,
		PriorityMedium: identity,
		PriorityLow:    identity,
		DueUrgent:      identity,
		DueSoon:        identity,
		DueFuture:      identity,
	}
}

// testModel creates a test model with sample todos
func testModel(todos []string) Model {
	fm := &markdown.FileModel{
		Todos: make([]markdown.Todo, len(todos)),
	}
	for i, text := range todos {
		fm.Todos[i] = markdown.Todo{Text: text, Checked: false}
	}
	return New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
}

func TestNew_InitializesCorrectly(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task 1", Checked: false},
			{Text: "Task 2", Checked: true},
		},
	}

	m := New("/tmp/test.md", fm, false, true, 5, testConfig(), testStyles(), "1.0.0")

	if m.FilePath != "/tmp/test.md" {
		t.Errorf("FilePath = %q, want %q", m.FilePath, "/tmp/test.md")
	}
	if len(m.FileModel.Todos) != 2 {
		t.Errorf("Todos count = %d, want 2", len(m.FileModel.Todos))
	}
	if m.ReadOnly != false {
		t.Error("ReadOnly should be false")
	}
	if m.ShowHeadings != true {
		t.Error("ShowHeadings should be true")
	}
	if m.MaxVisibleOverride != 5 {
		t.Errorf("MaxVisibleOverride = %d, want 5", m.MaxVisibleOverride)
	}
	if m.Version() != "1.0.0" {
		t.Errorf("Version = %q, want %q", m.Version(), "1.0.0")
	}
	if m.WordWrap != true {
		t.Error("WordWrap should default to true")
	}
	// Note: headingsDirty is false because showHeadings=true causes GetDocumentTree() to be called during New()
	if m.headingsDirty != false {
		t.Error("headingsDirty should be false after initialization with showHeadings=true")
	}
}

func TestNew_ExtractsAvailableTags(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task with @work tag", Checked: false, Tags: []string{"@work"}},
			{Text: "Task with @home tag", Checked: false, Tags: []string{"@home"}},
			{Text: "Task with both @work and @urgent", Checked: false, Tags: []string{"@work", "@urgent"}},
		},
	}

	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	if len(m.AvailableTags) != 3 {
		t.Errorf("AvailableTags count = %d, want 3", len(m.AvailableTags))
	}
}

func TestConfig_FallsBackToGlobal(t *testing.T) {
	// Set global config
	oldConfig := Config
	Config = testConfig()
	Config.Display.CheckSymbol = "global"
	defer func() { Config = oldConfig }()

	// Create model without injected config
	fm := &markdown.FileModel{}
	m := New("/tmp/test.md", fm, false, false, -1, nil, nil, "")

	if m.Config().Display.CheckSymbol != "global" {
		t.Errorf("Should fall back to global Config")
	}
}

func TestStyles_FallsBackToGlobal(t *testing.T) {
	// Set global styles
	oldStyles := StyleFuncs
	StyleFuncs = testStyles()
	defer func() { StyleFuncs = oldStyles }()

	// Create model without injected styles
	fm := &markdown.FileModel{}
	m := New("/tmp/test.md", fm, false, false, -1, nil, nil, "")

	if m.Styles() == nil {
		t.Error("Should fall back to global StyleFuncs")
	}
}

func TestGetHeadings_CachesResults(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task 1", Checked: false},
		},
	}

	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	// First call should populate cache
	_ = m.GetHeadings()
	if m.headingsDirty {
		t.Error("headingsDirty should be false after GetHeadings")
	}

	// Second call should return cached result (not repopulate)
	_ = m.GetHeadings()
	if m.headingsDirty {
		t.Error("headingsDirty should still be false after second GetHeadings")
	}
}

func TestInvalidateHeadingsCache_SetsFlag(t *testing.T) {
	fm := &markdown.FileModel{}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	// Get headings to clear dirty flag
	_ = m.GetHeadings()
	if m.headingsDirty {
		t.Error("headingsDirty should be false after GetHeadings")
	}

	// Invalidate cache
	m.InvalidateHeadingsCache()
	if !m.headingsDirty {
		t.Error("headingsDirty should be true after InvalidateHeadingsCache")
	}
}

func TestModel_LocallyModifiedTracking(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task 1", Checked: false},
		},
	}

	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "")

	// Initially empty
	if len(m.LocallyModified) != 0 {
		t.Errorf("LocallyModified should be empty, got %d items", len(m.LocallyModified))
	}

	// Can track modifications
	m.LocallyModified["Task 1"] = true
	if !m.LocallyModified["Task 1"] {
		t.Error("Should track locally modified tasks")
	}
}

func TestModel_InitialModeStates(t *testing.T) {
	m := testModel([]string{"Task 1"})

	if m.InputMode {
		t.Error("InputMode should be false initially")
	}
	if m.EditMode {
		t.Error("EditMode should be false initially")
	}
	if m.MoveMode {
		t.Error("MoveMode should be false initially")
	}
	if m.HelpMode {
		t.Error("HelpMode should be false initially")
	}
	if m.SearchMode {
		t.Error("SearchMode should be false initially")
	}
	if m.CommandMode {
		t.Error("CommandMode should be false initially")
	}
	if m.FilterMode {
		t.Error("FilterMode should be false initially")
	}
}

func TestModel_SelectedIndexInitialization(t *testing.T) {
	m := testModel([]string{"Task 1", "Task 2", "Task 3"})

	if m.SelectedIndex != 0 {
		t.Errorf("SelectedIndex = %d, want 0", m.SelectedIndex)
	}
}

func TestModel_CommandsInitialized(t *testing.T) {
	m := testModel([]string{"Task 1"})

	if len(m.Commands) == 0 {
		t.Error("Commands should be initialized")
	}
}
