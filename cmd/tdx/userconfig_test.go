package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestNewStyleFuncs(t *testing.T) {
	config := DefaultConfig()
	styles := NewStyles(config)

	styleFuncs := NewStyleFuncs(styles)

	if styleFuncs == nil {
		t.Fatal("NewStyleFuncs returned nil")
	}

	// Test that each function exists and returns non-empty for non-empty input
	tests := []struct {
		name string
		fn   func(string) string
	}{
		{"Magenta", styleFuncs.Magenta},
		{"Cyan", styleFuncs.Cyan},
		{"Dim", styleFuncs.Dim},
		{"Green", styleFuncs.Green},
		{"Yellow", styleFuncs.Yellow},
		{"Code", styleFuncs.Code},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn("test")
			if result == "" {
				t.Errorf("%s function returned empty string", tt.name)
			}
		})
	}
}

func TestGetConfigPath_Coverage(t *testing.T) {
	// Save original env
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", origXDG) }()

	tmpDir := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	path, err := getConfigPath()
	if err != nil {
		t.Fatalf("getConfigPath failed: %v", err)
	}

	expected := filepath.Join(tmpDir, "tdx", "config.toml")
	if path != expected {
		t.Errorf("getConfigPath() = %q, want %q", path, expected)
	}

	// Verify directory was created
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("getConfigPath should create the directory")
	}
}

func TestSaveTheme_Coverage(t *testing.T) {
	isolateHome(t)
	// Save original env
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", origXDG) }()

	tmpDir := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	err := SaveTheme("tokyo-night")
	if err != nil {
		t.Fatalf("SaveTheme failed: %v", err)
	}

	// Verify file was created
	configPath := filepath.Join(tmpDir, "tdx", "config.toml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Config file is empty")
	}

	// Verify theme name is in the file
	if !strings.Contains(string(content), "tokyo-night") {
		t.Errorf("Config should contain theme name, got: %s", string(content))
	}
}

func TestSaveTheme_PreservesExisting_Coverage(t *testing.T) {
	isolateHome(t)
	// Save original env
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", origXDG) }()

	tmpDir := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create initial config with display settings
	configDir := filepath.Join(tmpDir, "tdx")
	_ = os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, "config.toml")

	initialConfig := `[theme]
name = "nord"

[display]
check_symbol = "x"
`
	_ = os.WriteFile(configPath, []byte(initialConfig), 0644)

	// Save new theme
	err := SaveTheme("dracula")
	if err != nil {
		t.Fatalf("SaveTheme failed: %v", err)
	}

	// Verify new theme was saved
	content, _ := os.ReadFile(configPath)
	if !strings.Contains(string(content), "dracula") {
		t.Errorf("Config should contain new theme name 'dracula', got: %s", string(content))
	}
}

func TestLoadUserThemes_NoDirectory_Coverage(t *testing.T) {
	// Save original env
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", origXDG) }()

	// Use a non-existent directory
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "does-not-exist")
	_ = os.Setenv("XDG_CONFIG_HOME", nonExistent)

	// Should not panic, just return empty or nil
	themes := loadUserThemes()
	_ = themes // Just verify it doesn't panic
}

func TestGetAllThemes_Coverage(t *testing.T) {
	themes := getAllThemes()

	if len(themes) == 0 {
		t.Error("getAllThemes should return at least builtin themes")
	}

	// Check for some expected builtin themes
	expectedThemes := []string{"tokyo-night", "dracula", "nord"}
	for _, expected := range expectedThemes {
		found := false
		for name := range themes {
			if name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected theme %q not found", expected)
		}
	}
}

// Tests for DefaultConfig and new config structure

