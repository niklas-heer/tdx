package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// RecentFile represents a recently opened file with tracking metadata
type RecentFile struct {
	Path          string    `json:"path"`            // Absolute path to the file
	LastAccessed  time.Time `json:"last_accessed"`   // Last time this file was opened
	AccessCount   int       `json:"access_count"`    // Number of times opened
	LastCursorPos int       `json:"last_cursor_pos"` // Last cursor position in the file
	ContentHash   string    `json:"content_hash"`    // SHA256 hash of file content when cursor was saved
	LastModified  time.Time `json:"last_modified"`   // Last modification time when accessed
}

// RecentFiles manages the list of recently opened files
type RecentFiles struct {
	Files     []RecentFile `json:"files"`
	MaxRecent int          `json:"max_recent"`
}

// getConfigDir is a variable so it can be mocked in tests
var getConfigDir = GetConfigDir

// originalGetConfigDir stores the original function for reset
var originalGetConfigDir = GetConfigDir

// SetConfigDirForTesting allows tests to override the config directory
func SetConfigDirForTesting(dir string) {
	getConfigDir = func() (string, error) {
		return dir, nil
	}
}

// ResetConfigDirForTesting restores the original config directory function
func ResetConfigDirForTesting() {
	getConfigDir = originalGetConfigDir
}

// DefaultMaxRecent is the default maximum number of recent files to track
const DefaultMaxRecent = 20

// computeFileHash computes SHA256 hash of file content
func computeFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// SaveRecentFile adds or updates a file in the recent files list
func SaveRecentFile(filePath string, cursorPos int) error {
	// Get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}

	// Check if file exists
	fileInfo, err := os.Stat(absPath)
	if err != nil {
		return err
	}

	// Compute content hash
	contentHash, err := computeFileHash(absPath)
	if err != nil {
		// Non-fatal: continue without hash
		contentHash = ""
	}

	// Load existing recent files
	recent, err := LoadRecentFiles()
	if err != nil {
		// If file doesn't exist, create new
		recent = &RecentFiles{
			Files:     []RecentFile{},
			MaxRecent: DefaultMaxRecent,
		}
	}

	// Find if file already exists in list
	found := false
	for i := range recent.Files {
		if recent.Files[i].Path == absPath {
			// Update existing entry
			recent.Files[i].LastAccessed = time.Now()
			recent.Files[i].AccessCount++
			recent.Files[i].LastCursorPos = cursorPos
			recent.Files[i].ContentHash = contentHash
			recent.Files[i].LastModified = fileInfo.ModTime()
			found = true
			break
		}
	}

	if !found {
		// Add new entry
		recent.Files = append(recent.Files, RecentFile{
			Path:          absPath,
			LastAccessed:  time.Now(),
			AccessCount:   1,
			LastCursorPos: cursorPos,
			ContentHash:   contentHash,
			LastModified:  fileInfo.ModTime(),
		})
	}

	// Clean up non-existent files
	recent.CleanupMissing()

	// Sort and trim to max size
	recent.SortByScore()
	if len(recent.Files) > recent.MaxRecent {
		recent.Files = recent.Files[:recent.MaxRecent]
	}

	return recent.Save()
}

// LoadRecentFiles loads the recent files list from disk
func LoadRecentFiles() (*RecentFiles, error) {
	path, err := GetRecentFilesPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &RecentFiles{
				Files:     []RecentFile{},
				MaxRecent: getMaxRecentFromConfig(),
			}, nil
		}
		return nil, err
	}

	var recent RecentFiles
	if err := json.Unmarshal(data, &recent); err != nil {
		return nil, err
	}

	// Set from config or default if not specified
	if recent.MaxRecent == 0 {
		recent.MaxRecent = getMaxRecentFromConfig()
	}

	return &recent, nil
}

// getMaxRecentFromConfig gets the max recent files value from config or returns default
func getMaxRecentFromConfig() int {
	cfg, err := Load()
	if err != nil || cfg == nil {
		return DefaultMaxRecent
	}
	return cfg.GetInt("max-recent-files", DefaultMaxRecent)
}

// Save writes the recent files list to disk
func (r *RecentFiles) Save() error {
	path, err := GetRecentFilesPath()
	if err != nil {
		return err
	}

	// Ensure config directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// GetRecentFilesPath returns the path to the recent files JSON file
func GetRecentFilesPath() (string, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "recent.json"), nil
}

// CleanupMissing removes files that no longer exist from the list
func (r *RecentFiles) CleanupMissing() {
	valid := []RecentFile{}
	for _, file := range r.Files {
		if _, err := os.Stat(file.Path); err == nil {
			valid = append(valid, file)
		}
	}
	r.Files = valid
}

// SortByScore sorts files by a combination of recency and frequency
// More recent and more frequently accessed files score higher
func (r *RecentFiles) SortByScore() {
	now := time.Now()

	sort.Slice(r.Files, func(i, j int) bool {
		scoreI := r.calculateScore(r.Files[i], now)
		scoreJ := r.calculateScore(r.Files[j], now)
		return scoreI > scoreJ
	})
}

// calculateScore computes a score for a file based on recency and frequency
// Score formula: accessCount * (1 / (hoursSinceAccess + 1))
// This gives higher weight to recent files, with frequency as a multiplier
func (r *RecentFiles) calculateScore(file RecentFile, now time.Time) float64 {
	hoursSinceAccess := now.Sub(file.LastAccessed).Hours()

	// Recency factor: decays exponentially with time
	// After 24 hours, factor is 0.5; after 168 hours (1 week), factor is ~0.14
	recencyFactor := 1.0 / (1.0 + hoursSinceAccess/24.0)

	// Frequency factor: logarithmic to prevent very high access counts from dominating
	// This ensures a file opened 100 times isn't weighted 10x more than one opened 10 times
	frequencyFactor := float64(file.AccessCount)
	if frequencyFactor > 1 {
		frequencyFactor = 1 + (frequencyFactor-1)*0.5 // Dampen frequency impact
	}

	return recencyFactor * frequencyFactor
}

// GetCursorPosition returns the saved cursor position for a file
// Returns -1 if file not found or content has changed
func (r *RecentFiles) GetCursorPosition(filePath string) int {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return -1
	}

	for _, file := range r.Files {
		if file.Path == absPath {
			// Check if file content has changed
			if file.ContentHash != "" {
				currentHash, err := computeFileHash(absPath)
				if err != nil || currentHash != file.ContentHash {
					// File changed, don't restore cursor
					return -1
				}
			}
			return file.LastCursorPos
		}
	}

	return -1
}

// GetRecentFilesList returns a list of recent files sorted by score
func GetRecentFilesList() ([]RecentFile, error) {
	recent, err := LoadRecentFiles()
	if err != nil {
		return nil, err
	}

	recent.CleanupMissing()
	recent.SortByScore()

	return recent.Files, nil
}

// GetRecentFilePath returns the nth most recent file path (1-indexed)
func GetRecentFilePath(index int) (string, error) {
	files, err := GetRecentFilesList()
	if err != nil {
		return "", err
	}

	if index < 1 || index > len(files) {
		return "", fmt.Errorf("index %d out of range (1-%d)", index, len(files))
	}

	return files[index-1].Path, nil
}

// ClearRecentFiles removes all recent files from the list
func ClearRecentFiles() error {
	recent := &RecentFiles{
		Files:     []RecentFile{},
		MaxRecent: DefaultMaxRecent,
	}
	return recent.Save()
}
