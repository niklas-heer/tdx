package markdown

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
)

// SerializeAST converts an AST back to markdown text
// This is a custom implementation because goldmark's renderer had issues
func SerializeAST(doc *ASTDocument) string {
	if doc.sourceValid {
		return string(doc.Source)
	}
	var buf bytes.Buffer
	serializeNode(doc, doc.AST, &buf, 0)
	return buf.String()
}

func writeInlineNodeText(buf *bytes.Buffer, doc *ASTDocument, node ast.Node) bool {
	switch n := node.(type) {
	case *ast.AutoLink:
		buf.WriteByte('<')
		buf.Write(n.Label(doc.Source))
		buf.WriteByte('>')
		return true
	case *ast.RawHTML:
		buf.Write(n.Segments.Value(doc.Source))
		return true
	default:
		return false
	}
}

func serializeNode(doc *ASTDocument, node ast.Node, buf *bytes.Buffer, depth int, ordinal ...int) {
	if writeInlineNodeText(buf, doc, node) {
		return
	}
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

		if !inListItem && n.Lines().Len() > 0 {
			buf.Write(n.Lines().Value(doc.Source))
			if buf.Len() > 0 && buf.Bytes()[buf.Len()-1] != '\n' {
				buf.WriteByte('\n')
			}
			if n.NextSibling() != nil {
				buf.WriteByte('\n')
			}
			return
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
		// Carry the ordinal forward in one pass, including lists starting above one.
		number := n.Start
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			serializeNode(doc, child, buf, depth, number)
			number++
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
			if list.IsOrdered() {
				number := list.Start
				if len(ordinal) > 0 {
					number = ordinal[0]
				}
				marker = strconv.Itoa(number) + marker
			}
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
					var nested bytes.Buffer
					serializeNode(doc, child, &nested, 0)
					padding := indent + strings.Repeat(" ", len(marker)+1)
					for _, line := range strings.SplitAfter(nested.String(), "\n") {
						if line != "" {
							buf.WriteString(padding)
							buf.WriteString(line)
						}
					}
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
		// Prefix every rendered line, including nested lists and paragraph breaks.
		var quoted bytes.Buffer
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			serializeNode(doc, child, &quoted, depth)
		}
		for _, line := range strings.SplitAfter(quoted.String(), "\n") {
			if line != "" {
				buf.WriteString("> ")
				buf.WriteString(line)
			}
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

	case *ast.HTMLBlock:
		buf.Write(n.Lines().Value(doc.Source))
		if n.HasClosure() {
			buf.Write(n.ClosureLine.Value(doc.Source))
		}
		buf.WriteString("\n\n")

	case *extast.Table:
		for row := n.FirstChild(); row != nil; row = row.NextSibling() {
			buf.WriteString("| ")
			for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
				for inline := cell.FirstChild(); inline != nil; inline = inline.NextSibling() {
					serializeNode(doc, inline, buf, depth)
				}
				buf.WriteString(" |")
				if cell.NextSibling() != nil {
					buf.WriteByte(' ')
				}
			}
			buf.WriteByte('\n')
			if row.Kind() == extast.KindTableHeader {
				buf.WriteByte('|')
				for _, alignment := range n.Alignments {
					delimiter := " --- "
					switch alignment {
					case extast.AlignLeft:
						delimiter = " :--- "
					case extast.AlignRight:
						delimiter = " ---: "
					case extast.AlignCenter:
						delimiter = " :---: "
					}
					buf.WriteString(delimiter)
					buf.WriteByte('|')
				}
				buf.WriteByte('\n')
			}
		}
		buf.WriteByte('\n')

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
