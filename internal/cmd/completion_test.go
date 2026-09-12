package cmd

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func TestCompletionShellSyntax(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish"} {
		t.Run(shell, func(t *testing.T) {
			var out bytes.Buffer
			if err := WriteCompletion(&out, shell); err != nil {
				t.Fatal(err)
			}
			if _, err := exec.LookPath(shell); err != nil {
				t.Skipf("%s unavailable for syntax check", shell)
			}
			command := exec.Command(shell, "-n")
			command.Stdin = &out
			if result, err := command.CombinedOutput(); err != nil {
				t.Fatalf("invalid %s syntax: %v\n%s", shell, err, result)
			}
		})
	}
}

func TestBashCompletionValuesAndLiteralText(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	for _, tt := range []struct{ words, want string }{
		{"tdx list --status o", "open"}, {"tdx completion z", "zsh"}, {"tdx add -- --st", ""},
	} {
		script := bashCompletion + "\nCOMP_WORDS=(" + tt.words + "); COMP_CWORD=$((${#COMP_WORDS[@]}-1)); _tdx_complete; printf '%s' \"${COMPREPLY[*]}\"\n"
		command := exec.Command("bash")
		command.Stdin = strings.NewReader(script)
		result, err := command.CombinedOutput()
		if err != nil || string(result) != tt.want {
			t.Fatalf("%s: %q %v", tt.words, result, err)
		}
	}
}
