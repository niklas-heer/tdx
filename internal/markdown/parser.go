package markdown

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Todo represents a single todo item
type Todo struct {
	Index   int
	Checked bool
	Text    string
	LineNo  int
	Tags    []string // Tags extracted from the text (e.g., #urgent #backend)
}

// FileModel holds parsed file content with AST backend
type FileModel struct {
	Lines    []string     // Deprecated: kept for compatibility, use AST instead
	Todos    []Todo       // Cached todos extracted from AST
	ast      *ASTDocument // The goldmark AST (source of truth)
	dirty    bool         // Whether todos have been modified
	FilePath string       // Path to the file
	ModTime  time.Time    // File modification time when loaded
	Metadata *Metadata    // Per-file configuration from YAML frontmatter
}

// GetAST returns the underlying AST document
func (fm *FileModel) GetAST() *ASTDocument {
	return fm.ast
}

// ReadFile reads and parses a markdown file using AST
func ReadFile(filePath string) (*FileModel, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default document
			fm := ParseMarkdown("# Todos\n\n")
			fm.FilePath = filePath
			fm.ModTime = time.Time{} // Zero time for new file
			fm.Metadata = &Metadata{}
			return fm, nil
		}
		return nil, err
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Parse metadata first
	metadata, contentWithoutMeta, metaErr := ParseMetadata(string(content))
	if metaErr != nil {
		// Log warning but continue with empty metadata
		// The error is logged but doesn't block file loading
		metadata = &Metadata{}
	}

	fm := ParseMarkdown(contentWithoutMeta)
	fm.FilePath = filePath
	fm.ModTime = fileInfo.ModTime()
	fm.Metadata = metadata
	return fm, nil
}

// CheckFileModified checks if the file has been modified since we loaded it
func (fm *FileModel) CheckFileModified() (bool, error) {
	if fm.FilePath == "" {
		return false, nil // No file path, can't check
	}

	fileInfo, err := os.Stat(fm.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // File doesn't exist, no conflict
		}
		return false, err
	}

	// Compare modification times (with 1 second tolerance for filesystem precision)
	diskModTime := fileInfo.ModTime()
	if diskModTime.Sub(fm.ModTime).Abs() > time.Second {
		return true, nil
	}

	return false, nil
}

// WriteFile writes a FileModel to disk using AST serialization
// Returns an error if the file was modified externally
func WriteFile(filePath string, fm *FileModel) error {
	// Check if file was modified externally
	modified, err := fm.CheckFileModified()
	if err != nil {
		return err
	}
	if modified {
		return fmt.Errorf("file changed externally")
	}

	return WriteFileUnchecked(filePath, fm)
}

// WriteFileUnchecked writes a FileModel to disk without checking for external modifications
// Use this when you've already checked for conflicts and handled them
func WriteFileUnchecked(filePath string, fm *FileModel) error {
	content := SerializeMarkdown(fm)

	// Atomic write: temp file + rename
	dir := filepath.Dir(filePath)
	tmpFile := filepath.Join(dir, fmt.Sprintf(".tmp.%d", os.Getpid()))

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		return err
	}

	if err := os.Rename(tmpFile, filePath); err != nil {
		return err
	}

	// Update modification time after successful write
	fileInfo, err := os.Stat(filePath)
	if err == nil {
		fm.ModTime = fileInfo.ModTime()
	}

	return nil
}

// ParseMarkdown parses markdown content into a FileModel with AST backend
func ParseMarkdown(content string) *FileModel {
	// Parse with goldmark AST
	astDoc, err := ParseAST(content)
	if err != nil {
		// Fallback to empty document
		astDoc, _ = ParseAST("# Todos\n\n")
	}

	// Extract todos from AST
	todos := astDoc.ExtractTodos()

	// Keep Lines for backward compatibility (will be deprecated)
	lines := strings.Split(content, "\n")

	return &FileModel{
		Lines:    lines,
		Todos:    todos,
		ast:      astDoc,
		dirty:    false,
		Metadata: &Metadata{}, // Initialize with empty metadata
	}
}

// SerializeMarkdown converts a FileModel back to markdown using AST
func SerializeMarkdown(fm *FileModel) string {
	if fm.ast == nil {
		// Fallback to old serialization if no AST
		return serializeMarkdownLegacy(fm)
	}

	// If todos were modified, rebuild AST
	if fm.dirty {
		fm.syncTodosToAST()
	}

	// Serialize AST to markdown
	content := SerializeAST(fm.ast)

	// Ensure proper formatting
	content = EnsureHeader(content)
	content = EnsureTrailingNewline(content)

	// Add metadata frontmatter if present
	if fm.Metadata != nil && !fm.Metadata.IsEmpty() {
		content = SerializeMetadata(fm.Metadata, content)
	}

	return content
}

