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
	Source      []byte
	AST         ast.Node
	sourceValid bool                   // Source still represents the complete current tree.
	checkboxes  []*extast.TaskCheckBox // Populated in extraction order.
}

// TodoNode represents a todo item in the AST with its associated checkbox
type TodoNode struct {
	ListItem *ast.ListItem
	CheckBox *extast.TaskCheckBox
	TextNode *ast.Text
	Checked  bool
}

func newMarkdownParser() goldmark.Markdown {
	// Linkify turns source text into childless AutoLink nodes. The editor does
	// not need HTML-style linkification, and retaining text nodes preserves the
	// exact URL or email spelling through mutations and serialization.
	return goldmark.New(
		goldmark.WithExtensions(
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
		),
	)
}

// ParseAST parses markdown content into a goldmark AST
func ParseAST(content string) (*ASTDocument, error) {
	source := []byte(content)
	md := newMarkdownParser()
	doc := md.Parser().Parse(text.NewReader(source))

	return &ASTDocument{
		Source:      source,
		AST:         doc,
		sourceValid: true,
	}, nil
}

// ExtractTodos walks the AST and extracts all task list items with nesting information
func (doc *ASTDocument) ExtractTodos() []Todo {
	var todos []Todo
	doc.checkboxes = nil
	todoIndex := 0

	// Forward declare walkList so walkListItem can call it
	var walkList func(list *ast.List, depth int, parentIdx int)

	// walkListItem processes a single list item and its nested lists
	var walkListItem func(listItem *ast.ListItem, depth int, parentIdx int) //nolint:unparam,staticcheck
	walkListItem = func(listItem *ast.ListItem, depth int, parentIdx int) {
		// Check if this list item has a checkbox (task item)
		var checkbox *extast.TaskCheckBox
		var textBlock ast.Node

		for child := listItem.FirstChild(); child != nil; child = child.NextSibling() {
			// Look for checkbox in the first text block/paragraph
			if child.Kind() == ast.KindTextBlock || child.Kind() == ast.KindParagraph {
				_ = ast.Walk(child, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
					if entering && n.Kind() == extast.KindTaskCheckBox {
						checkbox = n.(*extast.TaskCheckBox)
						textBlock = child
						return ast.WalkStop, nil
					}
					return ast.WalkContinue, nil
				})
				if checkbox != nil {
					break
				}
			}
		}

		// If this is a task item, extract it
		currentIdx := -1
		if checkbox != nil {
			doc.checkboxes = append(doc.checkboxes, checkbox)
			text := doc.extractTodoText(listItem, checkbox)
			tags := ExtractTags(text)
			priority := ExtractPriority(text)

			// Get line number from textBlock
			lineNo := 0
			if textBlock != nil && textBlock.Lines().Len() > 0 {
				lineNo = textBlock.Lines().At(0).Start
			}

			todo := Todo{
				Index:       todoIndex + 1,
				Checked:     checkbox.IsChecked,
				Text:        text,
				LineNo:      lineNo,
				Tags:        tags,
				Priority:    priority,
				Depth:       depth,
				ParentIndex: parentIdx,
				DueDate:     ExtractDueDate(text),
			}
			todos = append(todos, todo)
			currentIdx = todoIndex
			todoIndex++
		}

		// Process nested lists within this list item
		for child := listItem.FirstChild(); child != nil; child = child.NextSibling() {
			if nestedList, ok := child.(*ast.List); ok {
				walkList(nestedList, depth+1, currentIdx)
			}
		}
	}

	// walkList processes all items in a list
	walkList = func(list *ast.List, depth int, parentIdx int) {
		for child := list.FirstChild(); child != nil; child = child.NextSibling() {
			if listItem, ok := child.(*ast.ListItem); ok {
				walkListItem(listItem, depth, parentIdx)
			}
		}
	}

	// Walk the document and find all top-level lists
	_ = ast.Walk(doc.AST, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if list, ok := node.(*ast.List); ok {
				// Only process lists that are NOT inside a ListItem (top-level lists)
				parent := list.Parent()
				if _, parentIsListItem := parent.(*ast.ListItem); !parentIsListItem {
					walkList(list, 0, -1)
					return ast.WalkSkipChildren, nil
				}
			}
		}
		return ast.WalkContinue, nil
	})

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
	// Count the same exposed tasks as ExtractTodos. Checkboxes in opaque nested
	// containers (for example a quoted example inside a task body) are not indexes.
	if doc.checkboxes == nil {
		doc.ExtractTodos()
	}
	indexed := make(map[ast.Node]bool, len(doc.checkboxes))
	for _, checkbox := range doc.checkboxes {
		indexed[checkbox] = true
	}
	var headings []Heading
	nextTodoIndex := 0

	// Use structure-based walk to respect modified parent-child relationships
	var walkNode func(ast.Node)
	walkNode = func(node ast.Node) {
		// Process headings before processing their children
		if node.Kind() == ast.KindHeading {
			heading := node.(*ast.Heading)

			// Retain inline Markdown so renaming does not silently drop links or emphasis.
			var headingText bytes.Buffer
			for child := heading.FirstChild(); child != nil; child = child.NextSibling() {
				serializeNode(doc, child, &headingText, 0)
			}

			// Get line number
			lineNo := 0
			if heading.Lines().Len() > 0 {
				lineNo = heading.Lines().At(0).Start
			}

			// The heading appears before the next todo we'll encounter
			headings = append(headings, Heading{
				Level:           heading.Level,
				Text:            headingText.String(),
				LineNo:          lineNo,
				BeforeTodoIndex: nextTodoIndex,
			})
		}

		// Count todos as we encounter them
		if indexed[node] {
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
		// Skip nested lists - they are separate todos, not part of this todo's text
		if child.Kind() == ast.KindList {
			continue
		}

		// Walk this child and collect text nodes
		_ = ast.Walk(child, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			// Skip the checkbox itself
			if n == checkbox {
				return ast.WalkSkipChildren, nil
			}

			// Skip any nested lists encountered during walk
			if n.Kind() == ast.KindList {
				return ast.WalkSkipChildren, nil
			}
			if entering && writeInlineNodeText(&buf, doc, n) {
				return ast.WalkSkipChildren, nil
			}

			// Collect text content
			switch node := n.(type) {
			case *ast.Text:
				if entering {
					segment := node.Segment
					buf.Write(segment.Value(doc.Source))
					// Add space if this is a soft line break
					if node.SoftLineBreak() {
						buf.WriteByte(' ')
					}
				}
			case *ast.String:
				if entering {
					buf.Write(node.Value)
				}
			case *ast.CodeSpan:
				if entering {
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
				}
			case *ast.Link:
				// Preserve full markdown link syntax [text](url)
				if entering {
					buf.WriteByte('[')
					// Collect link text from children
					for linkChild := node.FirstChild(); linkChild != nil; linkChild = linkChild.NextSibling() {
						if textNode, ok := linkChild.(*ast.Text); ok {
							segment := textNode.Segment
							buf.Write(segment.Value(doc.Source))
						}
					}
					buf.WriteByte(']')
					buf.WriteByte('(')
					buf.Write(node.Destination)
					buf.WriteByte(')')
					return ast.WalkSkipChildren, nil
				}
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
	if doc.checkboxes == nil {
		doc.ExtractTodos()
	}
	if todoIndex < 0 || todoIndex >= len(doc.checkboxes) {
		return nil, fmt.Errorf("todo at index %d not found", todoIndex)
	}
	checkbox := doc.checkboxes[todoIndex]
	container := checkbox.Parent()
	if container == nil {
		return nil, fmt.Errorf("task checkbox has no text container")
	}
	item, ok := container.Parent().(*ast.ListItem)
	if !ok {
		return nil, fmt.Errorf("task checkbox has no list item")
	}
	var textNode *ast.Text
	_ = ast.Walk(container, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if text, ok := node.(*ast.Text); ok {
				textNode = text
				return ast.WalkStop, nil
			}
		}
		return ast.WalkContinue, nil
	})
	return &TodoNode{ListItem: item, CheckBox: checkbox, TextNode: textNode, Checked: checkbox.IsChecked}, nil
}

// ToggleTodo toggles the checked state of a todo
func (doc *ASTDocument) ToggleTodo(todoIndex int) error {
	var checkbox *extast.TaskCheckBox
	if todoIndex >= 0 && todoIndex < len(doc.checkboxes) {
		checkbox = doc.checkboxes[todoIndex]
	} else {
		node, err := doc.FindTodoNode(todoIndex)
		if err != nil {
			return err
		}
		checkbox = node.CheckBox
	}
	if doc.sourceValid {
		parent := checkbox.Parent()
		if parent == nil || parent.Lines().Len() == 0 {
			return fmt.Errorf("checkbox source location unavailable")
		}
		start := parent.Lines().At(0).Start
		if start+2 >= len(doc.Source) || doc.Source[start] != '[' || doc.Source[start+2] != ']' {
			return fmt.Errorf("checkbox source location invalid")
		}
		mark := byte('x')
		if checkbox.IsChecked {
			mark = ' '
		}
		doc.Source[start+1] = mark
	}
	checkbox.IsChecked = !checkbox.IsChecked
	return nil
}
