# tdx

**Your todos, in markdown, done fast.**

<p align="center">
  <img src="assets/demo.gif" alt="tdx demo" width="600">
</p>

A fast, single-binary CLI todo manager focused on developer experience. Features vim-style navigation, an interactive TUI, and scriptable commands—all stored in plain markdown you can version control.

## Features

- ⚡ **Fast** - Single binary (4MB), instant startup, 30-40x faster than alternatives
- 📝 **Markdown-native** - Todos live in `todo.md`, version control friendly
- ⌨️ **Vim-style navigation** - `j/k`, relative jumps (`5j`), number keys
- 🖥️ **Interactive TUI** - Toggle, create, edit, delete, undo, move, copy
- 🔧 **Scriptable** - `list`, `add`, `toggle`, `edit`, `delete` commands
- 🌍 **Cross-platform** - macOS, Linux, Windows

## Installation

### Homebrew (macOS/Linux)

```bash
brew install niklas-heer/tap/tdx
```

### Quick Install Script

```bash
curl -fsSL https://niklas-heer.github.io/tdx/install.sh | bash
```

### Download Binary

Download the latest binary for your platform from [Releases](https://github.com/niklas-heer/tdx/releases):

- `tdx-darwin-arm64` - macOS Apple Silicon
- `tdx-darwin-amd64` - macOS Intel
- `tdx-linux-amd64` - Linux x64
- `tdx-linux-arm64` - Linux ARM64
- `tdx-windows-amd64.exe` - Windows x64

### From Source

Requires Go 1.21+:

```bash
git clone https://github.com/niklas-heer/tdx.git
cd tdx
just build
just install
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
| `j` / `k` | Move down / up |
| `Space` / `Enter` | Toggle completion |
| `n` | New todo |
| `e` | Edit todo |
| `d` | Delete todo |
| `c` | Copy to clipboard |
| `m` | Move mode |
| `/` | Fuzzy search |
| `u` | Undo |
| `?` | Help menu |
| `Esc` | Quit |
| `Cmd+V` / `Ctrl+Y` | Paste (in edit mode) |

**Vim-style jumps:**
- `5j` - Move down 5 lines
- `3k` - Move up 3 lines

**Fuzzy Search:**
Press `/` to enter search mode. Type to filter todos with live highlighting. Press `Enter` to select or `Esc` to cancel.

### CLI Commands

```bash
# List all todos
tdx list

# Add a new todo
tdx add "Buy milk"

# Toggle completion (1-based index)
tdx toggle 1

# Edit a todo
tdx edit 2 "Updated text"

# Delete a todo
tdx delete 3

# Use custom file
tdx ~/notes/work.md list
tdx project.md add "Task"
```

## File Format

Todos are stored in `todo.md` using standard Markdown:

```markdown
# Todos

- [x] Completed task
- [ ] Incomplete task
- [ ] Another task

Other markdown content is preserved.
```

## Development

### Prerequisites

- Go 1.21+
- [just](https://github.com/casey/just) (command runner)

### Building

```bash
# Build binary
just build

# Build for all platforms
just build-all

# Install to /usr/local/bin
just install
```

### Project Structure

```
tdx/
├── main.go          # Main application
├── config.go        # Version/description variables
├── tdx.toml         # Build configuration
├── go.mod           # Go modules
├── justfile         # Build commands
└── todo.md          # Your todos
```

### Commands

```bash
just build      # Build binary
just install    # Install to PATH
just tui        # Run TUI
just list       # List todos
just add "X"    # Add todo
just toggle 1   # Toggle todo
just check      # Run go vet
just fmt        # Format code
just clean      # Clean artifacts
```

## Configuration

### Config File

Create `~/.config/tdx/config.toml` to customize tdx:

```toml
[theme]
name = "tokyo-night"  # or any builtin theme

[display]
max_visible = 10       # limit visible todos (0 = unlimited)
check_symbol = "✓"     # symbol for completed items
select_marker = "➜"    # symbol for selected item

# Optional: override theme colors
[colors]
Base = "#c0caf5"
Dim = "#565f89"
Accent = "#7aa2f7"
Success = "#9ece6a"
Warning = "#e0af68"
Important = "#bb9af7"
AlertError = "#f7768e"
```

### Builtin Themes

- `tokyo-night` (default)
- `gruvbox-dark`
- `catppuccin-mocha`
- `nord`
- `dracula`
- `solarized-dark`
- `one-dark`
- `monokai`
- `github-dark`
- `rose-pine`

### Custom File Path

```bash
tdx ~/notes/work.md           # Use specific file
tdx project.md add "Task"     # All commands work
```

### Build Configuration

Build metadata in `tdx.toml`:
```toml
version = "0.4.0"
description = "your todos, in markdown, done fast"
```

## License

MIT - see [LICENSE](LICENSE)