// syncTodosToAST synchronizes the Todos slice back to the AST
// This is called when todos have been modified through the legacy API
func (fm *FileModel) syncTodosToAST() {
	// Extract current todos from AST to compare
	astTodos := fm.ast.ExtractTodos()

	// Check for modifications
	for i := range fm.Todos {
		if i >= len(astTodos) {
			// Todo was added
			fm.ast.AddTodo(fm.Todos[i].Text, fm.Todos[i].Checked)
		} else if fm.Todos[i].Text != astTodos[i].Text {
			// Text was modified
			fm.ast.UpdateTodoText(i, fm.Todos[i].Text)
			if fm.Todos[i].Checked != astTodos[i].Checked {
				fm.ast.ToggleTodo(i)
			}
		} else if fm.Todos[i].Checked != astTodos[i].Checked {
			// Checkbox was toggled
			fm.ast.ToggleTodo(i)
		}
	}

	// Check for deletions (Todos slice is shorter than AST)
	if len(fm.Todos) < len(astTodos) {
		// Delete from the end backwards to preserve indices
		for i := len(astTodos) - 1; i >= len(fm.Todos); i-- {
			fm.ast.DeleteTodo(i)
		}
	}

	fm.dirty = false
}

// AddTodoItem adds a new todo at the end of the file
func (fm *FileModel) AddTodoItem(text string, checked bool) {
	if fm.ast != nil {
		// Use AST for adding
		fm.ast.AddTodo(text, checked)
		// Re-extract todos to keep cache in sync
		fm.Todos = fm.ast.ExtractTodos()
	} else {
		// Legacy fallback
		newTodo := Todo{
			Index:   len(fm.Todos) + 1,
			Checked: checked,
			Text:    text,
			LineNo:  len(fm.Lines),
		}
		fm.Todos = append(fm.Todos, newTodo)
		fm.Lines = append(fm.Lines, fmt.Sprintf("- [%s] %s", map[bool]string{true: "x", false: " "}[checked], text))
	}
}

// InsertTodoItemAfter inserts a new todo after the specified index
// If afterIndex is -1, inserts at the beginning
// Returns the index of the newly inserted todo
func (fm *FileModel) InsertTodoItemAfter(afterIndex int, text string, checked bool) int {
	if fm.ast != nil {
		// Use AST for inserting
		fm.ast.InsertTodoAfter(afterIndex, text, checked)
		// Re-extract todos to keep cache in sync
		fm.Todos = fm.ast.ExtractTodos()
		// Return the new index (afterIndex + 1, or 0 if inserting at beginning)
		if afterIndex < 0 {
			return 0
		}
		return afterIndex + 1
	}
	// Legacy fallback - just append
	fm.AddTodoItem(text, checked)
	return len(fm.Todos) - 1
}

// UpdateTodoItem updates an existing todo
func (fm *FileModel) UpdateTodoItem(index int, text string, checked bool) error {
	if index < 0 || index >= len(fm.Todos) {
		return fmt.Errorf("invalid todo index: %d", index)
	}

	if fm.ast != nil {
		// Update via AST
		if fm.Todos[index].Text != text {
			if err := fm.ast.UpdateTodoText(index, text); err != nil {
				return err
			}
		}
		if fm.Todos[index].Checked != checked {
			if err := fm.ast.ToggleTodo(index); err != nil {
				return err
			}
		}
		// Re-extract todos to keep cache in sync
		fm.Todos = fm.ast.ExtractTodos()
	} else {
		// Legacy fallback
		fm.Todos[index].Text = text
		fm.Todos[index].Checked = checked
		fm.dirty = true
	}
	return nil
}

// DeleteTodoItem removes a todo
func (fm *FileModel) DeleteTodoItem(index int) error {
	if index < 0 || index >= len(fm.Todos) {
		return fmt.Errorf("invalid todo index: %d", index)
	}

	if fm.ast != nil {
		// Delete via AST
		if err := fm.ast.DeleteTodo(index); err != nil {
			return err
		}
		// Re-extract todos to keep cache in sync
		fm.Todos = fm.ast.ExtractTodos()
	} else {
		// Legacy fallback
		todo := fm.Todos[index]
		// Remove the line from the file
		newLines := make([]string, 0, len(fm.Lines)-1)
		for i, line := range fm.Lines {
			if i != todo.LineNo {
				newLines = append(newLines, line)
			}
		}
		fm.Lines = newLines
		// Re-parse to update line numbers
		content := strings.Join(fm.Lines, "\n")

		// Preserve metadata before re-parsing
		oldMetadata := fm.Metadata
		oldFilePath := fm.FilePath
		oldModTime := fm.ModTime

		*fm = *ParseMarkdown(content)

		// Restore preserved fields
		fm.Metadata = oldMetadata
		fm.FilePath = oldFilePath
		fm.ModTime = oldModTime
	}
	return nil
}

