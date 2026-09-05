package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/niklas-heer/tdx/internal/config"
)

func TestTUIUnicodeEditing(t *testing.T) {
	cases := []struct{ name, keys, want string }{
		{"bracketed paste", "n\x1b[200~enter ä😀\nignored\x1b[201~\r", "enter ä😀"},
		{"insert", "nKäse überprüfen\r", "Käse überprüfen"},
		{"delete", "nKäse\x1b[H\x1b[C\x1b[3~\r", "Kse"},
		{"right and backspace", "nKäse\x1b[H\x1b[C\x1b[C\x7f\r", "Kse"},
		{"three and four byte runes", "n茶😀\x7f\r", "茶"},
		{"search backspace", "/äü\x7f", "SEARCH   ä"},
		{"command backspace", ":üä\x7f", ":ü"},
		{"recent backspace", "rüä\x7f", "Recent: ü"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			config.SetConfigDirForTesting(dir)
			t.Cleanup(config.ResetConfigDirForTesting)
			file := filepath.Join(dir, "ü.md")
			if err := os.WriteFile(file, []byte("- [ ] Käse\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := config.SaveRecentFile(file, 0); err != nil {
				t.Fatal(err)
			}
			output := runPiped(t, file, tc.keys)
			if !utf8.ValidString(output) || !strings.Contains(output, tc.want) {
				t.Fatalf("want %q in valid UTF-8 output:\n%s", tc.want, output)
			}
			content := readTestFile(t, file)
			if !utf8.ValidString(content) {
				t.Fatalf("invalid UTF-8 on disk: %q", content)
			}
			if strings.HasPrefix(tc.keys, "n") && !strings.Contains(content, "- [ ] "+tc.want) {
				t.Fatalf("expected saved todo %q: %s", tc.want, content)
			}
		})
	}
}
