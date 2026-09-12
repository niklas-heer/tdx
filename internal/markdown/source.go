package markdown

import (
	"bytes"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/yuin/goldmark/ast"
)

// Source edits are prepared against the original bytes and parsed before being
// installed. Goldmark discards reference definitions from its tree, so rendering
// the tree cannot safely implement document edits.
type sourceEdit struct {
	start, end int
	text       string
}

type itemSource struct {
	start, end            int
	indent, marker, space string
}

var listPrefix = regexp.MustCompile(`^( *)([-+*]|[0-9]{1,9}[.)])([ \t]+)`)

func (doc *ASTDocument) lineStart(pos int) int {
	return bytes.LastIndexByte(doc.Source[:pos], '\n') + 1
}

func (doc *ASTDocument) lineEnd(pos int) int {
	if n := bytes.IndexByte(doc.Source[pos:], '\n'); n >= 0 {
		return pos + n + 1
	}
	return len(doc.Source)
}

func (doc *ASTDocument) newline() string {
	if bytes.Contains(doc.Source, []byte("\r\n")) {
		return "\r\n"
	}
	return "\n"
}

// blockExtent includes lazy continuation lines, code and HTML bodies. Container
// markers themselves are recovered from the corresponding physical source line.
func (doc *ASTDocument) blockExtent(node ast.Node) (start, end int) {
	start = len(doc.Source)
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && n.Type() != ast.TypeInline {
			for i := 0; i < n.Lines().Len(); i++ {
				line := n.Lines().At(i)
				if line.Start < start {
					start = line.Start
				}
				if line.Stop > end {
					end = line.Stop
				}
			}
			if h, ok := n.(*ast.HTMLBlock); ok && h.HasClosure() && h.ClosureLine.Stop > end {
				end = h.ClosureLine.Stop
			}
		}
		return ast.WalkContinue, nil
	})
	return
}

func (doc *ASTDocument) itemSource(item *ast.ListItem) (itemSource, error) {
	var span itemSource
	if !doc.sourceValid {
		return span, fmt.Errorf("cannot safely edit a document without source locations")
	}
	for n := item.Parent(); n != nil; n = n.Parent() {
		if _, ok := n.(*ast.Blockquote); ok {
			return span, fmt.Errorf("structural edits inside block quotes are not supported")
		}
	}
	start, knownEnd := doc.blockExtent(item)
	if start >= len(doc.Source) {
		return span, fmt.Errorf("list item source location unavailable")
	}
	span.start = doc.lineStart(start)
	lineEnd := doc.lineEnd(span.start)
	match := listPrefix.FindStringSubmatch(string(doc.Source[span.start:lineEnd]))
	if match == nil {
		return span, fmt.Errorf("list item has unsupported source indentation")
	}
	span.indent, span.marker, span.space = match[1], match[2], match[3]
	if strings.ContainsRune(span.space, '\t') {
		return span, fmt.Errorf("structural edits of tab-indented list markers are not supported")
	}
	span.end = lineEnd
	// Include indented blocks and reference definitions missing from the AST.
	// Leave trailing whitespace and all following document blocks in place.
	for pos := lineEnd; pos < len(doc.Source); {
		end := doc.lineEnd(pos)
		line := string(doc.Source[pos:end])
		if strings.TrimSpace(line) != "" {
			indent := len(line) - len(strings.TrimLeft(line, " "))
			if pos >= knownEnd && indent < len(span.indent)+len(span.marker)+len(span.space) {
				break
			}
			if strings.HasPrefix(line, "\t") {
				return span, fmt.Errorf("structural edits of tab-indented continuations are not supported")
			}
			span.end = end
		}
		pos = end
	}
	return span, nil
}

func sameTask(a, b Todo) bool {
	return a.Text == b.Text && a.Checked == b.Checked && a.Depth == b.Depth
}

