package main

import (
	"embed"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/lipgloss"
)

//go:embed themes/*.toml
var themesFS embed.FS

// UserConfig holds user configuration
type UserConfig struct {
	Theme   ThemeConfig   `toml:"theme"`
	Colors  ColorsConfig  // Populated from builtin theme, not from config file
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

	// Tag colors
	Tag string `toml:"Tag"` // hashtag color (#tag)

	// Priority colors (for !p1, !p2, !p3, !p4+)
	PriorityHigh   string `toml:"PriorityHigh"`   // !p1 - critical
	PriorityMedium string `toml:"PriorityMedium"` // !p2 - high
	PriorityLow    string `toml:"PriorityLow"`    // !p3, !p4+ - medium/low

	// Due date colors (for @due(...))
	DueUrgent string `toml:"DueUrgent"` // overdue or due today
	DueSoon   string `toml:"DueSoon"`   // due within 3 days
	DueFuture string `toml:"DueFuture"` // due later
}

// DisplayConfig holds display settings
type DisplayConfig struct {
	MaxVisible   int    `toml:"max_visible"`   // max todos to show (0 = unlimited)
	CheckSymbol  string `toml:"check_symbol"`  // symbol for checked items (default: ✓)
	SelectMarker string `toml:"select_marker"` // symbol for selected item (default: ➜)
}

// loadBuiltinThemes loads themes from embedded TOML files
func loadBuiltinThemes() map[string]ColorsConfig {
	themes := make(map[string]ColorsConfig)

	entries, err := themesFS.ReadDir("themes")
	if err != nil {
		return themes
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		data, err := themesFS.ReadFile("themes/" + entry.Name())
		if err != nil {
			continue
		}

		var config UserConfig
		if _, err := toml.Decode(string(data), &config); err != nil {
			continue
		}

		if config.Theme.Name != "" {
			themes[config.Theme.Name] = config.Colors
		}
	}

	return themes
}

// builtinThemes holds all embedded themes
var builtinThemes = loadBuiltinThemes()

// loadUserThemes loads themes from ~/.config/tdx/themes/ directory
func loadUserThemes() map[string]ColorsConfig {
	themes := make(map[string]ColorsConfig)

	// Get user themes directory
	var themesDir string
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		themesDir = filepath.Join(xdgConfig, "tdx", "themes")
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return themes
		}
		themesDir = filepath.Join(homeDir, ".config", "tdx", "themes")
	}

	// Check if directory exists
	entries, err := os.ReadDir(themesDir)
	if err != nil {
		return themes // Directory doesn't exist or not readable
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process .toml files
		if filepath.Ext(entry.Name()) != ".toml" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(themesDir, entry.Name()))
		if err != nil {
			continue
		}

		var config UserConfig
		if _, err := toml.Decode(string(data), &config); err != nil {
			continue
		}

		if config.Theme.Name != "" {
			themes[config.Theme.Name] = config.Colors
		}
	}

	return themes
}

// allThemes combines builtin and user themes (user themes override builtin)
func getAllThemes() map[string]ColorsConfig {
	themes := make(map[string]ColorsConfig)

	// Start with builtin themes
	for name, colors := range builtinThemes {
		themes[name] = colors
	}

	// Override/add with user themes
	for name, colors := range loadUserThemes() {
		themes[name] = colors
	}

	return themes
}

