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
	cfg.Defaults.WordWrap = true
	cfg.Defaults.FilterDone = false
	cfg.Defaults.ShowHeadings = false
	cfg.Defaults.ReadOnly = false
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

// Tests for ConfigType.Defaults

func TestConfigType_HasDefaultsStruct(t *testing.T) {
	cfg := &ConfigType{}

	// Defaults should be accessible
	cfg.Defaults.WordWrap = true
	cfg.Defaults.FilterDone = true
	cfg.Defaults.ShowHeadings = true
	cfg.Defaults.ReadOnly = true

	if !cfg.Defaults.WordWrap {
		t.Error("Defaults.WordWrap should be settable")
	}
	if !cfg.Defaults.FilterDone {
		t.Error("Defaults.FilterDone should be settable")
	}
	if !cfg.Defaults.ShowHeadings {
		t.Error("Defaults.ShowHeadings should be settable")
	}
	if !cfg.Defaults.ReadOnly {
		t.Error("Defaults.ReadOnly should be settable")
	}
}

func TestConfigType_DefaultsUsedInRun(t *testing.T) {
	// This test verifies the ConfigType structure includes Defaults
	// which Run() uses to set initial model state
	cfg := testConfig()

	// Verify testConfig sets Defaults
	if !cfg.Defaults.WordWrap {
		t.Error("testConfig should set WordWrap=true")
	}
	if cfg.Defaults.FilterDone {
		t.Error("testConfig should set FilterDone=false")
	}
	if cfg.Defaults.ShowHeadings {
		t.Error("testConfig should set ShowHeadings=false")
	}
	if cfg.Defaults.ReadOnly {
		t.Error("testConfig should set ReadOnly=false")
	}
}

func TestModel_UsesInjectedConfig(t *testing.T) {
	cfg := &ConfigType{}
	cfg.Display.CheckSymbol = "✓"
	cfg.Display.SelectMarker = "→"
	cfg.Display.MaxVisible = 25
	cfg.Defaults.WordWrap = false  // Non-default value
	cfg.Defaults.FilterDone = true // Non-default value

	fm := &markdown.FileModel{
		Todos: []markdown.Todo{{Text: "Test", Checked: false}},
	}

	m := New("/tmp/test.md", fm, false, false, -1, cfg, testStyles(), "")

	// Verify model uses injected config
	if m.Config().Display.CheckSymbol != "✓" {
		t.Errorf("Model should use injected config CheckSymbol")
	}
	if m.Config().Display.SelectMarker != "→" {
		t.Errorf("Model should use injected config SelectMarker")
	}
	if m.Config().Display.MaxVisible != 25 {
		t.Errorf("Model should use injected config MaxVisible")
	}
}

func TestStyleFuncsType_HasAllRequiredFields(t *testing.T) {
	styles := testStyles()

	// Core styles
	if styles.Magenta == nil {
		t.Error("Magenta style function should not be nil")
	}
	if styles.Cyan == nil {
		t.Error("Cyan style function should not be nil")
	}
	if styles.Dim == nil {
		t.Error("Dim style function should not be nil")
	}
	if styles.Green == nil {
		t.Error("Green style function should not be nil")
	}
	if styles.Yellow == nil {
		t.Error("Yellow style function should not be nil")
	}
	if styles.Code == nil {
		t.Error("Code style function should not be nil")
	}

	// New style functions for tags, priorities, and due dates
	if styles.Tag == nil {
		t.Error("Tag style function should not be nil")
	}
	if styles.PriorityHigh == nil {
		t.Error("PriorityHigh style function should not be nil")
	}
	if styles.PriorityMedium == nil {
		t.Error("PriorityMedium style function should not be nil")
	}
	if styles.PriorityLow == nil {
		t.Error("PriorityLow style function should not be nil")
	}
	if styles.DueUrgent == nil {
		t.Error("DueUrgent style function should not be nil")
	}
	if styles.DueSoon == nil {
		t.Error("DueSoon style function should not be nil")
	}
	if styles.DueFuture == nil {
		t.Error("DueFuture style function should not be nil")
	}
}

func TestStyleFuncsType_FunctionsWork(t *testing.T) {
	styles := testStyles()
	testInput := "test text"

	// Test all style functions return non-empty output
	styleFuncs := map[string]func(string) string{
		"Tag":            styles.Tag,
		"PriorityHigh":   styles.PriorityHigh,
		"PriorityMedium": styles.PriorityMedium,
		"PriorityLow":    styles.PriorityLow,
		"DueUrgent":      styles.DueUrgent,
		"DueSoon":        styles.DueSoon,
		"DueFuture":      styles.DueFuture,
	}

	for name, fn := range styleFuncs {
		result := fn(testInput)
		if result == "" {
			t.Errorf("%s style function returned empty string", name)
		}
	}
}