func (doc *ASTDocument) patch(edits []sourceEdit, expected []Todo, editedIndex int) error {
	if !doc.sourceValid {
		return fmt.Errorf("cannot safely edit a document without source locations")
	}
	sort.SliceStable(edits, func(i, j int) bool { return edits[i].start < edits[j].start })
	var b strings.Builder
	pos := 0
	for _, e := range edits {
		if e.start < pos || e.end < e.start || e.end > len(doc.Source) {
			return fmt.Errorf("overlapping or invalid source edits")
		}
		b.Write(doc.Source[pos:e.start])
		b.WriteString(e.text)
		pos = e.end
	}
	b.Write(doc.Source[pos:])
	next, err := ParseAST(b.String())
	if err != nil {
		return err
	}
	actual := next.ExtractTodos()
	if expected != nil {
		if len(actual) != len(expected) {
			return fmt.Errorf("edit would change unrelated task structure; document left unchanged")
		}
		for i := range actual {
			if i == editedIndex {
				if actual[i].Depth == expected[i].Depth && actual[i].Checked == expected[i].Checked {
					continue
				}
			}
			if !sameTask(actual[i], expected[i]) {
				return fmt.Errorf("edit would change unrelated task %d; document left unchanged", i+1)
			}
		}
	}
	*doc = *next
	return nil
}

func (doc *ASTDocument) UpdateTodoText(index int, value string) error {
	node, err := doc.FindTodoNode(index)
	if err != nil {
		return err
	}
	container := node.CheckBox.Parent()
	if !doc.sourceValid || container == nil || container.Lines().Len() == 0 {
		return fmt.Errorf("task text source location unavailable")
	}
	lines := container.Lines()
	start := lines.At(0).Start + 3
	end := lines.At(lines.Len() - 1).Stop
	for end > start && (doc.Source[end-1] == '\n' || doc.Source[end-1] == '\r') {
		end--
	}
	// Preserve spacing between the checkbox and title.
	for start < end && (doc.Source[start] == ' ' || doc.Source[start] == '\t') {
		start++
	}
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("task title must be a single line")
	}
	if start == lines.At(0).Start+3 && value != "" {
		value = " " + value
	}
	return doc.patch([]sourceEdit{{start, end, value}}, doc.ExtractTodos(), index)
}

func (doc *ASTDocument) InsertTodoAfter(index int, value string, checked bool) error {
	if index < -1 {
		return fmt.Errorf("invalid todo index: %d", index)
	}
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("task title must be a single line")
	}
	todos := doc.ExtractTodos()
	if len(todos) == 0 && index < 0 {
		return doc.appendTask(value, checked)
	}
	lookup := index
	if lookup < 0 {
		lookup = 0
	}
	node, err := doc.FindTodoNode(lookup)
	if err != nil {
		return err
	}
	span, err := doc.itemSource(node.ListItem)
	if err != nil {
		return err
	}
	at := span.end
	insert := index + 1
	for insert < len(todos) && todos[insert].Depth > todos[indexOrZero(index)].Depth {
		insert++
	}
	marker := span.marker
	if index < 0 {
		at = span.start
		insert = 0
	} else if len(marker) > 1 {
		n, _ := strconv.Atoi(marker[:len(marker)-1])
		marker = strconv.Itoa(n+1) + marker[len(marker)-1:]
	}
	mark := " "
	if checked {
		mark = "x"
	}
	text := span.indent + marker + span.space + "[" + mark + "] " + value + doc.newline()
	if at > 0 && doc.Source[at-1] != '\n' {
		text = doc.newline() + text
	}
	expected := slices.Insert(slices.Clone(todos), insert, Todo{Text: value, Checked: checked, Depth: todos[lookup].Depth})
	return doc.patch([]sourceEdit{{at, at, text}}, expected, insert)
}

func indexOrZero(index int) int {
	if index < 0 {
		return 0
	}
	return index
}

func (doc *ASTDocument) appendTask(value string, checked bool) error {
	mark := " "
	if checked {
		mark = "x"
	}
	text := "- [" + mark + "] " + value + doc.newline()
	if len(doc.Source) > 0 {
		if doc.Source[len(doc.Source)-1] != '\n' {
			text = doc.newline() + text
		}
		text = doc.newline() + text
	}
	return doc.patch([]sourceEdit{{len(doc.Source), len(doc.Source), text}}, []Todo{{Text: value, Checked: checked}}, 0)
}

func (doc *ASTDocument) AddTodo(value string, checked bool) error {
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("task title must be a single line")
	}
	todos := doc.ExtractTodos()
	if len(todos) == 0 {
		return doc.appendTask(value, checked)
	}
	return doc.InsertTodoAfter(len(todos)-1, value, checked)
}

