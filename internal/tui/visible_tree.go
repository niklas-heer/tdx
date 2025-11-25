package tui

import (
	"github.com/niklas-heer/tdx/internal/markdown"
)

// VisibleNodeType represents the type of node in the visible tree
type VisibleNodeType string

const (
	NodeTypeTodo            VisibleNodeType = "todo"
	NodeTypeHeading         VisibleNodeType = "heading"
	NodeTypeFilteredHeading VisibleNodeType = "filtered-heading"
)

// VisibleNode represents an item in the visible tree
type VisibleNode struct {
	Type      VisibleNodeType
	TodoIndex int    // Index in FileModel.Todos (-1 for headings)
	Level     int    // Heading level (1-6, 0 for todos)
	Text      string // Display text
}

// VisibleTree maintains the current visible state of the TUI
// This is the source of truth for what the user sees and how they interact with it
type VisibleTree struct {
	Nodes         []VisibleNode
	SelectedIndex int // Index in Nodes array
}

// BuildVisibleTree constructs the visible tree from the current model state
func (m *Model) BuildVisibleTree() *VisibleTree {
	tree := &VisibleTree{
		Nodes:         []VisibleNode{},
		SelectedIndex: 0,
	}

	if !m.ShowHeadings {
		// Simple case: just todos, filtered by visibility
		for i, todo := range m.FileModel.Todos {
			if m.isTodoVisible(i) {
				tree.Nodes = append(tree.Nodes, VisibleNode{
					Type:      NodeTypeTodo,
					TodoIndex: i,
					Level:     0,
					Text:      todo.Text,
				})
			}
		}
	} else {
		// Complex case: interleave headings and todos
		headings := m.GetHeadings()
		headingIdx := 0

		for i, todo := range m.FileModel.Todos {
			// Insert headings before this todo
			for headingIdx < len(headings) && headings[headingIdx].BeforeTodoIndex == i {
				// Check if this heading has any visible todos
				hasVisibleTodos := m.headingHasVisibleTodos(headingIdx, headings)

				nodeType := NodeTypeHeading
				if !hasVisibleTodos {
					nodeType = NodeTypeFilteredHeading
				}

				tree.Nodes = append(tree.Nodes, VisibleNode{
					Type:      nodeType,
					TodoIndex: -1,
					Level:     headings[headingIdx].Level,
					Text:      headings[headingIdx].Text,
				})

				headingIdx++
			}

			// Add todo if visible
			if m.isTodoVisible(i) {
				tree.Nodes = append(tree.Nodes, VisibleNode{
					Type:      NodeTypeTodo,
					TodoIndex: i,
					Level:     0,
					Text:      todo.Text,
				})
			}
		}

		// Add any remaining headings at the end
		for headingIdx < len(headings) {
			tree.Nodes = append(tree.Nodes, VisibleNode{
				Type:      NodeTypeFilteredHeading,
				TodoIndex: -1,
				Level:     headings[headingIdx].Level,
				Text:      headings[headingIdx].Text,
			})
			headingIdx++
		}
	}

	// Map the current SelectedIndex (in FileModel.Todos) to the visible tree
	tree.SelectedIndex = tree.findNodeByTodoIndex(m.SelectedIndex)
	if tree.SelectedIndex == -1 && len(tree.Nodes) > 0 {
		// If current selection is not visible, find nearest visible todo
		tree.SelectedIndex = tree.findNearestTodo(0)
	}

	return tree
}

// headingHasVisibleTodos checks if a heading section has any visible todos
func (m *Model) headingHasVisibleTodos(headingIdx int, headings []markdown.Heading) bool {
	if headingIdx >= len(headings) {
		return false
	}

	startIdx := headings[headingIdx].BeforeTodoIndex
	endIdx := len(m.FileModel.Todos)
	if headingIdx+1 < len(headings) {
		endIdx = headings[headingIdx+1].BeforeTodoIndex
	}

	for i := startIdx; i < endIdx; i++ {
		if m.isTodoVisible(i) {
			return true
		}
	}

	return false
}

// findNodeByTodoIndex finds the index in the visible tree for a given todo index
// Returns -1 if not found
func (tree *VisibleTree) findNodeByTodoIndex(todoIndex int) int {
	for i, node := range tree.Nodes {
		if node.Type == NodeTypeTodo && node.TodoIndex == todoIndex {
			return i
		}
	}
	return -1
}

