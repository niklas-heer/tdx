package main

import (
	"os"
	"strings"
	"testing"
)

// TestE2E_DailyWorkflow simulates a realistic day of using tdx
func TestE2E_DailyWorkflow(t *testing.T) {
	file := tempTestFile(t)

	// Morning: Create today's tasks
	runCLI(t, file, "add", "Review pull requests")
	runCLI(t, file, "add", "Update documentation")
	runCLI(t, file, "add", "Fix bug #123")
	runCLI(t, file, "add", "Team meeting at 2pm")

	// Start working - complete first task
	runCLI(t, file, "toggle", "1")

	// Realize need to add urgent task
	runCLI(t, file, "add", "URGENT: Deploy hotfix")

	// During meeting, mark it done
	runCLI(t, file, "toggle", "4")

	// Realize documentation needs more detail
	runCLI(t, file, "edit", "2", "Update API documentation with new endpoints")

	// Complete urgent task
	runCLI(t, file, "toggle", "5")

	// End of day: verify state
	todos := getTodos(t, file)
	if len(todos) != 5 {
		t.Fatalf("Expected 5 tasks, got %d", len(todos))
	}

	// Count completed tasks
	completed := 0
	for _, todo := range todos {
		if strings.HasPrefix(todo, "- [x] ") {
			completed++
		}
	}

	if completed != 3 {
		t.Errorf("Expected 3 completed tasks, got %d", completed)
	}

	// Verify specific edits preserved
	if !strings.Contains(strings.Join(todos, "\n"), "API documentation with new endpoints") {
		t.Error("Edit not preserved")
	}
}

// TestE2E_ProjectManagement simulates managing a project with sections
func TestE2E_ProjectManagement(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Project: Website Redesign

## Phase 1: Planning

- [ ] Gather requirements
- [ ] Create mockups
- [ ] Get stakeholder approval

## Phase 2: Development

- [ ] Set up development environment
- [ ] Implement new design
- [ ] Write tests

## Phase 3: Launch

- [ ] Deploy to staging
- [ ] QA testing
- [ ] Deploy to production`

	_ = os.WriteFile(file, []byte(initial), 0644)

	// Complete planning phase
	runCLI(t, file, "toggle", "1")
	runCLI(t, file, "toggle", "2")
	runCLI(t, file, "toggle", "3")

	// Start development
	runCLI(t, file, "toggle", "4")
	runCLI(t, file, "toggle", "5")

	// Add forgotten task
	runCLI(t, file, "add", "Security review")

	content, _ := os.ReadFile(file)
	result := string(content)

	// All sections should be preserved
	if !strings.Contains(result, "## Phase 1: Planning") {
		t.Error("Lost Phase 1 section")
	}
	if !strings.Contains(result, "## Phase 2: Development") {
		t.Error("Lost Phase 2 section")
	}
	if !strings.Contains(result, "## Phase 3: Launch") {
		t.Error("Lost Phase 3 section")
	}

	// Verify progress
	todos := getTodos(t, file)
	completed := 0
	for _, todo := range todos {
		if strings.HasPrefix(todo, "- [x] ") {
			completed++
		}
	}

	if completed != 5 {
		t.Errorf("Expected 5 completed tasks, got %d", completed)
	}
}

// TestE2E_WeeklyCleanup simulates weekly cleanup routine
func TestE2E_WeeklyCleanup(t *testing.T) {
	file := tempTestFile(t)

	// Week's accumulation of tasks
	runCLI(t, file, "add", "Monday task")
	runCLI(t, file, "add", "Tuesday task")
	runCLI(t, file, "add", "Wednesday task")
	runCLI(t, file, "add", "Thursday task")
	runCLI(t, file, "add", "Friday task")

	// Some completed during week
	runCLI(t, file, "toggle", "1")
	runCLI(t, file, "toggle", "3")
	runCLI(t, file, "toggle", "4")

	// Friday: clean up completed tasks
	runPiped(t, file, ":clear-done\r")

	todos := getTodos(t, file)
	if len(todos) != 2 {
		t.Errorf("Expected 2 remaining tasks after cleanup, got %d", len(todos))
	}

	// Remaining should be Tuesday and Friday
	if !strings.Contains(todos[0], "Tuesday task") {
		t.Error("Wrong task remained after cleanup")
	}
	if !strings.Contains(todos[1], "Friday task") {
		t.Error("Wrong task remained after cleanup")
	}
}

// TestE2E_CollaborativeDocument simulates a shared todo list
func TestE2E_CollaborativeDocument(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Team Tasks

**Instructions:** Check off your tasks when done!

## Alice's Tasks
- [ ] Review PRs
- [ ] Update dependencies

## Bob's Tasks
- [ ] Fix CI pipeline
- [ ] Write release notes

## Shared Tasks
- [ ] Team retrospective
- [ ] Update roadmap

---

*Last updated: 2024-01-15*`

	_ = os.WriteFile(file, []byte(initial), 0644)

	// Alice completes her tasks
	runCLI(t, file, "toggle", "1")
	runCLI(t, file, "toggle", "2")

	// Bob completes one task
	runCLI(t, file, "toggle", "3")

	// Team completes shared task
	runCLI(t, file, "toggle", "5")

	content, _ := os.ReadFile(file)
	result := string(content)

	// All sections and metadata preserved
	if !strings.Contains(result, "**Instructions:**") {
		t.Error("Lost instructions")
	}
	if !strings.Contains(result, "## Alice's Tasks") {
		t.Error("Lost Alice's section")
	}
	if !strings.Contains(result, "## Bob's Tasks") {
		t.Error("Lost Bob's section")
	}
	if !strings.Contains(result, "---") {
		t.Error("Lost horizontal rule")
	}
	if !strings.Contains(result, "*Last updated:") {
		t.Error("Lost timestamp")
	}
}