// relocate retains the raw body, changing only the indentation and list marker
// required by its new parent. Lazy continuations are made explicit when needed.
func (doc *ASTDocument) relocated(span itemSource, indent, marker, space string) string {
	raw := string(doc.Source[span.start:span.end])
	lines := strings.SplitAfter(raw, "\n")
	oldPrefix := len(span.indent) + len(span.marker) + len(span.space)
	newPrefix := len(indent) + len(marker) + len(space)
	lines[0] = indent + marker + space + lines[0][oldPrefix:]
	if oldPrefix == newPrefix {
		return strings.Join(lines, "")
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}
		n := len(lines[i]) - len(strings.TrimLeft(lines[i], " "))
		padding := n + newPrefix - oldPrefix
		if n < oldPrefix {
			padding = newPrefix
		}
		lines[i] = strings.Repeat(" ", padding) + lines[i][n:]
	}
	return strings.Join(lines, "")
}

func (doc *ASTDocument) DeleteTodo(index int) error {
	node, err := doc.FindTodoNode(index)
	if err != nil {
		return err
	}
	span, err := doc.itemSource(node.ListItem)
	if err != nil {
		return err
	}
	var promoted strings.Builder
	for child := node.ListItem.FirstChild(); child != nil; child = child.NextSibling() {
		if list, ok := child.(*ast.List); ok {
			for item := list.FirstChild(); item != nil; item = item.NextSibling() {
				s, err := doc.itemSource(item.(*ast.ListItem))
				if err != nil {
					return err
				}
				promoted.WriteString(doc.relocated(s, span.indent, span.marker, span.space))
			}
		}
	}
	todos := doc.ExtractTodos()
	expected := slices.Clone(todos)
	for i := index + 1; i < len(expected) && expected[i].Depth > todos[index].Depth; i++ {
		expected[i].Depth--
	}
	expected = slices.Delete(expected, index, index+1)
	if expected == nil {
		expected = []Todo{}
	}
	replacement := promoted.String()
	if replacement == "" {
		replacement = doc.removalSeparator(span)
	}
	return doc.patch([]sourceEdit{{span.start, span.end, replacement}}, expected, -1)
}

func (doc *ASTDocument) MoveTodoToPosition(from, target int, after bool) error {
	if from == target {
		return nil
	}
	node, err := doc.FindTodoNode(from)
	if err != nil {
		return err
	}
	dest, err := doc.FindTodoNode(target)
	if err != nil {
		return err
	}
	s, err := doc.itemSource(node.ListItem)
	if err != nil {
		return err
	}
	t, err := doc.itemSource(dest.ListItem)
	if err != nil {
		return err
	}
	if t.start >= s.start && t.start < s.end {
		return fmt.Errorf("cannot move a task into its own descendants")
	}
	at := t.start
	if after {
		at = t.end
	}
	todos := doc.ExtractTodos()
	end := from + 1
	for end < len(todos) && todos[end].Depth > todos[from].Depth {
		end++
	}
	insert := target
	if after {
		insert++
		for insert < len(todos) && todos[insert].Depth > todos[target].Depth {
			insert++
		}
	}
	group := slices.Clone(todos[from:end])
	delta := todos[target].Depth - todos[from].Depth
	for i := range group {
		group[i].Depth += delta
	}
	expected := slices.Delete(slices.Clone(todos), from, end)
	if insert > from {
		insert -= end - from
	}
	expected = slices.Insert(expected, insert, group...)
	return doc.moveSpan(s, at, t.indent, t.marker, t.space, expected)
}

func (doc *ASTDocument) moveSpan(span itemSource, at int, indent, marker, space string, expected []Todo) error {
	if at > span.start && at < span.end {
		return fmt.Errorf("overlapping move")
	}
	text := doc.relocated(span, indent, marker, space)
	if at < len(doc.Source) && !strings.HasSuffix(text, "\n") {
		text += doc.newline()
	}
	if at > 0 && at != span.end && doc.Source[at-1] != '\n' {
		text = doc.newline() + text
	}
	return doc.patch([]sourceEdit{{at, at, text}, {span.start, span.end, doc.removalSeparator(span)}}, expected, -1)
}