func TestDefaultConfig_HasAllFields(t *testing.T) {
	config := DefaultConfig()

	// Theme
	if config.Theme.Name == "" {
		t.Error("DefaultConfig should have a theme name")
	}
	if config.Theme.Name != "tokyo-night" {
		t.Errorf("Default theme should be 'tokyo-night', got %q", config.Theme.Name)
	}

	// Display
	if config.Display.CheckSymbol == "" {
		t.Error("DefaultConfig should have a check symbol")
	}
	if config.Display.SelectMarker == "" {
		t.Error("DefaultConfig should have a select marker")
	}

	// Defaults
	if config.Defaults.File != "todo.md" {
		t.Errorf("Default file should be 'todo.md', got %q", config.Defaults.File)
	}
	if config.Defaults.WordWrap != true {
		t.Error("Default WordWrap should be true")
	}
	if config.Defaults.ShowHeadings != false {
		t.Error("Default ShowHeadings should be false")
	}
	if config.Defaults.ReadOnly != false {
		t.Error("Default ReadOnly should be false")
	}
	if config.Defaults.FilterDone != false {
		t.Error("Default FilterDone should be false")
	}

	// Recent
	if config.Recent.MaxFiles != 20 {
		t.Errorf("Default MaxFiles should be 20, got %d", config.Recent.MaxFiles)
	}
	if config.Versioning.MaxVersions != 100 {
		t.Errorf("Default MaxVersions should be 100, got %d", config.Versioning.MaxVersions)
	}
}

func TestLoadConfig_DefaultsWhenNoFile(t *testing.T) {
	isolateHome(t)
	// Save original env
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", origXDG) }()

	// Use a non-existent directory
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "does-not-exist")
	_ = os.Setenv("XDG_CONFIG_HOME", nonExistent)

	config := LoadConfig()

	// Should return defaults
	if config.Theme.Name != "tokyo-night" {
		t.Errorf("LoadConfig without file should default to 'tokyo-night', got %q", config.Theme.Name)
	}
	if config.Defaults.File != "todo.md" {
		t.Errorf("LoadConfig without file should default file to 'todo.md', got %q", config.Defaults.File)
	}
	if config.Defaults.WordWrap != true {
		t.Error("LoadConfig without file should default WordWrap to true")
	}
}

func TestLoadConfig_ParsesDefaultsSection(t *testing.T) {
	isolateHome(t)
	// Save original env
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", origXDG) }()

	tmpDir := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create config with defaults section
	configDir := filepath.Join(tmpDir, "tdx")
	_ = os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, "config.toml")

	configContent := `[theme]
name = "dracula"

[defaults]
file = "~/todos/main.md"
max_visible = 50
word_wrap = false
show_headings = true
read_only = true
filter_done = true

[recent]
max_files = 30
`
	_ = os.WriteFile(configPath, []byte(configContent), 0644)

	config := LoadConfig()

	if config.Theme.Name != "dracula" {
		t.Errorf("Theme should be 'dracula', got %q", config.Theme.Name)
	}
	if config.Defaults.File != "~/todos/main.md" {
		t.Errorf("Defaults.File should be '~/todos/main.md', got %q", config.Defaults.File)
	}
	if config.Defaults.MaxVisible != 50 {
		t.Errorf("Defaults.MaxVisible should be 50, got %d", config.Defaults.MaxVisible)
	}
	if config.Defaults.WordWrap != false {
		t.Error("Defaults.WordWrap should be false")
	}
	if config.Defaults.ShowHeadings != true {
		t.Error("Defaults.ShowHeadings should be true")
	}
	if config.Defaults.ReadOnly != true {
		t.Error("Defaults.ReadOnly should be true")
	}
	if config.Defaults.FilterDone != true {
		t.Error("Defaults.FilterDone should be true")
	}
	if config.Recent.MaxFiles != 30 {
		t.Errorf("Recent.MaxFiles should be 30, got %d", config.Recent.MaxFiles)
	}
}

