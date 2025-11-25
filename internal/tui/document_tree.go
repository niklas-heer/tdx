package tui

// DocumentNodeType represents the type of node in the document tree
type DocumentNodeType string

const (
	DocNodeHeading DocumentNodeType = "heading"
	DocNodeTodo    DocumentNodeType = "todo"
)

// DocumentNode represents a node in the document tree (either a heading or a todo)
type DocumentNode struct {
	Type DocumentNodeType

	// For headings
	Level int    // 1-6 for headings, 0 for todos
	Text  string // Heading text or todo text

	// For todos
	TodoIndex int  // Index in FileModel.Todos array (-1 for headings)
	Checked   bool // Whether todo is checked
	Tags      []string

	// Tree structure
	Parent   *DocumentNode
	Children []*DocumentNode

	// Visibility
	Visible bool // Whether this node should be shown given current filters
}

// DocumentTree represents the entire document as a hierarchical tree
type DocumentTree struct {
	Root     *DocumentNode   // Synthetic root node
	Flat     []*DocumentNode // Flattened view for rendering
	Selected int             // Index in Flat array
}

// BuildDocumentTree constructs a document tree from the model state
func (m *Model) BuildDocumentTree() *DocumentTree {
	tree := &DocumentTree{
		Root: &DocumentNode{
			Type:     "root",
			Children: []*DocumentNode{},
			Visible:  true,
		},
		Flat:     []*DocumentNode{},
		Selected: 0,
	}

	if !m.ShowHeadings {
		// Simple mode: flat list of todos
		for i, todo := range m.FileModel.Todos {
			node := &DocumentNode{
				Type:      DocNodeTodo,
				Text:      todo.Text,
				TodoIndex: i,
				Checked:   todo.Checked,
				Tags:      todo.Tags,
				Parent:    tree.Root,
				Children:  []*DocumentNode{},
				Visible:   m.isTodoVisible(i),
			}
			tree.Root.Children = append(tree.Root.Children, node)
		}
	} else {
		// Complex mode: build hierarchy with headings
		headings := m.GetHeadings()

		// Stack to track current heading hierarchy
		headingStack := []*DocumentNode{tree.Root}

		headingIdx := 0

		for todoIdx, todo := range m.FileModel.Todos {
			// Insert headings that come before this todo
			for headingIdx < len(headings) && headings[headingIdx].BeforeTodoIndex == todoIdx {
				h := headings[headingIdx]

				// Create heading node
				headingNode := &DocumentNode{
					Type:      DocNodeHeading,
					Level:     h.Level,
					Text:      h.Text,
					TodoIndex: -1,
					Children:  []*DocumentNode{},
					Visible:   true, // Headings are always structurally visible
				}

				// Pop stack until we find the right parent level
				for len(headingStack) > 1 && headingStack[len(headingStack)-1].Level >= h.Level {
					headingStack = headingStack[:len(headingStack)-1]
				}

				// Add to parent
				parent := headingStack[len(headingStack)-1]
				headingNode.Parent = parent
				parent.Children = append(parent.Children, headingNode)

				// Push onto stack
				headingStack = append(headingStack, headingNode)

				headingIdx++
			}

			// Create todo node
			todoNode := &DocumentNode{
				Type:      DocNodeTodo,
				Text:      todo.Text,
				TodoIndex: todoIdx,
				Checked:   todo.Checked,
				Tags:      todo.Tags,
				Children:  []*DocumentNode{},
				Visible:   m.isTodoVisible(todoIdx),
			}

			// Add to current heading parent
			parent := headingStack[len(headingStack)-1]
			todoNode.Parent = parent
			parent.Children = append(parent.Children, todoNode)
		}

		// Add any remaining headings at the end
		for headingIdx < len(headings) {
			h := headings[headingIdx]

			headingNode := &DocumentNode{
				Type:      DocNodeHeading,
				Level:     h.Level,
				Text:      h.Text,
				TodoIndex: -1,
				Children:  []*DocumentNode{},
				Visible:   true,
			}

			// Pop stack until we find the right parent level
			for len(headingStack) > 1 && headingStack[len(headingStack)-1].Level >= h.Level {
				headingStack = headingStack[:len(headingStack)-1]
			}

			parent := headingStack[len(headingStack)-1]
			headingNode.Parent = parent
			parent.Children = append(parent.Children, headingNode)
			headingStack = append(headingStack, headingNode)

			headingIdx++
		}
	}

	// Flatten the tree for rendering and navigation
	tree.Flat = tree.flattenTree()

	// Find the selected node
	tree.Selected = tree.findNodeByTodoIndex(m.SelectedIndex)
	if tree.Selected == -1 && len(tree.Flat) > 0 {
		// Find first visible todo
		tree.Selected = tree.findNextVisibleTodo(0)
	}

	return tree
}

// flattenTree creates a depth-first flattened view of the tree
func (tree *DocumentTree) flattenTree() []*DocumentNode {
	var flat []*DocumentNode

	var walk func(*DocumentNode)
	walk = func(node *DocumentNode) {
		// Don't include the synthetic root
		if node.Type != "root" {
			flat = append(flat, node)
		}
		for _, child := range node.Children {
			walk(child)
		}
	}

	walk(tree.Root)
	return flat
}

// findNodeByTodoIndex finds the flat index of a node with the given todo index
func (tree *DocumentTree) findNodeByTodoIndex(todoIndex int) int {
	for i, node := range tree.Flat {
		if node.Type == DocNodeTodo && node.TodoIndex == todoIndex {
			return i
		}
	}
	return -1
}