// GetBuiltinThemeNames returns a sorted list of available theme names (builtin + user)
func GetBuiltinThemeNames() []string {
	allThemes := getAllThemes()
	names := make([]string, 0, len(allThemes))
	for name := range allThemes {
		names = append(names, name)
	}
	// Sort alphabetically for consistent display
	for i := 0; i < len(names)-1; i++ {
		for j := i + 1; j < len(names); j++ {
			if names[i] > names[j] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	return names
}

// GetBuiltinTheme returns the colors for a theme by name (checks user themes first, then builtin)
func GetBuiltinTheme(name string) (ColorsConfig, bool) {
	// Check user themes first (allows overriding builtin themes)
	userThemes := loadUserThemes()
	if colors, ok := userThemes[name]; ok {
		return colors, true
	}
	// Fall back to builtin themes
	colors, ok := builtinThemes[name]
	return colors, ok
}

// DefaultConfig returns Tokyo Night theme as default
func DefaultConfig() *UserConfig {
	return &UserConfig{
		Theme: ThemeConfig{
			Name: "tokyo-night",
		},
		Colors: builtinThemes["tokyo-night"],
		Display: DisplayConfig{
			MaxVisible:   0,   // unlimited by default
			CheckSymbol:  "✓", // default check symbol
			SelectMarker: "➜", // default select marker
		},
	}
}

// LoadConfig loads config from ~/.config/tdx/config.toml or returns defaults
func LoadConfig() *UserConfig {
	defaults := DefaultConfig()
	config := &UserConfig{}

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
		return defaults
	}

	// Load and parse config (theme name and display settings only)
	if _, err := toml.DecodeFile(configPath, config); err != nil {
		return defaults
	}

	// Apply defaults for any missing values
	if config.Theme.Name == "" {
		config.Theme.Name = defaults.Theme.Name
	}
	if config.Display.CheckSymbol == "" {
		config.Display.CheckSymbol = defaults.Display.CheckSymbol
	}
	if config.Display.SelectMarker == "" {
		config.Display.SelectMarker = defaults.Display.SelectMarker
	}
	// MaxVisible 0 is valid (unlimited), so we don't override it

	// Apply colors from theme (user themes override builtin)
	if config.Theme.Name != "" {
		if theme, ok := GetBuiltinTheme(config.Theme.Name); ok {
			config.Colors = theme
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

	// Tag style
	Tag lipgloss.Style

	// Priority styles
	PriorityHigh   lipgloss.Style
	PriorityMedium lipgloss.Style
	PriorityLow    lipgloss.Style

	// Due date styles
	DueUrgent lipgloss.Style
	DueSoon   lipgloss.Style
	DueFuture lipgloss.Style
}

// NewStyles creates lipgloss styles from config colors
func NewStyles(config *UserConfig) *Styles {
	// Helper to get color with fallback
	colorOrFallback := func(color, fallback string) string {
		if color != "" {
			return color
		}
		return fallback
	}

	// Fallback colors for new fields (Tokyo Night inspired defaults)
	tagColor := colorOrFallback(config.Colors.Tag, config.Colors.Warning)        // Yellow/orange for tags
	priorityHigh := colorOrFallback(config.Colors.PriorityHigh, "#f7768e")       // Red
	priorityMedium := colorOrFallback(config.Colors.PriorityMedium, "#bb9af7")   // Purple/magenta
	priorityLow := colorOrFallback(config.Colors.PriorityLow, config.Colors.Dim) // Dim
	dueUrgent := colorOrFallback(config.Colors.DueUrgent, "#7dcfff")             // Cyan/blue
	dueSoon := colorOrFallback(config.Colors.DueSoon, "#7aa2f7")                 // Blue
	dueFuture := colorOrFallback(config.Colors.DueFuture, config.Colors.Dim)     // Dim

	return &Styles{
		Base:      lipgloss.NewStyle().Foreground(lipgloss.Color(config.Colors.Base)),
		Dim:       lipgloss.NewStyle().Foreground(lipgloss.Color(config.Colors.Dim)),
		Accent:    lipgloss.NewStyle().Foreground(lipgloss.Color(config.Colors.Accent)),
		Success:   lipgloss.NewStyle().Foreground(lipgloss.Color(config.Colors.Success)),
		Warning:   lipgloss.NewStyle().Foreground(lipgloss.Color(config.Colors.Warning)),
		Important: lipgloss.NewStyle().Foreground(lipgloss.Color(config.Colors.Important)),
		Error:     lipgloss.NewStyle().Foreground(lipgloss.Color(config.Colors.AlertError)),
		Code:      lipgloss.NewStyle().Background(lipgloss.Color(config.Colors.Dim)).Foreground(lipgloss.Color(config.Colors.Base)),

		// New styles for tags, priorities, and due dates
		Tag:            lipgloss.NewStyle().Foreground(lipgloss.Color(tagColor)),
		PriorityHigh:   lipgloss.NewStyle().Foreground(lipgloss.Color(priorityHigh)).Bold(true),
		PriorityMedium: lipgloss.NewStyle().Foreground(lipgloss.Color(priorityMedium)),
		PriorityLow:    lipgloss.NewStyle().Foreground(lipgloss.Color(priorityLow)),
		DueUrgent:      lipgloss.NewStyle().Foreground(lipgloss.Color(dueUrgent)).Bold(true),
		DueSoon:        lipgloss.NewStyle().Foreground(lipgloss.Color(dueSoon)),
		DueFuture:      lipgloss.NewStyle().Foreground(lipgloss.Color(dueFuture)),
	}
}

// NewStyleFuncs creates StyleFuncsType from Styles
func NewStyleFuncs(styles *Styles) *StyleFuncsType {
	return &StyleFuncsType{
		Magenta: func(s string) string { return styles.Important.Render(s) },
		Cyan:    func(s string) string { return styles.Accent.Render(s) },
		Dim:     func(s string) string { return styles.Dim.Render(s) },
		Green:   func(s string) string { return styles.Success.Render(s) },
		Yellow:  func(s string) string { return styles.Warning.Render(s) },
		Code:    func(s string) string { return styles.Code.Render(s) },

		// New style functions for tags, priorities, and due dates
		Tag:            func(s string) string { return styles.Tag.Render(s) },
		PriorityHigh:   func(s string) string { return styles.PriorityHigh.Render(s) },
		PriorityMedium: func(s string) string { return styles.PriorityMedium.Render(s) },
		PriorityLow:    func(s string) string { return styles.PriorityLow.Render(s) },
		DueUrgent:      func(s string) string { return styles.DueUrgent.Render(s) },
		DueSoon:        func(s string) string { return styles.DueSoon.Render(s) },
		DueFuture:      func(s string) string { return styles.DueFuture.Render(s) },
	}
}

// StyleFuncsType holds style functions for rendering (duplicated here to avoid import cycle)
type StyleFuncsType struct {
	Magenta func(string) string
	Cyan    func(string) string
	Dim     func(string) string
	Green   func(string) string
	Yellow  func(string) string
	Code    func(string) string

	// New style functions for tags, priorities, and due dates
	Tag            func(string) string
	PriorityHigh   func(string) string
	PriorityMedium func(string) string
	PriorityLow    func(string) string
	DueUrgent      func(string) string
	DueSoon        func(string) string
	DueFuture      func(string) string
}

// getConfigPath returns the path to the TOML config file, creating directory if needed
func getConfigPath() (string, error) {
	var configDir string

	// Check XDG_CONFIG_HOME first
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		configDir = filepath.Join(xdgConfig, "tdx")
	} else {
		// Fall back to ~/.config/tdx
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(homeDir, ".config", "tdx")
	}

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(configDir, "config.toml"), nil
}

// minimalThemeConfig is used for saving only the theme name without colors
type minimalThemeConfig struct {
	Theme struct {
		Name string `toml:"name"`
	} `toml:"theme"`
	Display *DisplayConfig `toml:"display,omitempty"`
}

// SaveTheme saves the theme name to the config file
// Only saves the theme name, not colors, so the builtin theme colors are used
func SaveTheme(themeName string) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	// Load existing config to preserve display settings
	existingConfig := &UserConfig{}
	if _, err := os.Stat(configPath); err == nil {
		// File exists, load it to preserve display settings
		_, _ = toml.DecodeFile(configPath, existingConfig)
	}

	// Create minimal config with only theme name and display settings
	minConfig := &minimalThemeConfig{}
	minConfig.Theme.Name = themeName

	// Preserve display settings if they were customized
	if existingConfig.Display.MaxVisible != 0 ||
		existingConfig.Display.CheckSymbol != "" ||
		existingConfig.Display.SelectMarker != "" {
		minConfig.Display = &existingConfig.Display
	}

	// Write config to file
	f, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	encoder := toml.NewEncoder(f)
	return encoder.Encode(minConfig)
}
