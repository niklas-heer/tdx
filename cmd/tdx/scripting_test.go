package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestCLIFollowsTheFilesReadOnlySetting(t *testing.T) {
	root := t.TempDir()
	locked := filepath.Join(root, "locked.md")
	lockedContent := "---\nread-only: true\n---\n- [ ] Keep\n"
	open := filepath.Join(root, "open.md")
	if err := os.WriteFile(locked, []byte(lockedContent), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(open, []byte("---\nread-only: false\n---\n- [ ] Change\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// History cannot open under a file, so the refusal must come first.
	blocked := filepath.Join(root, "blocked")
	if err := os.WriteFile(blocked, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"add", "new"}, {"toggle", "1"}, {"edit", "1", "renamed"},
		{"delete", "1"}, {"done", "1"}, {"undone", "1"},
	} {
		_, stderr, err := scriptCLI(t, blocked, append([]string{locked}, args...)...)
		if err == nil || !strings.Contains(stderr, "read-only file") {
			t.Fatalf("%s on a read-only file: err=%v stderr=%q", args[0], err, stderr)
		}
	}
	if got, _ := os.ReadFile(locked); string(got) != lockedContent {
		t.Fatalf("read-only file changed: %q", got)
	}

	// read-only: false in the file outranks manual_save = true in config.toml.
	configRoot := filepath.Join(root, "config")
	if err := os.MkdirAll(filepath.Join(configRoot, "tdx"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configRoot, "tdx", "config.toml"), []byte("[defaults]\nmanual_save = true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, stderr, err := scriptCLI(t, configRoot, open, "done", "1"); err != nil {
		t.Fatalf("done on a file that opts out of manual save: %v %s", err, stderr)
	}
	if got, _ := os.ReadFile(open); !strings.Contains(string(got), "- [x] Change") {
		t.Fatalf("file that opts out of manual save was not changed: %q", got)
	}
	plain := filepath.Join(root, "plain.md")
	if err := os.WriteFile(plain, []byte("- [ ] Default\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, stderr, err := scriptCLI(t, configRoot, plain, "done", "1"); err == nil || !strings.Contains(stderr, "read-only mode") {
		t.Fatalf("config manual_save no longer applies: err=%v stderr=%q", err, stderr)
	}
	if _, stderr, err := scriptCLI(t, configRoot, "--read-only", open, "undone", "1"); err == nil || !strings.Contains(stderr, "read-only mode") {
		t.Fatalf("flag did not outrank read-only: false: err=%v stderr=%q", err, stderr)
	}
}

// Regression: when the TUI could not open its file, tdx printed the error to
// stdout and exited 0, including for `tdx last` and `tdx recent <n>`.
func TestOpeningFailuresExitNonzero(t *testing.T) {
	root := t.TempDir()
	configRoot := filepath.Join(root, "config")
	dir := filepath.Join(root, "notes")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	// A directory exists, so recent-file cleanup keeps it, but it cannot be opened.
	recent := fmt.Sprintf(`{"files":[{"path":%q,"last_accessed":%q,"access_count":1}],"max_recent":20}`,
		dir, time.Now().Format(time.RFC3339))
	if err := os.MkdirAll(filepath.Join(configRoot, "tdx"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configRoot, "tdx", "recent.json"), []byte(recent), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{dir}, {"last"}, {"recent", "1"}, {"recent", "9"}} {
		out, stderr, err := scriptCLI(t, configRoot, args...)
		if err == nil || out != "" || !strings.HasPrefix(stderr, "tdx: ") {
			t.Fatalf("tdx %v: err=%v out=%q stderr=%q; want a failing exit with the error on stderr", args, err, out, stderr)
		}
	}
}
