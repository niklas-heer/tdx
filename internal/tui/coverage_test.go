package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/niklas-heer/tdx/internal/markdown"
)

// ==================== visible_tree.go tests ====================

func TestVisibleTree_GetSelectedTodo(t *testing.T) {
	tree := &VisibleTree{
		Nodes: []VisibleNode{
			{Type: NodeTypeHeading, TodoIndex: -1, Level: 1, Text: "Heading"},
			{Type: NodeTypeTodo, TodoIndex: 0, Level: 0, Text: "Task 1"},
			{Type: NodeTypeTodo, TodoIndex: 1, Level: 0, Text: "Task 2"},
		},
		SelectedIndex: 1,
	}

	node := tree.GetSelectedTodo()
	if node == nil {
		t.Fatal("GetSelectedTodo() returned nil")
	}
	if node.Text != "Task 1" {
		t.Errorf("GetSelectedTodo().Text = %q, want 'Task 1'", node.Text)
	}
}

func TestVisibleTree_GetSelectedTodo_OnHeading(t *testing.T) {
	tree := &VisibleTree{
		Nodes: []VisibleNode{
			{Type: NodeTypeHeading, TodoIndex: -1, Level: 1, Text: "Heading"},
			{Type: NodeTypeTodo, TodoIndex: 0, Level: 0, Text: "Task 1"},
		},
		SelectedIndex: 0, // On heading
	}

	node := tree.GetSelectedTodo()
	if node != nil {
		t.Error("GetSelectedTodo() should return nil when on heading")
	}
}

func TestVisibleTree_GetSelectedTodo_OutOfBounds(t *testing.T) {
	tree := &VisibleTree{
		Nodes:         []VisibleNode{},
		SelectedIndex: 5,
	}

	node := tree.GetSelectedTodo()
	if node != nil {
		t.Error("GetSelectedTodo() should return nil when out of bounds")
	}
}

func TestVisibleTree_MoveUp(t *testing.T) {
	tree := &VisibleTree{
		Nodes: []VisibleNode{
			{Type: NodeTypeTodo, TodoIndex: 0, Text: "Task 1"},
			{Type: NodeTypeTodo, TodoIndex: 1, Text: "Task 2"},
			{Type: NodeTypeTodo, TodoIndex: 2, Text: "Task 3"},
		},
		SelectedIndex: 2, // Select Task 3
	}

	result := tree.MoveUp()
	if !result {
		t.Error("MoveUp() should return true")
	}
	if tree.SelectedIndex != 1 {
		t.Errorf("SelectedIndex = %d, want 1", tree.SelectedIndex)
	}
	if tree.Nodes[1].Text != "Task 3" {
		t.Errorf("Node at 1 = %q, want 'Task 3'", tree.Nodes[1].Text)
	}
	if tree.Nodes[2].Text != "Task 2" {
		t.Errorf("Node at 2 = %q, want 'Task 2'", tree.Nodes[2].Text)
	}
}

func TestVisibleTree_MoveUp_AtTop(t *testing.T) {
	tree := &VisibleTree{
		Nodes: []VisibleNode{
			{Type: NodeTypeTodo, TodoIndex: 0, Text: "Task 1"},
			{Type: NodeTypeTodo, TodoIndex: 1, Text: "Task 2"},
		},
		SelectedIndex: 0,
	}

	result := tree.MoveUp()
	if result {
		t.Error("MoveUp() should return false when at top")
	}
}

func TestVisibleTree_MoveUp_SkipsHeadings(t *testing.T) {
	tree := &VisibleTree{
		Nodes: []VisibleNode{
			{Type: NodeTypeTodo, TodoIndex: 0, Text: "Task 1"},
			{Type: NodeTypeHeading, TodoIndex: -1, Text: "Heading"},
			{Type: NodeTypeTodo, TodoIndex: 1, Text: "Task 2"},
		},
		SelectedIndex: 2, // Task 2
	}

	result := tree.MoveUp()
	if !result {
		t.Error("MoveUp() should return true")
	}
	// Task 2 should swap with Task 1, skipping the heading
	if tree.SelectedIndex != 0 {
		t.Errorf("SelectedIndex = %d, want 0", tree.SelectedIndex)
	}
}

