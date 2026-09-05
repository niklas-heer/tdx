package main

import (
	"github.com/BurntSushi/toml"
	"os"
	"path/filepath"
	"testing"
)

func TestThemeSavePreservesVersioningAndUnknownSettings(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := filepath.Join(dir, "tdx", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	source := "[versioning]\nmax_versions=7\n[custom]\nkeep=true\n[defaults]\nword_wrap=false\n"
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	if err := SaveTheme("nord"); err != nil {
		t.Fatal(err)
	}
	values := map[string]interface{}{}
	if _, err := toml.DecodeFile(path, &values); err != nil {
		t.Fatal(err)
	}
	if values["versioning"].(map[string]interface{})["max_versions"] != int64(7) {
		t.Fatal(values)
	}
	if values["custom"].(map[string]interface{})["keep"] != true {
		t.Fatal(values)
	}
	if values["defaults"].(map[string]interface{})["word_wrap"] != false {
		t.Fatal(values)
	}
}
