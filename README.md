# tdx

**Your todos, in markdown, done fast.**

<p align="center">
  <img src="assets/demo.gif" alt="tdx demo" width="600">
</p>

A fast, single-binary CLI todo manager focused on developer experience. Features vim-style navigation, an interactive TUI, and scriptable commands—all stored in plain markdown you can version control.

## Features

- ⚡ **Fast** - Single binary, instant startup
- 📝 **Markdown-native** - Todos live in `todo.md`, version control friendly
- ⌨️ **Vim-style navigation** - `j/k`, relative jumps (`5j`), number keys
- 🎨 **Interactive TUI** - Toggle, create, edit, delete, undo, move, copy
- 🔧 **Scriptable** - `list`, `add`, `toggle`, `edit`, `delete` commands
- 🔄 **Auto-updates** - Notifies when new versions are available
- 🌍 **Cross-platform** - macOS, Linux, Windows

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap niklas-heer/tap
brew install tdx
```

### Quick Install Script

```bash
curl -fsSL https://raw.githubusercontent.com/niklas-heer/tdx/main/scripts/install.sh | bash
```

### Download Binary

Download the latest binary for your platform from [Releases](https://github.com/niklas-heer/tdx/releases):

- `tdx-darwin-arm64` - macOS Apple Silicon
- `tdx-darwin-x64` - macOS Intel
- `tdx-linux-x64` - Linux x64
- `tdx-linux-arm64` - Linux ARM64
- `tdx-windows-x64.exe` - Windows x64

### From Source

```bash
git clone https://github.com/niklas-heer/tdx.git
cd tdx
bun install
just build
just install-bin
```

## Usage

### Interactive TUI (Default)

Launch the interactive todo manager:

```bash
tdx
```

**Keyboard Shortcuts:**

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `Space` / `Enter` | Toggle completion |
| `n` | New todo |
| `e` | Edit todo |
| `d` | Delete todo |
| `u` | Undo |
| `y` | Copy todo |
| `m` | Move mode |
| `?` | Help menu |
| `q` / `Esc` | Quit |

**Vim-style jumps:**
- `5j` - Move down 5 lines
- `3k` - Move up 3 lines
- `1-9` - Jump to todo by number

**Display Format:**
```
  [✓] Feed the kitten
➜ [ ] Bake cookies
  [ ] Water the plants
  [ ] Organize the desk

j/k: nav  |  space: toggle  |  n: new  |  e: edit  |  d: delete  |  u: undo  |  q: quit
```

- All todos: `  [✓]` or `  [ ]` (consistent indentation)
- Selected todo: `➜ [✓]` or `➜ [ ]` (arrow indicates selection)
- Checked items: magenta color
- Unchecked items: dim white
- Selected text: bold and bright (color highlighting)
- Arrow and color are sufficient to show which item is selected

### List Todos

Display all todos without entering interactive mode:

```bash
tdx list
```

Output:
```
  1. [✓] Feed the kitten
  2. [ ] Bake cookies
  3. [ ] Water the plants
```

### Add a Todo

Add a new unchecked todo:

```bash
tdx add "Buy milk"
tdx add "Call the dentist"
```

### Toggle a Todo

Toggle the completion status of a todo by index (1-based):

```bash
tdx toggle 1
tdx toggle 3
```

### Edit a Todo

Modify the text of an existing todo:

```bash
tdx edit 2 "Bake chocolate chip cookies"
tdx edit 1 "Feed all the kittens"
```

## File Format

Todos are stored in `todo.md` in your current working directory. The file uses standard Markdown:

```markdown
# Todos

- [x] Completed task
- [ ] Incomplete task
- [ ] Another task to do

You can have other markdown content here too.
It will be preserved exactly when you modify todos.

- [x] Even with mixed content
```

**Format Rules:**
- Todo lines start with `- [ ] ` (unchecked) or `- [x] ` (checked)
- Everything after the checkbox is the todo text
- Non-todo content (headers, paragraphs, etc.) is preserved exactly
- All modifications use atomic writes (temp file → rename)

## Development

### Project Structure

```
tdx/
├── src/
│   ├── cli.ts              # Main entry point
│   ├── commands/           # CLI commands (list, add, toggle, edit)
│   ├── todos/              # Todo model, parser, and writer
│   ├── tui/                # Ink-based TUI component
│   └── fs/                 # File store utilities
├── package.json            # Dependencies
├── tsconfig.json           # TypeScript configuration
├── README.md               # This file
└── todo.md                 # Your todos (created on first run)
```

### Building

**Development:**
```bash
bun run src/cli.ts
```

**Compiled Binary:**
```bash
bun build --compile --minify src/cli.ts --outfile tdx
./tdx list
```

### Testing

Round-trip consistency (read → write with no changes):

```bash
bun run src/cli.ts list
# Edit a todo
bun run src/cli.ts toggle 1
# Verify file is still valid
bun run src/cli.ts list
```

## Examples

### Daily Workflow

```bash
# Start the day - check what's on your plate
tdx list

# Add new tasks
tdx add "Review PR #42"
tdx add "Update documentation"

# Work through the day with interactive mode
tdx

# From another terminal, add urgent items
tdx add "URGENT: Fix production bug"

# Mark items as complete from command line
tdx toggle 3
```

### Scripting

```bash
#!/bin/bash
# Add daily standup tasks
tdx add "Daily standup at 10am"
tdx add "Send standup notes to Slack"

# Toggle them when done
tdx toggle 1
tdx toggle 2
```

### Integration with Other Tools

```bash
# Export todos for reporting
tdx list | grep "^\[x\]" > completed.txt

# Get todo count
PENDING=$(tdx list | grep "\[ \]" | wc -l)
echo "You have $PENDING pending todos"
```

## Error Handling

All commands exit with status `0` on success, non-zero on error.

**Example error cases:**
```bash
# Invalid index
$ tdx toggle 999
Error: Todo index 999 out of range (1-5)

# Empty text
$ tdx add ""
Error: Todo text cannot be empty

# File permission issues
$ chmod 000 todo.md && tdx list
Error: Failed to read todo.md: Permission denied
```

## Configuration

**Environment variables:**
- `TDX_NO_UPDATE_CHECK=1` - Disable automatic update checks

**Custom file path:**
```bash
tdx ~/notes/work.md           # Use specific file
tdx project.md add "Task"     # Commands work with custom files too
```

## Notes

- The `todo.md` file is created automatically on first use
- File modifications are atomic - no risk of corruption
- Non-todo content (headers, paragraphs) is preserved exactly
- Version check results are cached in `~/.tdx/` for 24 hours

## License

MIT - see [LICENSE](LICENSE)