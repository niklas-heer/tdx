package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	commandpkg "github.com/niklas-heer/tdx/internal/cmd"
)

func scriptCLI(t *testing.T, configRoot string, args ...string) (string, string, error) {
	t.Helper()
	command := exec.Command(testBinary, args...)
	command.Env = append(os.Environ(), "XDG_CONFIG_HOME="+configRoot, "CLICOLOR_FORCE=1")
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	return stdout.String(), stderr.String(), err
}

func TestCLIJSONWithUnwritableHistory(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "TASKS")
	if err := os.WriteFile(file, []byte("- [x] Parent #backend\n    - [ ] Fix Käse #backend !p2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// A file cannot hold the history directory, even under privileged test users.
	blocked := filepath.Join(root, "blocked")
	if err := os.WriteFile(blocked, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	out, stderr, err := scriptCLI(t, blocked, "--file", file, "list", "--json", "--status=open", "--tag=#backend")
	if err != nil || stderr != "" {
		t.Fatalf("failed: %v, stderr: %s", err, stderr)
	}
	var tasks []commandpkg.Task
	if err := json.Unmarshal([]byte(out), &tasks); err != nil {
		t.Fatalf("invalid JSON: %v: %q", err, out)
	}
	if len(tasks) != 1 || tasks[0].Index != 2 || tasks[0].Priority != 2 || tasks[0].ParentIndex == nil || *tasks[0].ParentIndex != 1 {
		t.Fatal(out)
	}
	out, stderr, err = scriptCLI(t, blocked, "-f", file, "list", "--status=done")
	if err != nil || stderr != "" || !strings.Contains(out, "1.") || strings.Contains(out, "2.") {
		t.Fatalf("text query: %q, %q, %v", out, stderr, err)
	}
	missing := filepath.Join(root, "missing.md")
	out, stderr, err = scriptCLI(t, blocked, missing, "list", "--json")
	if err != nil || stderr != "" || out != "[]\n" {
		t.Fatalf("empty query: %q, %q, %v", out, stderr, err)
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatal("list created a file")
	}
}

func TestCLIScriptErrorsBeforeSideEffects(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "tasks.md")
	original := "- [ ] Keep me\n"
	if err := os.WriteFile(file, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	configRoot := filepath.Join(root, "config")
	for _, args := range [][]string{
		{"--read-only", "add", "no"}, {"--read-only", "toggle", "1"},
		{"--read-only", "edit", "1", "no"}, {"--read-only", "delete", "1"},
		{"list", "--status=typo"}, {"add", "--json", "no"}, {"toggle", "one"},
		{"add"}, {"list", "extra"}, {"--typo"}, {"delete", "1", "2"},
	} {
		out, stderr, err := scriptCLI(t, configRoot, append([]string{"--file", file}, args...)...)
		if err == nil || out != "" || !strings.HasPrefix(stderr, "tdx:") {
			t.Fatalf("%q: stdout=%q stderr=%q err=%v", args, out, stderr, err)
		}
	}
	actual, _ := os.ReadFile(file)
	if string(actual) != original {
		t.Fatal("failed command modified document")
	}
	if _, err := os.Stat(configRoot); !os.IsNotExist(err) {
		t.Fatal("invalid command opened history storage")
	}
}

func TestCLILiteralArgumentsAndReadOnlyConfig(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "tasks.md")
	configRoot := filepath.Join(root, "config")
	configDir := filepath.Join(configRoot, "tdx")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	configFile := filepath.Join(configDir, "config.toml")
	if err := os.WriteFile(configFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"--read-only", `"quoted Käse"`, "--json"} {
		out, stderr, err := scriptCLI(t, configRoot, "--file", file, "add", "--", text)
		if err != nil || stderr != "" || !strings.Contains(out, text) {
			t.Fatalf("literal %q: %q %q %v", text, out, stderr, err)
		}
	}
	out, stderr, err := scriptCLI(t, configRoot, "--file", file, "list", "--json")
	var tasks []commandpkg.Task
	if err != nil || stderr != "" {
		t.Fatalf("list: %v %s", err, stderr)
	}
	if err := json.Unmarshal([]byte(out), &tasks); err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 3 || tasks[0].Text != "--read-only" || tasks[1].Text != `"quoted Käse"` || tasks[2].Text != "--json" {
		t.Fatal(out)
	}
	if err := os.WriteFile(configFile, []byte("[defaults]\nread_only = true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, stderr, err = scriptCLI(t, configRoot, "--file", file, "delete", "1")
	if err == nil || out != "" || !strings.Contains(stderr, "read-only") {
		t.Fatalf("config bypassed: %q %q %v", out, stderr, err)
	}
}