func TestLoadConfig_PartialDefaults(t *testing.T) {
	isolateHome(t)
	// Save original env
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", origXDG) }()

	tmpDir := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create config with only some defaults set
	configDir := filepath.Join(tmpDir, "tdx")
	_ = os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, "config.toml")

	// Only set file and filter_done, leave others to defaults
	configContent := `[defaults]
file = "TASKS.md"
filter_done = true
`
	_ = os.WriteFile(configPath, []byte(configContent), 0644)

	config := LoadConfig()

	// Custom values
	if config.Defaults.File != "TASKS.md" {
		t.Errorf("Defaults.File should be 'TASKS.md', got %q", config.Defaults.File)
	}
	if config.Defaults.FilterDone != true {
		t.Error("Defaults.FilterDone should be true")
	}

	// Default values for unset fields
	if config.Defaults.WordWrap != true {
		t.Error("Defaults.WordWrap should remain default (true)")
	}
	if config.Defaults.ShowHeadings != false {
		t.Error("Defaults.ShowHeadings should remain default (false)")
	}
	if config.Defaults.ReadOnly != false {
		t.Error("Defaults.ReadOnly should remain default (false)")
	}
	if config.Theme.Name != "tokyo-night" {
		t.Errorf("Theme should default to 'tokyo-night', got %q", config.Theme.Name)
	}
}

func TestLoadConfig_MaxVisibleZeroIsValid(t *testing.T) {
	isolateHome(t)
	// Save original env
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", origXDG) }()

	tmpDir := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create config with max_visible explicitly set to 0
	configDir := filepath.Join(tmpDir, "tdx")
	_ = os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, "config.toml")

	configContent := `[defaults]
max_visible = 0
`
	_ = os.WriteFile(configPath, []byte(configContent), 0644)

	config := LoadConfig()

	// 0 means unlimited and should be preserved, not replaced with default
	if config.Defaults.MaxVisible != 0 {
		t.Errorf("Defaults.MaxVisible should be 0 (unlimited), got %d", config.Defaults.MaxVisible)
	}
}

func TestSaveTheme_PreservesDefaultsSection(t *testing.T) {
	isolateHome(t)
	// Save original env
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", origXDG) }()

	tmpDir := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create initial config with defaults section
	configDir := filepath.Join(tmpDir, "tdx")
	_ = os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, "config.toml")

	initialConfig := `[theme]
name = "nord"

[defaults]
file = "~/my-todos.md"
filter_done = true
`
	_ = os.WriteFile(configPath, []byte(initialConfig), 0644)

	// Save new theme
	err := SaveTheme("dracula")
	if err != nil {
		t.Fatalf("SaveTheme failed: %v", err)
	}

	// Load and verify defaults were preserved
	config := LoadConfig()

	if config.Theme.Name != "dracula" {
		t.Errorf("Theme should be 'dracula', got %q", config.Theme.Name)
	}
	if config.Defaults.File != "~/my-todos.md" {
		t.Errorf("Defaults.File should be preserved as '~/my-todos.md', got %q", config.Defaults.File)
	}
	if config.Defaults.FilterDone != true {
		t.Error("Defaults.FilterDone should be preserved as true")
	}
}

// Tests for new style functions (Tag, Priority, Due)

func TestNewStyleFuncs_IncludesNewStyles(t *testing.T) {
	config := DefaultConfig()
	styles := NewStyles(config)
	styleFuncs := NewStyleFuncs(styles)

	// Test new style functions exist and work
	newStyleTests := []struct {
		name string
		fn   func(string) string
	}{
		{"Tag", styleFuncs.Tag},
		{"PriorityHigh", styleFuncs.PriorityHigh},
		{"PriorityMedium", styleFuncs.PriorityMedium},
		{"PriorityLow", styleFuncs.PriorityLow},
		{"DueUrgent", styleFuncs.DueUrgent},
		{"DueSoon", styleFuncs.DueSoon},
		{"DueFuture", styleFuncs.DueFuture},
	}

	for _, tt := range newStyleTests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.fn == nil {
				t.Errorf("%s function is nil", tt.name)
				return
			}
			result := tt.fn("test")
			if result == "" {
				t.Errorf("%s function returned empty string", tt.name)
			}
		})
	}
}

