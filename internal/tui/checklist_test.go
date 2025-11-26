package tui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/niklas-heer/tdx/internal/markdown"
)

// TestChecklistTapeSequence reproduces the sequence from checklists.tape
// to find what triggers a panic
func TestChecklistTapeSequence(t *testing.T) {
	content := `---
show-headings: true
filter-done: false
---

# Weekly Sprint Checklist

## Monday
- [ ] Review pending PRs #review #urgent
- [ ] Team standup notes #meeting
- [x] Check CI pipeline #devops

## Tuesday
- [ ] Write unit tests #testing
- [ ] Code review session #review
- [x] Update documentation #docs

## Wednesday
- [ ] Deploy to staging #devops #urgent
- [ ] Performance testing #testing
`
	fm := markdown.ParseMarkdown(content)

	// Apply frontmatter settings from parsed metadata
	showHeadings := true // Explicitly set to true as per frontmatter
	if fm.Metadata != nil && fm.Metadata.ShowHeadings != nil {
		showHeadings = *fm.Metadata.ShowHeadings
	}

	m := New("/tmp/checklist.md", fm, false, showHeadings, -1, testConfig(), testStyles(), "")
	m.TermWidth = 80

	t.Log("Initial state:")
	t.Logf("  Todos: %d", len(m.FileModel.Todos))
	t.Logf("  ShowHeadings: %v", m.ShowHeadings)
	t.Logf("  SelectedIndex: %d", m.SelectedIndex)

	// Helper to render and check for panic
	renderView := func(step string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("PANIC at %s: %v", step, r)
			}
		}()
		_ = m.View()
	}

	renderView("initial")

	// Step 1: gg - go to top
	t.Log("Step 1: gg (go to top)")
	m = pressKey(t, m, "g")
	m = pressKey(t, m, "g")
	renderView("step1")

	// Step 2: n then Escape (try to add, then cancel)
	t.Log("Step 2: n then Escape")
	m = pressKey(t, m, "n")
	renderView("step2-n")
	m = pressKeyType(t, m, tea.KeyEsc)
	renderView("step2-esc")

	// Step 3: Navigate down with j (4 times)
	t.Log("Step 3: Navigate down 4 times")
	for i := 0; i < 4; i++ {
		m = pressKey(t, m, "j")
		renderView(fmt.Sprintf("step3-j%d", i+1))
		t.Logf("  After j #%d: SelectedIndex=%d", i+1, m.SelectedIndex)
	}

	// Step 4: Toggle with space
	t.Log("Step 4: Toggle with space")
	m = pressKey(t, m, " ")
	renderView("step4")

	// Step 5: N then Escape (try to add at end, then cancel)
	t.Log("Step 5: N then Escape")
	m = pressKey(t, m, "N")
	renderView("step5-N")
	m = pressKeyType(t, m, tea.KeyEsc)
	renderView("step5-esc")

	// Step 6: gg then navigate down 4 times
	t.Log("Step 6: gg then navigate down 4 times")
	m = pressKey(t, m, "g")
	m = pressKey(t, m, "g")
	for i := 0; i < 4; i++ {
		m = pressKey(t, m, "j")
		renderView(fmt.Sprintf("step6-j%d", i+1))
	}

	// Step 7: Toggle with space
	t.Log("Step 7: Toggle with space")
	m = pressKey(t, m, " ")
	renderView("step7")

	// Step 8: Navigate down 2 times then toggle
	t.Log("Step 8: Navigate down 2 times then toggle")
	m = pressKey(t, m, "j")
	m = pressKey(t, m, "j")
	m = pressKey(t, m, " ")
	renderView("step8")

	// Step 9: Filter mode - f, j, j, space
	t.Log("Step 9: Tag filter mode")
	m = pressKey(t, m, "f")
	renderView("step9-f")
	if !m.FilterMode {
		t.Log("  FilterMode not activated (no tags?)")
	} else {
		m = pressKey(t, m, "j")
		m = pressKey(t, m, "j")
		m = pressKey(t, m, " ")
		renderView("step9-filter")
	}

	// Step 10: Clear filter - f, c, Escape
	t.Log("Step 10: Clear filter")
	m = pressKey(t, m, "f")
	if m.FilterMode {
		m = pressKey(t, m, "c")
		m = pressKeyType(t, m, tea.KeyEsc)
		renderView("step10")
	}

	// Step 11: gg then n to add new item
	t.Log("Step 11: gg then add new item")
	m = pressKey(t, m, "g")
	m = pressKey(t, m, "g")
	m = pressKey(t, m, "n")
	renderView("step11-n")
	// Type some text
	for _, c := range "Morning coffee" {
		m = pressKey(t, m, string(c))
	}
	m = pressKeyType(t, m, tea.KeyEnter)
	renderView("step11-enter")
	t.Logf("  After adding: %d todos", len(m.FileModel.Todos))

	// Step 12: Command palette - filter-done
	t.Log("Step 12: filter-done command")
	m = pressKey(t, m, ":")
	for _, c := range "filter-done" {
		m = pressKey(t, m, string(c))
	}
	m = pressKeyType(t, m, tea.KeyEnter)
	renderView("step12")
	t.Logf("  FilterDone: %v", m.FilterDone)

	// Step 13: N then Escape
	t.Log("Step 13: N then Escape")
	m = pressKey(t, m, "N")
	renderView("step13-N")
	m = pressKeyType(t, m, tea.KeyEsc)
	renderView("step13-esc")

	// Step 14: Toggle filter-done again
	t.Log("Step 14: filter-done command again")
	m = pressKey(t, m, ":")
	for _, c := range "filter-done" {
		m = pressKey(t, m, string(c))
	}
	m = pressKeyType(t, m, tea.KeyEnter)
	renderView("step14")
	t.Logf("  FilterDone: %v", m.FilterDone)

	// Step 15: N then Escape
	t.Log("Step 15: N then Escape")
	m = pressKey(t, m, "N")
	renderView("step15-N")
	m = pressKeyType(t, m, tea.KeyEsc)
	renderView("step15-esc")

	// Step 16: check-all command - THIS IS WHERE PANIC HAPPENS IN GIF
	t.Log("Step 16: check-all command")
	m = pressKey(t, m, ":")
	for _, c := range "check-all" {
		m = pressKey(t, m, string(c))
	}
	m = pressKeyType(t, m, tea.KeyEnter)
	t.Logf("  After check-all: SelectedIndex=%d, Todos=%d", m.SelectedIndex, len(m.FileModel.Todos))
	for i, todo := range m.FileModel.Todos {
		t.Logf("    Todo[%d]: %v - %s", i, todo.Checked, todo.Text)
	}
	renderView("step16-check-all")

	// Step 17: N then Escape
	t.Log("Step 17: N then Escape")
	m = pressKey(t, m, "N")
	renderView("step17-N")
	m = pressKeyType(t, m, tea.KeyEsc)
	renderView("step17-esc")

	// Step 18: uncheck-all command
	t.Log("Step 18: uncheck-all command")
	m = pressKey(t, m, ":")
	for _, c := range "uncheck-all" {
		m = pressKey(t, m, string(c))
	}
	m = pressKeyType(t, m, tea.KeyEnter)
	t.Logf("  After uncheck-all: SelectedIndex=%d", m.SelectedIndex)
	renderView("step18-uncheck-all")

	// Step 19: Help then Escape
	t.Log("Step 19: Help then Escape")
	m = pressKey(t, m, "?")
	renderView("step19-help")
	m = pressKeyType(t, m, tea.KeyEsc)
	renderView("step19-esc")

	// Step 20: Escape to quit
	t.Log("Step 20: Escape to quit")
	// Don't actually quit in test
	renderView("step20")

	t.Log("Test completed successfully - no panic!")
}