// findNextVisibleTodo finds the next visible todo starting from index
func (tree *DocumentTree) findNextVisibleTodo(startIdx int) int {
	for i := startIdx; i < len(tree.Flat); i++ {
		if tree.Flat[i].Type == DocNodeTodo && tree.Flat[i].Visible {
			return i
		}
	}
	return -1
}

// findPrevVisibleTodo finds the previous visible todo starting from index
func (tree *DocumentTree) findPrevVisibleTodo(startIdx int) int {
	for i := startIdx; i >= 0; i-- {
		if tree.Flat[i].Type == DocNodeTodo && tree.Flat[i].Visible {
			return i
		}
	}
	return -1
}

// GetSelectedNode returns the currently selected node
func (tree *DocumentTree) GetSelectedNode() *DocumentNode {
	if tree.Selected < 0 || tree.Selected >= len(tree.Flat) {
		return nil
	}
	return tree.Flat[tree.Selected]
}

// NavigateDown moves selection to next visible todo
func (tree *DocumentTree) NavigateDown(count int) {
	for i := 0; i < count; i++ {
		next := tree.findNextVisibleTodo(tree.Selected + 1)
		if next == -1 {
			break
		}
		tree.Selected = next
	}
}

// NavigateUp moves selection to previous visible todo
func (tree *DocumentTree) NavigateUp(count int) {
	for i := 0; i < count; i++ {
		prev := tree.findPrevVisibleTodo(tree.Selected - 1)
		if prev == -1 {
			break
		}
		tree.Selected = prev
	}
}

// NavigateToTop moves to first visible todo
func (tree *DocumentTree) NavigateToTop() {
	first := tree.findNextVisibleTodo(0)
	if first != -1 {
		tree.Selected = first
	}
}

// NavigateToBottom moves to last visible todo
func (tree *DocumentTree) NavigateToBottom() {
	last := tree.findPrevVisibleTodo(len(tree.Flat) - 1)
	if last != -1 {
		tree.Selected = last
	}
}

// MoveUp moves the selected todo up in the visible list
// This is insertion-based: the todo moves to the position of the previous visible todo
// Returns the target TodoIndex if move was successful, -1 otherwise
func (tree *DocumentTree) MoveUp() int {
	selectedNode := tree.GetSelectedNode()
	if selectedNode == nil || selectedNode.Type != DocNodeTodo || !selectedNode.Visible {
		return -1
	}

	// Find previous visible todo in the flat list (crosses section boundaries)
	prevVisibleIdx := tree.findPrevVisibleTodo(tree.Selected - 1)
	if prevVisibleIdx == -1 {
		return -1 // Already at top of visible list
	}

	prevVisibleNode := tree.Flat[prevVisibleIdx]
	if prevVisibleNode.Type != DocNodeTodo {
		return -1
	}

	// Return the target TodoIndex where we want to move
	// The selected todo will be inserted at the position of prevVisibleNode
	return prevVisibleNode.TodoIndex
}

// MoveDown moves the selected todo down in the visible list
// This is insertion-based: the todo moves to the position of the next visible todo
// Returns the target TodoIndex if move was successful, -1 otherwise
func (tree *DocumentTree) MoveDown() int {
	selectedNode := tree.GetSelectedNode()
	if selectedNode == nil || selectedNode.Type != DocNodeTodo || !selectedNode.Visible {
		return -1
	}

	// Find next visible todo in the flat list (crosses section boundaries)
	nextVisibleIdx := tree.findNextVisibleTodo(tree.Selected + 1)
	if nextVisibleIdx == -1 {
		return -1 // Already at bottom of visible list
	}

	nextVisibleNode := tree.Flat[nextVisibleIdx]
	if nextVisibleNode.Type != DocNodeTodo {
		return -1
	}

	// Return the target TodoIndex where we want to move
	// The selected todo will be inserted at the position of nextVisibleNode
	return nextVisibleNode.TodoIndex
}

// DeleteSelected deletes the currently selected todo
// Returns the deleted todo index, or -1 if nothing deleted
func (tree *DocumentTree) DeleteSelected() int {
	selectedNode := tree.GetSelectedNode()
	if selectedNode == nil || selectedNode.Type != DocNodeTodo {
		return -1
	}

	deletedIdx := selectedNode.TodoIndex
	parent := selectedNode.Parent

	// Remove from parent's children
	for i, child := range parent.Children {
		if child == selectedNode {
			parent.Children = append(parent.Children[:i], parent.Children[i+1:]...)
			break
		}
	}

	// Rebuild flat view
	tree.Flat = tree.flattenTree()

	// Find next visible todo for selection
	nextVisible := tree.findNextVisibleTodo(tree.Selected)
	if nextVisible == -1 {
		nextVisible = tree.findPrevVisibleTodo(tree.Selected - 1)
	}

	if nextVisible != -1 {
		tree.Selected = nextVisible
	} else if len(tree.Flat) > 0 {
		tree.Selected = 0
	}

	return deletedIdx
}

// SyncToModel is deprecated - movement now uses direct AST operations via MoveTodoItem
// This method is kept for backward compatibility but should not be used
func (tree *DocumentTree) SyncToModel(m *Model) {
	// No-op: movement is now handled directly in update.go using FileModel.MoveTodoItem
}
