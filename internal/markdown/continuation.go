package markdown

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"

	"github.com/yuin/goldmark/ast"
)

// sourceCopy materializes pending AST edits once, then edits exact source bytes.
// Metadata and the loaded disk revision stay owned by the FileModel.
func (fm *FileModel) sourceCopy() (*ASTDocument, error) {
	if fm.ast == nil || fm.dirty {
		return nil, fmt.Errorf("source editing requires a current Markdown tree")
	}
	doc, err := ParseAST(SerializeAST(fm.ast))
	if err == nil {
		doc.ExtractTodos()
	}
	return doc, err
}

func validTaskTitle(title string) bool {
	return strings.TrimSpace(title) != "" && !strings.ContainsFunc(title, unicode.IsControl)
}

func sourceLineEnd(source []byte, start int) int {
	if offset := bytes.IndexByte(source[start:], '\n'); offset >= 0 {
		return start + offset
	}
	return len(source)
}

func sourceNewline(source []byte) string {
	if bytes.Contains(source, []byte("\r\n")) {
		return "\r\n"
	}
	return "\n"
}

func appendSource(source []byte, block string) string {
	nl := sourceNewline(source)
	separator := ""
	if len(source) > 0 {
		separator = nl + nl
		if bytes.HasSuffix(source, []byte(nl)) {
			separator = nl
		}
		if bytes.HasSuffix(source, []byte(nl+nl)) {
			separator = ""
		}
	}
	return string(source) + separator + block
}

func (fm *FileModel) acceptSource(doc *ASTDocument) {
	fm.ast = doc
	fm.Todos = doc.ExtractTodos()
	fm.Lines = strings.Split(string(doc.Source), "\n")
	fm.dirty = false
}

// AppendTodoSource appends without serializing or normalizing existing blocks.
// Parsing the result prevents an unclosed fence or HTML block swallowing a task.
func (fm *FileModel) AppendTodoSource(title string) (int, error) {
	if !validTaskTitle(title) {
		return -1, fmt.Errorf("enter a nonempty single-line task title")
	}
	doc, err := fm.sourceCopy()
	if err != nil {
		return -1, err
	}
	before := doc.ExtractTodos()
	next, _ := ParseAST(appendSource(doc.Source, "- [ ] "+title+sourceNewline(doc.Source)))
	after := next.ExtractTodos()
	if len(after) != len(before)+1 || after[len(before)].Depth != 0 || after[len(before)].Checked {
		return -1, fmt.Errorf("cannot safely append here: close trailing Markdown containers first")
	}
	for i := range before {
		if !sameTask(before[i], after[i]) {
			return -1, fmt.Errorf("appending would change existing Markdown tasks")
		}
	}
	fm.acceptSource(next)
	return len(before), nil
}

func sameTask(a, b Todo) bool {
	return a.Text == b.Text && a.Checked == b.Checked && a.Depth == b.Depth && a.ParentIndex == b.ParentIndex
}

// ContinuationTitle returns only the source title line, keeping multiline notes
// out of the single-line editor. It never rewrites the document.
func (fm *FileModel) ContinuationTitle(index int) (string, error) {
	doc, err := fm.sourceCopy()
	if err != nil {
		return "", err
	}
	node, err := doc.FindTodoNode(index)
	if err != nil {
		return "", err
	}
	start := node.CheckBox.Parent().Lines().At(0).Start
	end := sourceLineEnd(doc.Source, start)
	return strings.TrimSpace(string(doc.Source[start+3 : end])), nil
}