func TestNewStyles_HasNewColorStyles(t *testing.T) {
	config := DefaultConfig()
	styles := NewStyles(config)

	// Verify new styles can render text (styles are initialized)
	// We test by rendering text and checking it's not empty
	testText := "test"

	if styles.Tag.Render(testText) == "" {
		t.Error("Tag style should render text")
	}
	if styles.PriorityHigh.Render(testText) == "" {
		t.Error("PriorityHigh style should render text")
	}
	if styles.PriorityMedium.Render(testText) == "" {
		t.Error("PriorityMedium style should render text")
	}
	if styles.PriorityLow.Render(testText) == "" {
		t.Error("PriorityLow style should render text")
	}
	if styles.DueUrgent.Render(testText) == "" {
		t.Error("DueUrgent style should render text")
	}
	if styles.DueSoon.Render(testText) == "" {
		t.Error("DueSoon style should render text")
	}
	if styles.DueFuture.Render(testText) == "" {
		t.Error("DueFuture style should render text")
	}
}

func TestColorsConfig_HasNewFields(t *testing.T) {
	// Test that themes have the new color fields
	theme, ok := GetBuiltinTheme("tokyo-night")
	if !ok {
		t.Fatal("tokyo-night theme should exist")
	}

	// New fields should be set (not empty)
	if theme.Tag == "" {
		t.Error("Theme should have Tag color")
	}
	if theme.PriorityHigh == "" {
		t.Error("Theme should have PriorityHigh color")
	}
	if theme.PriorityMedium == "" {
		t.Error("Theme should have PriorityMedium color")
	}
	if theme.PriorityLow == "" {
		t.Error("Theme should have PriorityLow color")
	}
	if theme.DueUrgent == "" {
		t.Error("Theme should have DueUrgent color")
	}
	if theme.DueSoon == "" {
		t.Error("Theme should have DueSoon color")
	}
	if theme.DueFuture == "" {
		t.Error("Theme should have DueFuture color")
	}
}

func TestAllBuiltinThemes_HaveNewColors(t *testing.T) {
	themeNames := GetBuiltinThemeNames()

	for _, name := range themeNames {
		t.Run(name, func(t *testing.T) {
			theme, ok := GetBuiltinTheme(name)
			if !ok {
				t.Fatalf("Theme %q should exist", name)
			}

			// All themes should have the new color fields
			if theme.Tag == "" {
				t.Errorf("Theme %q missing Tag color", name)
			}
			if theme.PriorityHigh == "" {
				t.Errorf("Theme %q missing PriorityHigh color", name)
			}
			if theme.DueUrgent == "" {
				t.Errorf("Theme %q missing DueUrgent color", name)
			}
		})
	}
}

// Tests for resolveFilePath (~ expansion and relative path resolution)

func TestResolveFilePath_TildeExpansion(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	result := resolveFilePath("~/todos.md")
	expected := filepath.Join(home, "todos.md")

	if result != expected {
		t.Errorf("resolveFilePath(\"~/todos.md\") = %q, want %q", result, expected)
	}
}

func TestResolveFilePath_TildeWithSubdirectory(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	result := resolveFilePath("~/notes/work/todos.md")
	expected := filepath.Join(home, "notes", "work", "todos.md")

	if result != expected {
		t.Errorf("resolveFilePath(\"~/notes/work/todos.md\") = %q, want %q", result, expected)
	}
}

func TestResolveFilePath_RelativePath(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Skip("Cannot get current working directory")
	}

	result := resolveFilePath("todo.md")
	expected := filepath.Join(cwd, "todo.md")

	if result != expected {
		t.Errorf("resolveFilePath(\"todo.md\") = %q, want %q", result, expected)
	}
}

func TestResolveFilePath_AbsolutePathUnchanged(t *testing.T) {
	absPath := filepath.Join(t.TempDir(), "my-todos.md")
	result := resolveFilePath(absPath)

	if result != absPath {
		t.Errorf("resolveFilePath(%q) = %q, want %q (absolute paths should be unchanged)", absPath, result, absPath)
	}
}

