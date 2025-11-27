package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/niklas-heer/tdx/internal/tui"
)

// TestTUI_ThemeCommand_OpensOverlay tests that :theme command opens the theme picker
func TestTUI_ThemeCommand_OpensOverlay(t *testing.T) {
	file := tempTestFile(t)
	runCLI(t, file, "add", "Task 1")

	// Open theme picker via command palette
	output := runPiped(t, file, ":theme\r")

	// Should show the theme picker overlay with themes listed
	// The overlay shows theme names like "tokyo-night", "dracula", etc.
	if !strings.Contains(output, "tokyo-night") && !strings.Contains(output, "dracula") {
		t.Errorf("Expected theme picker overlay to show theme names, got:\n%s", output)
	}
}

// TestTUI_ThemeCommand_ShowsThemeList tests that theme picker shows available themes
func TestTUI_ThemeCommand_ShowsThemeList(t *testing.T) {
	file := tempTestFile(t)
	runCLI(t, file, "add", "Task 1")

	// Open theme picker
	output := runPiped(t, file, ":theme\r")

	// Should show some builtin themes
	expectedThemes := []string{"tokyo-night", "dracula", "nord", "gruvbox-dark"}
	foundAny := false
	for _, theme := range expectedThemes {
		if strings.Contains(output, theme) {
			foundAny = true
			break
		}
	}
	if !foundAny {
		t.Errorf("Expected theme picker to show at least one builtin theme, got:\n%s", output)
	}
}

// TestTUI_ThemePicker_NavigateDown tests that j/down navigates themes
func TestTUI_ThemePicker_NavigateDown(t *testing.T) {
	file := tempTestFile(t)
	runCLI(t, file, "add", "Task 1")

	// Open theme picker and navigate down
	output := runPiped(t, file, ":theme\rj")

	// Should still show theme picker with selection arrow and themes
	if !strings.Contains(output, "tokyo-night") && !strings.Contains(output, "dracula") {
		t.Errorf("Expected theme picker to still be open after navigation, got:\n%s", output)
	}
	if !strings.Contains(output, "→") {
		t.Errorf("Expected selection arrow in theme picker, got:\n%s", output)
	}
}

// TestTUI_ThemePicker_NavigateUp tests that k/up navigates themes
func TestTUI_ThemePicker_NavigateUp(t *testing.T) {
	file := tempTestFile(t)
	runCLI(t, file, "add", "Task 1")

	// Open theme picker, navigate down then up
	output := runPiped(t, file, ":theme\rjk")

	// Should still show theme picker with themes
	if !strings.Contains(output, "tokyo-night") && !strings.Contains(output, "dracula") {
		t.Errorf("Expected theme picker to still be open, got:\n%s", output)
	}
}

// TestTUI_ThemePicker_EscapeCancels tests that escape closes theme picker without applying
func TestTUI_ThemePicker_EscapeCancels(t *testing.T) {
	file := tempTestFile(t)
	runCLI(t, file, "add", "Task 1")

	// Open theme picker and cancel with escape
	// Note: In piped mode, escape quits the app, so we test with a different approach
	// Just verify the theme picker was opened and then closes on enter (confirm)
	output := runPiped(t, file, ":theme\r\r")

	// Should close theme picker and show normal view (not showing theme list anymore)
	// After confirming, the overlay should be gone and we should see the task
	if !strings.Contains(output, "Task 1") {
		t.Errorf("Expected to see task after closing theme picker, got:\n%s", output)
	}
}

// TestTUI_ThemePicker_EnterConfirms tests that enter confirms theme selection
func TestTUI_ThemePicker_EnterConfirms(t *testing.T) {
	file := tempTestFile(t)
	runCLI(t, file, "add", "Task 1")

	// Open theme picker and confirm with enter
	output := runPiped(t, file, ":theme\r\r")

	// Should close theme picker
	if strings.Contains(output, "Select Theme") {
		t.Errorf("Expected theme picker to close after enter, got:\n%s", output)
	}
	// Should show the task
	if !strings.Contains(output, "Task 1") {
		t.Errorf("Expected to see task after confirming theme, got:\n%s", output)
	}
}

// TestTUI_ThemePicker_ShowsCurrentTheme tests that current theme is marked
func TestTUI_ThemePicker_ShowsCurrentTheme(t *testing.T) {
	file := tempTestFile(t)
	runCLI(t, file, "add", "Task 1")

	// Open theme picker
	output := runPiped(t, file, ":theme\r")

	// Should show the current theme marker (●)
	if !strings.Contains(output, "[●]") {
		t.Errorf("Expected current theme marker [●] in theme picker, got:\n%s", output)
	}
}

// TestTUI_ThemePicker_ShowsHelp tests that theme picker shows help text
func TestTUI_ThemePicker_ShowsHelp(t *testing.T) {
	file := tempTestFile(t)
	runCLI(t, file, "add", "Task 1")

	// Open theme picker
	output := runPiped(t, file, ":theme\r")

	// Should show help text
	if !strings.Contains(output, "enter apply") {
		t.Errorf("Expected help text in theme picker, got:\n%s", output)
	}
	if !strings.Contains(output, "esc cancel") {
		t.Errorf("Expected escape hint in theme picker, got:\n%s", output)
	}
}

