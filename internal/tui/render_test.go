package tui

import (
	"strings"
	"testing"
)

// TestRenderInlineCode_Links tests that markdown links are properly rendered
func TestRenderInlineCode_Links(t *testing.T) {
	identity := func(s string) string { return s }

	tests := []struct {
		name     string
		input    string
		expected string // What we expect to see in output (text part)
	}{
		{
			name:     "Simple link",
			input:    "Check [Google](https://google.com) for info",
			expected: "Google", // Should render as clickable "Google"
		},
		{
			name:     "Link at start",
			input:    "[Documentation](https://example.com/docs) is here",
			expected: "Documentation",
		},
		{
			name:     "Multiple links",
			input:    "See [Link1](http://a.com) and [Link2](http://b.com)",
			expected: "Link1",
		},
		{
			name:     "Link with code",
			input:    "Use `code` and [link](http://example.com) together",
			expected: "code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderInlineCode(tt.input, false, identity, identity, identity)

			// The result should contain the link text but not the URL in brackets
			if !strings.Contains(result, tt.expected) {
				t.Errorf("Expected result to contain '%s', got: %s", tt.expected, result)
			}

			// Should NOT contain the markdown syntax like ](http
			if strings.Contains(result, "](") {
				t.Errorf("Result still contains markdown link syntax: %s", result)
			}

			// For links, should contain OSC 8 hyperlink escape codes
			if strings.Contains(tt.input, "[") && strings.Contains(tt.input, "](") {
				// OSC 8 format: \x1b]8;;URL\x1b\\TEXT\x1b]8;;\x1b\\
				if !strings.Contains(result, "\x1b]8;;") {
					t.Errorf("Expected OSC 8 hyperlink codes in result, got: %q", result)
				}
			}
		})
	}
}

// TestRenderInlineCode_Code tests that inline code is properly rendered
func TestRenderInlineCode_Code(t *testing.T) {
	identity := func(s string) string { return s }
	codeStyle := func(s string) string { return "[CODE:" + s + "]" }

	input := "Use `grep` to search"
	result := RenderInlineCode(input, false, identity, identity, codeStyle)

	// Should contain styled code
	if !strings.Contains(result, "[CODE: grep ]") {
		t.Errorf("Expected styled code block, got: %s", result)
	}

	// Should NOT contain backticks
	if strings.Contains(result, "`") {
		t.Errorf("Result still contains backticks: %s", result)
	}
}

// TestRenderInlineCode_NoMarkdown tests plain text passes through
func TestRenderInlineCode_NoMarkdown(t *testing.T) {
	identity := func(s string) string { return s }

	input := "Plain text with no formatting"
	result := RenderInlineCode(input, false, identity, identity, identity)

	if result != input {
		t.Errorf("Expected plain text to pass through unchanged, got: %s", result)
	}
}