// Helper functions
func pressKey(t *testing.T, m Model, key string) Model {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	result, _ := m.Update(msg)
	return result.(Model)
}

func pressKeyType(t *testing.T, m Model, keyType tea.KeyType) Model {
	msg := tea.KeyMsg{Type: keyType}
	result, _ := m.Update(msg)
	return result.(Model)
}

// executeCommand directly executes a command by name (bypasses command mode debouncing)
func executeCommand(m *Model, cmdName string) {
	for _, cmd := range m.Commands {
		if cmd.Name == cmdName {
			cmd.Handler(m)
			return
		}
	}
}

// TestChecklistWithShowHeadingsNavigation tests navigation with headings enabled
func TestChecklistWithShowHeadingsNavigation(t *testing.T) {
	content := `# Checklist

## Section 1
- [ ] Task 1
- [ ] Task 2

## Section 2
- [ ] Task 3
- [ ] Task 4
`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, true, -1, testConfig(), testStyles(), "") // showHeadings=true
	m.TermWidth = 80

	t.Logf("Initial SelectedIndex: %d", m.SelectedIndex)
	t.Logf("Todos: %d", len(m.FileModel.Todos))

	// Navigate through all items
	for i := 0; i < len(m.FileModel.Todos)+2; i++ {
		m = pressKey(t, m, "j")
		t.Logf("After j #%d: SelectedIndex=%d", i+1, m.SelectedIndex)
	}

	// Navigate back up
	for i := 0; i < len(m.FileModel.Todos)+2; i++ {
		m = pressKey(t, m, "k")
		t.Logf("After k #%d: SelectedIndex=%d", i+1, m.SelectedIndex)
	}
}