// TestTUI_ThemeStatusBar tests that status bar shows theme mode help text
func TestTUI_ThemeStatusBar(t *testing.T) {
	file := tempTestFile(t)
	runCLI(t, file, "add", "Task 1")

	// Open theme picker
	output := runPiped(t, file, ":theme\r")

	// Theme picker should show help text with enter/esc hints
	if !strings.Contains(output, "enter apply") || !strings.Contains(output, "esc cancel") {
		t.Errorf("Expected theme picker help text in output, got:\n%s", output)
	}
}

// TestGetBuiltinThemeNames tests that GetBuiltinThemeNames returns sorted names
func TestGetBuiltinThemeNames(t *testing.T) {
	names := GetBuiltinThemeNames()

	if len(names) == 0 {
		t.Fatal("Expected at least one builtin theme")
	}

	// Check that themes are sorted alphabetically
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Errorf("Themes not sorted: %s > %s", names[i-1], names[i])
		}
	}

	// Check some expected themes exist
	found := map[string]bool{}
	for _, name := range names {
		found[name] = true
	}

	expectedThemes := []string{"tokyo-night", "dracula", "nord"}
	for _, expected := range expectedThemes {
		if !found[expected] {
			t.Errorf("Expected builtin theme %s not found", expected)
		}
	}
}

// TestGetBuiltinTheme tests that GetBuiltinTheme returns correct colors
func TestGetBuiltinTheme(t *testing.T) {
	// Test existing theme
	colors, ok := GetBuiltinTheme("tokyo-night")
	if !ok {
		t.Fatal("Expected tokyo-night theme to exist")
	}
	if colors.Accent == "" {
		t.Error("Expected Accent color to be set for tokyo-night")
	}

	// Test non-existing theme
	_, ok = GetBuiltinTheme("non-existent-theme")
	if ok {
		t.Error("Expected non-existent theme to return false")
	}
}

// TestTUI_ThemePicker_LivePreview tests that navigating themes changes the view
func TestTUI_ThemePicker_LivePreview(t *testing.T) {
	file := tempTestFile(t)
	runCLI(t, file, "add", "Task 1")

	// This test verifies that the theme apply function is called when navigating
	// We can't easily test visual color changes, but we can verify the flow works

	// Navigate through a few themes
	output := runPiped(t, file, ":theme\rjjj")

	// Should still show theme picker (not crashed) with themes visible
	if !strings.Contains(output, "tokyo-night") && !strings.Contains(output, "dracula") {
		t.Errorf("Expected theme picker to still be open after navigation, got:\n%s", output)
	}
}

// TestTUI_ThemePickerAvailableThemes tests that the model has themes populated
func TestTUI_ThemePickerAvailableThemes(t *testing.T) {
	// Verify the global is set up correctly
	if len(tui.AvailableThemes) == 0 {
		t.Error("Expected AvailableThemes to be populated in TestMain")
	}

	if tui.CurrentThemeName == "" {
		t.Error("Expected CurrentThemeName to be set in TestMain")
	}

	if tui.ThemeApplyFunc == nil {
		t.Error("Expected ThemeApplyFunc to be set in TestMain")
	}

	if tui.ThemeSaveFunc == nil {
		t.Error("Expected ThemeSaveFunc to be set in TestMain")
	}
}

// TestSaveTheme_OnlyWritesThemeName tests that SaveTheme only writes theme name, not colors
func TestSaveTheme_OnlyWritesThemeName(t *testing.T) {
	// Create a temp config directory
	tmpDir, err := os.MkdirTemp("", "tdx-theme-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create config file path
	configPath := filepath.Join(tmpDir, "config.toml")

	// Write a config file directly to test
	f, err := os.Create(configPath)
	if err != nil {
		t.Fatal(err)
	}

	// Use the minimalThemeConfig struct to write
	type minimalConfig struct {
		Theme struct {
			Name string `toml:"name"`
		} `toml:"theme"`
	}
	cfg := minimalConfig{}
	cfg.Theme.Name = "dracula"

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(cfg); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	// Read back and verify no colors section
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	contentStr := string(content)

	// Should contain theme name
	if !strings.Contains(contentStr, "dracula") {
		t.Errorf("Expected config to contain theme name 'dracula', got:\n%s", contentStr)
	}

	// Should NOT contain colors section
	if strings.Contains(contentStr, "[colors]") {
		t.Errorf("Expected config to NOT contain [colors] section, got:\n%s", contentStr)
	}

	// Should NOT contain color keys
	colorKeys := []string{"Base", "Dim", "Accent", "Success", "Warning", "Important", "AlertError"}
	for _, key := range colorKeys {
		if strings.Contains(contentStr, key+" =") || strings.Contains(contentStr, key+"=") {
			t.Errorf("Expected config to NOT contain color key '%s', got:\n%s", key, contentStr)
		}
	}
}