// MoveTodoItem moves a todo from fromIndex to toIndex (shifts other todos)
func (fm *FileModel) MoveTodoItem(fromIndex, toIndex int) error {
	if fromIndex < 0 || fromIndex >= len(fm.Todos) || toIndex < 0 || toIndex >= len(fm.Todos) {
		return fmt.Errorf("invalid todo indices: %d, %d", fromIndex, toIndex)
	}

	if fromIndex == toIndex {
		return nil // No-op
	}

	if fm.ast != nil {
		// Move via AST
		if err := fm.ast.MoveTodo(fromIndex, toIndex); err != nil {
			return err
		}
		// Re-extract todos to keep cache in sync
		fm.Todos = fm.ast.ExtractTodos()
	} else {
		// Legacy fallback: array manipulation
		todo := fm.Todos[fromIndex]
		// Remove from old position
		fm.Todos = append(fm.Todos[:fromIndex], fm.Todos[fromIndex+1:]...)
		// Insert at new position (adjusted if moving down)
		insertIdx := toIndex
		if fromIndex < toIndex {
			insertIdx-- // Account for the removal
		}
		fm.Todos = append(fm.Todos[:insertIdx+1], append([]Todo{todo}, fm.Todos[insertIdx+1:]...)...)
		// Update all indices
		for i := range fm.Todos {
			fm.Todos[i].Index = i + 1
		}
		fm.dirty = true
	}
	return nil
}

// SwapTodoItems swaps two todos (deprecated: use MoveTodoItem for better UX)
func (fm *FileModel) SwapTodoItems(index1, index2 int) error {
	if index1 < 0 || index1 >= len(fm.Todos) || index2 < 0 || index2 >= len(fm.Todos) {
		return fmt.Errorf("invalid todo indices: %d, %d", index1, index2)
	}

	if fm.ast != nil {
		// Swap via AST
		if err := fm.ast.SwapTodos(index1, index2); err != nil {
			return err
		}
		// Re-extract todos to keep cache in sync
		fm.Todos = fm.ast.ExtractTodos()
	} else {
		// Legacy fallback
		fm.Todos[index1], fm.Todos[index2] = fm.Todos[index2], fm.Todos[index1]
		// Update indexes
		fm.Todos[index1].Index = index1 + 1
		fm.Todos[index2].Index = index2 + 1
		// Swap lines
		lineI := fm.Todos[index1].LineNo
		lineJ := fm.Todos[index2].LineNo
		fm.Lines[lineI], fm.Lines[lineJ] = fm.Lines[lineJ], fm.Lines[lineI]
		// Swap line numbers
		fm.Todos[index1].LineNo, fm.Todos[index2].LineNo = fm.Todos[index2].LineNo, fm.Todos[index1].LineNo
		fm.dirty = true
	}
	return nil
}

// GetHeadings extracts headings from the AST with their positions
func (fm *FileModel) GetHeadings() []Heading {
	if fm.ast == nil {
		return []Heading{}
	}
	return fm.ast.ExtractHeadings()
}

// Clone creates a deep copy of the FileModel
func (fm *FileModel) Clone() *FileModel {
	if fm == nil {
		return nil
	}

	// Deep copy todos
	todos := make([]Todo, len(fm.Todos))
	copy(todos, fm.Todos)

	// Deep copy lines (for backward compatibility)
	lines := make([]string, len(fm.Lines))
	copy(lines, fm.Lines)

	// Deep copy AST if present
	var astCopy *ASTDocument
	if fm.ast != nil {
		// Copy source bytes
		source := make([]byte, len(fm.ast.Source))
		copy(source, fm.ast.Source)

		// Parse again to get a fresh AST (simplest way to deep copy)
		astCopy, _ = ParseAST(string(source))
	}

	return &FileModel{
		Lines: lines,
		Todos: todos,
		ast:   astCopy,
		dirty: fm.dirty,
	}
}