// TestCheckAllThenN tests the exact sequence that triggers panic in VHS
// check-all command followed by N (append new task)
func TestCheckAllThenN(t *testing.T) {
	content := `---
show-headings: true
filter-done: false
---

# Weekly Sprint Checklist

## Monday
- [ ] Review pending PRs #review #urgent
- [ ] Team standup notes #meeting
- [x] Check CI pipeline #devops

## Tuesday
- [ ] Write unit tests #testing
- [ ] Code review session #review
- [x] Update documentation #docs

## Wednesday
- [ ] Deploy to staging #devops #urgent
- [ ] Performance testing #testing
`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/checklist.md", fm, false, true, -1, testConfig(), testStyles(), "")
	m.TermWidth = 80

	t.Logf("Initial: Todos=%d, SelectedIndex=%d, ShowHeadings=%v",
		len(m.FileModel.Todos), m.SelectedIndex, m.ShowHeadings)

	// Render initial view
	view := m.View()
	t.Logf("Initial view rendered OK, length=%d", len(view))

	// Execute check-all command
	t.Log("Executing :check-all")
	m = pressKey(t, m, ":")
	for _, c := range "check-all" {
		m = pressKey(t, m, string(c))
	}
	m = pressKeyType(t, m, tea.KeyEnter)

	t.Logf("After check-all: SelectedIndex=%d", m.SelectedIndex)
	for i, todo := range m.FileModel.Todos {
		t.Logf("  Todo[%d]: checked=%v, text=%s", i, todo.Checked, todo.Text)
	}

	// Render after check-all
	view = m.View()
	t.Logf("View after check-all OK, length=%d", len(view))

	// Press N (append new task at end) - THIS IS THE CRITICAL MOMENT
	t.Log("Pressing N (append new task)")
	m = pressKey(t, m, "N")

	t.Logf("After N: InputMode=%v, InsertAfterCursor=%v, SelectedIndex=%d",
		m.InputMode, m.InsertAfterCursor, m.SelectedIndex)

	// This is where the panic likely occurs - rendering in input mode after check-all
	t.Log("Rendering view in input mode...")
	view = m.View()
	t.Logf("View after N OK, length=%d", len(view))

	// Press Escape to cancel
	t.Log("Pressing Escape")
	m = pressKeyType(t, m, tea.KeyEsc)

	view = m.View()
	t.Logf("View after Escape OK, length=%d", len(view))

	t.Log("Test passed - no panic!")
}