func TestVisibleTree_MoveDown(t *testing.T) {
	tree := &VisibleTree{
		Nodes: []VisibleNode{
			{Type: NodeTypeTodo, TodoIndex: 0, Text: "Task 1"},
			{Type: NodeTypeTodo, TodoIndex: 1, Text: "Task 2"},
			{Type: NodeTypeTodo, TodoIndex: 2, Text: "Task 3"},
		},
		SelectedIndex: 0, // Select Task 1
	}

	result := tree.MoveDown()
	if !result {
		t.Error("MoveDown() should return true")
	}
	if tree.SelectedIndex != 1 {
		t.Errorf("SelectedIndex = %d, want 1", tree.SelectedIndex)
	}
	if tree.Nodes[0].Text != "Task 2" {
		t.Errorf("Node at 0 = %q, want 'Task 2'", tree.Nodes[0].Text)
	}
	if tree.Nodes[1].Text != "Task 1" {
		t.Errorf("Node at 1 = %q, want 'Task 1'", tree.Nodes[1].Text)
	}
}

func TestVisibleTree_MoveDown_AtBottom(t *testing.T) {
	tree := &VisibleTree{
		Nodes: []VisibleNode{
			{Type: NodeTypeTodo, TodoIndex: 0, Text: "Task 1"},
			{Type: NodeTypeTodo, TodoIndex: 1, Text: "Task 2"},
		},
		SelectedIndex: 1,
	}

	result := tree.MoveDown()
	if result {
		t.Error("MoveDown() should return false when at bottom")
	}
}

func TestVisibleTree_NavigateUp(t *testing.T) {
	tree := &VisibleTree{
		Nodes: []VisibleNode{
			{Type: NodeTypeTodo, TodoIndex: 0, Text: "Task 1"},
			{Type: NodeTypeTodo, TodoIndex: 1, Text: "Task 2"},
			{Type: NodeTypeTodo, TodoIndex: 2, Text: "Task 3"},
		},
		SelectedIndex: 2,
	}

	tree.NavigateUp(2)
	if tree.SelectedIndex != 0 {
		t.Errorf("After NavigateUp(2), SelectedIndex = %d, want 0", tree.SelectedIndex)
	}
}

func TestVisibleTree_NavigateDown(t *testing.T) {
	tree := &VisibleTree{
		Nodes: []VisibleNode{
			{Type: NodeTypeTodo, TodoIndex: 0, Text: "Task 1"},
			{Type: NodeTypeTodo, TodoIndex: 1, Text: "Task 2"},
			{Type: NodeTypeTodo, TodoIndex: 2, Text: "Task 3"},
		},
		SelectedIndex: 0,
	}

	tree.NavigateDown(2)
	if tree.SelectedIndex != 2 {
		t.Errorf("After NavigateDown(2), SelectedIndex = %d, want 2", tree.SelectedIndex)
	}
}

func TestVisibleTree_NavigateToTop(t *testing.T) {
	tree := &VisibleTree{
		Nodes: []VisibleNode{
			{Type: NodeTypeHeading, TodoIndex: -1, Text: "Heading"},
			{Type: NodeTypeTodo, TodoIndex: 0, Text: "Task 1"},
			{Type: NodeTypeTodo, TodoIndex: 1, Text: "Task 2"},
		},
		SelectedIndex: 2,
	}

	tree.NavigateToTop()
	if tree.SelectedIndex != 1 { // First todo, not heading
		t.Errorf("After NavigateToTop(), SelectedIndex = %d, want 1", tree.SelectedIndex)
	}
}

func TestVisibleTree_NavigateToBottom(t *testing.T) {
	tree := &VisibleTree{
		Nodes: []VisibleNode{
			{Type: NodeTypeTodo, TodoIndex: 0, Text: "Task 1"},
			{Type: NodeTypeTodo, TodoIndex: 1, Text: "Task 2"},
			{Type: NodeTypeHeading, TodoIndex: -1, Text: "Heading"},
		},
		SelectedIndex: 0,
	}

	tree.NavigateToBottom()
	if tree.SelectedIndex != 1 { // Last todo, not heading
		t.Errorf("After NavigateToBottom(), SelectedIndex = %d, want 1", tree.SelectedIndex)
	}
}

func TestVisibleTree_NavigateToBottom_Empty(t *testing.T) {
	tree := &VisibleTree{
		Nodes:         []VisibleNode{},
		SelectedIndex: 0,
	}

	tree.NavigateToBottom() // Should not panic
}

