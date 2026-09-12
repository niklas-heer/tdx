package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	commandpkg "github.com/niklas-heer/tdx/internal/cmd"
)

func TestCLIRevisionWorkflow(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "tasks.md")
	configRoot := filepath.Join(root, "config")
	original := "# Project\n\n- [ ] First !p2\n## API\n- [ ] Second !p1 @due(2026-10-01)\n"
	if err := os.WriteFile(file, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	out, stderr, err := scriptCLI(t, configRoot, file, "list", "--with-revision", "--section=Project", "--priority=1", "--due=2026-10-01")
	if err != nil || stderr != "" {
		t.Fatalf("query: %v %s", err, stderr)
	}
	var snapshot commandpkg.ListSnapshot
	if err := json.Unmarshal([]byte(out), &snapshot); err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Tasks) != 1 || snapshot.Tasks[0].Index != 2 {
		t.Fatal(out)
	}
	token, stderr, err := scriptCLI(t, configRoot, file, "revision")
	if err != nil || stderr != "" || strings.TrimSpace(token) != snapshot.Revision {
		t.Fatalf("revision: %q %q %v", token, stderr, err)
	}
	if _, err := os.Stat(configRoot); !os.IsNotExist(err) {
		t.Fatal("read-only query created configuration")
	}
	_, stderr, err = scriptCLI(t, configRoot, file, "done", "2", "--if-revision", snapshot.Revision)
	if err != nil {
		t.Fatalf("done: %v %s", err, stderr)
	}
	before, _ := os.ReadFile(file)
	_, stderr, err = scriptCLI(t, configRoot, file, "delete", "1", "--if-revision", snapshot.Revision)
	if err == nil || !strings.Contains(stderr, "revision does not match") {
		t.Fatalf("accepted stale write: %v %s", err, stderr)
	}
	after, _ := os.ReadFile(file)
	if string(before) != string(after) {
		t.Fatal("stale write changed file")
	}
	for _, command := range []string{"done", "done", "undone", "undone"} {
		_, stderr, err = scriptCLI(t, configRoot, file, command, "2")
		if err != nil {
			t.Fatalf("%s: %v %s", command, err, stderr)
		}
	}
	after, _ = os.ReadFile(file)
	if string(after) != original {
		t.Fatalf("idempotent round trip altered file: %q", after)
	}
}

func TestCLIAutomationValidationAndCompletions(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "missing.md")
	configRoot := filepath.Join(root, "config")
	for _, args := range [][]string{
		{"list", "--due=tomorow"}, {"list", "--priority=-1"}, {"list", "--section="},
		{"list", "--if-revision=missing"}, {"done", "1", "--if-revision=bad"},
		{"done", "1", "--read-only"}, {"undone", "1", "--manual-save"},
		{"revision", "extra"}, {"add", "hi", "--with-revision"},
		{"completion", "bad"}, {"completion"},
	} {
		out, stderr, err := scriptCLI(t, configRoot, append([]string{file}, args...)...)
		if err == nil || out != "" || stderr == "" {
			t.Fatalf("%q: %q %q %v", args, out, stderr, err)
		}
	}
	for _, shell := range []string{"bash", "zsh", "fish"} {
		out, stderr, err := scriptCLI(t, configRoot, "completion", shell)
		if err != nil || stderr != "" || !strings.Contains(out, "with-revision") {
			t.Fatalf("%s completion: %s %s %v", shell, out, stderr, err)
		}
	}
	if _, err := os.Stat(configRoot); !os.IsNotExist(err) {
		t.Fatal("validation/completion created configuration")
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatal("validation/completion created task file")
	}
}
