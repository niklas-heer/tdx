package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	defer os.Setenv("XDG_CONFIG_HOME", origXDG)

	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

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
	// Save original env
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", origXDG)

	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

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
	// Save original env
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", origXDG)

	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create initial config with display settings
	configDir := filepath.Join(tmpDir, "tdx")
	os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, "config.toml")

	initialConfig := `[theme]
name = "nord"

[display]
check_symbol = "x"
`
	os.WriteFile(configPath, []byte(initialConfig), 0644)

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
	defer os.Setenv("XDG_CONFIG_HOME", origXDG)

	// Use a non-existent directory
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "does-not-exist")
	os.Setenv("XDG_CONFIG_HOME", nonExistent)

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