// TestCheckAllWithFilterDoneThenN tests the EXACT VHS sequence that might trigger panic
// This simulates: filter-done ON -> filter-done OFF -> check-all -> N
func TestCheckAllWithFilterDoneThenN(t *testing.T) {
	content := `---
show-headings: true
filter-done: false
---

# Weekly Sprint Checklist

## Monday
- [ ] Review pending PRs #review #urgent
- [ ] Team standup notes #meeting
- [x] Check CI pipeline #devops

## Tuesday
- [ ] Write unit tests #testing
- [ ] Code review session #review
- [x] Update documentation #docs

## Wednesday
- [ ] Deploy to staging #devops #urgent
- [ ] Performance testing #testing
`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/checklist.md", fm, false, true, -1, testConfig(), testStyles(), "")
	m.TermWidth = 80

	t.Logf("Initial: Todos=%d, FilterDone=%v", len(m.FileModel.Todos), m.FilterDone)

	// Step 1: Toggle filter-done ON (using direct command execution)
	t.Log("Step 1: filter-done ON")
	executeCommand(&m, "filter-done")
	t.Logf("  FilterDone=%v", m.FilterDone)
	_ = m.View() // render

	// Step 2: Toggle filter-done OFF
	t.Log("Step 2: filter-done OFF")
	executeCommand(&m, "filter-done")
	t.Logf("  FilterDone=%v", m.FilterDone)
	_ = m.View() // render

	// Step 3: check-all
	t.Log("Step 3: check-all")
	executeCommand(&m, "check-all")
	t.Logf("  All checked, FilterDone=%v, SelectedIndex=%d", m.FilterDone, m.SelectedIndex)

	// Verify all are checked
	allChecked := true
	for _, todo := range m.FileModel.Todos {
		if !todo.Checked {
			allChecked = false
			break
		}
	}
	t.Logf("  All todos checked: %v", allChecked)

	view := m.View()
	t.Logf("  View after check-all: length=%d", len(view))

	// Step 4: Press N - this is where it might panic
	t.Log("Step 4: Press N (append mode)")
	m = pressKey(t, m, "N")
	t.Logf("  InputMode=%v, InsertAfterCursor=%v, SelectedIndex=%d",
		m.InputMode, m.InsertAfterCursor, m.SelectedIndex)

	// This render might panic
	t.Log("Step 5: Render in input mode")
	view = m.View()
	t.Logf("  View in input mode: length=%d", len(view))

	// Type some text
	t.Log("Step 6: Type text")
	for _, c := range "# Reset checklist" {
		m = pressKey(t, m, string(c))
	}
	view = m.View()
	t.Logf("  View after typing: length=%d", len(view))

	// Escape
	t.Log("Step 7: Escape")
	m = pressKeyType(t, m, tea.KeyEsc)
	view = m.View()
	t.Logf("  View after escape: length=%d", len(view))

	t.Log("Test passed - no panic!")
}

// TestCheckAllWithFilterDoneON tests what happens when filter-done is ON and we check-all
// This is a potential panic scenario: all todos checked + filter-done ON = empty todosToShow
func TestCheckAllWithFilterDoneON(t *testing.T) {
	content := `---
show-headings: true
---

# Checklist

## Section 1
- [ ] Task 1
- [ ] Task 2

## Section 2
- [ ] Task 3
- [ ] Task 4
`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/checklist.md", fm, false, true, -1, testConfig(), testStyles(), "")
	m.TermWidth = 80

	t.Logf("Initial: Todos=%d, FilterDone=%v", len(m.FileModel.Todos), m.FilterDone)
	_ = m.View()

	// Turn ON filter-done
	t.Log("Step 1: filter-done ON")
	executeCommand(&m, "filter-done")
	t.Logf("  FilterDone=%v", m.FilterDone)
	_ = m.View()

	// check-all - now ALL todos are checked AND filter-done is ON
	// This means todosToShow will be EMPTY!
	t.Log("Step 2: check-all (with filter-done ON)")
	executeCommand(&m, "check-all")
	t.Logf("  FilterDone=%v, SelectedIndex=%d", m.FilterDone, m.SelectedIndex)

	// This render might panic because todosToShow is empty
	t.Log("Step 3: Render after check-all with filter-done ON")
	view := m.View()
	t.Logf("  View length=%d", len(view))

	// Press N (append mode) - potential panic
	t.Log("Step 4: Press N")
	m = pressKey(t, m, "N")
	t.Logf("  InputMode=%v, SelectedIndex=%d", m.InputMode, m.SelectedIndex)

	// Render in input mode with empty todosToShow
	t.Log("Step 5: Render in input mode")
	view = m.View()
	t.Logf("  View length=%d", len(view))

	t.Log("Test passed - no panic!")
}