func TestResolveFilePath_CustomFilename(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Skip("Cannot get current working directory")
	}

	// Test with different filenames users might configure
	testCases := []string{"TODO.md", "TASKS.md", "todos.md", "checklist.md"}

	for _, filename := range testCases {
		t.Run(filename, func(t *testing.T) {
			result := resolveFilePath(filename)
			expected := filepath.Join(cwd, filename)

			if result != expected {
				t.Errorf("resolveFilePath(%q) = %q, want %q", filename, result, expected)
			}
		})
	}
}

func TestSavedViewsConfigAndManualSaveAlias(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	configDir := filepath.Join(dir, "tdx")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(configDir, "config.toml")
	if err := os.WriteFile(path, []byte("[defaults]\nread_only = true\nmanual_save = false\n[views]\nrestore = true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := LoadConfig()
	if cfg.Defaults.ReadOnly || !cfg.Views.Restore {
		t.Fatalf("aliases/settings not applied: %+v", cfg)
	}
	if err := SaveTheme("tokyo-night"); err != nil {
		t.Fatal(err)
	}
	cfg = LoadConfig()
	if cfg.Defaults.ManualSave == nil || *cfg.Defaults.ManualSave || !cfg.Views.Restore {
		t.Fatal("theme change dropped workflow settings")
	}
}

// isolateHome points every home and config directory lookup at fresh temp
// directories so tests never read or write a developer's real config. It
// returns the stand-ins for the home directory and os.UserConfigDir.
func isolateHome(t *testing.T) (home, userConfig string) {
	t.Helper()
	root := t.TempDir()
	home = filepath.Join(root, "home")
	userConfig = filepath.Join(root, "user-config")
	for _, dir := range []string{home, userConfig} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("APPDATA", userConfig)
	t.Setenv("XDG_CONFIG_HOME", "")
	orig := userConfigDir
	userConfigDir = func() (string, error) { return userConfig, nil }
	t.Cleanup(func() { userConfigDir = orig })
	return home, userConfig
}

func writeConfig(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

func decodeRaw(t *testing.T, path string) map[string]any {
	t.Helper()
	raw := map[string]any{}
	if _, err := toml.DecodeFile(path, &raw); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return raw
}

func TestSaveTheme_KeepsUnrelatedSettings(t *testing.T) {
	home, _ := isolateHome(t)
	path := filepath.Join(home, ".config", "tdx", "config.toml")
	writeConfig(t, path, `[theme]
name = "nord"

[versioning]
max_versions = 5

[colors]
Base = "#ffffff"
`, 0o644)

	if err := SaveTheme("dracula"); err != nil {
		t.Fatalf("SaveTheme: %v", err)
	}

	raw := decodeRaw(t, path)
	if got := raw["theme"].(map[string]any)["name"]; got != "dracula" {
		t.Errorf("theme.name = %v, want dracula", got)
	}
	if got := raw["versioning"].(map[string]any)["max_versions"]; got != int64(5) {
		t.Errorf("versioning.max_versions = %v, want 5", got)
	}
	if got := raw["colors"].(map[string]any)["Base"]; got != "#ffffff" {
		t.Errorf("colors.Base = %v, want #ffffff", got)
	}
	if _, ok := raw["defaults"]; ok {
		content, _ := os.ReadFile(path)
		t.Errorf("SaveTheme added a [defaults] table:\n%s", content)
	}

	cfg := LoadConfig()
	if cfg.Versioning.MaxVersions != 5 {
		t.Errorf("MaxVersions = %d, want 5", cfg.Versioning.MaxVersions)
	}
	if !cfg.Defaults.WordWrap || cfg.Defaults.File != "todo.md" {
		t.Errorf("defaults changed after theme save: %+v", cfg.Defaults)
	}
}

func TestSaveTheme_RefusesUnparseableConfig(t *testing.T) {
	home, _ := isolateHome(t)
	dir := filepath.Join(home, ".config", "tdx")
	path := filepath.Join(dir, "config.toml")
	broken := "# my settings\n[theme\nname = \"nord\"\n[versioning]\nmax_versions = 5\n"
	writeConfig(t, path, broken, 0o644)

	err := SaveTheme("dracula")
	if err == nil {
		t.Fatal("SaveTheme succeeded on a config that does not parse")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error should name the config path, got %v", err)
	}
	content, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(content) != broken {
		t.Errorf("config changed:\n%s", content)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("expected only config.toml in %s, found %d entries", dir, len(entries))
	}
}

func TestSaveTheme_WritesToTheConfigLoadConfigRead(t *testing.T) {
	home, userConfig := isolateHome(t)
	path := filepath.Join(userConfig, "tdx", "config.toml")
	writeConfig(t, path, "[theme]\nname = \"nord\"\n\n[versioning]\nmax_versions = 5\n", 0o644)

	if err := SaveTheme("dracula"); err != nil {
		t.Fatalf("SaveTheme: %v", err)
	}

	shadow := filepath.Join(home, ".config", "tdx", "config.toml")
	if _, err := os.Stat(shadow); !os.IsNotExist(err) {
		t.Errorf("SaveTheme created %s, which would shadow %s", shadow, path)
	}
	if got := decodeRaw(t, path)["theme"].(map[string]any)["name"]; got != "dracula" {
		t.Errorf("theme.name in %s = %v, want dracula", path, got)
	}
	cfg := LoadConfig()
	if cfg.Theme.Name != "dracula" || cfg.Versioning.MaxVersions != 5 {
		t.Errorf("LoadConfig after save: theme %q, max_versions %d", cfg.Theme.Name, cfg.Versioning.MaxVersions)
	}
}

func TestSaveTheme_CreatesConfigWhenNoneExists(t *testing.T) {
	home, userConfig := isolateHome(t)

	if err := SaveTheme("dracula"); err != nil {
		t.Fatalf("SaveTheme: %v", err)
	}

	path := filepath.Join(home, ".config", "tdx", "config.toml")
	raw := decodeRaw(t, path)
	if got := raw["theme"].(map[string]any)["name"]; got != "dracula" {
		t.Errorf("theme.name = %v, want dracula", got)
	}
	if len(raw) != 1 {
		t.Errorf("new config should hold only [theme], got %v", raw)
	}
	if _, err := os.Stat(filepath.Join(userConfig, "tdx")); !os.IsNotExist(err) {
		t.Error("SaveTheme should not create the OS-specific config directory")
	}
}

func TestSaveTheme_KeepsFileModeAndSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file modes and symlinks differ on Windows")
	}
	home, _ := isolateHome(t)
	target := filepath.Join(home, "dotfiles", "tdx.toml")
	writeConfig(t, target, "[theme]\nname = \"nord\"\n", 0o600)
	link := filepath.Join(home, ".config", "tdx", "config.toml")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	if err := SaveTheme("dracula"); err != nil {
		t.Fatalf("SaveTheme: %v", err)
	}

	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("config symlink was replaced: %v", err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want 0600", info.Mode().Perm())
	}
	if got := decodeRaw(t, target)["theme"].(map[string]any)["name"]; got != "dracula" {
		t.Errorf("theme.name = %v, want dracula", got)
	}
}

func TestLoadConfig_WarnsWhenConfigDoesNotParse(t *testing.T) {
	home, _ := isolateHome(t)
	path := filepath.Join(home, ".config", "tdx", "config.toml")
	writeConfig(t, path, "[theme\nname = \"nord\"\n", 0o644)
	var warnings bytes.Buffer
	orig := configWarnings
	configWarnings = &warnings
	t.Cleanup(func() { configWarnings = orig })

	cfg := LoadConfig()

	if cfg.Theme.Name != "tokyo-night" {
		t.Errorf("broken config should fall back to defaults, got theme %q", cfg.Theme.Name)
	}
	got := warnings.String()
	if strings.Count(got, "\n") != 1 || !strings.HasPrefix(got, "tdx: warning:") {
		t.Errorf("expected one warning line, got %q", got)
	}
	if !strings.Contains(got, path) {
		t.Errorf("warning should name %s, got %q", path, got)
	}
	if !strings.Contains(got, "toml: line 2") {
		t.Errorf("warning should include the parse error, got %q", got)
	}
}

func TestLoadConfig_NoWarningForValidOrMissingConfig(t *testing.T) {
	home, _ := isolateHome(t)
	var warnings bytes.Buffer
	orig := configWarnings
	configWarnings = &warnings
	t.Cleanup(func() { configWarnings = orig })

	LoadConfig()
	writeConfig(t, filepath.Join(home, ".config", "tdx", "config.toml"), "[theme]\nname = \"nord\"\n", 0o644)
	LoadConfig()

	if warnings.Len() != 0 {
		t.Errorf("unexpected warning: %q", warnings.String())
	}
}

// Round-2 council cases for SaveTheme.
func TestSaveTheme_RefusesConfigLoadConfigRejects(t *testing.T) {
	home, _ := isolateHome(t)
	path := filepath.Join(home, ".config", "tdx", "config.toml")
	const bad = "[defaults]\nword_wrap = \"no\"\n"
	writeConfig(t, path, bad, 0o644)
	if err := SaveTheme("dracula"); err == nil {
		t.Fatal("SaveTheme accepted a config whose values LoadConfig rejects")
	}
	if got, _ := os.ReadFile(path); string(got) != bad {
		t.Fatalf("config changed: %q", got)
	}
}

func TestSaveTheme_UpdatesExistingThemeTableWhateverItsCase(t *testing.T) {
	home, _ := isolateHome(t)
	path := filepath.Join(home, ".config", "tdx", "config.toml")
	writeConfig(t, path, "[Theme]\nName = \"nord\"\n", 0o644)
	if err := SaveTheme("dracula"); err != nil {
		t.Fatal(err)
	}
	raw := decodeRaw(t, path)
	if len(raw) != 1 {
		t.Fatalf("theme table duplicated: %v", raw)
	}
	theme, _ := raw["Theme"].(map[string]any)
	if len(theme) != 1 || theme["Name"] != "dracula" {
		t.Fatalf("theme table = %v, want only Name = dracula", raw)
	}
	if cfg := LoadConfig(); cfg.Theme.Name != "dracula" {
		t.Fatalf("LoadConfig reads theme %q", cfg.Theme.Name)
	}
}

func TestSaveTheme_WritesThroughDanglingSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks differ on Windows")
	}
	home, _ := isolateHome(t)
	target := filepath.Join(home, "dotfiles", "tdx.toml")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, ".config", "tdx", "config.toml")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := SaveTheme("dracula"); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("config symlink was replaced: %v", err)
	}
	if got := decodeRaw(t, target)["theme"].(map[string]any)["name"]; got != "dracula" {
		t.Fatalf("link target theme.name = %v", got)
	}
}

