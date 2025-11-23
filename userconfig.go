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

// Built-in themes
var builtinThemes = map[string]ColorsConfig{
	"tokyo-night": {
		Base:       "#c0caf5",
		Dim:        "#565f89",
		Accent:     "#7aa2f7",
		Success:    "#c3e88d",
		Warning:    "#ff9e64",
		Important:  "#bb9af7",
		AlertError: "#ff007c",
	},
	"gruvbox-dark": {
		Base:       "#ebdbb2",
		Dim:        "#928374",
		Accent:     "#83a598",
		Success:    "#b8bb26",
		Warning:    "#fe8019",
		Important:  "#d3869b",
		AlertError: "#fb4934",
	},
	"catppuccin-mocha": {
		Base:       "#cdd6f4",
		Dim:        "#6c7086",
		Accent:     "#89b4fa",
		Success:    "#a6e3a1",
		Warning:    "#fab387",
		Important:  "#cba6f7",
		AlertError: "#f38ba8",
	},
	"nord": {
		Base:       "#eceff4",
		Dim:        "#4c566a",
		Accent:     "#88c0d0",
		Success:    "#a3be8c",
		Warning:    "#ebcb8b",
		Important:  "#b48ead",
		AlertError: "#bf616a",
	},
	"dracula": {
		Base:       "#f8f8f2",
		Dim:        "#6272a4",
		Accent:     "#8be9fd",
		Success:    "#50fa7b",
		Warning:    "#ffb86c",
		Important:  "#ff79c6",
		AlertError: "#ff5555",
	},
}

// DefaultConfig returns Tokyo Night theme as default
func DefaultConfig() *UserConfig {
	return &UserConfig{
		Theme: ThemeConfig{
			Name: "tokyo-night",
		},
		Colors: builtinThemes["tokyo-night"],
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
	meta, err := toml.DecodeFile(configPath, config)
	if err != nil {
		return DefaultConfig()
	}

	// Apply builtin theme if name is set and no custom colors defined
	if config.Theme.Name != "" {
		if theme, ok := builtinThemes[config.Theme.Name]; ok {
			// Only apply theme colors if [colors] section wasn't in config
			if !meta.IsDefined("colors") {
				config.Colors = theme
			}
		}
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