// TestE2E_MigrationFromPlainMarkdown simulates converting existing markdown
func TestE2E_MigrationFromPlainMarkdown(t *testing.T) {
	file := tempTestFile(t)

	// Existing markdown file with various content
	initial := `# My Notes

Some random thoughts and ideas.

## Todo

Here are things I need to do:
- Buy groceries
- Call mom
- Fix the leaky faucet

## Ideas

- Start a blog
- Learn a new language
- Take up photography

## References

[Useful link](https://example.com)
> Important quote here`

	_ = os.WriteFile(file, []byte(initial), 0644)

	// Convert regular list items to task items manually
	content, _ := os.ReadFile(file)
	markdown := string(content)
	markdown = strings.ReplaceAll(markdown, "- Buy groceries", "- [ ] Buy groceries")
	markdown = strings.ReplaceAll(markdown, "- Call mom", "- [ ] Call mom")
	markdown = strings.ReplaceAll(markdown, "- Fix the leaky faucet", "- [ ] Fix the leaky faucet")
	_ = os.WriteFile(file, []byte(markdown), 0644)

	// Now use tdx to manage
	runCLI(t, file, "toggle", "1")
	runCLI(t, file, "add", "Water the plants")

	content, _ = os.ReadFile(file)
	result := string(content)

	// All original content preserved
	if !strings.Contains(result, "Some random thoughts and ideas.") {
		t.Error("Lost original content")
	}
	if !strings.Contains(result, "## Ideas") {
		t.Error("Lost Ideas section")
	}
	if !strings.Contains(result, "[Useful link]") {
		t.Error("Lost reference link")
	}
	if !strings.Contains(result, "> Important quote") {
		t.Error("Lost blockquote")
	}

	todos := getTodos(t, file)
	if len(todos) != 4 {
		t.Errorf("Expected 4 tasks, got %d", len(todos))
	}
}

