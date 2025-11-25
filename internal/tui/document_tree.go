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

// MoveUp moves the selected todo up ONE POSITION in the visible list
// Returns (fromTodoIndex, toTodoIndex) for the AST move operation, or (-1, -1) if cannot move
func (tree *DocumentTree) MoveUp() (int, int) {
	selectedNode := tree.GetSelectedNode()
	if selectedNode == nil || selectedNode.Type != DocNodeTodo || !selectedNode.Visible {
		return -1, -1
	}

	// Find previous visible item in the flat list (can be heading or todo)
	targetVisualPos := -1
	for i := tree.Selected - 1; i >= 0; i-- {
		if tree.Flat[i].Visible {
			targetVisualPos = i
			break
		}
	}

	if targetVisualPos == -1 {
		return -1, -1 // Already at top of visible list
	}

	targetNode := tree.Flat[targetVisualPos]

	if targetNode.Type == DocNodeTodo {
		// Simple case: target is a todo, insert before it
		return selectedNode.TodoIndex, targetNode.TodoIndex
	} else {
		// Target is a heading - we want to appear above it (just before the heading)
		// Find the last todo BEFORE this heading
		insertAfterTodoIndex := -1
		for i := targetVisualPos - 1; i >= 0; i-- {
			if tree.Flat[i].Type == DocNodeTodo {
				insertAfterTodoIndex = tree.Flat[i].TodoIndex
				break
			}
		}

		if insertAfterTodoIndex == -1 {
			// No todos before heading - insert at beginning (position 0)
			return selectedNode.TodoIndex, 0
		}

		// We want to insert AFTER the last todo before the heading
		// MoveTodo with fromIndex > toIndex inserts BEFORE toIndex
		// MoveTodo with fromIndex < toIndex inserts AFTER toIndex
		// Since we want to insert AFTER insertAfterTodoIndex, and our fromIndex is likely > insertAfterTodoIndex,
		// we need to return it such that the AST operation does the right thing
		// Actually, just return the todo we want to insert after - the handler will figure it out
		return selectedNode.TodoIndex, insertAfterTodoIndex
	}
}

// MoveDown moves the selected todo down ONE POSITION in the visible list
// Returns (fromTodoIndex, toTodoIndex) for the AST move operation, or (-1, -1) if cannot move
func (tree *DocumentTree) MoveDown() (int, int) {
	selectedNode := tree.GetSelectedNode()
	if selectedNode == nil || selectedNode.Type != DocNodeTodo || !selectedNode.Visible {
		return -1, -1
	}

	// Find next visible item in the flat list (can be heading or todo)
	targetVisualPos := -1
	for i := tree.Selected + 1; i < len(tree.Flat); i++ {
		if tree.Flat[i].Visible {
			targetVisualPos = i
			break
		}
	}

	if targetVisualPos == -1 {
		return -1, -1 // Already at bottom of visible list
	}

	// Now we know: selectedNode should visually appear at position targetVisualPos
	// We need to determine where in the FILE this todo should be inserted

	targetNode := tree.Flat[targetVisualPos]

	// If target is a heading, we can't move "onto" a heading when moving down
	// We need to skip to the first todo under that heading, or the next item after it

	if targetNode.Type == DocNodeHeading {
		// Find the first todo under this heading, or the next visible item
		insertAfterTodoIndex := -1
		for i := targetVisualPos + 1; i < len(tree.Flat); i++ {
			if tree.Flat[i].Type == DocNodeTodo {
				// Insert after this todo (which is under the heading)
				insertAfterTodoIndex = tree.Flat[i].TodoIndex
				break
			}
			if tree.Flat[i].Visible && tree.Flat[i].Type == DocNodeHeading {
				// Hit another heading - insert before the first todo under it
				// Actually, this means we should keep looking
				continue
			}
		}

		if insertAfterTodoIndex == -1 {
			// No todos found - insert at end
			return selectedNode.TodoIndex, len(tree.Flat) - 1
		}

		return selectedNode.TodoIndex, insertAfterTodoIndex
	} else {
		// Target is a todo - insert after it (to appear below it visually)
		return selectedNode.TodoIndex, targetNode.TodoIndex
	}
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