// ContinueTodoSource retires a task subtree and appends its continuation as one
// in-memory action. The caller saves once and owns undo. Original notes remain
// byte-exact; the copy is dedented only when promoting a nested task to file end.
func (fm *FileModel) ContinueTodoSource(index int, title string) (int, error) {
	if !validTaskTitle(title) {
		return -1, fmt.Errorf("enter a nonempty single-line task title")
	}
	doc, err := fm.sourceCopy()
	if err != nil {
		return -1, err
	}
	before := doc.ExtractTodos()
	if index < 0 || index >= len(before) || before[index].Checked {
		return -1, fmt.Errorf("choose an open task to continue")
	}
	node, err := doc.FindTodoNode(index)
	if err != nil {
		return -1, err
	}
	start := node.CheckBox.Parent().Lines().At(0).Start
	lineStart := bytes.LastIndexByte(doc.Source[:start], '\n') + 1
	prefix := string(doc.Source[lineStart:start])
	indent := len(prefix) - len(strings.TrimLeft(prefix, " "))
	marker := strings.TrimSpace(prefix)
	// Quoted tasks need their enclosing quote carried too. Refuse ambiguous
	// containers instead of silently dropping or moving unrelated content.
	if strings.ContainsAny(prefix, ">\t") || (marker != "-" && marker != "+" && marker != "*" && !strings.HasSuffix(marker, ".") && !strings.HasSuffix(marker, ")")) {
		return -1, fmt.Errorf("cannot safely re-enter this task container; edit the Markdown in your editor")
	}
	end := start
	_ = ast.Walk(node.ListItem, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && n.Type() == ast.TypeBlock {
			for i := 0; i < n.Lines().Len(); i++ {
				end = max(end, n.Lines().At(i).Stop)
			}
		}
		return ast.WalkContinue, nil
	})
	// Include the remainder of the final content line and indented closing fences
	// omitted from Goldmark's line segments, but no following sibling or heading.
	if end < len(doc.Source) && (end == 0 || doc.Source[end-1] != '\n') {
		end = sourceLineEnd(doc.Source, end)
		if end < len(doc.Source) {
			end++
		}
	}
	for end < len(doc.Source) {
		stop := sourceLineEnd(doc.Source, end)
		line := string(doc.Source[end:stop])
		spaces := len(line) - len(strings.TrimLeft(line, " "))
		if strings.TrimSpace(line) != "" && spaces <= indent {
			break
		}
		end = stop
		if end < len(doc.Source) {
			end++
		}
	}
	firstEnd := sourceLineEnd(doc.Source, start)
	nl := sourceNewline(doc.Source)
	tailStart := firstEnd
	if tailStart < len(doc.Source) {
		tailStart++
	}
	var tail strings.Builder
	if tailStart < end {
		for _, line := range strings.SplitAfter(string(doc.Source[tailStart:end]), "\n") {
			tail.WriteString(strings.TrimPrefix(line, strings.Repeat(" ", indent)))
		}
	}
	block := prefix[indent:] + "[ ] " + title + nl + tail.String()
	blockDoc, _ := ParseAST(block)
	copied := blockDoc.ExtractTodos()
	subtreeEnd := index + 1
	for subtreeEnd < len(before) && before[subtreeEnd].Depth > before[index].Depth {
		subtreeEnd++
	}
	if len(copied) != subtreeEnd-index || copied[0].Depth != 0 || copied[0].Checked {
		return -1, fmt.Errorf("cannot preserve this task's nested Markdown safely")
	}
	for j := 1; j < len(copied); j++ {
		original := before[index+j]
		if copied[j].Checked != original.Checked || copied[j].Depth != original.Depth-before[index].Depth {
			return -1, fmt.Errorf("cannot preserve this task's nested Markdown safely")
		}
	}
	for i := index; i < subtreeEnd; i++ {
		if !before[i].Checked {
			if err := doc.ToggleTodo(i); err != nil {
				return -1, err
			}
		}
	}
	next, _ := ParseAST(appendSource(doc.Source, block))
	after := next.ExtractTodos()
	if len(after) != len(before)+len(copied) {
		return -1, fmt.Errorf("cannot safely append continuation to this Markdown document")
	}
	for i, original := range before {
		if i >= index && i < subtreeEnd {
			original.Checked = true
		}
		if !sameTask(original, after[i]) {
			return -1, fmt.Errorf("continuation would change unrelated tasks")
		}
	}
	for j, copy := range copied {
		got := after[len(before)+j]
		parent := copy.ParentIndex
		if parent >= 0 {
			parent += len(before)
		}
		if got.Checked != copy.Checked || got.Depth != copy.Depth || got.ParentIndex != parent || (j > 0 && got.Text != before[index+j].Text) {
			return -1, fmt.Errorf("continuation would change task structure")
		}
	}
	fm.acceptSource(next)
	return len(before), nil
}
