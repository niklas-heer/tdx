package markdown

import (
	"slices"
	"sort"

	"github.com/yuin/goldmark/ast"
)

// SortTodos sorts sibling tasks while carrying each task's complete body and
// descendants. Lists, sections and non-task siblings remain separate groups.
// All groups are prepared in a private document, so a rejected group rolls back
// the whole action instead of leaving a partially sorted document.
func (fm *FileModel) SortTodos(less func(Todo, Todo) bool) error {
	if fm.ast == nil {
		sort.SliceStable(fm.Todos, func(i, j int) bool { return less(fm.Todos[i], fm.Todos[j]) })
		fm.dirty = true
		return nil
	}
	doc, _ := ParseAST(SerializeAST(fm.ast))
	type location struct{ start, depth int }
	var lists []location
	_ = ast.Walk(doc.AST, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if list, ok := node.(*ast.List); entering && ok && list.FirstChild() != nil {
			start, _ := doc.blockExtent(list)
			depth := 0
			for n := list.Parent(); n != nil; n = n.Parent() {
				depth++
			}
			lists = append(lists, location{doc.lineStart(start), depth})
		}
		return ast.WalkContinue, nil
	})
	sort.Slice(lists, func(i, j int) bool {
		if lists[i].depth != lists[j].depth {
			return lists[i].depth > lists[j].depth
		}
		return lists[i].start > lists[j].start
	})
	for _, loc := range lists {
		todos := doc.ExtractTodos()
		indexes := map[*ast.ListItem]int{}
		for i, checkbox := range doc.checkboxes {
			indexes[checkbox.Parent().Parent().(*ast.ListItem)] = i
		}
		var list *ast.List
		_ = ast.Walk(doc.AST, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
			if n, ok := node.(*ast.List); entering && ok && n.FirstChild() != nil {
				start, _ := doc.blockExtent(n)
				if doc.lineStart(start) == loc.start {
					list = n
					return ast.WalkStop, nil
				}
			}
			return ast.WalkContinue, nil
		})
		if list == nil {
			continue
		}
		expected := slices.Clone(todos)
		var edits []sourceEdit
		var run []*ast.ListItem
		flush := func() error {
			if len(run) < 2 {
				run = nil
				return nil
			}
			ordered := slices.Clone(run)
			sort.SliceStable(ordered, func(i, j int) bool { return less(todos[indexes[ordered[i]]], todos[indexes[ordered[j]]]) })
			if slices.Equal(run, ordered) {
				run = nil
				return nil
			}
			out := indexes[run[0]]
			for i, item := range ordered {
				from, err := doc.itemSource(item)
				if err != nil {
					return err
				}
				to, err := doc.itemSource(run[i])
				if err != nil {
					return err
				}
				text := doc.relocated(from, to.indent, to.marker, to.space)
				if to.end < len(doc.Source) && len(text) > 0 && text[len(text)-1] != '\n' {
					text += doc.newline()
				}
				edits = append(edits, sourceEdit{to.start, to.end, text})
				start := indexes[item]
				end := start + 1
				for end < len(todos) && todos[end].Depth > todos[start].Depth {
					end++
				}
				copy(expected[out:], todos[start:end])
				out += end - start
			}
			run = nil
			return nil
		}
		for child := list.FirstChild(); child != nil; child = child.NextSibling() {
			item, ok := child.(*ast.ListItem)
			if _, task := indexes[item]; ok && task {
				run = append(run, item)
			} else if err := flush(); err != nil {
				return err
			}
		}
		if err := flush(); err != nil {
			return err
		}
		if len(edits) > 0 {
			if err := doc.patch(edits, expected, -1); err != nil {
				return err
			}
		}
	}
	fm.ast = doc
	fm.Todos = doc.ExtractTodos()
	return nil
}
