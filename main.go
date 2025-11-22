package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

const defaultFile = "todo.md"

// Todo represents a single todo item
type Todo struct {
	Index   int
	Checked bool
	Text    string
	LineNo  int
}

// FileModel holds parsed file content
type FileModel struct {
	Lines []string
	Todos []Todo
}

// App state for TUI
type model struct {
	filePath      string
	fileModel     FileModel
	selectedIndex int
	inputMode     bool
	editMode      bool
	moveMode      bool
	helpMode      bool
	inputBuffer   string
	cursorPos     int
	numberBuffer  string
	history       *FileModel
	copyFeedback  bool
	err           error
}

// Styles
var (
	magentaStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	cyanStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	greenStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	yellowStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	codeStyle    = lipgloss.NewStyle().Background(lipgloss.Color("8")).Foreground(lipgloss.Color("15"))
)

// Message to clear copy feedback
type clearCopyFeedbackMsg struct{}

func main() {
	args := os.Args[1:]

	// Determine file path and command
	filePath := defaultFile
	var command string
	var cmdArgs []string

	if len(args) > 0 {
		// Check if first arg is a .md file
		if strings.HasSuffix(args[0], ".md") {
			filePath = args[0]
			args = args[1:]
		}

		if len(args) > 0 {
			command = args[0]
			cmdArgs = args[1:]
		}
	}

	// Resolve to absolute path
	if !filepath.IsAbs(filePath) {
		cwd, _ := os.Getwd()
		filePath = filepath.Join(cwd, filePath)
	}

	// Handle commands
	switch command {
	case "help", "--help", "-h":
		printHelp()
	case "--version", "-v":
		fmt.Printf("tdx v%s\n", Version)
	case "list":
		listTodos(filePath)
	case "add":
		if len(cmdArgs) < 1 {
			fmt.Println("Error: add requires text argument")
			os.Exit(1)
		}
		addTodo(filePath, strings.Join(cmdArgs, " "))
	case "toggle":
		if len(cmdArgs) < 1 {
			fmt.Println("Error: toggle requires index argument")
			os.Exit(1)
		}
		idx, err := strconv.Atoi(cmdArgs[0])
		if err != nil {
			fmt.Println("Error: invalid index")
			os.Exit(1)
		}
		toggleTodo(filePath, idx)
	case "edit":
		if len(cmdArgs) < 2 {
			fmt.Println("Error: edit requires index and text arguments")
			os.Exit(1)
		}
		idx, err := strconv.Atoi(cmdArgs[0])
		if err != nil {
			fmt.Println("Error: invalid index")
			os.Exit(1)
		}
		editTodo(filePath, idx, strings.Join(cmdArgs[1:], " "))
	case "delete":
		if len(cmdArgs) < 1 {
			fmt.Println("Error: delete requires index argument")
			os.Exit(1)
		}
		idx, err := strconv.Atoi(cmdArgs[0])
		if err != nil {
			fmt.Println("Error: invalid index")
			os.Exit(1)
		}
		deleteTodo(filePath, idx)
	case "":
		// Launch TUI
		runTUI(filePath)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	help := fmt.Sprintf(`tdx - %s

Usage:
  tdx [file.md] [command] [args]

Commands:
  (none)              Launch interactive TUI
  list                List all todos
  add "text"          Add a new todo
  toggle <index>      Toggle todo completion
  edit <index> "text" Edit todo text
  delete <index>      Delete a todo
  help                Show this help

TUI Controls:
  j/k, ↑/↓            Navigate up/down
  Space, Enter        Toggle completion
  n                   New todo
  e                   Edit todo
  d                   Delete todo
  c                   Copy to clipboard
  m                   Move todo
  u                   Undo
  ?                   Toggle help
  q, Esc              Quit`, Description)
	fmt.Println(help)
}

// File operations

func readFile(filePath string) (*FileModel, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &FileModel{
				Lines: []string{"# Todos", ""},
				Todos: []Todo{},
			}, nil
		}
		return nil, err
	}

	return parseMarkdown(string(content)), nil
}

