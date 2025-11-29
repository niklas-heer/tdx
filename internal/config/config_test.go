package config

import (
	"os"
	"path/filepath"
	"testing"
)

// Helper to create bool pointer
func boolPtr(b bool) *bool {
	return &b
}

// Helper to create int pointer
func intPtr(i int) *int {
	return &i
}

func TestConfig_GetBool(t *testing.T) {
	tests := []struct {
		name       string
		config     *Config
		field      string
		defaultVal bool
		expected   bool
	}{
		{
			name:       "empty config returns default",
			config:     &Config{},
			field:      "filter-done",
			defaultVal: true,
			expected:   true,
		},
		{
			name:       "unknown field returns default",
			config:     &Config{FilterDone: boolPtr(true)},
			field:      "unknown-field",
			defaultVal: false,
			expected:   false,
		},
		{
			name:       "filter-done true",
			config:     &Config{FilterDone: boolPtr(true)},
			field:      "filter-done",
			defaultVal: false,
			expected:   true,
		},
		{
			name:       "filter-done false",
			config:     &Config{FilterDone: boolPtr(false)},
			field:      "filter-done",
			defaultVal: true,
			expected:   false,
		},
		{
			name:       "show-headings true",
			config:     &Config{ShowHeadings: boolPtr(true)},
			field:      "show-headings",
			defaultVal: false,
			expected:   true,
		},
		{
			name:       "read-only true",
			config:     &Config{ReadOnly: boolPtr(true)},
			field:      "read-only",
			defaultVal: false,
			expected:   true,
		},
		{
			name:       "word-wrap true",
			config:     &Config{WordWrap: boolPtr(true)},
			field:      "word-wrap",
			defaultVal: false,
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetBool(tt.field, tt.defaultVal)
			if got != tt.expected {
				t.Errorf("GetBool(%q, %v) = %v, want %v", tt.field, tt.defaultVal, got, tt.expected)
			}
		})
	}
}

func TestConfig_GetInt(t *testing.T) {
	tests := []struct {
		name       string
		config     *Config
		field      string
		defaultVal int
		expected   int
	}{
		{
			name:       "empty config returns default",
			config:     &Config{},
			field:      "max-visible",
			defaultVal: 10,
			expected:   10,
		},
		{
			name:       "unknown field returns default",
			config:     &Config{MaxVisible: intPtr(50)},
			field:      "unknown-field",
			defaultVal: 20,
			expected:   20,
		},
		{
			name:       "max-visible set",
			config:     &Config{MaxVisible: intPtr(42)},
			field:      "max-visible",
			defaultVal: 10,
			expected:   42,
		},
		{
			name:       "max-visible zero",
			config:     &Config{MaxVisible: intPtr(0)},
			field:      "max-visible",
			defaultVal: 10,
			expected:   0,
		},
		{
			name:       "max-recent-files set",
			config:     &Config{MaxRecentFiles: intPtr(25)},
			field:      "max-recent-files",
			defaultVal: 10,
			expected:   25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetInt(tt.field, tt.defaultVal)
			if got != tt.expected {
				t.Errorf("GetInt(%q, %v) = %v, want %v", tt.field, tt.defaultVal, got, tt.expected)
			}
		})
	}
}

func TestGetConfigDir(t *testing.T) {
	// Test with XDG_CONFIG_HOME set
	tmpDir := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	configDir, err := GetConfigDir()
	if err != nil {
		t.Fatalf("GetConfigDir failed: %v", err)
	}

	expected := filepath.Join(tmpDir, "tdx")
	if configDir != expected {
		t.Errorf("GetConfigDir() = %q, want %q", configDir, expected)
	}
}

func TestGetConfigPath(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	configPath, err := GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath failed: %v", err)
	}

	expected := filepath.Join(tmpDir, "tdx", "config.yaml")
	if configPath != expected {
		t.Errorf("GetConfigPath() = %q, want %q", configPath, expected)
	}
}

func TestLoad_NoConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}
	// All fields should be nil (empty config)
	if cfg.FilterDone != nil {
		t.Error("Expected FilterDone to be nil")
	}
}

func TestLoad_ValidConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	// Create config directory and file
	configDir := filepath.Join(tmpDir, "tdx")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configContent := `filter-done: true
max-visible: 50
show-headings: false
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.FilterDone == nil || *cfg.FilterDone != true {
		t.Error("Expected FilterDone to be true")
	}
	if cfg.MaxVisible == nil || *cfg.MaxVisible != 50 {
		t.Error("Expected MaxVisible to be 50")
	}
	if cfg.ShowHeadings == nil || *cfg.ShowHeadings != false {
		t.Error("Expected ShowHeadings to be false")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	// Create config directory and file with invalid YAML
	configDir := filepath.Join(tmpDir, "tdx")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configContent := `invalid yaml: [not: closed`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}