// findNearestTodo finds the nearest todo node starting from the given index
// Returns -1 if no todos exist in the tree
func (tree *VisibleTree) findNearestTodo(startIdx int) int {
	if len(tree.Nodes) == 0 {
		return -1
	}

	// Search forward first
	for i := startIdx; i < len(tree.Nodes); i++ {
		if tree.Nodes[i].Type == NodeTypeTodo {
			return i
		}
	}

	// Search backward
	for i := startIdx - 1; i >= 0; i-- {
		if tree.Nodes[i].Type == NodeTypeTodo {
			return i
		}
	}

	return -1
}

// GetSelectedTodo returns the currently selected todo node
// Returns nil if selection is not on a todo
func (tree *VisibleTree) GetSelectedTodo() *VisibleNode {
	if tree.SelectedIndex < 0 || tree.SelectedIndex >= len(tree.Nodes) {
		return nil
	}
	node := &tree.Nodes[tree.SelectedIndex]
	if node.Type != NodeTypeTodo {
		return nil
	}
	return node
}

// MoveUp moves the selected todo up in the visible list
// Returns true if the move was successful
func (tree *VisibleTree) MoveUp() bool {
	selectedNode := tree.GetSelectedTodo()
	if selectedNode == nil {
		return false
	}

	// Find previous todo node
	prevTodoIdx := -1
	for i := tree.SelectedIndex - 1; i >= 0; i-- {
		if tree.Nodes[i].Type == NodeTypeTodo {
			prevTodoIdx = i
			break
		}
	}

	if prevTodoIdx == -1 {
		return false // Already at top
	}

	// Swap the nodes in the visible tree
	tree.Nodes[tree.SelectedIndex], tree.Nodes[prevTodoIdx] = tree.Nodes[prevTodoIdx], tree.Nodes[tree.SelectedIndex]
	tree.SelectedIndex = prevTodoIdx

	return true
}

// MoveDown moves the selected todo down in the visible list
// Returns true if the move was successful
func (tree *VisibleTree) MoveDown() bool {
	selectedNode := tree.GetSelectedTodo()
	if selectedNode == nil {
		return false
	}

	// Find next todo node
	nextTodoIdx := -1
	for i := tree.SelectedIndex + 1; i < len(tree.Nodes); i++ {
		if tree.Nodes[i].Type == NodeTypeTodo {
			nextTodoIdx = i
			break
		}
	}

	if nextTodoIdx == -1 {
		return false // Already at bottom
	}

	// Swap the nodes in the visible tree
	tree.Nodes[tree.SelectedIndex], tree.Nodes[nextTodoIdx] = tree.Nodes[nextTodoIdx], tree.Nodes[tree.SelectedIndex]
	tree.SelectedIndex = nextTodoIdx

	return true
}

// NavigateUp moves the selection up to the previous visible todo
// Does NOT modify the tree, only changes selection
func (tree *VisibleTree) NavigateUp(count int) {
	for i := 0; i < count; i++ {
		prevTodoIdx := -1
		for j := tree.SelectedIndex - 1; j >= 0; j-- {
			if tree.Nodes[j].Type == NodeTypeTodo {
				prevTodoIdx = j
				break
			}
		}
		if prevTodoIdx == -1 {
			break // Already at top
		}
		tree.SelectedIndex = prevTodoIdx
	}
}

// NavigateDown moves the selection down to the next visible todo
// Does NOT modify the tree, only changes selection
func (tree *VisibleTree) NavigateDown(count int) {
	for i := 0; i < count; i++ {
		nextTodoIdx := -1
		for j := tree.SelectedIndex + 1; j < len(tree.Nodes); j++ {
			if tree.Nodes[j].Type == NodeTypeTodo {
				nextTodoIdx = j
				break
			}
		}
		if nextTodoIdx == -1 {
			break // Already at bottom
		}
		tree.SelectedIndex = nextTodoIdx
	}
}

// NavigateToTop moves selection to the first visible todo
func (tree *VisibleTree) NavigateToTop() {
	tree.SelectedIndex = tree.findNearestTodo(0)
}

// NavigateToBottom moves selection to the last visible todo
func (tree *VisibleTree) NavigateToBottom() {
	if len(tree.Nodes) == 0 {
		return
	}
	for i := len(tree.Nodes) - 1; i >= 0; i-- {
		if tree.Nodes[i].Type == NodeTypeTodo {
			tree.SelectedIndex = i
			return
		}
	}
}

