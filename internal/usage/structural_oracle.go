package usage

import (
	"fmt"
	"slices"
	"strings"

	"github.com/niklas-heer/tdx/internal/editor"
	"github.com/niklas-heer/tdx/internal/markdown"
	"github.com/yuin/goldmark/ast"
)

// Structural campaigns use unique titles as identities. This oracle checks
// ownership independently of the production source ranges and move algorithm.
// Ambiguous hand-written traces retain the multiset and source checks.
func checkStructuralOwnership(beforeSource, afterSource string, a editor.Action, protected []string) error {
	before, _ := markdown.ParseAST(beforeSource)
	after, _ := markdown.ParseAST(afterSource)
	old, current := before.ExtractTodos(), after.ExtractTodos()
	switch a.Kind {
	case editor.Edit, editor.Toggle, editor.SetChecked, editor.Delete, editor.Move, editor.MoveToPosition, editor.Indent, editor.Outdent, editor.Insert:
		if a.Index < 0 || a.Index >= len(old) {
			return fmt.Errorf("successful action has invalid task index")
		}
	}
	if (a.Kind == editor.Move || a.Kind == editor.MoveToPosition) && (a.Target < 0 || a.Target >= len(old)) {
		return fmt.Errorf("successful move has invalid destination")
	}
	if a.Kind == editor.Outdent && old[a.Index].ParentIndex < 0 {
		return fmt.Errorf("successful outdent has no parent")
	}
	oldHeadings, newHeadings := before.ExtractHeadings(), after.ExtractHeadings()
	if a.Kind == editor.CreateHeading {
		insert := len(oldHeadings)
		if a.Index >= 0 && a.Index < len(oldHeadings) {
			for i := a.Index + 1; i < len(oldHeadings); i++ {
				if oldHeadings[i].Level <= oldHeadings[a.Index].Level {
					insert = i
					break
				}
			}
		}
		if insert >= len(newHeadings) || newHeadings[insert].Text != strings.TrimSpace(a.Text) || newHeadings[insert].Level != a.Level {
			return fmt.Errorf("new heading missing from requested position")
		}
		newHeadings = slices.Delete(newHeadings, insert, insert+1)
	}
	if len(oldHeadings) != len(newHeadings) {
		return fmt.Errorf("action changed unrelated heading structure")
	}
	for i, h := range oldHeadings {
		want := h.Text
		if a.Kind == editor.RenameHeading && a.Index == i {
			want = strings.TrimSpace(a.Text)
		}
		if newHeadings[i].Text != want || newHeadings[i].Level != h.Level {
			return fmt.Errorf("action changed unrelated heading %d", i)
		}
	}
	for _, block := range protected {
		left, right := strings.Index(beforeSource, block), strings.Index(afterSource, block)
		if left < 0 || right < 0 {
			continue // The byte-preservation check reports missing blocks.
		}
		for i := range oldHeadings {
			if (left < oldHeadings[i].LineNo) != (right < newHeadings[i].LineNo) {
				return fmt.Errorf("protected block crossed heading %d", i)
			}
		}
		for _, other := range protected {
			if (left < strings.Index(beforeSource, other)) != (right < strings.Index(afterSource, other)) {
				return fmt.Errorf("protected blocks changed document order")
			}
		}
	}
	identities := map[string]int{}
	for i, task := range old {
		if _, duplicate := identities[task.Text]; duplicate {
			return nil
		}
		identities[task.Text] = i
	}
	if a.Kind == editor.Edit {
		if len(current) != len(old) {
			return fmt.Errorf("edit changed task count")
		}
		node, _ := after.FindTodoNode(a.Index)
		lines := node.CheckBox.Parent().Lines()
		line := lines.At(0)
		if current[a.Index].Checked != old[a.Index].Checked || (a.Text != old[a.Index].Text && (lines.Len() != 1 || strings.TrimSpace(string(line.Value(after.Source))[3:]) != strings.TrimSpace(a.Text))) {
			return fmt.Errorf("edit did not replace the selected paragraph")
		}
		if other, collision := identities[current[a.Index].Text]; collision && other != a.Index {
			return nil // This edit made title-based identity ambiguous.
		}
		delete(identities, old[a.Index].Text)
		identities[current[a.Index].Text] = a.Index
	}
	ids := make([]int, len(current))
	positions := map[int]int{}
	for i, task := range current {
		id, known := identities[task.Text]
		if !known {
			id = -1 // The conservation oracle validates the one inserted task.
		}
		if _, duplicate := positions[id]; duplicate && id >= 0 {
			return fmt.Errorf("task identity duplicated")
		}
		ids[i] = id
		positions[id] = i
	}
	parents, depths := make([]int, len(old)), make([]int, len(old))
	for i, task := range old {
		parents[i], depths[i] = task.ParentIndex, task.Depth
	}
	removed := map[int]bool{}
	switch a.Kind {
	case editor.Delete:
		removed[a.Index] = true
	case editor.ClearDone:
		for i, task := range old {
			removed[i] = task.Checked
		}
	}
	for i := range old {
		for p := old[i].ParentIndex; p >= 0; p = old[p].ParentIndex {
			if removed[p] {
				depths[i]--
			}
		}
		for parents[i] >= 0 && removed[parents[i]] {
			parents[i] = old[parents[i]].ParentIndex
		}
	}
	moved := -1
	switch a.Kind {
	case editor.Move, editor.MoveToPosition:
		moved = a.Index
		parents[moved] = old[a.Target].ParentIndex
		if a.Index == a.Target {
			moved = -1
		}
	case editor.Indent:
		moved = a.Index
		selected, _ := before.FindTodoNode(moved)
		parents[moved] = -1
		for i := moved - 1; i >= 0; i-- {
			node, _ := before.FindTodoNode(i)
			if node.ListItem == selected.ListItem.PreviousSibling() {
				parents[moved] = i
				break
			}
		}
	case editor.Outdent:
		moved = a.Index
		parents[moved] = old[old[moved].ParentIndex].ParentIndex
	}
	if moved >= 0 {
		depth := old[moved].Depth
		switch a.Kind {
		case editor.Indent:
			depth++
		case editor.Outdent:
			depth--
		default:
			depth = old[a.Target].Depth
		}
		delta := depth - old[moved].Depth
		for i := moved; i < len(old) && (i == moved || old[i].Depth > old[moved].Depth); i++ {
			depths[i] += delta
		}
	}
	section := func(headings []markdown.Heading, offset int) int {
		index := -1
		for i, h := range headings {
			if h.LineNo < offset {
				index = i
			}
		}
		return index
	}
	if a.Kind == editor.Add || a.Kind == editor.Insert || a.Kind == editor.AddInSection {
		pos, exists := positions[-1]
		if !exists {
			return fmt.Errorf("inserted task identity missing")
		}
		parent, depth, heading := -1, 0, a.Index
		anchor := a.Index
		if a.Kind == editor.Add {
			anchor = len(old) - 1
		}
		if a.Kind != editor.AddInSection {
			heading = len(newHeadings) - 1
			if anchor >= 0 {
				parent, depth = old[anchor].ParentIndex, old[anchor].Depth
				heading = section(oldHeadings, old[anchor].LineNo)
			}
		}
		actualParent := -1
		if current[pos].ParentIndex >= 0 {
			actualParent = ids[current[pos].ParentIndex]
		}
		if current[pos].Depth != depth || actualParent != parent || section(newHeadings, current[pos].LineNo) != heading {
			return fmt.Errorf("inserted task has incorrect depth, parent or section")
		}
		if a.Kind == editor.Add && pos != len(current)-1 {
			return fmt.Errorf("add did not append after the last task")
		}
		if a.Kind == editor.Insert {
			end := a.Index + 1
			for end < len(old) && old[end].Depth > old[a.Index].Depth {
				end++
			}
			if pos != end {
				return fmt.Errorf("insert did not follow the selected subtree")
			}
		}
	}
	if a.Kind != editor.SortDone && a.Kind != editor.SortDue && a.Kind != editor.SortPriority {
		var wantOrder, actualOrder []int
		for i := range old {
			if !removed[i] {
				wantOrder = append(wantOrder, i)
			}
		}
		for _, id := range ids {
			if id >= 0 {
				actualOrder = append(actualOrder, id)
			}
		}
		if moved >= 0 && a.Kind != editor.Indent {
			end := moved + 1
			for end < len(old) && old[end].Depth > old[moved].Depth {
				end++
			}
			at := a.Target
			if a.Kind == editor.Outdent {
				at = old[moved].ParentIndex + 1
				for at < len(old) && old[at].Depth > old[old[moved].ParentIndex].Depth {
					at++
				}
			} else if (a.Kind == editor.Move && moved < a.Target) || (a.Kind == editor.MoveToPosition && a.InsertAfter) {
				at++
				for at < len(old) && old[at].Depth > old[a.Target].Depth {
					at++
				}
			}
			group := slices.Clone(wantOrder[moved:end])
			wantOrder = slices.Delete(wantOrder, moved, end)
			insert := 0
			for insert < len(wantOrder) && wantOrder[insert] < at {
				insert++
			}
			wantOrder = slices.Insert(wantOrder, insert, group...)
		}
		if !slices.Equal(wantOrder, actualOrder) {
			return fmt.Errorf("action produced incorrect subtree order")
		}
	}
	for id, task := range old {
		if removed[id] {
			continue
		}
		pos, exists := positions[id]
		if !exists {
			return fmt.Errorf("task %d lost its identity", id)
		}
		actual := current[pos]
		parent := -1
		if actual.ParentIndex >= 0 {
			parent = ids[actual.ParentIndex]
		}
		if actual.Depth != depths[id] || parent != parents[id] {
			return fmt.Errorf("task %d changed depth or parent ownership", id)
		}
		wantSection := section(oldHeadings, task.LineNo)
		if moved >= 0 && (a.Kind == editor.Move || a.Kind == editor.MoveToPosition) {
			for ancestor := id; ancestor >= 0; ancestor = old[ancestor].ParentIndex {
				if ancestor == moved {
					wantSection = section(oldHeadings, old[a.Target].LineNo)
					break
				}
			}
		}
		if section(newHeadings, actual.LineNo) != wantSection {
			return fmt.Errorf("task %d changed section ownership", id)
		}
		if taskBody(before, id) != taskBody(after, pos) {
			return fmt.Errorf("task %d changed body ownership or content", id)
		}
	}
	return nil
}

// First paragraphs are editable task titles; direct nested lists have their own
// owners. Other body blocks belong to this task. Ignore only relocation indent.
func taskBody(doc *markdown.ASTDocument, index int) string {
	node, err := doc.FindTodoNode(index)
	if err != nil {
		return ""
	}
	var body strings.Builder
	for child := node.ListItem.FirstChild(); child != nil; child = child.NextSibling() {
		if child == node.CheckBox.Parent() || child.Kind() == ast.KindList {
			continue
		}
		_ = ast.Walk(child, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if entering && n.Type() != ast.TypeInline {
				fmt.Fprintln(&body, n.Kind())
				if code, ok := n.(*ast.FencedCodeBlock); ok && code.Info != nil {
					body.Write(code.Info.Segment.Value(doc.Source))
				}
				for i := 0; i < n.Lines().Len(); i++ {
					line := n.Lines().At(i)
					body.WriteString(strings.TrimLeft(string(line.Value(doc.Source)), " \t"))
				}
				if html, ok := n.(*ast.HTMLBlock); ok && html.HasClosure() {
					body.WriteString(strings.TrimLeft(string(html.ClosureLine.Value(doc.Source)), " \t"))
				}
			}
			return ast.WalkContinue, nil
		})
	}
	return body.String()
}
