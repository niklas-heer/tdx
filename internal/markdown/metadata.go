package markdown

import (
	"bytes"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Metadata represents per-file configuration options from YAML frontmatter
type Metadata struct {
	FilterDone   *bool `yaml:"filter-done,omitempty"`   // Filter out completed tasks
	MaxVisible   *int  `yaml:"max-visible,omitempty"`   // Maximum visible tasks
	ShowHeadings *bool `yaml:"show-headings,omitempty"` // Show headings between tasks
	ReadOnly     *bool `yaml:"read-only,omitempty"`     // Open in read-only mode
	WordWrap     *bool `yaml:"word-wrap,omitempty"`     // Enable word wrapping
}

// frontmatterRegex matches YAML frontmatter at the start of a file
// Format: ---\n...yaml...\n---\n
var frontmatterRegex = regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\s*\n`)

// ParseMetadata extracts YAML frontmatter from markdown content
// Returns the metadata and the content without frontmatter
func ParseMetadata(content string) (*Metadata, string, error) {
	matches := frontmatterRegex.FindStringSubmatch(content)
	if matches == nil || len(matches) < 2 {
		// No frontmatter found
		return &Metadata{}, content, nil
	}

	yamlContent := matches[1]
	contentWithoutFrontmatter := strings.TrimPrefix(content, matches[0])

	var metadata Metadata
	decoder := yaml.NewDecoder(bytes.NewBufferString(yamlContent))
	decoder.KnownFields(true) // Reject unknown fields to catch typos

	if err := decoder.Decode(&metadata); err != nil {
		// Return empty metadata but still strip frontmatter to avoid parse issues
		return &Metadata{}, contentWithoutFrontmatter, err
	}

	return &metadata, contentWithoutFrontmatter, nil
}

// SerializeMetadata adds YAML frontmatter to markdown content
func SerializeMetadata(metadata *Metadata, content string) string {
	// Don't add frontmatter if all fields are nil (no configuration)
	if metadata.IsEmpty() {
		return content
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")

	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(metadata); err != nil {
		// If encoding fails, just return content without frontmatter
		return content
	}
	encoder.Close()

	buf.WriteString("---\n")
	buf.WriteString(content)

	return buf.String()
}

// IsEmpty returns true if all metadata fields are nil
func (m *Metadata) IsEmpty() bool {
	return m.FilterDone == nil &&
		m.MaxVisible == nil &&
		m.ShowHeadings == nil &&
		m.ReadOnly == nil &&
		m.WordWrap == nil
}

// GetBool returns the value of a bool pointer or the default if nil
func (m *Metadata) GetBool(field string, defaultValue bool) bool {
	switch field {
	case "filter-done":
		if m.FilterDone != nil {
			return *m.FilterDone
		}
	case "show-headings":
		if m.ShowHeadings != nil {
			return *m.ShowHeadings
		}
	case "read-only":
		if m.ReadOnly != nil {
			return *m.ReadOnly
		}
	case "word-wrap":
		if m.WordWrap != nil {
			return *m.WordWrap
		}
	}
	return defaultValue
}

// GetInt returns the value of an int pointer or the default if nil
func (m *Metadata) GetInt(field string, defaultValue int) int {
	switch field {
	case "max-visible":
		if m.MaxVisible != nil {
			return *m.MaxVisible
		}
	}
	return defaultValue
}