func writeFile(filePath string, fm *FileModel) error {
	content := serializeMarkdown(fm)

	// Atomic write: temp file + rename
	dir := filepath.Dir(filePath)
	tmpFile := filepath.Join(dir, fmt.Sprintf(".tmp.%d", os.Getpid()))

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		return err
	}

	return os.Rename(tmpFile, filePath)
}

func parseMarkdown(content string) *FileModel {
	lines := strings.Split(content, "\n")
	var todos []Todo
	todoIndex := 1

	for lineNo, line := range lines {
		if strings.HasPrefix(line, "- [ ] ") {
			todos = append(todos, Todo{
				Index:   todoIndex,
				Checked: false,
				Text:    strings.TrimPrefix(line, "- [ ] "),
				LineNo:  lineNo,
			})
			todoIndex++
		} else if strings.HasPrefix(line, "- [x] ") {
			todos = append(todos, Todo{
				Index:   todoIndex,
				Checked: true,
				Text:    strings.TrimPrefix(line, "- [x] "),
				LineNo:  lineNo,
			})
			todoIndex++
		}
	}

	return &FileModel{
		Lines: lines,
		Todos: todos,
	}
}

func serializeMarkdown(fm *FileModel) string {
	lines := make([]string, len(fm.Lines))
	copy(lines, fm.Lines)

	for _, todo := range fm.Todos {
		if todo.Checked {
			lines[todo.LineNo] = fmt.Sprintf("- [x] %s", todo.Text)
		} else {
			lines[todo.LineNo] = fmt.Sprintf("- [ ] %s", todo.Text)
		}
	}

	return strings.Join(lines, "\n")
}

// CLI commands