// RebuildFileStructure reconstructs Lines from Todos, preserving non-todo content
// This uses AST to preserve all structure perfectly
func RebuildFileStructure(fm *FileModel) {
	if fm.ast == nil {
		rebuildFileStructureLegacy(fm)
		return
	}

	// With AST, we don't need to rebuild - the AST is already correct
	// Just mark as dirty so it gets synced on next save
	fm.dirty = true
}

// Legacy fallback functions for when AST is not available

func serializeMarkdownLegacy(fm *FileModel) string {
	// Build a map of line numbers to updated todo content
	todoLineMap := make(map[int]string)
	var todosToAppend []Todo

	for _, todo := range fm.Todos {
		var line string
		if todo.Checked {
			line = fmt.Sprintf("- [x] %s", todo.Text)
		} else {
			line = fmt.Sprintf("- [ ] %s", todo.Text)
		}

		// If todo.LineNo is beyond current Lines array, we need to append it
		if todo.LineNo >= len(fm.Lines) {
			todosToAppend = append(todosToAppend, todo)
		} else {
			todoLineMap[todo.LineNo] = line
		}
	}

	// Build result, updating todo lines in place and preserving everything else
	var result []string
	for i, line := range fm.Lines {
		if updatedLine, exists := todoLineMap[i]; exists {
			// This line is a todo - use updated version
			result = append(result, updatedLine)
		} else {
			// Preserve non-todo lines as-is
			result = append(result, line)
		}
	}

	// Append todos that were beyond the current Lines array
	for _, todo := range todosToAppend {
		if todo.Checked {
			result = append(result, fmt.Sprintf("- [x] %s", todo.Text))
		} else {
			result = append(result, fmt.Sprintf("- [ ] %s", todo.Text))
		}
	}

	// If there are no lines at all, create minimal structure
	if len(result) == 0 {
		result = append(result, "# Todos", "")
		for _, todo := range fm.Todos {
			if todo.Checked {
				result = append(result, fmt.Sprintf("- [x] %s", todo.Text))
			} else {
				result = append(result, fmt.Sprintf("- [ ] %s", todo.Text))
			}
		}
	}

	// Check if header is missing and add it
	hasHeader := false
	for _, line := range result {
		if strings.HasPrefix(line, "#") {
			hasHeader = true
			break
		}
	}
	if !hasHeader {
		result = append([]string{"# Todos", ""}, result...)
	}

	// Ensure trailing newline
	output := strings.Join(result, "\n")
	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}
	return output
}

func rebuildFileStructureLegacy(fm *FileModel) {
	// Find all todo lines in the current file (before modification)
	oldTodoLines := make(map[int]bool)
	for i, line := range fm.Lines {
		if strings.HasPrefix(line, "- [ ] ") || strings.HasPrefix(line, "- [x] ") {
			oldTodoLines[i] = true
		}
	}

	if len(oldTodoLines) == 0 {
		// No original todos, just append new ones at the end
		for i := range fm.Todos {
			fm.Todos[i].Index = i + 1
			fm.Todos[i].LineNo = len(fm.Lines)
			var line string
			if fm.Todos[i].Checked {
				line = fmt.Sprintf("- [x] %s", fm.Todos[i].Text)
			} else {
				line = fmt.Sprintf("- [ ] %s", fm.Todos[i].Text)
			}
			fm.Lines = append(fm.Lines, line)
		}
		return
	}

	// Find first and last todo positions
	firstTodoLine := len(fm.Lines)
	lastTodoLine := -1
	for lineNo := range oldTodoLines {
		if lineNo < firstTodoLine {
			firstTodoLine = lineNo
		}
		if lineNo > lastTodoLine {
			lastTodoLine = lineNo
		}
	}

	// Rebuild: keep non-todo lines, insert new todos at original position
	var newLines []string

	// Keep everything before first todo
	for i := 0; i < firstTodoLine; i++ {
		newLines = append(newLines, fm.Lines[i])
	}

	// Insert new todos
	for i := range fm.Todos {
		fm.Todos[i].Index = i + 1
		fm.Todos[i].LineNo = len(newLines)
		var line string
		if fm.Todos[i].Checked {
			line = fmt.Sprintf("- [x] %s", fm.Todos[i].Text)
		} else {
			line = fmt.Sprintf("- [ ] %s", fm.Todos[i].Text)
		}
		newLines = append(newLines, line)
	}

	// Keep everything after the last original todo line
	for i := lastTodoLine + 1; i < len(fm.Lines); i++ {
		newLines = append(newLines, fm.Lines[i])
	}

	fm.Lines = newLines
}