// TestE2E_BugTracking simulates using tdx for bug tracking
func TestE2E_BugTracking(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Bug Tracker

## Critical Bugs 🔥
- [ ] Login fails on Safari [Bug #401](https://issues.example.com/401)
- [ ] Data loss on network error [Bug #402](https://issues.example.com/402)

## High Priority
- [ ] Slow dashboard load [Bug #403](https://issues.example.com/403)

## Medium Priority
- [ ] Minor UI glitch [Bug #404](https://issues.example.com/404)

## Investigation Notes

**Bug #401**: Reproduced on Safari 16.x, works on Chrome.
**Bug #402**: Need to add retry logic.`

	_ = os.WriteFile(file, []byte(initial), 0644)

	// Fix critical bugs
	runCLI(t, file, "toggle", "1")
	runCLI(t, file, "toggle", "2")

	// Add new bug
	runCLI(t, file, "add", "Search not working [Bug #405](https://issues.example.com/405)")

	content, _ := os.ReadFile(file)
	result := string(content)

	// Verify structure preserved
	if !strings.Contains(result, "## Critical Bugs 🔥") {
		t.Error("Lost critical bugs section")
	}
	if !strings.Contains(result, "## Investigation Notes") {
		t.Error("Lost investigation notes")
	}
	if !strings.Contains(result, "**Bug #401**:") {
		t.Error("Lost bug notes")
	}

	// Verify emoji preserved
	if !strings.Contains(result, "🔥") {
		t.Error("Lost emoji")
	}
}

// TestE2E_RecipeChecklist simulates using tdx for recipe instructions
func TestE2E_RecipeChecklist(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Chocolate Chip Cookies 🍪

## Ingredients
- 2 cups flour
- 1 cup butter
- 1 cup chocolate chips

## Instructions

- [ ] Preheat oven to 350°F
- [ ] Mix dry ingredients
- [ ] Cream butter and sugar
- [ ] Combine wet and dry ingredients
- [ ] Add chocolate chips
- [ ] Bake for 12 minutes
- [ ] Let cool for 5 minutes

## Notes
Best served warm with milk!`

	_ = os.WriteFile(file, []byte(initial), 0644)

	// Check off steps as cooking
	for i := 1; i <= 7; i++ {
		runCLI(t, file, "toggle", string(rune('0'+i)))
	}

	content, _ := os.ReadFile(file)
	result := string(content)

	// All recipe content preserved
	if !strings.Contains(result, "## Ingredients") {
		t.Error("Lost ingredients section")
	}
	if !strings.Contains(result, "## Instructions") {
		t.Error("Lost instructions section")
	}
	if !strings.Contains(result, "Best served warm") {
		t.Error("Lost notes")
	}

	// All tasks checked
	todos := getTodos(t, file)
	for i, todo := range todos {
		if !strings.HasPrefix(todo, "- [x] ") {
			t.Errorf("Step %d not checked: %s", i+1, todo)
		}
	}
}

// TestE2E_ErrorRecovery tests handling of interrupted operations
func TestE2E_ErrorRecovery(t *testing.T) {
	file := tempTestFile(t)

	// Create initial tasks
	runCLI(t, file, "add", "Task 1")
	runCLI(t, file, "add", "Task 2")
	runCLI(t, file, "add", "Task 3")

	// Save current state
	beforeContent, _ := os.ReadFile(file)

	// Make some changes in TUI with undo
	runPiped(t, file, " u") // Toggle and undo

	afterContent, _ := os.ReadFile(file)

	// State should be reverted
	if string(beforeContent) != string(afterContent) {
		t.Error("Undo did not restore previous state")
	}

	// Try invalid operations (should not crash or corrupt)
	runCLI(t, file, "toggle", "99")                  // Invalid index
	runCLI(t, file, "delete", "99")                  // Invalid index
	runCLI(t, file, "edit", "99", "Should not work") // Invalid index

	todos := getTodos(t, file)
	if len(todos) != 3 {
		t.Error("Invalid operations corrupted the file")
	}
}

// TestE2E_LargeDocument tests performance with large documents
func TestE2E_LargeDocument(t *testing.T) {
	file := tempTestFile(t)

	// Create a document with lots of content
	var content strings.Builder
	content.WriteString("# Large Project\n\n")

	for section := 1; section <= 10; section++ {
		content.WriteString("## Section " + string(rune('0'+section)) + "\n\n")
		content.WriteString("Some description for this section.\n\n")

		for task := 1; task <= 10; task++ {
			content.WriteString("- [ ] Task " + string(rune('0'+task)) + " in section " + string(rune('0'+section)) + "\n")
		}

		content.WriteString("\nNotes for this section.\n\n")
	}

	_ = os.WriteFile(file, []byte(content.String()), 0644)

	// Should have 100 tasks
	todos := getTodos(t, file)
	if len(todos) != 100 {
		t.Errorf("Expected 100 tasks, got %d", len(todos))
	}

	// Operations should still work efficiently
	runCLI(t, file, "toggle", "50")
	runCLI(t, file, "edit", "25", "Modified task")
	runCLI(t, file, "delete", "75")

	todos = getTodos(t, file)
	if len(todos) != 99 {
		t.Errorf("Expected 99 tasks after delete, got %d", len(todos))
	}
}

// TestE2E_MarkdownFeatures tests various markdown features together
func TestE2E_MarkdownFeatures(t *testing.T) {
	file := tempTestFile(t)

	initial := `# Feature Demo

## Formatting

- [ ] Task with **bold** text
- [ ] Task with *italic* text
- [ ] Task with ` + "`code`" + ` inline
- [ ] Task with [link](https://example.com)

## Code Example

` + "```go" + `
func main() {
    fmt.Println("Hello")
}
` + "```" + `

## Quote

> This is an important note
> that spans multiple lines

---

## Lists

Regular list:
1. First item
2. Second item

---

*Last updated: 2024*`

	_ = os.WriteFile(file, []byte(initial), 0644)

	// Operate on tasks
	runCLI(t, file, "toggle", "1")
	runCLI(t, file, "toggle", "3")
	runCLI(t, file, "add", "Task with **mixed** *formatting* and `code`")

	content, _ := os.ReadFile(file)
	result := string(content)

	// Verify all markdown features preserved (note: numbered lists may be normalized)
	features := []string{
		"**bold**", "*italic*", "`code`", "[link]",
		"```go", "func main()", "```",
		"> This is an important note",
		"---",
		"First item", // List content preserved even if numbering changes
		"*Last updated:",
	}

	for _, feature := range features {
		if !strings.Contains(result, feature) {
			t.Errorf("Lost markdown feature: %s", feature)
		}
	}
}
