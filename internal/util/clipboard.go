package util

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Clipboard is injected into each TUI session so failures can be reported truthfully.
type Clipboard interface {
	Copy(string) error
	Paste() (string, error)
}

type SystemClipboard struct {
	Platform string
	LookPath func(string) (string, error)
	Run      func(context.Context, string, []string, string) (string, error)
	Getenv   func(string) string
	Timeout  time.Duration
}

func (c SystemClipboard) command(copying bool) (string, []string, error) {
	platform := c.Platform
	if platform == "" {
		platform = runtime.GOOS
	}
	lookup := c.LookPath
	if lookup == nil {
		lookup = exec.LookPath
	}
	env := c.Getenv
	if env == nil {
		env = os.Getenv
	}
	type command struct {
		name string
		args []string
	}
	var candidates []command
	switch platform {
	case "darwin":
		name := "pbpaste"
		if copying {
			name = "pbcopy"
		}
		candidates = []command{{name, nil}}
	case "windows":
		script := "[Console]::OutputEncoding = [System.Text.Encoding]::UTF8; Get-Clipboard -Raw"
		if copying {
			script = "[Console]::InputEncoding = [System.Text.Encoding]::UTF8; Set-Clipboard -Value ([Console]::In.ReadToEnd())"
		}
		candidates = []command{{"powershell.exe", []string{"-NoProfile", "-NonInteractive", "-Command", script}}}
	case "linux", "freebsd", "openbsd":
		if env("WAYLAND_DISPLAY") != "" {
			if copying {
				candidates = append(candidates, command{"wl-copy", nil})
			} else {
				candidates = append(candidates, command{"wl-paste", []string{"--no-newline"}})
			}
		}
		if env("DISPLAY") != "" {
			args := []string{"-selection", "clipboard"}
			if !copying {
				args = append(args, "-o")
			}
			candidates = append(candidates, command{"xclip", args})
			flag := "--input"
			if !copying {
				flag = "--output"
			}
			candidates = append(candidates, command{"xsel", []string{"--clipboard", flag}})
		}
	}
	for _, candidate := range candidates {
		if path, err := lookup(candidate.name); err == nil {
			return path, candidate.args, nil
		}
	}
	return "", nil, fmt.Errorf("clipboard unavailable: use terminal paste, or install a clipboard helper (wl-clipboard/xclip on Linux; PowerShell on Windows)")
}

func (c SystemClipboard) execute(copying bool, input string) (string, error) {
	name, args, err := c.command(copying)
	if err != nil {
		return "", err
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	run := c.Run
	if run == nil {
		run = func(ctx context.Context, name string, args []string, input string) (string, error) {
			cmd := exec.CommandContext(ctx, name, args...)
			cmd.Stdin = strings.NewReader(input)
			cmd.WaitDelay = 100 * time.Millisecond
			output, err := cmd.Output()
			return string(output), err
		}
	}
	output, err := run(ctx, name, args, input)
	if ctx.Err() != nil {
		return "", fmt.Errorf("clipboard timed out: check your desktop clipboard service")
	}
	if err != nil {
		return "", fmt.Errorf("clipboard operation failed (%s): %w", name, err)
	}
	return output, nil
}
func (c SystemClipboard) Copy(text string) error { _, err := c.execute(true, text); return err }
func (c SystemClipboard) Paste() (string, error) { return c.execute(false, "") }

// Legacy wrappers retain source compatibility for existing callers.
func CopyToClipboard(text string) { _ = (SystemClipboard{}).Copy(text) }
func PasteFromClipboard() string {
	text, _ := (SystemClipboard{}).Paste()
	text, _, _ = strings.Cut(strings.TrimRight(text, "\r\n"), "\n")
	return text
}
