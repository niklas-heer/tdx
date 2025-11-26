package util

import (
	"os/exec"
	"strings"
)

// CopyToClipboard copies text to the system clipboard
func CopyToClipboard(text string) {
	// Use pbcopy on macOS
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	_ = cmd.Run() // Ignore error - clipboard may not be available
}

// PasteFromClipboard retrieves text from the system clipboard
func PasteFromClipboard() string {
	// Use pbpaste on macOS
	cmd := exec.Command("pbpaste")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	// Remove trailing newline and return first line only
	text := strings.TrimRight(string(out), "\n\r")
	if idx := strings.Index(text, "\n"); idx != -1 {
		text = text[:idx]
	}
	return text
}
