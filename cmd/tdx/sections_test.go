package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/niklas-heer/tdx/internal/config"
)

func TestTUISections(t *testing.T) {
	const document = "# Work\n\n- [ ] Main task\n\n## Backend\n\n- [ ] Child task\n\n# Empty\n\n# Home\n\n- [ ] Home task\n"
	cases := []struct{ name, keys, want, absent, saved string }{
		{"overview", "s", "Empty", "", ""},
		{"parent focus", "s\r", "Child task", "Home task", ""},
		{"child focus", "sj\r", "Child task", "Main task", ""},
		{"empty focus and add", "sjj\rnNew task\r", "New task", "Home task", "# Empty\n\n- [ ] New task"},
		{"rename heading", "se\x1b[H\x1b[3~\x1b[3~\x1b[3~\x1b[3~Büro\r", "Büro", "", "# Büro\n"},
		{"create sibling", "snLater\r", "Later", "", "# Later\n"},
		{"create child", "sNPlanning\r", "Planning", "", "## Planning\n"},
		{"fold", "s \x1b", "Home task", "Child task", ""},
		{"clear focus", "sj\rS", "Home task", "", ""},
		{"empty cannot delete hidden task", "sjj\rd", "Section: Empty", "Home task", "- [ ] Main task"},
		{"search stays in focus", "sj\r/Home", "No matches", "Home task", ""},
		{"undo rename", "se\x1b[HNew \r\x1bu", "Main task", "New Work", "# Work\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			config.SetConfigDirForTesting(dir)
			t.Cleanup(config.ResetConfigDirForTesting)
			file := filepath.Join(dir, "todo.md")
			if err := os.WriteFile(file, []byte(document), 0600); err != nil {
				t.Fatal(err)
			}
			got := runPiped(t, file, tc.keys)
			if tc.want != "" && !strings.Contains(got, tc.want) {
				t.Fatalf("want %q:\n%s", tc.want, got)
			}
			if tc.absent != "" && strings.Contains(got, tc.absent) {
				t.Fatalf("unexpected %q:\n%s", tc.absent, got)
			}
			saved := readTestFile(t, file)
			if tc.saved != "" && !strings.Contains(saved, tc.saved) {
				t.Fatalf("want saved %q:\n%s", tc.saved, saved)
			}
			if !strings.Contains(saved, "Home task") || !strings.Contains(saved, "Child task") {
				t.Fatalf("unrelated tasks changed: %s", saved)
			}
		})
	}
}

func TestTUISectionReadOnly(t *testing.T) {
	dir := t.TempDir()
	config.SetConfigDirForTesting(dir)
	t.Cleanup(config.ResetConfigDirForTesting)
	file := filepath.Join(dir, "todo.md")
	content := "---\nread-only: true\n---\n\n# Work\n\n- [ ] Task\n"
	if err := os.WriteFile(file, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	output := runPiped(t, file, "se")
	if !strings.Contains(output, "read-only") {
		t.Fatalf("expected read-only feedback: %s", output)
	}
	if got := readTestFile(t, file); got != content {
		t.Fatalf("read-only file changed: %s", got)
	}
}

func TestTUIUndoAfterCancelAndReload(t *testing.T) {
	for _, keys := range []string{" n\x1bu", "  :reload\r uuu"} {
		t.Run(keys, func(t *testing.T) {
			dir := t.TempDir()
			config.SetConfigDirForTesting(dir)
			t.Cleanup(config.ResetConfigDirForTesting)
			file := filepath.Join(dir, "todo.md")
			if err := os.WriteFile(file, []byte("- [ ] Task\n"), 0600); err != nil {
				t.Fatal(err)
			}
			runPiped(t, file, keys)
			if got := readTestFile(t, file); !strings.Contains(got, "- [ ] Task") {
				t.Fatalf("undo crossed a discarded snapshot: %s", got)
			}
		})
	}
}
