# tdx - A Fast Markdown Todo Manager

A lightweight, single-binary CLI tool for managing todos in a `todo.md` file using Bun and TypeScript. Features an interactive TUI built with Ink and minimal non-interactive commands for scripting.

## Features

- ⚡ **Fast & Lightweight** - Built with Bun, compiles to a single binary
- 📝 **Markdown-based** - Todos stored in a simple `todo.md` file
- 🎨 **Interactive TUI** - Navigate, create, delete, toggle, and undo with keyboard shortcuts
- 🔒 **Atomic writes** - Safe file operations, no corruption
- 📦 **Zero dependencies** - Minimal custom parser, no heavy libraries
- 🛠️ **CLI commands** - `list`, `add`, `toggle`, `edit` for automation

## Installation

### From Source

Clone the repository and build:

```bash
git clone <repository>
cd tdx
bun install
bun build --compile --minify src/cli.ts --outfile tdx
sudo mv tdx /usr/local/bin/  # Optional: add to PATH
```

### Development Mode

```bash
bun run src/cli.ts
```

## Usage

### Interactive TUI (Default)

Launch the interactive todo manager:

```bash
tdx
```

**Keyboard Shortcuts:**
- `j` or **Down** - Move selection down
- `k` or **Up** - Move selection up
- **Space** or **Enter** - Toggle todo completion
- `n` - Create a new todo (opens input mode)
- `d` - Delete selected todo
- `u` - Undo the last action
- `q` or **Esc** - Exit

**Display Format:**
```
  [✓] Feed the kitten
➜ [ ] Bake cookies
  [ ] Water the plants
  [ ] Organize the desk

j/k: nav  |  space: toggle  |  n: new  |  d: delete  |  u: undo  |  q: quit
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

## Notes

- The `todo.md` file is created automatically on first use with a simple header
- File modifications are atomic - no risk of corruption
- Only todo lines are rewritten; all other content is preserved byte-for-byte
- The tool works with any `todo.md` file in your current working directory
- No configuration files or environment variables needed

## License

MIT