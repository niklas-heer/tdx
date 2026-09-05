package util

import (
	"os/exec"
	"runtime"
	"strings"
)

func clipboardCommands(paste bool) [][]string {
	switch runtime.GOOS {
	case "darwin":
		if paste {
			return [][]string{{"pbpaste"}}
		}
		return [][]string{{"pbcopy"}}
	case "windows":
		script := "[Console]::InputEncoding = [System.Text.UTF8Encoding]::new(); $text = [Console]::In.ReadToEnd(); Set-Clipboard -Value $text"
		if paste {
			script = "[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new(); Get-Clipboard -Raw"
		}
		return [][]string{{"powershell", "-NoProfile", "-NonInteractive", "-Command", script}}
	default:
		if paste {
			return [][]string{{"wl-paste", "--no-newline"}, {"xclip", "-selection", "clipboard", "-o"}, {"xsel", "--clipboard", "--output"}}
		}
		return [][]string{{"wl-copy"}, {"xclip", "-selection", "clipboard"}, {"xsel", "--clipboard", "--input"}}
	}
}

// CopyToClipboard tries the native desktop clipboard, tolerating headless sessions.
func CopyToClipboard(text string) {
	for _, args := range clipboardCommands(false) {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdin = strings.NewReader(text)
		if cmd.Run() == nil {
			return
		}
	}
}

// PasteFromClipboard reads only the first line, as task inputs are single-line.
func PasteFromClipboard() string {
	for _, args := range clipboardCommands(true) {
		out, err := exec.Command(args[0], args[1:]...).Output()
		if err == nil {
			text := strings.TrimRight(string(out), "\n\r")
			if idx := strings.IndexAny(text, "\n\r"); idx != -1 {
				text = text[:idx]
			}
			return text
		}
	}
	return ""
}