func TestVisibleTree_DeleteSelected(t *testing.T) {
	tree := &VisibleTree{
		Nodes: []VisibleNode{
			{Type: NodeTypeTodo, TodoIndex: 0, Text: "Task 1"},
			{Type: NodeTypeTodo, TodoIndex: 1, Text: "Task 2"},
			{Type: NodeTypeTodo, TodoIndex: 2, Text: "Task 3"},
		},
		SelectedIndex: 1, // Delete Task 2
	}

	deletedIdx := tree.DeleteSelected()
	if deletedIdx != 1 {
		t.Errorf("DeleteSelected() = %d, want 1", deletedIdx)
	}
	if len(tree.Nodes) != 2 {
		t.Errorf("After delete, len(Nodes) = %d, want 2", len(tree.Nodes))
	}
	// Selection should move to next todo
	if tree.SelectedIndex != 1 {
		t.Errorf("After delete, SelectedIndex = %d, want 1", tree.SelectedIndex)
	}
}

func TestVisibleTree_DeleteSelected_Last(t *testing.T) {
	tree := &VisibleTree{
		Nodes: []VisibleNode{
			{Type: NodeTypeTodo, TodoIndex: 0, Text: "Task 1"},
			{Type: NodeTypeTodo, TodoIndex: 1, Text: "Task 2"},
		},
		SelectedIndex: 1, // Delete last
	}

	deletedIdx := tree.DeleteSelected()
	if deletedIdx != 1 {
		t.Errorf("DeleteSelected() = %d, want 1", deletedIdx)
	}
	// Selection should move to previous
	if tree.SelectedIndex != 0 {
		t.Errorf("After delete, SelectedIndex = %d, want 0", tree.SelectedIndex)
	}
}

func TestVisibleTree_DeleteSelected_OnHeading(t *testing.T) {
	tree := &VisibleTree{
		Nodes: []VisibleNode{
			{Type: NodeTypeHeading, TodoIndex: -1, Text: "Heading"},
			{Type: NodeTypeTodo, TodoIndex: 0, Text: "Task 1"},
		},
		SelectedIndex: 0, // On heading
	}

	deletedIdx := tree.DeleteSelected()
	if deletedIdx != -1 {
		t.Errorf("DeleteSelected() on heading = %d, want -1", deletedIdx)
	}
}

func TestVisibleTree_GetVisibleTodoIndices(t *testing.T) {
	tree := &VisibleTree{
		Nodes: []VisibleNode{
			{Type: NodeTypeHeading, TodoIndex: -1, Text: "Heading"},
			{Type: NodeTypeTodo, TodoIndex: 0, Text: "Task 1"},
			{Type: NodeTypeTodo, TodoIndex: 2, Text: "Task 3"},
			{Type: NodeTypeHeading, TodoIndex: -1, Text: "Another"},
			{Type: NodeTypeTodo, TodoIndex: 5, Text: "Task 6"},
		},
	}

	indices := tree.GetVisibleTodoIndices()
	expected := []int{0, 2, 5}
	if len(indices) != len(expected) {
		t.Fatalf("GetVisibleTodoIndices() = %v, want %v", indices, expected)
	}
	for i, v := range indices {
		if v != expected[i] {
			t.Errorf("indices[%d] = %d, want %d", i, v, expected[i])
		}
	}
}

// ==================== commands.go tests ====================

func TestGetTodoSections_NoHeadings(t *testing.T) {
	todos := []markdown.Todo{
		{Text: "Task 1"},
		{Text: "Task 2"},
		{Text: "Task 3"},
	}
	headings := []markdown.Heading{}

	sections := getTodoSections(todos, headings)
	if len(sections) != 1 {
		t.Fatalf("Expected 1 section, got %d", len(sections))
	}
	if sections[0].startIndex != 0 || sections[0].endIndex != 3 {
		t.Errorf("Section = {%d, %d}, want {0, 3}", sections[0].startIndex, sections[0].endIndex)
	}
}

