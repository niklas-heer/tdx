package markdown

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// ASTDocument holds the goldmark AST and provides operations on it
type ASTDocument struct {
	Source []byte
	AST    ast.Node
}

// TodoNode represents a todo item in the AST with its associated checkbox
type TodoNode struct {
	ListItem *ast.ListItem
	CheckBox *extast.TaskCheckBox
	TextNode *ast.Text
	Checked  bool
}

// ParseAST parses markdown content into a goldmark AST
func ParseAST(content string) (*ASTDocument, error) {
	source := []byte(content)

	// Create parser with GFM extension for task list support
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM, // GitHub Flavored Markdown
		),
	)

	doc := md.Parser().Parse(text.NewReader(source))

	return &ASTDocument{
		Source: source,
		AST:    doc,
	}, nil
}

// ExtractTodos walks the AST and extracts all task list items
func (doc *ASTDocument) ExtractTodos() []Todo {
	var todos []Todo
	todoIndex := 1

	// Walk the document structure in order, processing List nodes
	// This ensures we respect the parent-child relationships we've modified
	var walkNode func(ast.Node)
	walkNode = func(node ast.Node) {
		// Process current node
		if node.Kind() == extast.KindTaskCheckBox {
			checkbox := node.(*extast.TaskCheckBox)

			// Navigate: TaskCheckBox -> TextBlock -> ListItem
			textBlock := checkbox.Parent()
			if textBlock != nil {
				listItem := textBlock.Parent()
				if listItem != nil {
					// Extract text content (everything after the checkbox)
					text := doc.extractTodoText(listItem, checkbox)

					// Extract tags from the text
					tags := ExtractTags(text)

					// Get line number from textBlock (ListItem doesn't have Lines())
					lineNo := 0
					if textBlock.Lines().Len() > 0 {
						lineNo = textBlock.Lines().At(0).Start
					}

					todo := Todo{
						Index:   todoIndex,
						Checked: checkbox.IsChecked,
						Text:    text,
						LineNo:  lineNo,
						Tags:    tags,
					}
					todos = append(todos, todo)
					todoIndex++
				}
			}
		}

		// Walk children in document order by iterating through siblings
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			walkNode(child)
		}
	}

	walkNode(doc.AST)

	return todos
}

// Heading represents a markdown heading with its position
type Heading struct {
	Level           int // 1-6 for h1-h6
	Text            string
	LineNo          int
	BeforeTodoIndex int // Which todo this heading appears before (0 = before first todo, -1 = after all todos)
}

// ExtractHeadings walks the AST and extracts all headings with their positions relative to todos
func (doc *ASTDocument) ExtractHeadings() []Heading {
	var headings []Heading
	nextTodoIndex := 0

	// Use structure-based walk to respect modified parent-child relationships
	var walkNode func(ast.Node)
	walkNode = func(node ast.Node) {
		// Process headings before processing their children
		if node.Kind() == ast.KindHeading {
			heading := node.(*ast.Heading)

			// Extract text from heading
			var text strings.Builder
			for child := heading.FirstChild(); child != nil; child = child.NextSibling() {
				if textNode, ok := child.(*ast.Text); ok {
					text.Write(textNode.Segment.Value(doc.Source))
				}
			}

			// Get line number
			lineNo := 0
			if heading.Lines().Len() > 0 {
				lineNo = heading.Lines().At(0).Start
			}

			// The heading appears before the next todo we'll encounter
			headings = append(headings, Heading{
				Level:           heading.Level,
				Text:            text.String(),
				LineNo:          lineNo,
				BeforeTodoIndex: nextTodoIndex,
			})
		}

		// Count todos as we encounter them
		if node.Kind() == extast.KindTaskCheckBox {
			nextTodoIndex++
		}

		// Walk children in document order by iterating through siblings
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			walkNode(child)
		}
	}

	walkNode(doc.AST)

	return headings
}

