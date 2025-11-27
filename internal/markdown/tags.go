package markdown

import (
	"regexp"
	"sort"
	"strings"
)

// tagRegex matches hashtags at the end of todo text
// Format: #tag (alphanumeric, dash, underscore)
var tagRegex = regexp.MustCompile(`#([a-zA-Z0-9_-]+)`)

// ExtractTags extracts all tags from todo text
// Tags are hashtags like #urgent #backend
func ExtractTags(text string) []string {
	matches := tagRegex.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return []string{}
	}

	tags := make([]string, 0, len(matches))
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 {
			tag := match[1] // Extract the tag without the # prefix
			// Deduplicate tags
			if !seen[tag] {
				tags = append(tags, tag)
				seen[tag] = true
			}
		}
	}

	return tags
}

// RemoveTags removes all tags from the text
// This is useful if you want to display text without tags
func RemoveTags(text string) string {
	return strings.TrimSpace(tagRegex.ReplaceAllString(text, ""))
}

// HasTag checks if a todo has a specific tag
func (t *Todo) HasTag(tag string) bool {
	for _, todoTag := range t.Tags {
		if strings.EqualFold(todoTag, tag) {
			return true
		}
	}
	return false
}

// HasAnyTag checks if a todo has any of the specified tags
func (t *Todo) HasAnyTag(tags []string) bool {
	if len(tags) == 0 {
		return true // No filter means match all
	}
	for _, tag := range tags {
		if t.HasTag(tag) {
			return true
		}
	}
	return false
}

// GetAllTags returns all unique tags from a list of todos, sorted alphabetically
func GetAllTags(todos []Todo) []string {
	tagSet := make(map[string]bool)
	for _, todo := range todos {
		for _, tag := range todo.Tags {
			tagSet[tag] = true
		}
	}

	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}

	// Sort tags alphabetically for deterministic ordering
	sort.Strings(tags)
	return tags
}