func TestGetTodoSections_WithHeadings(t *testing.T) {
	todos := []markdown.Todo{
		{Text: "Task 1"},
		{Text: "Task 2"},
		{Text: "Task 3"},
		{Text: "Task 4"},
	}
	headings := []markdown.Heading{
		{Text: "Section A", BeforeTodoIndex: 0},
		{Text: "Section B", BeforeTodoIndex: 2},
	}

	sections := getTodoSections(todos, headings)
	if len(sections) != 2 {
		t.Fatalf("Expected 2 sections, got %d", len(sections))
	}
	// Section A: todos 0-1
	if sections[0].startIndex != 0 || sections[0].endIndex != 2 {
		t.Errorf("Section 0 = {%d, %d}, want {0, 2}", sections[0].startIndex, sections[0].endIndex)
	}
	// Section B: todos 2-3
	if sections[1].startIndex != 2 || sections[1].endIndex != 4 {
		t.Errorf("Section 1 = {%d, %d}, want {2, 4}", sections[1].startIndex, sections[1].endIndex)
	}
}

func TestGetTodoSections_EmptyTodos(t *testing.T) {
	todos := []markdown.Todo{}
	headings := []markdown.Heading{{Text: "Heading", BeforeTodoIndex: 0}}

	sections := getTodoSections(todos, headings)
	if sections != nil {
		t.Errorf("Expected nil, got %v", sections)
	}
}

func TestSortTodosInSections(t *testing.T) {
	todos := []markdown.Todo{
		{Text: "B Task", Priority: 2},
		{Text: "A Task", Priority: 1},
		{Text: "D Task", Priority: 4},
		{Text: "C Task", Priority: 3},
	}
	headings := []markdown.Heading{
		{Text: "Section 1", BeforeTodoIndex: 0},
		{Text: "Section 2", BeforeTodoIndex: 2},
	}

	// Sort by priority within sections
	sortTodosInSections(todos, headings, func(slice []markdown.Todo) {
		// Simple bubble sort by priority
		for i := 0; i < len(slice); i++ {
			for j := i + 1; j < len(slice); j++ {
				if slice[j].Priority < slice[i].Priority {
					slice[i], slice[j] = slice[j], slice[i]
				}
			}
		}
	})

	// Section 1: A (p1), B (p2)
	if todos[0].Text != "A Task" {
		t.Errorf("todos[0] = %q, want 'A Task'", todos[0].Text)
	}
	if todos[1].Text != "B Task" {
		t.Errorf("todos[1] = %q, want 'B Task'", todos[1].Text)
	}
	// Section 2: C (p3), D (p4)
	if todos[2].Text != "C Task" {
		t.Errorf("todos[2] = %q, want 'C Task'", todos[2].Text)
	}
	if todos[3].Text != "D Task" {
		t.Errorf("todos[3] = %q, want 'D Task'", todos[3].Text)
	}
}

func TestHighlightMatches(t *testing.T) {
	// Identity function for styling
	highlight := func(s string) string { return "[" + s + "]" }

	tests := []struct {
		name     string
		text     string
		query    string
		hasMatch bool
	}{
		{
			name:     "simple match",
			text:     "hello world",
			query:    "world",
			hasMatch: true,
		},
		{
			name:     "no match",
			text:     "hello world",
			query:    "foo",
			hasMatch: false,
		},
		{
			name:     "case insensitive",
			text:     "Hello World",
			query:    "world",
			hasMatch: true,
		},
		{
			name:     "empty query",
			text:     "hello world",
			query:    "",
			hasMatch: false,
		},
		{
			name:     "multiple matches",
			text:     "test test test",
			query:    "test",
			hasMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HighlightMatches(tt.text, tt.query, highlight)

			if !tt.hasMatch {
				// No match expected - text should be unchanged
				if got != tt.text {
					t.Errorf("HighlightMatches(%q, %q) = %q, want %q", tt.text, tt.query, got, tt.text)
				}
			} else {
				// Match expected - result should contain brackets from our highlight fn
				if !strings.Contains(got, "[") {
					t.Errorf("HighlightMatches(%q, %q) = %q, should contain highlighted match", tt.text, tt.query, got)
				}
			}
		})
	}
}

// ==================== BuildVisibleTree tests ====================

func TestBuildVisibleTree_Simple(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task 1", Checked: false},
			{Text: "Task 2", Checked: false},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")

	tree := m.BuildVisibleTree()
	if tree == nil {
		t.Fatal("BuildVisibleTree() returned nil")
	}
	if len(tree.Nodes) != 2 {
		t.Errorf("len(Nodes) = %d, want 2", len(tree.Nodes))
	}
}

