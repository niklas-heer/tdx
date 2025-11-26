package markdown

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
)

// SerializeAST converts an AST back to markdown text
// This is a custom implementation because goldmark's renderer had issues
func SerializeAST(doc *ASTDocument) string {
	var buf bytes.Buffer
	serializeNode(doc, doc.AST, &buf, 0)
	return buf.String()
}

func serializeNode(doc *ASTDocument, node ast.Node, buf *bytes.Buffer, depth int) {
	switch n := node.(type) {
	case *ast.Document:
		// Serialize all children
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			serializeNode(doc, child, buf, depth)
		}

	case *ast.Heading:
		// Write heading markers
		buf.WriteString(strings.Repeat("#", n.Level))
		buf.WriteString(" ")
		// Write heading content
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			serializeNode(doc, child, buf, depth)
		}
		buf.WriteString("\n")
		// Add blank line after headings
		if n.NextSibling() != nil {
			buf.WriteString("\n")
		}

	case *ast.Paragraph:
		// Check if this is inside a list item (for task lists)
		inListItem := false
		for parent := n.Parent(); parent != nil; parent = parent.Parent() {
			if _, ok := parent.(*ast.ListItem); ok {
				inListItem = true
				break
			}
		}

		// Serialize children
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			serializeNode(doc, child, buf, depth)
		}

		// Add newline after paragraph (unless it's in a list item)
		if !inListItem {
			buf.WriteString("\n")
			// Add blank line after paragraphs if next node exists and isn't a list
			if n.NextSibling() != nil {
				if _, isList := n.NextSibling().(*ast.List); !isList {
					buf.WriteString("\n")
				}
			}
		}

	case *ast.List:
		// Serialize all list items
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			serializeNode(doc, child, buf, depth)
		}
		// Add blank line after list
		if n.NextSibling() != nil {
			buf.WriteString("\n")
		}

	case *ast.ListItem:
		// Write list marker with indentation
		indent := strings.Repeat("  ", depth)
		marker := "-"
		if list, ok := n.Parent().(*ast.List); ok {
			marker = string(list.Marker)
		}
		buf.WriteString(indent)
		buf.WriteString(marker)
		buf.WriteString(" ")

		// First pass: serialize non-list children (text content)
		hasNestedList := false
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			if _, isList := child.(*ast.List); isList {
				hasNestedList = true
			} else {
				serializeNode(doc, child, buf, depth)
			}
		}
		buf.WriteString("\n")

		// Second pass: serialize nested lists (after the newline)
		if hasNestedList {
			for child := n.FirstChild(); child != nil; child = child.NextSibling() {
				if _, isList := child.(*ast.List); isList {
					serializeNode(doc, child, buf, depth+1)
				}
			}
		}

	case *extast.TaskCheckBox:
		// Write checkbox with space after it
		if n.IsChecked {
			buf.WriteString("[x] ")
		} else {
			buf.WriteString("[ ] ")
		}

	case *ast.Text:
		// Write text content from source
		segment := n.Segment
		buf.Write(segment.Value(doc.Source))
		if n.SoftLineBreak() {
			buf.WriteString(" ")
		}

	case *ast.String:
		// Write string value
		buf.Write(n.Value)

	case *ast.CodeSpan:
		// Write inline code
		buf.WriteString("`")
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			serializeNode(doc, child, buf, depth)
		}
		buf.WriteString("`")

	case *ast.CodeBlock, *ast.FencedCodeBlock:
		// Handle code blocks
		buf.WriteString("```")
		if fenced, ok := n.(*ast.FencedCodeBlock); ok {
			// Write language if present
			if fenced.Language(doc.Source) != nil {
				buf.Write(fenced.Language(doc.Source))
			}
		}
		buf.WriteString("\n")

		// Write code content
		lines := n.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			buf.Write(line.Value(doc.Source))
		}
		buf.WriteString("```\n\n")

	case *ast.Blockquote:
		// Serialize blockquote
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			buf.WriteString("> ")
			serializeNode(doc, child, buf, depth)
		}
		if n.NextSibling() != nil {
			buf.WriteString("\n")
		}

	case *ast.Link:
		// Write link: [text](url)
		buf.WriteString("[")
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			serializeNode(doc, child, buf, depth)
		}
		buf.WriteString("](")
		buf.Write(n.Destination)
		if n.Title != nil {
			buf.WriteString(` "`)
			buf.Write(n.Title)
			buf.WriteString(`"`)
		}
		buf.WriteString(")")

	case *ast.Image:
		// Write image: ![alt](url)
		buf.WriteString("![")
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			serializeNode(doc, child, buf, depth)
		}
		buf.WriteString("](")
		buf.Write(n.Destination)
		if n.Title != nil {
			buf.WriteString(` "`)
			buf.Write(n.Title)
			buf.WriteString(`"`)
		}
		buf.WriteString(")")

	case *ast.Emphasis:
		// Check emphasis level
		if n.Level == 2 {
			// Strong (**text**)
			buf.WriteString("**")
			for child := n.FirstChild(); child != nil; child = child.NextSibling() {
				serializeNode(doc, child, buf, depth)
			}
			buf.WriteString("**")
		} else {
			// Regular emphasis (*text*)
			buf.WriteString("*")
			for child := n.FirstChild(); child != nil; child = child.NextSibling() {
				serializeNode(doc, child, buf, depth)
			}
			buf.WriteString("*")
		}

	case *ast.ThematicBreak:
		// Horizontal rule
		buf.WriteString("---\n\n")

	case *extast.Strikethrough:
		// Write strikethrough (~~text~~)
		buf.WriteString("~~")
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			serializeNode(doc, child, buf, depth)
		}
		buf.WriteString("~~")

	case *extast.Table:
		// TODO: Implement table serialization if needed
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			serializeNode(doc, child, buf, depth)
		}

	default:
		// For unknown node types, try to serialize children
		if node.HasChildren() {
			for child := node.FirstChild(); child != nil; child = child.NextSibling() {
				serializeNode(doc, child, buf, depth)
			}
		}
	}
}

// EnsureTrailingNewline ensures the markdown ends with a newline
func EnsureTrailingNewline(content string) string {
	if !strings.HasSuffix(content, "\n") {
		return content + "\n"
	}
	return content
}

// EnsureHeader ensures the document starts with a header
func EnsureHeader(content string) string {
	lines := strings.Split(content, "\n")
	hasHeader := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			hasHeader = true
			break
		}
		// If we hit content before a header, there's no header
		break
	}

	if !hasHeader {
		return "# Todos\n\n" + content
	}
	return content
}
