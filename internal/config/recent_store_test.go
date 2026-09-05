package config

import (
	"os"
	"path/filepath"
	"testing"
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
