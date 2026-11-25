package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRecentStoresHaveIndependentLimitsAndDestinations(t *testing.T) {
	a, b := RecentStore{Dir: t.TempDir(), Limit: 1}, RecentStore{Dir: t.TempDir(), Limit: 3}
	for i, name := range []string{"a.md", "b.md", "c.md"} {
		path := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(path, []byte("- [ ] Test"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := a.SaveFile(path, i); err != nil {
			t.Fatal(err)
		}
		if err := b.SaveFile(path, i); err != nil {
			t.Fatal(err)
		}
	}
	listA, err := a.Load()
	if err != nil {
		t.Fatal(err)
	}
	listB, err := b.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(listA.Files) != 1 || len(listB.Files) != 3 {
		t.Fatalf("shared retention: %d %d", len(listA.Files), len(listB.Files))
	}
	if err := a.Clear(); err != nil {
		t.Fatal(err)
	}
	listB, err = b.Load()
	if err != nil || len(listB.Files) != 3 {
		t.Fatal("cleared another store")
	}
}

// Regression: trimming by score dropped a newly opened file when busier files
// outscored it, so `tdx last` opened something else.
func TestSaveFileKeepsTheFileJustOpened(t *testing.T) {
	dir := t.TempDir()
	store := RecentStore{Dir: t.TempDir(), Limit: 2}
	busy := &RecentFiles{MaxRecent: 2}
	for _, name := range []string{"busy1.md", "busy2.md"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		busy.Files = append(busy.Files, RecentFile{Path: path, LastAccessed: time.Now(), AccessCount: 50})
	}
	if err := store.save(busy); err != nil {
		t.Fatal(err)
	}
	fresh := filepath.Join(dir, "fresh.md")
	if err := os.WriteFile(fresh, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveFile(fresh, 0); err != nil {
		t.Fatal(err)
	}
	recent, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(recent.Files) != 2 {
		t.Fatalf("kept %d files, want the limit of 2", len(recent.Files))
	}
	if got := recent.LatestPath(); got != fresh {
		t.Fatalf("LatestPath() = %q, want the file just opened %q", got, fresh)
	}
}