// extractTodoText extracts the text content from a list item, excluding the checkbox
func (doc *ASTDocument) extractTodoText(listItem ast.Node, checkbox ast.Node) string {
	var buf bytes.Buffer

	for child := listItem.FirstChild(); child != nil; child = child.NextSibling() {
		// Walk this child and collect text nodes
		ast.Walk(child, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering {
				return ast.WalkContinue, nil
			}

			// Skip the checkbox itself
			if n == checkbox {
				return ast.WalkSkipChildren, nil
			}

			// Collect text content
			switch node := n.(type) {
			case *ast.Text:
				segment := node.Segment
				buf.Write(segment.Value(doc.Source))
				// Add space if this is a soft line break
				if node.SoftLineBreak() {
					buf.WriteByte(' ')
				}
			case *ast.String:
				buf.Write(node.Value)
			case *ast.CodeSpan:
				// Code spans need special handling
				buf.WriteByte('`')
				for child := node.FirstChild(); child != nil; child = child.NextSibling() {
					if textNode, ok := child.(*ast.Text); ok {
						segment := textNode.Segment
						buf.Write(segment.Value(doc.Source))
					}
				}
				buf.WriteByte('`')
				return ast.WalkSkipChildren, nil
			case *ast.Link:
				// Preserve link syntax
				buf.WriteByte('[')
			case *ast.Emphasis:
				// Could preserve emphasis markers if needed
			}

			return ast.WalkContinue, nil
		})
	}

	return strings.TrimSpace(buf.String())
}

// FindTodoNode finds the TodoNode for a given todo index
func (doc *ASTDocument) FindTodoNode(todoIndex int) (*TodoNode, error) {
	currentIndex := 0

	var found *TodoNode
	ast.Walk(doc.AST, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || found != nil {
			return ast.WalkContinue, nil
		}

		if node.Kind() == extast.KindTaskCheckBox {
			if currentIndex == todoIndex {
				checkbox := node.(*extast.TaskCheckBox)
				textBlock := checkbox.Parent()
				if textBlock == nil {
					return ast.WalkContinue, nil
				}
				listItem := textBlock.Parent()
				if listItem == nil {
					return ast.WalkContinue, nil
				}

				// Find the text node containing the checkbox text
				var textNode *ast.Text
				ast.Walk(textBlock, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
					if entering {
						if tn, ok := n.(*ast.Text); ok {
							textNode = tn
							return ast.WalkStop, nil
						}
					}
					return ast.WalkContinue, nil
				})

				found = &TodoNode{
					ListItem: listItem.(*ast.ListItem),
					CheckBox: checkbox,
					TextNode: textNode,
					Checked:  checkbox.IsChecked,
				}
				return ast.WalkStop, nil
			}
			currentIndex++
		}

		return ast.WalkContinue, nil
	})

	if found == nil {
		return nil, fmt.Errorf("todo at index %d not found", todoIndex)
	}

	return found, nil
}

// ToggleTodo toggles the checked state of a todo
func (doc *ASTDocument) ToggleTodo(todoIndex int) error {
	node, err := doc.FindTodoNode(todoIndex)
	if err != nil {
		return err
	}

	// Toggle the checkbox state
	node.CheckBox.IsChecked = !node.CheckBox.IsChecked
	node.Checked = node.CheckBox.IsChecked

	return nil
}

