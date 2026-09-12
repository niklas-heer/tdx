package markdown

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
)

func (fm *FileModel) headingNodes() []*ast.Heading {
	var headings []*ast.Heading
	if fm.ast == nil {
		return headings
	}
	_ = ast.Walk(fm.ast.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if h, ok := n.(*ast.Heading); ok && entering {
			headings = append(headings, h)
		}
		return ast.WalkContinue, nil
	})
	return headings
}

func validHeading(title string, level int) error {
	if strings.TrimSpace(title) == "" || strings.ContainsAny(title, "\r\n") {
		return fmt.Errorf("heading must have a non-empty, single-line title")
	}
	if level < 1 || level > 6 {
		return fmt.Errorf("heading level must be between 1 and 6")
	}
	return nil
}

// RenameHeading changes exactly the selected heading and preserves its source style.
func (fm *FileModel) RenameHeading(index int, title string) error {
	headings := fm.headingNodes()
	if index < 0 || index >= len(headings) {
		return fmt.Errorf("section no longer exists")
	}
	h := headings[index]
	if err := validHeading(title, h.Level); err != nil {
		return err
	}
	if h.Lines().Len() == 0 {
		return fmt.Errorf("heading source location unavailable")
	}
	start := h.Lines().At(0).Start
	end := h.Lines().At(h.Lines().Len() - 1).Stop
	for end > start && (fm.ast.Source[end-1] == '\n' || fm.ast.Source[end-1] == '\r') {
		end--
	}
	if err := fm.ast.patch([]sourceEdit{{start, end, strings.TrimSpace(title)}}, fm.ast.ExtractTodos(), -1); err != nil {
		return err
	}
	fm.Todos = fm.ast.ExtractTodos()
	return nil
}

// CreateHeading inserts after a selected section and its descendants, or appends.
func (fm *FileModel) CreateHeading(afterIndex, level int, title string) (int, error) {
	if err := validHeading(title, level); err != nil {
		return -1, err
	}
	if fm.ast == nil {
		return -1, fmt.Errorf("document is unavailable")
	}
	headings := fm.headingNodes()
	if afterIndex < -1 || afterIndex >= len(headings) {
		return -1, fmt.Errorf("section no longer exists")
	}
	at := len(fm.ast.Source)
	index := len(headings)
	if afterIndex >= 0 {
		h := headings[afterIndex]
		if h.Lines().Len() == 0 {
			return -1, fmt.Errorf("empty heading source location unavailable")
		}
		if h.Parent() != fm.ast.AST {
			return -1, fmt.Errorf("creating sections inside nested Markdown blocks is not supported")
		}
		for i := afterIndex + 1; i < len(headings); i++ {
			if headings[i].Level <= h.Level {
				if headings[i].Lines().Len() == 0 {
					return -1, fmt.Errorf("empty heading source location unavailable")
				}
				at = fm.ast.lineStart(headings[i].Lines().At(0).Start)
				index = i
				break
			}
		}
	}
	newline := fm.ast.newline()
	text := strings.Repeat("#", level) + " " + strings.TrimSpace(title) + newline + newline
	if at > 0 {
		text = newline + text
		if fm.ast.Source[at-1] != '\n' {
			text = newline + text
		}
	}
	if err := fm.ast.patch([]sourceEdit{{at, at, text}}, fm.ast.ExtractTodos(), -1); err != nil {
		return -1, err
	}
	fm.Todos = fm.ast.ExtractTodos()
	return index, nil
}

// AddTodoInSection appends to the section's immediate list, including empty sections.
func (fm *FileModel) AddTodoInSection(index int, title string) (int, error) {
	headings := fm.headingNodes()
	if index < 0 || index >= len(headings) {
		return -1, fmt.Errorf("section no longer exists")
	}
	if strings.ContainsAny(title, "\r\n") {
		return -1, fmt.Errorf("task title must be a single line")
	}
	h := headings[index]
	if h.Lines().Len() == 0 {
		return -1, fmt.Errorf("empty heading source location unavailable")
	}
	if h.Parent() != fm.ast.AST {
		return -1, fmt.Errorf("adding tasks to nested Markdown sections is not supported")
	}
	at := fm.ast.lineEnd(h.Lines().At(h.Lines().Len()-1).Stop - 1)
	// Setext headings have an extra underline line after the text lines.
	if at < len(fm.ast.Source) {
		next := strings.TrimSpace(string(fm.ast.Source[at:fm.ast.lineEnd(at)]))
		if next != "" && strings.Trim(next, "=-") == "" {
			at = fm.ast.lineEnd(at)
		}
	}
	prefix := "- "
	if list, ok := h.NextSibling().(*ast.List); ok && list.LastChild() != nil {
		span, err := fm.ast.itemSource(list.LastChild().(*ast.ListItem))
		if err != nil {
			return -1, err
		}
		at = span.end
		prefix = span.indent + span.marker + span.space
	}
	todos := fm.ast.ExtractTodos()
	insert := 0
	for insert < len(todos) && todos[insert].LineNo < at {
		insert++
	}
	expected := append([]Todo{}, todos[:insert]...)
	expected = append(expected, Todo{Text: title})
	expected = append(expected, todos[insert:]...)
	newline := fm.ast.newline()
	text := prefix + "[ ] " + title + newline
	if _, ok := h.NextSibling().(*ast.List); !ok {
		text = newline + text + newline
	}
	if at > 0 && fm.ast.Source[at-1] != '\n' {
		text = newline + text
	}
	if err := fm.ast.patch([]sourceEdit{{at, at, text}}, expected, insert); err != nil {
		return -1, err
	}
	fm.Todos = fm.ast.ExtractTodos()
	return insert, nil
}