// An ordered list starting above one cannot interrupt a paragraph. Removing its
// first item must leave a separating blank line so remaining items stay a list.
func (doc *ASTDocument) removalSeparator(span itemSource) string {
	if len(span.marker) > 1 {
		return doc.newline()
	}
	return ""
}

func (doc *ASTDocument) MoveTodo(from, to int) error {
	return doc.MoveTodoToPosition(from, to, from < to)
}

func (doc *ASTDocument) IndentTodo(index int) error {
	node, err := doc.FindTodoNode(index)
	if err != nil {
		return err
	}
	prev, ok := node.ListItem.PreviousSibling().(*ast.ListItem)
	if !ok {
		return fmt.Errorf("cannot indent: no previous sibling")
	}
	s, err := doc.itemSource(node.ListItem)
	if err != nil {
		return err
	}
	p, err := doc.itemSource(prev)
	if err != nil {
		return err
	}
	indent := p.indent + strings.Repeat(" ", len(p.marker)+len(p.space))
	marker, space := "-", " "
	at := p.end
	for child := prev.FirstChild(); child != nil; child = child.NextSibling() {
		if list, ok := child.(*ast.List); ok && list.LastChild() != nil {
			nested, e := doc.itemSource(list.LastChild().(*ast.ListItem))
			if e != nil {
				return e
			}
			indent, marker, space = nested.indent, nested.marker, nested.space
			at = nested.end
			break
		}
	}
	expected := slices.Clone(doc.ExtractTodos())
	depth := expected[index].Depth
	for i := index; i < len(expected) && (i == index || expected[i].Depth > depth); i++ {
		expected[i].Depth++
	}
	return doc.moveSpan(s, at, indent, marker, space, expected)
}

func (doc *ASTDocument) OutdentTodo(index int) error {
	node, err := doc.FindTodoNode(index)
	if err != nil {
		return err
	}
	parent, ok := node.ListItem.Parent().Parent().(*ast.ListItem)
	if !ok {
		return fmt.Errorf("cannot outdent: already at top level")
	}
	s, err := doc.itemSource(node.ListItem)
	if err != nil {
		return err
	}
	p, err := doc.itemSource(parent)
	if err != nil {
		return err
	}
	todos := doc.ExtractTodos()
	end := index + 1
	for end < len(todos) && todos[end].Depth > todos[index].Depth {
		end++
	}
	group := slices.Clone(todos[index:end])
	for i := range group {
		group[i].Depth--
	}
	insert := end
	for insert < len(todos) && todos[insert].Depth >= todos[index].Depth {
		insert++
	}
	expected := slices.Delete(slices.Clone(todos), index, end)
	insert -= end - index
	expected = slices.Insert(expected, insert, group...)
	return doc.moveSpan(s, p.end, p.indent, p.marker, p.space, expected)
}

func (doc *ASTDocument) SwapTodos(a, b int) error {
	if a == b {
		return nil
	}
	n, err := doc.FindTodoNode(a)
	if err != nil {
		return err
	}
	m, err := doc.FindTodoNode(b)
	if err != nil {
		return err
	}
	if n.ListItem.Parent() != m.ListItem.Parent() {
		return fmt.Errorf("swap requires sibling tasks")
	}
	s, err := doc.itemSource(n.ListItem)
	if err != nil {
		return err
	}
	t, err := doc.itemSource(m.ListItem)
	if err != nil {
		return err
	}
	// Validate the permutation by moving complete subtrees in a private copy.
	copyDoc, _ := ParseAST(string(doc.Source))
	if a > b {
		a, b = b, a
	}
	todos := doc.ExtractTodos()
	ae, be := a+1, b+1
	for ae < len(todos) && todos[ae].Depth > todos[a].Depth {
		ae++
	}
	for be < len(todos) && todos[be].Depth > todos[b].Depth {
		be++
	}
	expected := slices.Clone(todos[:a])
	expected = append(expected, todos[b:be]...)
	expected = append(expected, todos[ae:b]...)
	expected = append(expected, todos[a:ae]...)
	expected = append(expected, todos[be:]...)
	if err = copyDoc.patch([]sourceEdit{{s.start, s.end, string(doc.Source[t.start:t.end])}, {t.start, t.end, string(doc.Source[s.start:s.end])}}, expected, -1); err != nil {
		return err
	}
	*doc = *copyDoc
	return nil
}
