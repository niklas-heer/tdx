package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/niklas-heer/tdx/internal/config"
)

// Regression: exit recorded the file tdx was launched with, so after switching
// to another file in the app, `tdx last` reopened the first one.
func TestExitRecordsTheFileShownLast(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()
	first, second := filepath.Join(dir, "first.md"), filepath.Join(dir, "second.md")
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte("# Todos\n\n- [ ] task\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg := testConfig()
	cfg.Recent = config.RecentStore{Dir: t.TempDir()}
	if err := cfg.Recent.SaveFile(second, 0); err != nil {
		t.Fatal(err)
	}
	runtime := Runtime{Config: cfg, Styles: testStyles(), Version: "test"}

	// Open the recent files overlay and switch to its only entry.
	runtime.RunPiped(first, []byte("r\r"), false)

	recent, err := cfg.Recent.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := recent.LatestPath(); got != second {
		t.Fatalf("LatestPath() = %q after switching to %q", got, second)
	}
}
