package main

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/lipgloss"
)

// UserConfig holds user configuration
type UserConfig struct {
	Theme   ThemeConfig   `toml:"theme"`
	Colors  ColorsConfig  `toml:"colors"`
	Display DisplayConfig `toml:"display"`
}

// ThemeConfig holds theme metadata
type ThemeConfig struct {
	Name string `toml:"name"`
}

// ColorsConfig holds all color definitions using hex codes
type ColorsConfig struct {
	// Core colors
	Base       string `toml:"Base"`       // default foreground
	Dim        string `toml:"Dim"`        // muted text
	Accent     string `toml:"Accent"`     // highlights, selections
	Success    string `toml:"Success"`    // completed items, matches
	Warning    string `toml:"Warning"`    // move mode
	Important  string `toml:"Important"`  // checked items
	AlertError string `toml:"AlertError"` // errors
}

// DisplayConfig holds display settings
type DisplayConfig struct {
	MaxVisible int `toml:"max_visible"` // max todos to show (0 = unlimited)
}

// DefaultConfig returns Tokyo Night theme as default
func DefaultConfig() *UserConfig {
	return &UserConfig{
		Theme: ThemeConfig{
			Name: "tokyo-night",
		},
		Colors: ColorsConfig{
			Base:       "#c0caf5",
			Dim:        "#565f89",
			Accent:     "#7aa2f7",
			Success:    "#c3e88d",
			Warning:    "#ff9e64",
			Important:  "#bb9af7",
			AlertError: "#ff007c",
		},
		Display: DisplayConfig{
			MaxVisible: 0, // unlimited by default
		},
	}
}

// LoadConfig loads config from ~/.config/tdx/config.toml or returns defaults
func LoadConfig() *UserConfig {
	config := DefaultConfig()

	// Try multiple config locations
	var configPaths []string

	// First try XDG style: ~/.config/tdx/config.toml
	if home, err := os.UserHomeDir(); err == nil {
		configPaths = append(configPaths, filepath.Join(home, ".config", "tdx", "config.toml"))
	}

	// Then try OS-specific config dir
	if configDir, err := os.UserConfigDir(); err == nil {
		configPaths = append(configPaths, filepath.Join(configDir, "tdx", "config.toml"))
	}

	// Find first existing config file
	var configPath string
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			configPath = path
			break
		}
	}

	if configPath == "" {
		return config
	}

	// Load and parse config
	_, err := toml.DecodeFile(configPath, config)
	if err != nil {
		return DefaultConfig()
	}

	return config
}

// Styles holds the lipgloss styles derived from config
type Styles struct {
	Base      lipgloss.Style
	Dim       lipgloss.Style
	Accent    lipgloss.Style
	Success   lipgloss.Style
	Warning   lipgloss.Style
	Important lipgloss.Style
	Error     lipgloss.Style
	Code      lipgloss.Style
}

// NewStyles creates lipgloss styles from config colors
func NewStyles(config *UserConfig) *Styles {
	return &Styles{
		Base:      lipgloss.NewStyle().Foreground(lipgloss.Color(config.Colors.Base)),
		Dim:       lipgloss.NewStyle().Foreground(lipgloss.Color(config.Colors.Dim)),
		Accent:    lipgloss.NewStyle().Foreground(lipgloss.Color(config.Colors.Accent)),
		Success:   lipgloss.NewStyle().Foreground(lipgloss.Color(config.Colors.Success)),
		Warning:   lipgloss.NewStyle().Foreground(lipgloss.Color(config.Colors.Warning)),
		Important: lipgloss.NewStyle().Foreground(lipgloss.Color(config.Colors.Important)),
		Error:     lipgloss.NewStyle().Foreground(lipgloss.Color(config.Colors.AlertError)),
		Code:      lipgloss.NewStyle().Background(lipgloss.Color(config.Colors.Dim)).Foreground(lipgloss.Color(config.Colors.Base)),
	}
}
