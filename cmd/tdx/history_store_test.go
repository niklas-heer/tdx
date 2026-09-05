package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/niklas-heer/tdx/internal/tui"
	"github.com/niklas-heer/tdx/internal/versioning"
)

func TestHistoryBrowserResolvesAliases(t *testing.T) {
	versioning.SetStoreDirForTesting(t.TempDir())
	t.Cleanup(versioning.ResetStoreDirForTesting)
	versions, err := versioning.Open(100)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = versions.Close() })
	dir := t.TempDir()
	target, alias := filepath.Join(dir, "tasks.md"), filepath.Join(dir, "alias.md")
	source := "- [ ] Task\n"
	if err := os.WriteFile(target, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, alias); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("symlinks unavailable: %v", err)
		}
		t.Fatal(err)
	}
	persistence := historyStore(versions)
	if _, err := persistence.ReadFile(alias); err != nil {
		t.Fatal(err)
	}
	cfg := &tui.ConfigType{}
	wireVersioningTUI(cfg, versions)
	for _, path := range []string{alias, target} {
		list, err := cfg.ListVersionsFunc(path)
		if err != nil || len(list) != 1 {
			t.Fatalf("history for %s: %v, %v", path, list, err)
		}
		got, err := cfg.ReadVersionFunc(path, list[0].ID)
		if err != nil || got != source {
			t.Fatalf("restore for %s: %q, %v", path, got, err)
		}
	}
	list, err := cfg.ListVersionsFunc(filepath.Join(dir, "new.md"))
	if err != nil || len(list) != 0 {
		t.Fatalf("new file: %v, %v", list, err)
	}
}
