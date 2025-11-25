package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the global configuration for tdx
type Config struct {
	FilterDone   *bool `yaml:"filter-done,omitempty"`   // Filter out completed tasks
	MaxVisible   *int  `yaml:"max-visible,omitempty"`   // Maximum visible tasks
	ShowHeadings *bool `yaml:"show-headings,omitempty"` // Show headings between tasks
	ReadOnly     *bool `yaml:"read-only,omitempty"`     // Open in read-only mode
	WordWrap     *bool `yaml:"word-wrap,omitempty"`     // Enable word wrapping
}

// GetConfigPath returns the path to the config file
// Follows XDG Base Directory specification on Unix-like systems
func GetConfigPath() (string, error) {
	var configDir string

	// Check XDG_CONFIG_HOME first
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		configDir = xdgConfig
	} else {
		// Fall back to ~/.config
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(homeDir, ".config")
	}

	return filepath.Join(configDir, "tdx", "config.yaml"), nil
}

// Load loads the global config file
// Returns an empty config if the file doesn't exist
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return &Config{}, nil // Return empty config on error
	}

	// If config doesn't exist, return empty config (not an error)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &Config{}, nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return &Config{}, nil // Return empty config on error
	}

	// Parse YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return &Config{}, err // Return error on parse failure
	}

	return &cfg, nil
}

// GetBool returns the value of a bool pointer or the default if nil
func (c *Config) GetBool(field string, defaultValue bool) bool {
	switch field {
	case "filter-done":
		if c.FilterDone != nil {
			return *c.FilterDone
		}
	case "show-headings":
		if c.ShowHeadings != nil {
			return *c.ShowHeadings
		}
	case "read-only":
		if c.ReadOnly != nil {
			return *c.ReadOnly
		}
	case "word-wrap":
		if c.WordWrap != nil {
			return *c.WordWrap
		}
	}
	return defaultValue
}

// GetInt returns the value of an int pointer or the default if nil
func (c *Config) GetInt(field string, defaultValue int) int {
	switch field {
	case "max-visible":
		if c.MaxVisible != nil {
			return *c.MaxVisible
		}
	}
	return defaultValue
}
