package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/niklas-heer/tdx/internal/config"
)

var inlineLayoutSizes = [][2]int{{24, 8}, {80, 24}, {120, 40}, {200, 60}}

func richLayoutModel() Model {
	var source strings.Builder
	source.WriteString("# Project\n\n")
	for i := 0; i < 80; i++ {
		fmt.Fprintf(&source, "## Group %03d — 世界\n\n- [ ] T%03da %s [guide日本語](https://example.com/%s) #work !p2\n  - [ ] T%03db %s\n\n", i, i,
			strings.Repeat("wide 世界 café 👩🏽‍💻 é ", 8), strings.Repeat("long-path/", 20), i,
			strings.Repeat("nested 日本語 🌍 task ", 8))
	}
	m := testModelWithMarkdown(source.String())
	m.ReadOnly = true
	m.ShowHeadings = true
	m.Config().Display.CheckSymbol = "x"
	m.styles.Cyan = func(s string) string { return "\x1b[36m" + s + "\x1b[0m" }
	m.styles.Dim = func(s string) string { return "\x1b[2m" + s + "\x1b[0m" }
	return m
}

// A full terminal-height inline frame makes the renderer scroll the terminal,
// leaving previous frame rows behind. Reserve one terminal row for its cursor.
func assertInlineFrameBounds(t *testing.T, m Model) string {
	t.Helper()
	content := m.View().Content
	if !utf8.ValidString(content) {
		t.Fatal("frame splits a UTF-8 sequence")
	}
	if height := lipgloss.Height(content); height != max(1, m.TermHeight-1) {
		t.Errorf("frame has %d display rows at %dx%d; want exactly %d to keep redraw height stable", height, m.TermWidth, m.TermHeight, max(1, m.TermHeight-1))
	}
	for i, row := range strings.Split(content, "\n") {
		if width := lipgloss.Width(row); width > m.TermWidth {
			t.Errorf("display row %d has width %d at terminal width %d: %q", i, width, m.TermWidth, ansi.Strip(row))
			break
		}
	}
	return ansi.Strip(content)
}

func TestInlineLayoutKeepsSelectionVisible(t *testing.T) {
	for _, size := range inlineLayoutSizes {
		for _, visible := range []int{0, 10, 1000} {
			for _, wrap := range []bool{false, true} {
				t.Run(fmt.Sprintf("%dx%d/max%d/wrap%t", size[0], size[1], visible, wrap), func(t *testing.T) {
					m := richLayoutModel()
					m.TermWidth, m.TermHeight = size[0], size[1]
					m.MaxVisibleOverride, m.WordWrap = visible, wrap
					for _, selected := range []int{0, 80, 159} {
						m.SelectedIndex = selected
						m.InvalidateDocumentTree()
						token := strings.Fields(m.FileModel.Todos[selected].Text)[0]
						for _, focus := range []bool{false, true, false} {
							if focus {
								m.focusSection(selected/2 + 1)
							} else {
								m.clearSections()
							}
							plain := assertInlineFrameBounds(t, m)
							if !strings.Contains(plain, token) {
								t.Errorf("selected task %s disappeared (focus=%t):\n%s", token, focus, plain)
							}
							if focus && !strings.Contains(plain, "Section:") {
								t.Errorf("focus banner disappeared: %q", plain)
							}
							if !focus && strings.Contains(plain, "Section:") {
								t.Errorf("cleared focus banner remains: %q", plain)
							}
						}
					}
				})
			}
		}
	}
}

func TestInlineLayoutKeepsLongInputCursorTailVisible(t *testing.T) {
	for _, size := range inlineLayoutSizes {
		for _, wrap := range []bool{false, true} {
			for _, mode := range []string{"edit", "append", "insert"} {
				t.Run(fmt.Sprintf("%dx%d/%s/wrap%t", size[0], size[1], mode, wrap), func(t *testing.T) {
					m := richLayoutModel()
					m.TermWidth, m.TermHeight = size[0], size[1]
					m.MaxVisibleOverride, m.WordWrap, m.SelectedIndex = 1000, wrap, 80
					m.InputBuffer = strings.Repeat("世界 résumé 👩🏽‍💻 long input ", 80) + "TAIL42"
					m.CursorPos = len(m.InputBuffer)
					m.EditMode = mode == "edit"
					m.InputMode = mode != "edit"
					m.InsertAfterCursor = mode == "insert"
					plain := assertInlineFrameBounds(t, m)
					if !strings.Contains(plain, "TAIL42") {
						t.Errorf("input tail at cursor is off-screen: %q", plain)
					}
				})
			}
		}
	}
}