func TestSaveTheme_LeavesOneThemeName(t *testing.T) {
	home, _ := isolateHome(t)
	path := filepath.Join(home, ".config", "tdx", "config.toml")
	writeConfig(t, path, "[theme]\nName = \"nord\"\nname = \"tokyo-night\"\n", 0o644)
	if err := SaveTheme("dracula"); err != nil {
		t.Fatal(err)
	}
	if cfg := LoadConfig(); cfg.Theme.Name != "dracula" {
		t.Fatalf("LoadConfig reads theme %q after saving dracula", cfg.Theme.Name)
	}
	if theme := decodeRaw(t, path)["theme"].(map[string]any); len(theme) != 1 {
		t.Fatalf("theme table keeps several names: %v", theme)
	}
}

func TestSaveTheme_FollowsSymlinkChainToMissingFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks differ on Windows")
	}
	home, _ := isolateHome(t)
	dir := filepath.Join(home, "dotfiles")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, ".config", "tdx", "config.toml")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	middle, target := filepath.Join(dir, "middle.toml"), filepath.Join(dir, "missing.toml")
	if err := os.Symlink("missing.toml", middle); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(middle, link); err != nil {
		t.Fatal(err)
	}
	if err := SaveTheme("dracula"); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(middle); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("intermediate symlink was replaced: %v", err)
	}
	if got := decodeRaw(t, target)["theme"].(map[string]any)["name"]; got != "dracula" {
		t.Fatalf("chain target theme.name = %v", got)
	}
}