func TestBuildVisibleTree_WithFilteredDone(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task 1", Checked: false},
			{Text: "Task 2", Checked: true},
			{Text: "Task 3", Checked: false},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
	m.FilterDone = true

	tree := m.BuildVisibleTree()
	if len(tree.Nodes) != 2 {
		t.Errorf("With FilterDone, len(Nodes) = %d, want 2", len(tree.Nodes))
	}
}

// ==================== SyncToModel tests ====================

func TestVisibleTree_SyncToModel(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task 1"},
			{Text: "Task 2"},
			{Text: "Task 3"},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")

	// Create a reordered visible tree (Task 3 moved to first)
	tree := &VisibleTree{
		Nodes: []VisibleNode{
			{Type: NodeTypeTodo, TodoIndex: 2, Text: "Task 3"},
			{Type: NodeTypeTodo, TodoIndex: 0, Text: "Task 1"},
			{Type: NodeTypeTodo, TodoIndex: 1, Text: "Task 2"},
		},
		SelectedIndex: 0,
	}

	tree.SyncToModel(&m)

	// Model should now have reordered todos
	if m.FileModel.Todos[0].Text != "Task 3" {
		t.Errorf("After sync, todos[0] = %q, want 'Task 3'", m.FileModel.Todos[0].Text)
	}
	if m.FileModel.Todos[1].Text != "Task 1" {
		t.Errorf("After sync, todos[1] = %q, want 'Task 1'", m.FileModel.Todos[1].Text)
	}
}

func TestVisibleTree_SyncToModel_Empty(t *testing.T) {
	fm := &markdown.FileModel{Todos: []markdown.Todo{}}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")

	tree := &VisibleTree{Nodes: []VisibleNode{}, SelectedIndex: 0}
	tree.SyncToModel(&m) // Should not panic
}

// ==================== Filter handler tests ====================

func TestHandleFilterKey_Navigation(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task #urgent", Tags: []string{"urgent"}},
			{Text: "Task #backend", Tags: []string{"backend"}},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
	m.FilterMode = true
	m.AvailableTags = []string{"urgent", "backend"}
	m.TagFilterCursor = 0

	// Navigate down
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	result, _ := m.handleFilterKey(msg)
	m = result.(Model)

	if m.TagFilterCursor != 1 {
		t.Errorf("After 'j', TagFilterCursor = %d, want 1", m.TagFilterCursor)
	}

	// Navigate up
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	result, _ = m.handleFilterKey(msg)
	m = result.(Model)

	if m.TagFilterCursor != 0 {
		t.Errorf("After 'k', TagFilterCursor = %d, want 0", m.TagFilterCursor)
	}
}

func TestHandleFilterKey_SelectTag(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task #urgent", Tags: []string{"urgent"}},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
	m.FilterMode = true
	m.AvailableTags = []string{"urgent", "backend"}
	m.TagFilterCursor = 0
	m.FilteredTags = []string{}

	// Select tag with enter
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	result, _ := m.handleFilterKey(msg)
	m = result.(Model)

	if len(m.FilteredTags) != 1 || m.FilteredTags[0] != "urgent" {
		t.Errorf("FilteredTags = %v, want [urgent]", m.FilteredTags)
	}
	if m.FilterMode {
		t.Error("FilterMode should be closed after selection")
	}
}

func TestHandleFilterKey_ClearFilters(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task #urgent", Tags: []string{"urgent"}},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
	m.FilterMode = true
	m.FilteredTags = []string{"urgent", "backend"}

	// Clear filters with 'c'
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	result, _ := m.handleFilterKey(msg)
	m = result.(Model)

	if len(m.FilteredTags) != 0 {
		t.Errorf("FilteredTags should be empty, got %v", m.FilteredTags)
	}
}

func TestHandleFilterKey_Escape(t *testing.T) {
	fm := &markdown.FileModel{Todos: []markdown.Todo{}}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
	m.FilterMode = true

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	result, _ := m.handleFilterKey(msg)
	m = result.(Model)

	if m.FilterMode {
		t.Error("FilterMode should be false after Esc")
	}
}

