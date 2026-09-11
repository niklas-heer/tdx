package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestViewStoreCanonicalFilesAndIsolation(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "todo.md")
	os.WriteFile(file, []byte("- [ ] task\n"), 0600)
	store := FileViewStore{Dir: t.TempDir()}
	state := &SavedViews{Active: "Work", Views: map[string]SavedView{"Work": {FilterDone: true}}}
	if err := store.Save(file, state); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(dir, "alias.md")
	if err := os.Symlink(file, alias); err == nil {
		got, err := store.Load(alias)
		if err != nil || got.Active != "Work" {
			t.Fatalf("canonical alias: %+v %v", got, err)
		}
	}
	other, err := store.Load(filepath.Join(dir, "other.md"))
	if err != nil || len(other.Views) != 0 {
		t.Fatal("views leaked across files")
	}
	got, _ := store.Load(file)
	delete(got.Views, "Work")
	got.Active = ""
	if err := store.Save(file, got); err != nil {
		t.Fatal(err)
	}
	got, _ = store.Load(file)
	if len(got.Views) != 0 {
		t.Fatal("delete did not persist")
	}
	path, _ := store.path(file)
	os.WriteFile(path, []byte("broken"), 0600)
	if _, err := store.Load(file); err == nil {
		t.Fatal("corrupt views silently ignored")
	}
}

func TestViewStoreValidationAndMissingCanonicalParent(t *testing.T) {
	store := FileViewStore{Dir: t.TempDir()}
	file := filepath.Join(t.TempDir(), "todo.md")
	for _, state := range []*SavedViews{nil, {Views: map[string]SavedView{"bad": {Due: "invalid"}}}, {Views: map[string]SavedView{"bad": {Focus: &SectionRef{}}}}} {
		if err := store.Save(file, state); err == nil {
			t.Fatal("invalid state accepted")
		}
	}
	dir := t.TempDir()
	real := filepath.Join(dir, "real")
	os.Mkdir(real, 0700)
	alias := filepath.Join(dir, "alias")
	if err := os.Symlink(real, alias); err != nil {
		t.Skip(err)
	}
	a, _ := store.path(filepath.Join(real, "new", "todo.md"))
	b, _ := store.path(filepath.Join(alias, "new", "todo.md"))
	if a != b {
		t.Fatal("nonexistent file under symlink parent not canonical")
	}
}