func TestInlineLayoutBoundsEveryModal(t *testing.T) {
	modes := map[string]func(*Model){
		"help":          func(m *Model) { m.HelpMode = true },
		"sections":      func(m *Model) { m.SectionsMode = true; m.SectionCursor = 40 },
		"heading-input": func(m *Model) { m.SectionsMode = true; m.HeadingInput = "rename"; m.SectionCursor = 40 },
		"saved-views":   func(m *Model) { m.ViewMode = "load"; m.ViewCursor = 40 },
		"save-view":     func(m *Model) { m.ViewMode = "save" },
		"delete-view":   func(m *Model) { m.ViewMode = "delete"; m.ViewCursor = 40; m.ViewConfirm = true },
		"recent":        func(m *Model) { m.RecentFilesMode = true; m.RecentFilesCursor = 40 },
		"tags":          func(m *Model) { m.FilterMode = true; m.TagFilterCursor = 4 },
		"priorities":    func(m *Model) { m.PriorityFilterMode = true },
		"due-dates":     func(m *Model) { m.DueFilterMode = true },
		"themes":        func(m *Model) { m.ThemeMode = true; m.ThemeCursor = 40 },
		"commands":      func(m *Model) { m.CommandMode = true; m.updateFilteredCommands() },
		"search": func(m *Model) {
			m.SearchMode = true
			m.InputBuffer = "世界"
			m.CursorPos = len(m.InputBuffer)
			m.updateSearchResults()
		},
		"max-visible":     func(m *Model) { m.MaxVisibleInputMode = true; m.InputBuffer = "1000"; m.CursorPos = 4 },
		"move":            func(m *Model) { m.MoveMode = true },
		"conflict":        func(m *Model) { m.ConflictDiffMode = true; m.ConflictPending = true },
		"versions":        func(m *Model) { m.VersionsMode = true; m.VersionsCursor = 30 },
		"restore-confirm": func(m *Model) { m.VersionsMode = true; m.VersionsCursor = 30; m.VersionsConfirmMode = true },
	}
	for _, size := range inlineLayoutSizes {
		for name, enter := range modes {
			t.Run(fmt.Sprintf("%dx%d/%s", size[0], size[1], name), func(t *testing.T) {
				m := richLayoutModel()
				m.TermWidth, m.TermHeight = size[0], size[1]
				m.MaxVisibleOverride, m.SelectedIndex = 1000, 80
				m.InputBuffer = strings.Repeat("long 世界 input ", 40)
				m.CursorPos = len(m.InputBuffer)
				m.ConflictLocalContent = strings.Repeat("local 世界 ", 120) + "\n"
				m.ConflictDiskContent = strings.Repeat("disk café ", 120) + "\n"
				m.Config().ReadVersionFunc = func(string, int64) (string, error) { return "- [ ] previous version\n", nil }
				for i := 0; i < 60; i++ {
					label := fmt.Sprintf("Entry %d %s", i, strings.Repeat("世界", 30))
					m.ViewNames = append(m.ViewNames, label)
					m.AvailableThemes = append(m.AvailableThemes, label)
					m.AvailableTags = append(m.AvailableTags, label)
					m.RecentFiles = append(m.RecentFiles, config.RecentFile{Path: "/tmp/" + label + ".md", AccessCount: 30})
					m.VersionsList = append(m.VersionsList, VersionInfo{ID: int64(i + 1), CreatedAt: time.Unix(0, 0)})
				}
				enter(&m)
				assertInlineFrameBounds(t, m)
			})
		}
	}
}