// UpdateTodoText updates the text of a todo
func (doc *ASTDocument) UpdateTodoText(todoIndex int, newText string) error {
	node, err := doc.FindTodoNode(todoIndex)
	if err != nil {
		return err
	}

	// Get the list item and its parent
	listItem := node.ListItem
	parentList := listItem.Parent()
	if parentList == nil {
		return fmt.Errorf("list item has no parent")
	}

	// Find the position of this list item in its parent
	var prevSibling ast.Node
	for child := parentList.FirstChild(); child != nil; child = child.NextSibling() {
		if child == listItem {
			break
		}
		prevSibling = child
	}

	// Create a complete markdown list item with checkbox to parse properly
	var tempMarkdown string
	if node.CheckBox.IsChecked {
		tempMarkdown = "- [x] " + newText
	} else {
		tempMarkdown = "- [ ] " + newText
	}

	// Append to source
	sourceStart := len(doc.Source)
	doc.Source = append(doc.Source, []byte(tempMarkdown)...)

	// Parse as a complete list item to get proper inline element handling
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
		),
	)
	tempDoc := md.Parser().Parse(text.NewReader([]byte(tempMarkdown)))

	// Find the parsed list item
	var newListItem *ast.ListItem
	ast.Walk(tempDoc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || newListItem != nil {
			return ast.WalkContinue, nil
		}

		if li, ok := n.(*ast.ListItem); ok {
			// Verify it has a checkbox
			hasCheckbox := false
			ast.Walk(li, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
				if entering && child.Kind() == extast.KindTaskCheckBox {
					hasCheckbox = true
					return ast.WalkStop, nil
				}
				return ast.WalkContinue, nil
			})
			if hasCheckbox {
				newListItem = li
				return ast.WalkStop, nil
			}
		}

		return ast.WalkContinue, nil
	})

	if newListItem == nil {
		return fmt.Errorf("failed to parse new todo text")
	}

	// Detach from temp document
	if newListItem.Parent() != nil {
		newListItem.Parent().RemoveChild(newListItem.Parent(), newListItem)
	}

	// Adjust all segments to point to our source
	adjustNodeSegments(newListItem, sourceStart)

	// Replace old list item with new one
	parentList.RemoveChild(parentList, listItem)

	if prevSibling == nil {
		// Insert at beginning
		if parentList.FirstChild() != nil {
			parentList.InsertBefore(parentList, parentList.FirstChild(), newListItem)
		} else {
			parentList.AppendChild(parentList, newListItem)
		}
	} else {
		// Insert after previous sibling
		parentList.InsertAfter(parentList, prevSibling, newListItem)
	}

	return nil
}

// adjustNodeSegments recursively adjusts all segment positions in a node tree
func adjustNodeSegments(node ast.Node, offset int) {
	// Adjust this node's segment if it has one
	if n, ok := node.(*ast.Text); ok {
		seg := n.Segment
		n.Segment = text.NewSegment(seg.Start+offset, seg.Stop+offset)
	}

	// Recursively adjust children
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		adjustNodeSegments(child, offset)
	}
}

// DeleteTodo removes a todo from the AST
func (doc *ASTDocument) DeleteTodo(todoIndex int) error {
	node, err := doc.FindTodoNode(todoIndex)
	if err != nil {
		return err
	}

	// Remove the list item from its parent
	parent := node.ListItem.Parent()
	if parent != nil {
		parent.RemoveChild(parent, node.ListItem)
	}

	return nil
}

// AddTodo adds a new todo to the AST
func (doc *ASTDocument) AddTodo(todoText string, checked bool) error {
	// Find the last list in the document, or create one
	var lastList *ast.List

	ast.Walk(doc.AST, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if list, ok := node.(*ast.List); ok {
				// Check if this list contains task items
				hasTaskItems := false
				for child := list.FirstChild(); child != nil; child = child.NextSibling() {
					if listItem, ok := child.(*ast.ListItem); ok {
						// Check if this list item has a checkbox
						ast.Walk(listItem, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
							if entering && n.Kind() == extast.KindTaskCheckBox {
								hasTaskItems = true
								return ast.WalkStop, nil
							}
							return ast.WalkContinue, nil
						})
						if hasTaskItems {
							break
						}
					}
				}
				if hasTaskItems {
					lastList = list
				}
			}
		}
		return ast.WalkContinue, nil
	})

	// If no list found, create one and append to document
	if lastList == nil {
		lastList = ast.NewList(0) // 0 = unordered list with '-'
		lastList.Marker = '-'
		doc.AST.AppendChild(doc.AST, lastList)
	}

	// Append new text to source (this is a bit of a hack but necessary for AST)
	newTodoText := []byte(todoText)
	sourceStart := len(doc.Source)
	doc.Source = append(doc.Source, newTodoText...)

	// Create new list item with checkbox and text
	listItem := ast.NewListItem(0)

	// Create paragraph (required for task list items)
	para := ast.NewParagraph()
	listItem.AppendChild(listItem, para)

	// Create checkbox
	checkbox := extast.NewTaskCheckBox(checked)
	para.AppendChild(para, checkbox)

	// Create text node pointing to the new source bytes
	textNode := ast.NewText()
	textNode.Segment = text.NewSegment(sourceStart, sourceStart+len(newTodoText))
	para.AppendChild(para, textNode)

	// Append to list
	lastList.AppendChild(lastList, listItem)

	return nil
}