func TestHandlePriorityFilterKey_Navigation(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task !p1", Priority: 1},
			{Text: "Task !p2", Priority: 2},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
	m.PriorityFilterMode = true
	m.AvailablePriorities = []int{1, 2, 3}
	m.PriorityFilterCursor = 0

	// Navigate down
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	result, _ := m.handlePriorityFilterKey(msg)
	m = result.(Model)

	if m.PriorityFilterCursor != 1 {
		t.Errorf("After 'j', PriorityFilterCursor = %d, want 1", m.PriorityFilterCursor)
	}
}

func TestHandlePriorityFilterKey_Select(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task !p1", Priority: 1},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
	m.PriorityFilterMode = true
	m.AvailablePriorities = []int{1, 2}
	m.PriorityFilterCursor = 0
	m.FilteredPriorities = []int{}

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	result, _ := m.handlePriorityFilterKey(msg)
	m = result.(Model)

	if len(m.FilteredPriorities) != 1 {
		t.Errorf("FilteredPriorities = %v, want [1]", m.FilteredPriorities)
	}
}

func TestHandlePriorityFilterKey_Escape(t *testing.T) {
	fm := &markdown.FileModel{Todos: []markdown.Todo{}}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
	m.PriorityFilterMode = true

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	result, _ := m.handlePriorityFilterKey(msg)
	m = result.(Model)

	if m.PriorityFilterMode {
		t.Error("PriorityFilterMode should be false after Esc")
	}
}

func TestHandleThemeKey_Escape(t *testing.T) {
	fm := &markdown.FileModel{Todos: []markdown.Todo{}}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
	m.ThemeMode = true

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	result, _ := m.handleThemeKey(msg)
	m = result.(Model)

	if m.ThemeMode {
		t.Error("ThemeMode should be false after Esc")
	}
}

func TestHandleMaxVisibleInputKey_Escape(t *testing.T) {
	fm := &markdown.FileModel{Todos: []markdown.Todo{}}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
	m.MaxVisibleInputMode = true

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	result, _ := m.handleMaxVisibleInputKey(msg)
	m = result.(Model)

	if m.MaxVisibleInputMode {
		t.Error("MaxVisibleInputMode should be false after Esc")
	}
}

// ==================== ColorizePriorities test ====================

func TestColorizePriorities(t *testing.T) {
	result := ColorizePriorities("Task !p1 with priority")
	if result == "" {
		t.Error("ColorizePriorities returned empty string")
	}
	// Should contain the original text parts
	if !strings.Contains(result, "Task") {
		t.Error("Result should contain 'Task'")
	}
}

func TestColorizePriorities_NoPriority(t *testing.T) {
	result := ColorizePriorities("Task without priority")
	if result != "Task without priority" {
		t.Errorf("Text without priority should be unchanged, got %q", result)
	}
}

// ==================== View rendering tests ====================

func TestView_HelpMode(t *testing.T) {
	fm := &markdown.FileModel{Todos: []markdown.Todo{}}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
	m.HelpMode = true

	view := m.View()
	if view == "" {
		t.Error("View() returned empty string")
	}
	// Help mode should show keyboard shortcuts
	if !strings.Contains(view, "j") && !strings.Contains(view, "k") {
		t.Error("Help view should contain key bindings")
	}
}

func TestView_NormalMode(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task 1", Checked: false},
			{Text: "Task 2", Checked: true},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
	m.TermWidth = 80
	m.TermHeight = 24

	view := m.View()
	if view == "" {
		t.Error("View() returned empty string")
	}
	// Should contain our tasks
	if !strings.Contains(view, "Task 1") {
		t.Error("View should contain 'Task 1'")
	}
}

func TestView_FilterMode(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task #urgent", Tags: []string{"urgent"}},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
	m.TermWidth = 80
	m.TermHeight = 24
	m.FilterMode = true
	m.AvailableTags = []string{"urgent", "backend"}

	view := m.View()
	if view == "" {
		t.Error("View() returned empty string in filter mode")
	}
}

func TestView_CommandMode(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task 1"},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
	m.TermWidth = 80
	m.TermHeight = 24
	m.CommandMode = true

	view := m.View()
	if view == "" {
		t.Error("View() returned empty string in command mode")
	}
}

func TestView_ThemeMode(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task 1"},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
	m.TermWidth = 80
	m.TermHeight = 24
	m.ThemeMode = true
	m.AvailableThemes = []string{"default", "dracula"}

	view := m.View()
	if view == "" {
		t.Error("View() returned empty string in theme mode")
	}
}

