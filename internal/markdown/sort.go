package markdown

import (
	"github.com/yuin/goldmark/ast"
	"sort"
)

// SortTodoSubtrees reorders sibling task items as complete subtrees. Ordinary
// list items, headings and all other document blocks retain their positions.
func (fm *FileModel) SortTodoSubtrees(less func(Todo, Todo) bool) {
	if fm.ast == nil {
		return
	}
	values := make(map[ast.Node]Todo)
	for i, todo := range fm.Todos {
		node, err := fm.ast.FindTodoNode(i)
		if err == nil {
			values[node.ListItem] = todo
		}
	}
	var visit func(ast.Node)
	visit = func(node ast.Node) {
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			visit(child)
		}
		if _, ok := node.(*ast.List); !ok {
			return
		}
		children := []ast.Node{}
		tasks := []ast.Node{}
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			children = append(children, child)
			if _, ok := values[child]; ok {
				tasks = append(tasks, child)
			}
		}
		sort.SliceStable(tasks, func(i, j int) bool { return less(values[tasks[i]], values[tasks[j]]) })
		index := 0
		for i, child := range children {
			if _, ok := values[child]; ok {
				children[i] = tasks[index]
				index++
			}
		}
		node.RemoveChildren(node)
		for _, child := range children {
			node.AppendChild(node, child)
		}
	}
	visit(fm.ast.AST)
	fm.ast.invalidateSource()
	fm.Todos = fm.ast.ExtractTodos()
}