// DeleteSelected deletes the currently selected todo and adjusts selection
// Returns the todo index that was deleted, or -1 if nothing was deleted
func (tree *VisibleTree) DeleteSelected() int {
	selectedNode := tree.GetSelectedTodo()
	if selectedNode == nil {
		return -1
	}

	deletedTodoIdx := selectedNode.TodoIndex

	// Find next todo for selection (prefer next, then previous)
	nextTodoIdx := -1
	for i := tree.SelectedIndex + 1; i < len(tree.Nodes); i++ {
		if tree.Nodes[i].Type == NodeTypeTodo {
			nextTodoIdx = i
			break
		}
	}

	if nextTodoIdx == -1 {
		// No next todo, try previous
		for i := tree.SelectedIndex - 1; i >= 0; i-- {
			if tree.Nodes[i].Type == NodeTypeTodo {
				nextTodoIdx = i
				break
			}
		}
	}

	// Remove the node from the tree
	tree.Nodes = append(tree.Nodes[:tree.SelectedIndex], tree.Nodes[tree.SelectedIndex+1:]...)

	// Adjust selection
	if nextTodoIdx == -1 {
		// No todos left
		tree.SelectedIndex = 0
	} else if nextTodoIdx > tree.SelectedIndex {
		// Next todo was after deleted, adjust for removal
		tree.SelectedIndex = nextTodoIdx - 1
	} else {
		// Next todo was before deleted
		tree.SelectedIndex = nextTodoIdx
	}

	return deletedTodoIdx
}

// SyncToModel applies the visible tree ordering back to the underlying FileModel
// This is called after moves to persist the new order
func (tree *VisibleTree) SyncToModel(m *Model) {
	if len(tree.Nodes) == 0 {
		return
	}

	// Build a mapping of current positions to todos
	oldTodos := make([]markdown.Todo, len(m.FileModel.Todos))
	copy(oldTodos, m.FileModel.Todos)

	// DEBUG: Log old todos
	// fmt.Printf("DEBUG SyncToModel: oldTodos count=%d\n", len(oldTodos))
	// for i, t := range oldTodos {
	// 	fmt.Printf("  old[%d]: %s (checked=%v)\n", i, t.Text, t.Checked)
	// }

	// Extract visible todo order from tree
	var visibleOrder []int
	for _, node := range tree.Nodes {
		if node.Type == NodeTypeTodo && node.TodoIndex >= 0 && node.TodoIndex < len(oldTodos) {
			visibleOrder = append(visibleOrder, node.TodoIndex)
		}
	}

	if len(visibleOrder) == 0 {
		return
	}

	// Rebuild the todos array with the new order
	newTodos := make([]markdown.Todo, 0, len(oldTodos))
	used := make([]bool, len(oldTodos))

	// Add visible todos in their new order
	for _, oldIdx := range visibleOrder {
		newTodos = append(newTodos, oldTodos[oldIdx])
		used[oldIdx] = true
	}

	// Add hidden/filtered todos at the end, preserving their relative order
	for i, todo := range oldTodos {
		if !used[i] {
			newTodos = append(newTodos, todo)
		}
	}

	// Update the model's todos
	m.FileModel.Todos = newTodos

	// DEBUG: Log new todos
	// fmt.Printf("DEBUG SyncToModel: newTodos count=%d\n", len(newTodos))
	// for i, t := range newTodos {
	// 	fmt.Printf("  new[%d]: %s (checked=%v)\n", i, t.Text, t.Checked)
	// }

	// Update TodoIndex for all visible nodes to reflect new positions
	newIdx := 0
	for i := range tree.Nodes {
		if tree.Nodes[i].Type == NodeTypeTodo {
			tree.Nodes[i].TodoIndex = newIdx
			newIdx++
		}
	}

	// Update the model's SelectedIndex
	if selectedNode := tree.GetSelectedTodo(); selectedNode != nil {
		m.SelectedIndex = selectedNode.TodoIndex
	}
}

// GetVisibleTodoIndices returns the indices of visible todos in the current FileModel
func (tree *VisibleTree) GetVisibleTodoIndices() []int {
	var indices []int
	for _, node := range tree.Nodes {
		if node.Type == NodeTypeTodo {
			indices = append(indices, node.TodoIndex)
		}
	}
	return indices
}