// A relative link target is resolved by the OS against the link's real
// directory, which differs from its path when that directory is a symlink.
func TestSaveTheme_ResolvesRelativeLinkThroughSymlinkedDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks differ on Windows")
	}
	home, _ := isolateHome(t)
	dotfiles := filepath.Join(home, "dotfiles")
	real := filepath.Join(dotfiles, "shared", "config.toml")
	writeConfig(t, real, "[theme]\nname = \"nord\"\n", 0o644)
	if err := os.MkdirAll(filepath.Join(dotfiles, "tdx"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../shared/config.toml", filepath.Join(dotfiles, "tdx", "config.toml")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dotfiles, "tdx"), filepath.Join(home, ".config", "tdx")); err != nil {
		t.Fatal(err)
	}
	if err := SaveTheme("dracula"); err != nil {
		t.Fatal(err)
	}
	if got := decodeRaw(t, real)["theme"].(map[string]any)["name"]; got != "dracula" {
		t.Fatalf("real config theme.name = %v", got)
	}
	if cfg := LoadConfig(); cfg.Theme.Name != "dracula" {
		t.Fatalf("LoadConfig reads theme %q", cfg.Theme.Name)
	}
}

func TestSaveTheme_WritesConfigInReadOnlyDirectory(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("directory permissions differ on Windows and for root")
	}
	home, _ := isolateHome(t)
	path := filepath.Join(home, ".config", "tdx", "config.toml")
	writeConfig(t, path, "[theme]\nname = \"nord\"\n", 0o644)
	dir := filepath.Dir(path)
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	if err := SaveTheme("dracula"); err != nil {
		t.Fatal(err)
	}
	if cfg := LoadConfig(); cfg.Theme.Name != "dracula" {
		t.Fatalf("LoadConfig reads theme %q", cfg.Theme.Name)
	}
}