// TestChecklistFilterDoneWithHeadings tests the combination that might cause issues
func TestChecklistFilterDoneWithHeadings(t *testing.T) {
	content := `# Checklist

## Section 1
- [ ] Task 1
- [x] Task 2 done

## Section 2
- [ ] Task 3
- [x] Task 4 done
`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/test.md", fm, false, true, -1, testConfig(), testStyles(), "") // showHeadings=true
	m.TermWidth = 80

	// Toggle filter-done
	t.Log("Toggling filter-done ON")
	m.FilterDone = true
	m.InvalidateDocumentTree()
	m.adjustSelectionForFilter()

	t.Logf("SelectedIndex after filter: %d", m.SelectedIndex)

	// Try to render
	view := m.View()
	t.Logf("View length: %d", len(view))

	// Toggle filter-done off
	t.Log("Toggling filter-done OFF")
	m.FilterDone = false
	m.InvalidateDocumentTree()

	view = m.View()
	t.Logf("View length after filter off: %d", len(view))

	// Navigate
	for i := 0; i < 4; i++ {
		m = pressKey(t, m, "j")
		t.Logf("After j #%d: SelectedIndex=%d", i+1, m.SelectedIndex)
	}
}

// TestExactVHSPanicSequence reproduces the EXACT sequence from the original VHS tape
// that was causing panics: N + comment text + Escape + check-all + N
func TestExactVHSPanicSequence(t *testing.T) {
	content := `---
show-headings: true
filter-done: false
---

# Weekly Sprint Checklist

## Monday
- [ ] Review pending PRs #review #urgent
- [ ] Team standup notes #meeting
- [x] Check CI pipeline #devops

## Tuesday
- [ ] Write unit tests #testing
- [ ] Code review session #review
- [x] Update documentation #docs

## Wednesday
- [ ] Deploy to staging #devops #urgent
- [ ] Performance testing #testing

## Commands
- [ ] echo 'Deploy complete!' && date
`
	fm := markdown.ParseMarkdown(content)
	m := New("/tmp/checklist.md", fm, false, true, -1, testConfig(), testStyles(), "")
	m.TermWidth = 80

	renderView := func(step string) {
		view := m.View()
		t.Logf("  [%s] View rendered OK, length=%d", step, len(view))
	}

	t.Logf("Initial: Todos=%d, SelectedIndex=%d", len(m.FileModel.Todos), m.SelectedIndex)
	renderView("initial")

	// VHS sequence from original tape:
	// 1. Navigate with gg, n (new task), type comment, Escape
	t.Log("Step 1: gg")
	m = pressKey(t, m, "g")
	m = pressKey(t, m, "g")
	renderView("after-gg")

	t.Log("Step 2: n (new task after cursor)")
	m = pressKey(t, m, "n")
	t.Logf("  InputMode=%v, InsertAfterCursor=%v", m.InputMode, m.InsertAfterCursor)
	renderView("after-n")

	t.Log("Step 3: Type comment text")
	for _, c := range "# Headings visible thanks to front matter" {
		m = pressKey(t, m, string(c))
	}
	renderView("after-typing")

	t.Log("Step 4: Escape (cancel input)")
	m = pressKeyType(t, m, tea.KeyEsc)
	t.Logf("  InputMode=%v, Todos=%d", m.InputMode, len(m.FileModel.Todos))
	renderView("after-escape1")

	// 2. Navigate, toggle items, filter
	t.Log("Step 5: Navigate jjj, space, jj, space")
	m = pressKey(t, m, "j")
	m = pressKey(t, m, "j")
	m = pressKey(t, m, "j")
	m = pressKey(t, m, " ")
	m = pressKey(t, m, "j")
	m = pressKey(t, m, "j")
	m = pressKey(t, m, " ")
	renderView("after-toggles")

	// 3. Filter by tag
	t.Log("Step 6: Filter by tag")
	m = pressKey(t, m, "f")
	m = pressKey(t, m, "j")
	m = pressKey(t, m, "j")
	m = pressKey(t, m, " ")
	m = pressKey(t, m, "f")
	m = pressKey(t, m, "c")
	m = pressKeyType(t, m, tea.KeyEsc)
	renderView("after-filter")

	// 4. gg, n + comment + Escape + Enter (add "Morning coffee ritual")
	t.Log("Step 7: gg, n, type, enter (add new task)")
	m = pressKey(t, m, "g")
	m = pressKey(t, m, "g")
	m = pressKey(t, m, "n")
	for _, c := range "Morning coffee ritual #urgent" {
		m = pressKey(t, m, string(c))
	}
	m = pressKeyType(t, m, tea.KeyEnter)
	t.Logf("  After adding task: Todos=%d", len(m.FileModel.Todos))
	renderView("after-add-task")

	// 5. Toggle filter-done on
	t.Log("Step 8: filter-done ON")
	executeCommand(&m, "filter-done")
	t.Logf("  FilterDone=%v", m.FilterDone)
	renderView("after-filter-done-on")

	// 6. N + comment + Escape (this is where VHS had Type "N")
	t.Log("Step 9: N (append mode), type comment, Escape")
	m = pressKey(t, m, "N")
	t.Logf("  InputMode=%v, InsertAfterCursor=%v", m.InputMode, m.InsertAfterCursor)
	for _, c := range "# Completed items now hidden" {
		m = pressKey(t, m, string(c))
	}
	m = pressKeyType(t, m, tea.KeyEsc)
	t.Logf("  After Escape: InputMode=%v, Todos=%d", m.InputMode, len(m.FileModel.Todos))
	renderView("after-N-escape")

	// 7. Toggle filter-done off
	t.Log("Step 10: filter-done OFF")
	executeCommand(&m, "filter-done")
	t.Logf("  FilterDone=%v", m.FilterDone)
	renderView("after-filter-done-off")

	// 8. N + comment + Escape BEFORE check-all (this was in the VHS tape)
	t.Log("Step 11: N, type '# Check all items at once', Escape")
	m = pressKey(t, m, "N")
	for _, c := range "# Check all items at once" {
		m = pressKey(t, m, string(c))
	}
	m = pressKeyType(t, m, tea.KeyEsc)
	renderView("after-comment-before-check-all")

	// 9. check-all - THIS IS THE CRITICAL COMMAND
	t.Log("Step 12: :check-all")
	executeCommand(&m, "check-all")
	t.Logf("  After check-all: SelectedIndex=%d", m.SelectedIndex)
	allChecked := true
	for i, todo := range m.FileModel.Todos {
		if !todo.Checked {
			allChecked = false
			t.Logf("  Todo[%d] NOT checked: %s", i, todo.Text)
		}
	}
	t.Logf("  All todos checked: %v", allChecked)
	renderView("after-check-all")

	// 10. N + comment + Escape AFTER check-all (THIS IS WHERE PANIC LIKELY OCCURS)
	t.Log("Step 13: N (after check-all), type comment, Escape")
	m = pressKey(t, m, "N")
	t.Logf("  InputMode=%v, InsertAfterCursor=%v, SelectedIndex=%d", m.InputMode, m.InsertAfterCursor, m.SelectedIndex)
	renderView("after-N-post-check-all")

	for _, c := range "# Reset checklist with uncheck-all" {
		m = pressKey(t, m, string(c))
	}
	renderView("after-typing-post-check-all")

	m = pressKeyType(t, m, tea.KeyEsc)
	t.Logf("  After Escape: InputMode=%v", m.InputMode)
	renderView("after-escape-post-check-all")

	// 11. uncheck-all
	t.Log("Step 14: :uncheck-all")
	executeCommand(&m, "uncheck-all")
	renderView("after-uncheck-all")

	// 12. More N + comment sequences
	t.Log("Step 15: N, type, Escape")
	m = pressKey(t, m, "N")
	for _, c := range "# Copy task to clipboard with 'c'" {
		m = pressKey(t, m, string(c))
	}
	m = pressKeyType(t, m, tea.KeyEsc)
	renderView("after-final-N")

	// 13. G (go to last)
	t.Log("Step 16: G (go to last)")
	m = pressKey(t, m, "G")
	renderView("after-G")

	// 14. c (copy)
	t.Log("Step 17: c (copy)")
	m = pressKey(t, m, "c")
	renderView("after-c")

	t.Log("Test passed - no panic!")
}