func listTodos(filePath string) {
	fm, err := readFile(filePath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if len(fm.Todos) == 0 {
		fmt.Println("No todos found")
		return
	}

	for _, todo := range fm.Todos {
		checkbox := "[ ]"
		if todo.Checked {
			checkbox = "[✓]"
		}
		fmt.Printf("  %d. %s %s\n", todo.Index, checkbox, todo.Text)
	}
}

func addTodo(filePath string, text string) {
	// Remove surrounding quotes if present
	text = strings.Trim(text, "\"")

	fm, err := readFile(filePath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	newTodo := Todo{
		Index:   len(fm.Todos) + 1,
		Checked: false,
		Text:    text,
		LineNo:  len(fm.Lines),
	}

	fm.Todos = append(fm.Todos, newTodo)
	fm.Lines = append(fm.Lines, fmt.Sprintf("- [ ] %s", text))

	if err := writeFile(filePath, fm); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s Added: %s\n", greenStyle.Render("✓"), text)
}

func toggleTodo(filePath string, index int) {
	fm, err := readFile(filePath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if index < 1 || index > len(fm.Todos) {
		fmt.Printf("Error: invalid index %d\n", index)
		os.Exit(1)
	}

	todo := &fm.Todos[index-1]
	todo.Checked = !todo.Checked

	if err := writeFile(filePath, fm); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	checkbox := "[ ]"
	if todo.Checked {
		checkbox = "[✓]"
	}
	fmt.Printf("%s Toggled: %s %s\n", greenStyle.Render("✓"), checkbox, todo.Text)
}

func editTodo(filePath string, index int, text string) {
	text = strings.Trim(text, "\"")

	fm, err := readFile(filePath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if index < 1 || index > len(fm.Todos) {
		fmt.Printf("Error: invalid index %d\n", index)
		os.Exit(1)
	}

	todo := &fm.Todos[index-1]
	todo.Text = text

	if err := writeFile(filePath, fm); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s Edited: %s\n", greenStyle.Render("✓"), text)
}

func deleteTodo(filePath string, index int) {
	fm, err := readFile(filePath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if index < 1 || index > len(fm.Todos) {
		fmt.Printf("Error: invalid index %d\n", index)
		os.Exit(1)
	}

	todo := fm.Todos[index-1]

	// Remove the line from the file
	newLines := make([]string, 0, len(fm.Lines)-1)
	for i, line := range fm.Lines {
		if i != todo.LineNo {
			newLines = append(newLines, line)
		}
	}
	fm.Lines = newLines

	// Re-parse to update line numbers
	fm = parseMarkdown(strings.Join(fm.Lines, "\n"))

	if err := writeFile(filePath, fm); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s Deleted: %s\n", greenStyle.Render("✓"), todo.Text)
}

// TUI implementation

func runTUI(filePath string) {
	fm, err := readFile(filePath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	m := model{
		filePath:      filePath,
		fileModel:     *fm,
		selectedIndex: 0,
	}

	// Check if we have a TTY
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// Piped input - process directly without Bubbletea event loop
		input, _ := io.ReadAll(os.Stdin)
		m.processPipedInput(input)
		fmt.Print(m.View())
		return
	}

	// Normal TTY - use Bubbletea (no alt screen to keep context visible)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

// processPipedInput handles input byte-by-byte for testing/scripting
func (m *model) processPipedInput(input []byte) {
	for i := 0; i < len(input); i++ {
		b := input[i]

		// Handle input/edit mode
		if m.inputMode || m.editMode {
			switch b {
			case '\r', '\n': // Enter
				if m.inputMode {
					if m.inputBuffer != "" {
						m.addNewTodo()
						m.inputMode = false
					}
					// If buffer empty, stay in input mode (allows n\rText\r format)
				} else if m.editMode {
					if m.inputBuffer != "" {
						m.fileModel.Todos[m.selectedIndex].Text = m.inputBuffer
						writeFile(m.filePath, &m.fileModel)
					}
					m.editMode = false
				}
			case 27: // Escape
				m.inputMode = false
				m.editMode = false
			case 127, 8: // Backspace
				if m.cursorPos > 0 {
					m.inputBuffer = m.inputBuffer[:m.cursorPos-1] + m.inputBuffer[m.cursorPos:]
					m.cursorPos--
				}
			default:
				if b >= 32 && b < 127 { // Printable ASCII
					m.inputBuffer = m.inputBuffer[:m.cursorPos] + string(b) + m.inputBuffer[m.cursorPos:]
					m.cursorPos++
				}
			}
			continue
		}

		// Handle move mode
		if m.moveMode {
			switch b {
			case 'j':
				if m.selectedIndex < len(m.fileModel.Todos)-1 {
					m.swapTodos(m.selectedIndex, m.selectedIndex+1)
					m.selectedIndex++
				}
			case 'k':
				if m.selectedIndex > 0 {
					m.swapTodos(m.selectedIndex, m.selectedIndex-1)
					m.selectedIndex--
				}
			case '\r', '\n':
				writeFile(m.filePath, &m.fileModel)
				m.moveMode = false
			case 27:
				m.moveMode = false
			}
			continue
		}

		// Normal mode
		switch b {
		case 'q', 27: // Quit
			return
		case 'j':
			count := m.getCount()
			m.selectedIndex = min(m.selectedIndex+count, len(m.fileModel.Todos)-1)
			if m.selectedIndex < 0 {
				m.selectedIndex = 0
			}
		case 'k':
			count := m.getCount()
			m.selectedIndex = max(m.selectedIndex-count, 0)
		case ' ', '\r', '\n':
			if len(m.fileModel.Todos) > 0 {
				m.saveHistory()
				todo := &m.fileModel.Todos[m.selectedIndex]
				todo.Checked = !todo.Checked
				writeFile(m.filePath, &m.fileModel)
			}
		case 'n':
			m.saveHistory()
			m.inputMode = true
			m.inputBuffer = ""
			m.cursorPos = 0
		case 'e':
			if len(m.fileModel.Todos) > 0 {
				m.saveHistory()
				m.editMode = true
				m.inputBuffer = m.fileModel.Todos[m.selectedIndex].Text
				m.cursorPos = len(m.inputBuffer)
			}
		case 'd':
			if len(m.fileModel.Todos) > 0 {
				m.saveHistory()
				m.deleteCurrent()
			}
		case 'm':
			if len(m.fileModel.Todos) > 0 {
				m.saveHistory()
				m.moveMode = true
			}
		case 'u':
			if m.history != nil {
				m.fileModel = *m.history
				m.history = nil
				writeFile(m.filePath, &m.fileModel)
				if m.selectedIndex >= len(m.fileModel.Todos) {
					m.selectedIndex = max(0, len(m.fileModel.Todos)-1)
				}
			}
		case 'c':
			if len(m.fileModel.Todos) > 0 {
				copyToClipboard(m.fileModel.Todos[m.selectedIndex].Text)
			}
		case '?':
			m.helpMode = !m.helpMode
		case '1', '2', '3', '4', '5', '6', '7', '8', '9':
			m.numberBuffer += string(b)
		}
	}
}

func (m *model) getCount() int {
	count := 1
	if m.numberBuffer != "" {
		count, _ = strconv.Atoi(m.numberBuffer)
		m.numberBuffer = ""
	}
	return count
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case clearCopyFeedbackMsg:
		m.copyFeedback = false
		return m, nil
	case tea.KeyMsg:
		// Handle EOF from piped input
		if msg.Type == tea.KeyCtrlD {
			return m, tea.Quit
		}
		return m.handleKey(msg)
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Handle input/edit mode
	if m.inputMode || m.editMode {
		return m.handleInputKey(msg)
	}

	// Handle move mode
	if m.moveMode {
		return m.handleMoveKey(msg)
	}

	// Handle help mode
	if m.helpMode {
		if key == "?" || key == "q" || key == "esc" {
			m.helpMode = false
		}
		return m, nil
	}

	// Number buffer for vim-style navigation
	if key >= "1" && key <= "9" {
		m.numberBuffer += key
		return m, nil
	}

	// Get count from number buffer
	count := 1
	if m.numberBuffer != "" {
		count, _ = strconv.Atoi(m.numberBuffer)
		m.numberBuffer = ""
	}

	switch key {
	case "q", "esc", "ctrl+c":
		return m, tea.Quit

	case "j", "down":
		m.selectedIndex = min(m.selectedIndex+count, len(m.fileModel.Todos)-1)
		if m.selectedIndex < 0 {
			m.selectedIndex = 0
		}

	case "k", "up":
		m.selectedIndex = max(m.selectedIndex-count, 0)

	case " ", "enter":
		if len(m.fileModel.Todos) > 0 {
			m.saveHistory()
			todo := &m.fileModel.Todos[m.selectedIndex]
			todo.Checked = !todo.Checked
			writeFile(m.filePath, &m.fileModel)
		}

	case "n":
		m.saveHistory()
		m.inputMode = true
		m.inputBuffer = ""
		m.cursorPos = 0

	case "e":
		if len(m.fileModel.Todos) > 0 {
			m.saveHistory()
			m.editMode = true
			m.inputBuffer = m.fileModel.Todos[m.selectedIndex].Text
			m.cursorPos = len(m.inputBuffer)
		}

	case "d":
		if len(m.fileModel.Todos) > 0 {
			m.saveHistory()
			m.deleteCurrent()
		}

	case "c":
		if len(m.fileModel.Todos) > 0 {
			copyToClipboard(m.fileModel.Todos[m.selectedIndex].Text)
			m.copyFeedback = true
			return m, tea.Tick(1500*time.Millisecond, func(t time.Time) tea.Msg {
				return clearCopyFeedbackMsg{}
			})
		}

	case "m":
		if len(m.fileModel.Todos) > 0 {
			m.saveHistory()
			m.moveMode = true
		}

	case "u":
		if m.history != nil {
			m.fileModel = *m.history
			m.history = nil
			writeFile(m.filePath, &m.fileModel)
			if m.selectedIndex >= len(m.fileModel.Todos) {
				m.selectedIndex = max(0, len(m.fileModel.Todos)-1)
			}
		}

	case "?":
		m.helpMode = true
	}

	return m, nil
}

func (m model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "enter", "ctrl+m":
		if m.inputMode {
			if m.inputBuffer != "" {
				m.addNewTodo()
			}
			m.inputMode = false
		} else if m.editMode {
			if m.inputBuffer != "" {
				m.fileModel.Todos[m.selectedIndex].Text = m.inputBuffer
				writeFile(m.filePath, &m.fileModel)
			}
			m.editMode = false
		}

	case "esc":
		m.inputMode = false
		m.editMode = false
		if m.history != nil {
			m.fileModel = *m.history
			m.history = nil
		}

	case "backspace", "ctrl+h":
		if m.cursorPos > 0 {
			m.inputBuffer = m.inputBuffer[:m.cursorPos-1] + m.inputBuffer[m.cursorPos:]
			m.cursorPos--
		}

	case "delete":
		if m.cursorPos < len(m.inputBuffer) {
			m.inputBuffer = m.inputBuffer[:m.cursorPos] + m.inputBuffer[m.cursorPos+1:]
		}

	case "left":
		if m.cursorPos > 0 {
			m.cursorPos--
		}

	case "right":
		if m.cursorPos < len(m.inputBuffer) {
			m.cursorPos++
		}

	case "home", "ctrl+a":
		m.cursorPos = 0

	case "end", "ctrl+e":
		m.cursorPos = len(m.inputBuffer)

	default:
		// Insert character
		if len(key) == 1 {
			m.inputBuffer = m.inputBuffer[:m.cursorPos] + key + m.inputBuffer[m.cursorPos:]
			m.cursorPos++
		}
	}

	return m, nil
}

func (m model) handleMoveKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "j", "down":
		if m.selectedIndex < len(m.fileModel.Todos)-1 {
			m.swapTodos(m.selectedIndex, m.selectedIndex+1)
			m.selectedIndex++
		}

	case "k", "up":
		if m.selectedIndex > 0 {
			m.swapTodos(m.selectedIndex, m.selectedIndex-1)
			m.selectedIndex--
		}

	case "enter":
		writeFile(m.filePath, &m.fileModel)
		m.moveMode = false

	case "esc":
		if m.history != nil {
			m.fileModel = *m.history
			m.history = nil
		}
		m.moveMode = false
	}

	return m, nil
}

func (m *model) saveHistory() {
	// Deep copy
	lines := make([]string, len(m.fileModel.Lines))
	copy(lines, m.fileModel.Lines)

	todos := make([]Todo, len(m.fileModel.Todos))
	copy(todos, m.fileModel.Todos)

	m.history = &FileModel{
		Lines: lines,
		Todos: todos,
	}
}

func (m *model) addNewTodo() {
	newTodo := Todo{
		Index:   len(m.fileModel.Todos) + 1,
		Checked: false,
		Text:    m.inputBuffer,
		LineNo:  len(m.fileModel.Lines),
	}

	m.fileModel.Todos = append(m.fileModel.Todos, newTodo)
	m.fileModel.Lines = append(m.fileModel.Lines, fmt.Sprintf("- [ ] %s", m.inputBuffer))

	writeFile(m.filePath, &m.fileModel)
	m.selectedIndex = len(m.fileModel.Todos) - 1
}

func (m *model) deleteCurrent() {
	if len(m.fileModel.Todos) == 0 {
		return
	}

	todo := m.fileModel.Todos[m.selectedIndex]

	// Remove line
	newLines := make([]string, 0, len(m.fileModel.Lines)-1)
	for i, line := range m.fileModel.Lines {
		if i != todo.LineNo {
			newLines = append(newLines, line)
		}
	}
	m.fileModel.Lines = newLines

	// Re-parse
	m.fileModel = *parseMarkdown(strings.Join(m.fileModel.Lines, "\n"))

	writeFile(m.filePath, &m.fileModel)

	// Adjust selection
	if m.selectedIndex >= len(m.fileModel.Todos) {
		m.selectedIndex = max(0, len(m.fileModel.Todos)-1)
	}
}

func (m *model) swapTodos(i, j int) {
	// Swap in the todos slice
	m.fileModel.Todos[i], m.fileModel.Todos[j] = m.fileModel.Todos[j], m.fileModel.Todos[i]

	// Swap line numbers
	m.fileModel.Todos[i].LineNo, m.fileModel.Todos[j].LineNo = m.fileModel.Todos[j].LineNo, m.fileModel.Todos[i].LineNo

	// Update indexes
	m.fileModel.Todos[i].Index = i + 1
	m.fileModel.Todos[j].Index = j + 1

	// Update actual lines
	lineI := m.fileModel.Todos[i].LineNo
	lineJ := m.fileModel.Todos[j].LineNo
	m.fileModel.Lines[lineI], m.fileModel.Lines[lineJ] = m.fileModel.Lines[lineJ], m.fileModel.Lines[lineI]
}

func (m model) View() string {
	if m.helpMode {
		return m.renderHelp()
	}

	var b strings.Builder

	if len(m.fileModel.Todos) == 0 && !m.inputMode {
		b.WriteString(dimStyle.Render("No todos. Press 'n' to create one."))
		b.WriteString("\n")
	}

	for i, todo := range m.fileModel.Todos {
		relIndex := i - m.selectedIndex
		isSelected := i == m.selectedIndex

		// Relative index
		indexStr := fmt.Sprintf("%+3d", relIndex)
		if relIndex == 0 {
			indexStr = "  0"
		}

		// Arrow - don't show on existing items when in input mode
		arrow := "   "
		if isSelected && !m.inputMode {
			arrow = cyanStyle.Render(" ➜ ")
		}

		// Checkbox
		var checkbox string
		if todo.Checked {
			checkbox = magentaStyle.Render("[✓]")
		} else {
			checkbox = dimStyle.Render("[ ]")
		}

		// Text with inline code rendering
		text := renderInlineCode(todo.Text, todo.Checked)

		// Show edit cursor if in edit mode on this item
		if m.editMode && isSelected {
			before := m.inputBuffer[:m.cursorPos]
			after := m.inputBuffer[m.cursorPos:]
			text = before + lipgloss.NewStyle().Reverse(true).Render(" ") + after
		}

		// Move indicator
		if m.moveMode && isSelected {
			arrow = yellowStyle.Render(" ≡ ")
		}

		b.WriteString(fmt.Sprintf("%s%s%s %s\n", dimStyle.Render(indexStr), arrow, checkbox, text))
	}

	// Input mode
	if m.inputMode {
		arrow := cyanStyle.Render(" ➜ ")
		checkbox := dimStyle.Render("[ ]")
		before := m.inputBuffer[:m.cursorPos]
		after := m.inputBuffer[m.cursorPos:]
		cursor := lipgloss.NewStyle().Reverse(true).Render(" ")
		b.WriteString(fmt.Sprintf("   %s%s %s%s%s\n", arrow, checkbox, before, cursor, after))
	}

	b.WriteString("\n")

	// Status bar
	if m.inputMode || m.editMode {
		b.WriteString(dimStyle.Render("(Press ") + cyanStyle.Render("Enter") + dimStyle.Render(" to confirm, ") + cyanStyle.Render("Esc") + dimStyle.Render(" to cancel)"))
	} else if m.moveMode {
		b.WriteString(yellowStyle.Render("≡ Moving: ") + cyanStyle.Render("j/k") + yellowStyle.Render(" move  |  ") + cyanStyle.Render("enter") + yellowStyle.Render(" confirm  |  ") + cyanStyle.Render("esc") + yellowStyle.Render(" cancel"))
	} else if m.copyFeedback {
		b.WriteString(greenStyle.Render("✓ Copied to clipboard!"))
	} else {
		b.WriteString(cyanStyle.Render("?") + dimStyle.Render(" help  |  ") + cyanStyle.Render("j/k") + dimStyle.Render(" nav  |  ") + cyanStyle.Render("n") + dimStyle.Render(" new  |  ") + cyanStyle.Render("␣") + dimStyle.Render(" toggle  |  ") + cyanStyle.Render("esc") + dimStyle.Render(" quit"))
	}

	return b.String()
}

func (m model) renderHelp() string {
	var b strings.Builder

	title := cyanStyle.Render("tdx") + " " + dimStyle.Render("v"+Version)
	b.WriteString("\n  " + title + "\n\n")

	// Define columns: header and entries (key, description)
	type entry struct {
		key  string
		desc string
	}
	type column struct {
		header  string
		entries []entry
	}

	columns := []column{
		{
			header: "NAVIGATION",
			entries: []entry{
				{"j", "Down"},
				{"k", "Up"},
				{"5j", "Jump 5 down"},
				{"3k", "Jump 3 up"},
			},
		},
		{
			header: "EDITING",
			entries: []entry{
				{"space", "Toggle"},
				{"n", "New"},
				{"e", "Edit"},
				{"d", "Delete"},
				{"c", "Copy"},
				{"m", "Move"},
			},
		},
		{
			header: "OTHER",
			entries: []entry{
				{"u", "Undo"},
				{"?", "Help"},
				{"esc", "Quit"},
			},
		},
	}

	// Helper to get display width (proper unicode width)
	displayWidth := func(s string) int {
		return runewidth.StringWidth(s)
	}

	// Calculate max key width and max desc width per column
	keyWidths := make([]int, len(columns))
	descWidths := make([]int, len(columns))
	for i, col := range columns {
		for _, e := range col.entries {
			kw := displayWidth(e.key)
			dw := displayWidth(e.desc)
			if kw > keyWidths[i] {
				keyWidths[i] = kw
			}
			if dw > descWidths[i] {
				descWidths[i] = dw
			}
		}
	}

	// Calculate column widths: key + gap + desc + padding
	colWidths := make([]int, len(columns))
	for i, col := range columns {
		// Column width is max of: header width OR (keyWidth + 2 + descWidth)
		entryWidth := keyWidths[i] + 2 + descWidths[i]
		headerWidth := displayWidth(col.header)
		if entryWidth > headerWidth {
			colWidths[i] = entryWidth
		} else {
			colWidths[i] = headerWidth
		}
		// Add padding between columns
		colWidths[i] += 2
	}

	// Find max rows
	maxRows := 0
	for _, col := range columns {
		if len(col.entries) > maxRows {
			maxRows = len(col.entries)
		}
	}

	// Render header row with centered headers
	b.WriteString("  ")
	for i, col := range columns {
		header := col.header
		padding := colWidths[i] - displayWidth(header)
		leftPad := padding / 2
		rightPad := padding - leftPad
		b.WriteString(strings.Repeat(" ", leftPad) + cyanStyle.Render(header) + strings.Repeat(" ", rightPad))
	}
	b.WriteString("\n")

	// Render entry rows
	for row := 0; row < maxRows; row++ {
		b.WriteString("  ")
		for i, col := range columns {
			if row < len(col.entries) {
				e := col.entries[row]
				// Pad key to max key width in this column
				keyPad := keyWidths[i] - displayWidth(e.key)
				content := cyanStyle.Render(e.key) + strings.Repeat(" ", keyPad) + "  " + e.desc
				// Pad to column width
				visibleLen := keyWidths[i] + 2 + displayWidth(e.desc)
				padding := colWidths[i] - visibleLen
				b.WriteString(content + strings.Repeat(" ", padding))
			} else {
				b.WriteString(strings.Repeat(" ", colWidths[i]))
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  Press ") + cyanStyle.Render("?") + dimStyle.Render(" or ") + cyanStyle.Render("q") + dimStyle.Render(" to close help"))

	return b.String()
}

func copyToClipboard(text string) {
	// Use pbcopy on macOS
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	cmd.Run()
}

// renderInlineCode renders text with backtick-enclosed code and markdown links highlighted
func renderInlineCode(text string, isChecked bool) string {
	// First pass: replace markdown links [text](url) with clickable hyperlinks
	linkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	text = linkRe.ReplaceAllStringFunc(text, func(match string) string {
		// Extract link text and URL
		submatch := linkRe.FindStringSubmatch(match)
		if len(submatch) > 2 {
			linkText := submatch[1]
			url := submatch[2]
			// OSC 8 hyperlink: \e]8;;URL\e\\TEXT\e]8;;\e\\
			return fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", url, cyanStyle.Render(linkText))
		}
		return match
	})

	// Second pass: handle inline code
	codeRe := regexp.MustCompile("`([^`]+)`")
	matches := codeRe.FindAllStringSubmatchIndex(text, -1)

	if len(matches) == 0 {
		if isChecked {
			return magentaStyle.Render(text)
		}
		return text
	}

	var result strings.Builder
	lastIndex := 0

	for _, match := range matches {
		// Text before the code block
		before := text[lastIndex:match[0]]
		if isChecked {
			result.WriteString(magentaStyle.Render(before))
		} else {
			result.WriteString(before)
		}

		// The code block (without backticks)
		code := text[match[2]:match[3]]
		result.WriteString(codeStyle.Render(" " + code + " "))

		lastIndex = match[1]
	}

	// Remaining text after last code block
	if lastIndex < len(text) {
		after := text[lastIndex:]
		if isChecked {
			result.WriteString(magentaStyle.Render(after))
		} else {
			result.WriteString(after)
		}
	}

	return result.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