// InsertTodoAfter inserts a new todo after the specified index
// If afterIndex is -1, inserts at the beginning of the first todo list
func (doc *ASTDocument) InsertTodoAfter(afterIndex int, todoText string, checked bool) error {
	// Append new text to source
	newTodoText := []byte(todoText)
	sourceStart := len(doc.Source)
	doc.Source = append(doc.Source, newTodoText...)

	// Create new list item with checkbox and text
	newListItem := ast.NewListItem(0)
	para := ast.NewParagraph()
	newListItem.AppendChild(newListItem, para)
	checkbox := extast.NewTaskCheckBox(checked)
	para.AppendChild(para, checkbox)
	textNode := ast.NewText()
	textNode.Segment = text.NewSegment(sourceStart, sourceStart+len(newTodoText))
	para.AppendChild(para, textNode)

	if afterIndex < 0 {
		// Insert at the beginning of the first todo list
		var firstList *ast.List
		ast.Walk(doc.AST, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
			if entering {
				if list, ok := node.(*ast.List); ok {
					hasTaskItems := false
					for child := list.FirstChild(); child != nil; child = child.NextSibling() {
						if listItem, ok := child.(*ast.ListItem); ok {
							ast.Walk(listItem, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
								if entering && n.Kind() == extast.KindTaskCheckBox {
									hasTaskItems = true
									return ast.WalkStop, nil
								}
								return ast.WalkContinue, nil
							})
							if hasTaskItems {
								break
							}
						}
					}
					if hasTaskItems {
						firstList = list
						return ast.WalkStop, nil
					}
				}
			}
			return ast.WalkContinue, nil
		})

		if firstList == nil {
			// No existing list, create one
			firstList = ast.NewList(0)
			firstList.Marker = '-'
			doc.AST.AppendChild(doc.AST, firstList)
			firstList.AppendChild(firstList, newListItem)
		} else {
			// Insert before the first child
			firstChild := firstList.FirstChild()
			if firstChild != nil {
				firstList.InsertBefore(firstList, firstChild, newListItem)
			} else {
				firstList.AppendChild(firstList, newListItem)
			}
		}
		return nil
	}

	// Find the todo node at afterIndex
	node, err := doc.FindTodoNode(afterIndex)
	if err != nil {
		return err
	}

	// Get the parent list
	parentList := node.ListItem.Parent()
	if parentList == nil {
		return fmt.Errorf("list item has no parent")
	}

	// Insert after the found list item
	nextSibling := node.ListItem.NextSibling()
	if nextSibling != nil {
		parentList.InsertBefore(parentList, nextSibling, newListItem)
	} else {
		parentList.AppendChild(parentList, newListItem)
	}

	return nil
}

// SwapTodos swaps two todos in the AST
// MoveTodoToPosition moves a todo by removing it and inserting before/after another todo
// insertAfter: if true, insert after targetIndex; if false, insert before targetIndex
func (doc *ASTDocument) MoveTodoToPosition(fromIndex, targetIndex int, insertAfter bool) error {
	if fromIndex == targetIndex {
		return nil // No-op
	}

	// Get the node to move
	nodeFrom, err := doc.FindTodoNode(fromIndex)
	if err != nil {
		return err
	}

	// Get the target node
	nodeTarget, err := doc.FindTodoNode(targetIndex)
	if err != nil {
		return err
	}

	// Remove from current parent
	parentFrom := nodeFrom.ListItem.Parent()
	parentFrom.RemoveChild(parentFrom, nodeFrom.ListItem)

	// Insert at target position
	parentTarget := nodeTarget.ListItem.Parent()
	if insertAfter {
		parentTarget.InsertAfter(parentTarget, nodeTarget.ListItem, nodeFrom.ListItem)
	} else {
		parentTarget.InsertBefore(parentTarget, nodeTarget.ListItem, nodeFrom.ListItem)
	}

	return nil
}