func TestView_PriorityFilterMode(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task !p1", Priority: 1},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
	m.TermWidth = 80
	m.TermHeight = 24
	m.PriorityFilterMode = true
	m.AvailablePriorities = []int{1, 2, 3}

	view := m.View()
	if view == "" {
		t.Error("View() returned empty string in priority filter mode")
	}
}

func TestView_SearchMode(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task 1"},
			{Text: "Task 2"},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
	m.TermWidth = 80
	m.TermHeight = 24
	m.SearchMode = true

	view := m.View()
	if view == "" {
		t.Error("View() returned empty string in search mode")
	}
}

func TestView_InputMode(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task 1"},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
	m.TermWidth = 80
	m.TermHeight = 24
	m.InputMode = true
	m.InputBuffer = "New task text"

	view := m.View()
	if view == "" {
		t.Error("View() returned empty string in input mode")
	}
}

// ==================== handleThemeKey tests ====================

func TestHandleThemeKey_NavigationCoverage(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task 1"},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
	m.ThemeMode = true
	m.AvailableThemes = []string{"default", "dracula", "nord"}
	m.ThemeCursor = 0

	// Navigate down
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	result, _ := m.handleThemeKey(msg)
	m = result.(Model)

	if m.ThemeCursor != 1 {
		t.Errorf("After 'j', ThemeCursor = %d, want 1", m.ThemeCursor)
	}
}

// ==================== hasActiveFilters tests ====================

func TestHasActiveFilters_NoneCoverage(t *testing.T) {
	fm := &markdown.FileModel{Todos: []markdown.Todo{}}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")

	if m.hasActiveFilters() {
		t.Error("hasActiveFilters() should be false with no filters")
	}
}

func TestHasActiveFilters_FilterDoneCoverage(t *testing.T) {
	fm := &markdown.FileModel{Todos: []markdown.Todo{}}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
	m.FilterDone = true

	if !m.hasActiveFilters() {
		t.Error("hasActiveFilters() should be true with FilterDone")
	}
}

func TestHasActiveFilters_TagFilterCoverage(t *testing.T) {
	fm := &markdown.FileModel{Todos: []markdown.Todo{}}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
	m.FilteredTags = []string{"urgent"}

	if !m.hasActiveFilters() {
		t.Error("hasActiveFilters() should be true with tag filter")
	}
}

func TestHasActiveFilters_PriorityFilterCoverage(t *testing.T) {
	fm := &markdown.FileModel{Todos: []markdown.Todo{}}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
	m.FilteredPriorities = []int{1}

	if !m.hasActiveFilters() {
		t.Error("hasActiveFilters() should be true with priority filter")
	}
}

// ==================== isTodoVisible tests ====================

func TestIsTodoVisible_NoFiltersCoverage(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task", Checked: false},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")

	if !m.isTodoVisible(0) {
		t.Error("Todo should be visible with no filters")
	}
}

func TestIsTodoVisible_FilterDone_UncheckedCoverage(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task", Checked: false},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
	m.FilterDone = true

	if !m.isTodoVisible(0) {
		t.Error("Unchecked todo should be visible with FilterDone")
	}
}

func TestIsTodoVisible_FilterDone_CheckedCoverage(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task", Checked: true},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
	m.FilterDone = true

	if m.isTodoVisible(0) {
		t.Error("Checked todo should not be visible with FilterDone")
	}
}

func TestIsTodoVisible_TagFilter_MatchCoverage(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task #urgent", Tags: []string{"urgent"}},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
	m.FilteredTags = []string{"urgent"}

	if !m.isTodoVisible(0) {
		t.Error("Todo with matching tag should be visible")
	}
}

func TestIsTodoVisible_TagFilter_NoMatchCoverage(t *testing.T) {
	fm := &markdown.FileModel{
		Todos: []markdown.Todo{
			{Text: "Task #backend", Tags: []string{"backend"}},
		},
	}
	m := New("/tmp/test.md", fm, false, false, -1, testConfig(), testStyles(), "test")
	m.FilteredTags = []string{"urgent"}

	if m.isTodoVisible(0) {
		t.Error("Todo without matching tag should not be visible")
	}
}
