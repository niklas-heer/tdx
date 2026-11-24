package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/niklas-heer/tdx/internal/config"
	"github.com/niklas-heer/tdx/internal/markdown"
)

// Regression: frontmatter outranked explicit command-line flags, so
// `tdx --manual-save file.md` autosaved a file whose frontmatter said
// `read-only: false`. CLI flags > frontmatter > config.toml defaults.
func TestCommandLineFlagsOutrankFrontmatter(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()
	write := func(name, text string) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	const frontmatter = "---\nread-only: false\nshow-headings: false\nmax-visible: 3\n---\n# Todos\n\n- [ ] a\n"
	first := write("first.md", frontmatter)
	second := write("second.md", frontmatter)

	check := func(t *testing.T, m Model, wantReadOnly, wantHeadings bool, wantMax int) {
		t.Helper()
		if m.ReadOnly != wantReadOnly || m.ShowHeadings != wantHeadings || m.MaxVisibleOverride != wantMax {
			t.Fatalf("read-only %v, headings %v, max %d; want %v, %v, %d",
				m.ReadOnly, m.ShowHeadings, m.MaxVisibleOverride, wantReadOnly, wantHeadings, wantMax)
		}
	}
	open := func(t *testing.T, flags FlagOverrides) Model {
		t.Helper()
		fm, err := markdown.ReadFile(first)
		if err != nil {
			t.Fatal(err)
		}
		cfg := testConfig()
		cfg.Flags = flags
		return New(first, fm, true, true, 7, cfg, testStyles(), "test")
	}

	t.Run("explicit flags win, also after switching files", func(t *testing.T) {
		m := open(t, FlagOverrides{ReadOnly: true, ShowHeadings: true, MaxVisible: true})
		check(t, m, true, true, 7)
		m.RecentFiles = []config.RecentFile{{Path: second}}
		m.RecentFilesMode = true
		result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		m = result.(Model)
		if m.FilePath != second {
			t.Fatalf("switch failed: %v", m.Err)
		}
		check(t, m, true, true, 7)
	})

	t.Run("frontmatter wins over config defaults", func(t *testing.T) {
		check(t, open(t, FlagOverrides{}), false, false, 3)
	})
}