// MoveTodo moves a todo from fromIndex to toIndex (shifts other todos)
// Deprecated: Use MoveTodoToPosition for more explicit control
func (doc *ASTDocument) MoveTodo(fromIndex, toIndex int) error {
	if fromIndex == toIndex {
		return nil // No-op
	}

	// Get both nodes BEFORE any modifications
	nodeFrom, err := doc.FindTodoNode(fromIndex)
	if err != nil {
		return err
	}

	nodeTo, err := doc.FindTodoNode(toIndex)
	if err != nil {
		return err
	}

	// Get parents
	parentFrom := nodeFrom.ListItem.Parent()
	parentTo := nodeTo.ListItem.Parent()

	// Remove the moving node from its current position
	parentFrom.RemoveChild(parentFrom, nodeFrom.ListItem)

	// Insert at new position relative to nodeTo
	if fromIndex < toIndex {
		// Moving down: insert AFTER nodeTo
		parentTo.InsertAfter(parentTo, nodeTo.ListItem, nodeFrom.ListItem)
	} else {
		// Moving up: insert BEFORE nodeTo
		parentTo.InsertBefore(parentTo, nodeTo.ListItem, nodeFrom.ListItem)
	}

	return nil
}

// SwapTodos swaps two todos in the AST (kept for backward compatibility but prefer MoveTodo)
func (doc *ASTDocument) SwapTodos(index1, index2 int) error {
	node1, err := doc.FindTodoNode(index1)
	if err != nil {
		return err
	}

	node2, err := doc.FindTodoNode(index2)
	if err != nil {
		return err
	}

	// Get parent (SwapTodos assumes same parent for simplicity)
	parent1 := node1.ListItem.Parent()

	// NOTE: SwapTodos is deprecated in favor of MoveTodo for better UX
	// MoveTodo provides better insertion behavior and handles cross-parent moves

	// Find what comes before each node (these will be our anchors)
	var prev1, prev2 ast.Node
	for child := parent1.FirstChild(); child != nil; child = child.NextSibling() {
		if child.NextSibling() == node1.ListItem {
			prev1 = child
		}
		if child.NextSibling() == node2.ListItem {
			prev2 = child
		}
	}

	// Check if they are adjacent
	adjacent := (node1.ListItem.NextSibling() == node2.ListItem) || (node2.ListItem.NextSibling() == node1.ListItem)

	if adjacent {
		// For adjacent nodes, we need special handling
		// Determine which comes first
		var firstNode, secondNode *TodoNode
		var prevFirst ast.Node

		if node1.ListItem.NextSibling() == node2.ListItem {
			firstNode, secondNode = node1, node2
			prevFirst = prev1
		} else {
			firstNode, secondNode = node2, node1
			prevFirst = prev2
		}

		// Remove both nodes
		parent1.RemoveChild(parent1, firstNode.ListItem)
		parent1.RemoveChild(parent1, secondNode.ListItem)

		// Insert in swapped order
		if prevFirst != nil {
			// Insert second after prevFirst
			parent1.InsertAfter(parent1, prevFirst, secondNode.ListItem)
			// Insert first after second
			parent1.InsertAfter(parent1, secondNode.ListItem, firstNode.ListItem)
		} else {
			// First node was at the beginning
			if parent1.FirstChild() != nil {
				// Insert at beginning
				parent1.InsertBefore(parent1, parent1.FirstChild(), secondNode.ListItem)
			} else {
				parent1.AppendChild(parent1, secondNode.ListItem)
			}
			parent1.InsertAfter(parent1, secondNode.ListItem, firstNode.ListItem)
		}
	} else {
		// Non-adjacent nodes: swap them independently
		// Remove both from parent
		parent1.RemoveChild(parent1, node1.ListItem)
		parent1.RemoveChild(parent1, node2.ListItem)

		// After removing node1, if prev2 was node1, we need to adjust
		if prev2 == node1.ListItem {
			prev2 = prev1
		}

		// Re-insert in swapped order
		if prev1 != nil {
			parent1.InsertAfter(parent1, prev1, node2.ListItem)
		} else {
			// node1 was first child
			if parent1.FirstChild() != nil {
				parent1.InsertBefore(parent1, parent1.FirstChild(), node2.ListItem)
			} else {
				parent1.AppendChild(parent1, node2.ListItem)
			}
		}

		if prev2 != nil {
			parent1.InsertAfter(parent1, prev2, node1.ListItem)
		} else {
			// node2 was first child
			if parent1.FirstChild() != nil {
				parent1.InsertBefore(parent1, parent1.FirstChild(), node1.ListItem)
			} else {
				parent1.AppendChild(parent1, node1.ListItem)
			}
		}
	}

	return nil
}
