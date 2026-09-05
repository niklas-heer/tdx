package markdown

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
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

func (fm *FileModel) setHeadingText(h *ast.Heading, title string) {
	h.RemoveChildren(h)
	start := len(fm.ast.Source)
	fm.ast.Source = append(fm.ast.Source, title...)
	h.AppendChild(h, ast.NewTextSegment(text.NewSegment(start, len(fm.ast.Source))))
}

// RenameHeading changes exactly the selected heading, even when titles repeat.
func (fm *FileModel) RenameHeading(index int, title string) error {
	headings := fm.headingNodes()
	if index < 0 || index >= len(headings) {
		return fmt.Errorf("section no longer exists")
	}
	if err := validHeading(title, headings[index].Level); err != nil {
		return err
	}
	fm.setHeadingText(headings[index], strings.TrimSpace(title))
	return nil
}

// CreateHeading inserts a heading after a selected section and its descendants.
// An index of -1 appends a new top-level section to the document.
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
	h := ast.NewHeading(level)
	fm.setHeadingText(h, strings.TrimSpace(title))
	if afterIndex == -1 {
		fm.ast.AST.AppendChild(fm.ast.AST, h)
	} else {
		selected := headings[afterIndex]
		parent := selected.Parent()
		var before ast.Node
		for n := selected.NextSibling(); n != nil; n = n.NextSibling() {
			if next, ok := n.(*ast.Heading); ok && next.Level <= selected.Level {
				before = n
				break
			}
		}
		if before != nil {
			parent.InsertBefore(parent, before, h)
		} else {
			parent.AppendChild(parent, h)
		}
	}
	for i, node := range fm.headingNodes() {
		if node == h {
			return i, nil
		}
	}
	return -1, fmt.Errorf("could not create section")
}

// AddTodoInSection inserts into the section's own task list, including empty sections.
func (fm *FileModel) AddTodoInSection(index int, title string) (int, error) {
	headings := fm.headingNodes()
	if index < 0 || index >= len(headings) {
		return -1, fmt.Errorf("section no longer exists")
	}
	h := headings[index]
	list, ok := h.NextSibling().(*ast.List)
	if !ok || list.IsOrdered() {
		list = ast.NewList('-')
		h.Parent().InsertAfter(h.Parent(), h, list)
	}
	start := len(fm.ast.Source)
	fm.ast.Source = append(fm.ast.Source, title...)
	item := ast.NewListItem(0)
	para := ast.NewParagraph()
	para.AppendChild(para, extast.NewTaskCheckBox(false))
	para.AppendChild(para, ast.NewTextSegment(text.NewSegment(start, len(fm.ast.Source))))
	item.AppendChild(item, para)
	list.AppendChild(list, item)
	fm.Todos = fm.ast.ExtractTodos()
	// The inserted task is the last task in this immediate list, before any later section.
	count := 0
	found := -1
	_ = ast.Walk(fm.ast.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && n.Kind() == extast.KindTaskCheckBox {
			if n.Parent() == para {
				found = count
			}
			count++
		}
		return ast.WalkContinue, nil
	})
	return found, nil
}