func TestInlineLayoutResizesWithoutLosingSelection(t *testing.T) {
	m := richLayoutModel()
	m.SelectedIndex = 80
	m.MaxVisibleOverride = 1000
	for _, size := range [][2]int{{200, 60}, {24, 8}, {120, 40}, {80, 24}, {24, 8}} {
		hadSize := m.TermWidth > 0 && m.TermHeight > 0
		next, command := m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		if hadSize && command == nil {
			t.Errorf("actual resize to %dx%d did not request clearing the previous frame", size[0], size[1])
		}
		if !hadSize && command != nil {
			t.Error("first size notification unnecessarily cleared terminal")
		}
		m = next.(Model)
		plain := assertInlineFrameBounds(t, m)
		if !strings.Contains(plain, "T040a") {
			t.Errorf("resize to %dx%d hid selection: %q", size[0], size[1], plain)
		}
		sameSize, command := m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		if command != nil {
			t.Errorf("repeated %dx%d notification unnecessarily cleared terminal", size[0], size[1])
		}
		if sameSize.(Model).SelectedIndex != m.SelectedIndex {
			t.Error("repeated size notification changed selection")
		}
		if sameSize.(Model).View().Content != m.View().Content {
			t.Error("repeated size notification changed the frame")
		}
	}
}

func TestInlineLayoutFrameHeightStableAcrossModeTransitions(t *testing.T) {
	for _, size := range inlineLayoutSizes {
		t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
			m := richLayoutModel()
			m.TermWidth, m.TermHeight = size[0], size[1]
			m.SelectedIndex, m.MaxVisibleOverride = 80, 1000
			assertInlineFrameBounds(t, m)
			// Main -> section picker -> focused main -> tag overlay -> focused
			// main -> all tasks -> help -> main must use the same redraw height.
			for _, key := range []rune{'s', tea.KeyEnter, 't', tea.KeyEsc, 'S', '?', tea.KeyEsc} {
				m = viewKey(m, key)
				plain := assertInlineFrameBounds(t, m)
				if !m.SectionsMode && !m.FilterMode && !m.HelpMode && !strings.Contains(plain, "T040a") {
					t.Errorf("mode transition %q hid selected task: %q", key, plain)
				}
			}
		})
	}
}

func TestInlineLayoutUnsizedPipedViewStillRenders(t *testing.T) {
	m := testModelWithMarkdown("# Work\n\n- [ ] PIPED_TASK\n")
	if got := m.View().Content; !strings.Contains(ansi.Strip(got), "PIPED_TASK") {
		t.Fatalf("unsized piped view disappeared: %q", got)
	}
}

func TestInlineLayoutInteractiveStartupWaitsForSize(t *testing.T) {
	m := testModelWithMarkdown("- [ ] STARTUP_TASK\n")
	m.waitForSize = true
	for _, size := range [][2]int{{0, 0}, {80, 0}, {0, 24}} {
		m.TermWidth, m.TermHeight = size[0], size[1]
		if got := m.View().Content; got != "" {
			t.Fatalf("interactive startup rendered before both dimensions arrived: %q", got)
		}
	}
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = next.(Model)
	if plain := assertInlineFrameBounds(t, m); !strings.Contains(plain, "STARTUP_TASK") {
		t.Fatalf("first sized frame omitted task: %q", plain)
	}
}

func TestInlineLayoutUnicodeAndOSCLinks(t *testing.T) {
	for _, size := range inlineLayoutSizes {
		for _, wrap := range []bool{false, true} {
			t.Run(fmt.Sprintf("%dx%d/wrap%t", size[0], size[1], wrap), func(t *testing.T) {
				m := testModelWithMarkdown("- [ ] LINK [世界café👩🏽‍💻](https://example.com/" + strings.Repeat("long-path/", 80) + ") " + strings.Repeat("世界 café 👩🏽‍💻 ", 40) + "\n")
				m.TermWidth, m.TermHeight = size[0], size[1]
				m.WordWrap = wrap
				m.Config().Display.SelectMarker = "👉🏽"
				m.Config().Display.CheckSymbol = "✓"
				m.FileModel.Todos[0].Checked = true
				plain := assertInlineFrameBounds(t, m)
				if !strings.Contains(plain, "LINK") {
					t.Fatalf("selected task label was hidden by Unicode marker or hyperlink width: %q", plain)
				}
				if strings.Contains(plain, "https://example.com") {
					t.Fatal("OSC hyperlink destination leaked into display columns")
				}
			})
		}
	}
}

func TestInlineLayoutVerySmallTerminalStillBoundsFrame(t *testing.T) {
	for _, size := range [][2]int{{1, 1}, {2, 2}, {8, 3}, {12, 4}} {
		m := testModelWithMarkdown("- [ ] tiny\n")
		m.TermWidth, m.TermHeight = size[0], size[1]
		assertInlineFrameBounds(t, m)
		m.HelpMode = true
		assertInlineFrameBounds(t, m)
	}
}