func TestSaveTheme_RefusesSymlinkChainLoadConfigCannotRead(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks differ on Windows")
	}
	home, _ := isolateHome(t)
	dir := filepath.Join(home, "links")
	target := filepath.Join(dir, "target.toml")
	const original = "[theme]\nname = \"nord\"\n"
	writeConfig(t, target, original, 0o644)
	next := target
	for i := 0; i < 65; i++ {
		link := filepath.Join(dir, fmt.Sprintf("link%02d.toml", i))
		if err := os.Symlink(next, link); err != nil {
			t.Fatal(err)
		}
		next = link
	}
	config := filepath.Join(home, ".config", "tdx", "config.toml")
	if err := os.MkdirAll(filepath.Dir(config), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(next, config); err != nil {
		t.Fatal(err)
	}
	_ = SaveTheme("dracula")
	if _, readErr := os.ReadFile(config); readErr == nil {
		t.Skip("this system follows 66 links; nothing to check")
	}
	if got, _ := os.ReadFile(target); string(got) != original {
		t.Fatalf("a chain LoadConfig cannot read was written through: %q", got)
	}
}

func TestSaveTheme_UpdatesEveryThemeTableLoadConfigMayRead(t *testing.T) {
	home, _ := isolateHome(t)
	path := filepath.Join(home, ".config", "tdx", "config.toml")
	writeConfig(t, path, "[Theme]\nName = \"nord\"\n\n[theme]\nname = \"nord\"\n", 0o644)
	for i := 0; i < 20; i++ { // map order varies between runs
		name := fmt.Sprintf("theme-%d", i)
		if err := SaveTheme(name); err != nil {
			t.Fatal(err)
		}
		if cfg := LoadConfig(); cfg.Theme.Name != name {
			t.Fatalf("LoadConfig reads theme %q after saving %q", cfg.Theme.Name, name)
		}
	}
}
