# tdx

[![CI](https://github.com/niklas-heer/tdx/actions/workflows/ci.yml/badge.svg)](https://github.com/niklas-heer/tdx/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/niklas-heer/tdx/main/.github/badges/coverage.json)](https://github.com/niklas-heer/tdx/actions/workflows/ci.yml)
[![GitHub Downloads](https://img.shields.io/github/downloads/niklas-heer/tdx/total?logo=github&label=downloads)](https://github.com/niklas-heer/tdx/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/niklas-heer/tdx)](https://goreportcard.com/report/github.com/niklas-heer/tdx)

**Your todos, in markdown, done fast.**

<p align="center">
  <img src="assets/demo-0.11.0.gif" alt="tdx demo" width="600">
</p>

A fast, single-binary CLI todo manager focused on developer experience. Features vim-style navigation, an interactive TUI, and scriptable commands—all stored in plain markdown you can version control.

## Features

- ⚡ **Fast** - Native binary with no application runtime to install
- 📝 **Markdown-native** - Todos live in `todo.md`, version control friendly
- ⌨️ **Vim-style navigation** - `j/k`, relative jumps (`5j`), number keys
- 🖥️ **Interactive TUI** - Toggle, create, edit, delete, undo, move, copy
- 🎯 **Command Palette** - Helix-style `:` commands with fuzzy search
- 📋 **Manual Save** - Reuse checklists without automatic writes; legacy read-only options remain supported
- 🔖 **Saved Views** - Save per-file filters and section focus; optionally restore the last view
- 🔧 **Scriptable** - Filtered JSON output and `list`, `add`, `toggle`, `edit`, `delete` commands
- 🔄 **Smart Conflict Handling** - Atomic saves, external-change detection, visual conflict diffs
- 🕘 **Version History** - Automatic snapshots with visual diffs and safe restore
- 📑 **Per-File Configuration** - YAML frontmatter for file-specific settings
- 📂 **Recent Files** - Jump to recently opened files with cursor position restoration
- 🌍 **Cross-platform** - macOS, Linux, Windows

## Sections and projects

Press **s** to open the section overview. It lists every Markdown heading, including empty sections, with nested headings indented and completion counts beside each project.

| Key in the section overview | Action |
| --- | --- |
| ↑ / ↓ or j / k | Select a section |
| Enter | Focus the section and its subsections |
| Space | Fold or unfold its tasks |
| e | Rename the selected heading |
| n | Create a section after the selected section |
| N | Create a subsection (up to heading level 6) |
| a | Show all tasks and clear folds |
| Esc | Return to the task list |

While focused, **n** adds a task after the selection; in an empty section it creates that section's first task. **N** adds to the focused section's own task list. Press **S** to return to all sections. Tag, priority, due-date, completed-task filters, and search respect the current section. Focus and folds last for the current session and reset when a file is reloaded, headings change, or an edit is undone. Save a named view to recall them later. **u** undoes up to 100 edits during the session.

Section editing uses the same guarded saves and version history as task editing. Read-only files support section browsing and focus without permitting heading edits.

### What's new in 0.14.0

[Release notes and upgrade guidance](docs/releases/0.14.0.md) · [Download v0.14.0](https://github.com/niklas-heer/tdx/releases/tag/v0.14.0)

- Preserve surrounding Markdown bytes during checkbox-only edits, with faster cached updates.
- Keep undo history intact when cancelling input and detect external edits made while typing.
- Validate sustained use with replayable sessions, CLI contracts and real-terminal tests.
- Manage projects directly through Markdown headings, without opening another editor.
- Compose tdx with scripts and editors using filtered JSON output and explicit file selection.
- Type and paste international text in task, search, command, and recent-file inputs.
- Use the current Bubble Tea and Lip Gloss v2 terminal renderer and input handling.
- Use mise for reproducible development tools and tasks. `mise tasks` lists available commands; `mise run check` runs local validation.
- Installation defaults to `~/.local/bin`; set `TDX_INSTALL_DIR` to choose another directory. Existing todo files and global configuration remain compatible.

## Saved views

Set your tag (`t`), priority (`p`), due-date (`D`), completed-task and section filters, then run `:save-view`. Enter a name such as `Today` or `Backend`. Use `:views` to select a saved view, `:delete-view` to remove one, and `:clear-view` to show all tasks and stop restoring that view. Replacing or deleting a view requires confirmation; `Esc` cancels.

The status bar shows the active view and marks it modified if you change its filters. Views include heading visibility, section focus and folds. Section references follow heading ancestry and repeated-title occurrence; removed or renamed headings are skipped when loading a view. Task text and Markdown files are never changed by saving a view.

Views are stored per canonical file path in the configuration directory, so opening a file through a symlink shares its views. To reopen the last saved/selected view automatically, opt in through `~/.config/tdx/config.toml`:

```toml
[views]
restore = true
```

Without this setting, views are loaded only when requested. Changes to an active view are not saved until `:save-view` is used again.

## Scripting and editor integrations

Query a project without opening the TUI or writing history:

```bash
tdx --file ./TASKS list --json --status open --tag backend
tdx tasks.md list --status done
tdx tasks.md list --json | jq -r '.[] | select(.priority == 1) | .text'
tdx add -- --read-only          # Add literal flag-like text
tdx --read-only tasks.md list   # Reads work; CLI writes are rejected
# Query a heading and its descendants without renumbering tasks
tdx tasks.md list --json --section Backend --priority 1 --due week
# Explicit completion is safe to retry
tdx tasks.md done 2
tdx tasks.md undone 2
```

`--file` (or `-f`) accepts any filename, including paths with spaces or without a `.md` extension. The positional `tdx tasks.md …` syntax still works. Options may appear before or after the command; use `--` before task text that starts with a dash. Shell quoting is preserved as received, including intentional quote characters in task text.

`list` supports `--status all|open|done` (default `all`) and exact, case-sensitive `--tag` filters, with or without the leading `#`. Repeat `--tag` to require every tag. Filters apply to both text and JSON output. `--priority N` matches an exact priority (`0` means unset). `--due` accepts `all` (has a due date), `none`, `overdue`, `today`, `week` (today through seven days ahead), or an exact `YYYY-MM-DD` date. Relative dates use the local calendar. `--section "Heading title"` matches the exact Markdown heading title and includes its subsections; repeated titles select all matching sections. These filters compose with status and tags. An empty JSON result is `[]`, including when the todo file does not exist; listing does not create that file or require a writable history directory.

Each JSON task contains:

| Field | Meaning |
| --- | --- |
| `index` | One-based position in the full file, preserved when filtering |
| `text`, `checked` | Markdown task text and completion state |
| `depth`, `parent_index` | Nesting depth and one-based parent position; `null` for root tasks |
| `tags`, `priority` | Tag array (empty is `[]`); priority `0` means unset |
| `due_date` | `YYYY-MM-DD` or `null` |

Indexes are positions, **not persistent IDs**: re-query after inserting, deleting, reordering, or externally editing tasks before using an index in a write command. JSON goes only to stdout; errors go to stderr with exit code 1. Successful commands exit 0. The existing array shape and field meanings are retained. Consumers should ignore additional fields. Removing or changing a field's type or meaning requires an explicitly versioned contract and migration guidance, including during pre-1.0 development.

### Guard scripts against intervening edits

Use `list --with-revision` for one consistent snapshot of both the query result and the document revision:

```json
{"schema_version": 1, "revision": "sha256:<64 lowercase hex digits>", "tasks": []}
```

`--with-revision` implies JSON and changes the top-level shape to this envelope; ordinary `--json` continues to return an array. The revision covers the entire file, including frontmatter and line endings. A nonexistent file returns `missing`, which differs from an existing empty file. `tdx tasks.md revision` prints only the revision token and does not create history or configuration files.

```bash
snapshot=$(tdx tasks.md list --with-revision --status open --tag backend)
revision=$(printf '%s' "$snapshot" | jq -r .revision)
index=$(printf '%s' "$snapshot" | jq -r '.tasks[0].index // empty')
if [ -n "$index" ]; then
  tdx tasks.md done "$index" --if-revision "$revision"
fi
```

All task mutation commands support `--if-revision`. A stale revision fails before changing the document; query again and reconsider the selected task. Changes that arrive after loading are still caught by the normal locked save. `done` and `undone` leave an already-correct file unchanged, but a supplied stale revision still fails. Revision tokens detect intervening content changes; they are not persistent task identities or authentication tokens.

### Shell completion

```bash
# Bash: add to ~/.bashrc
source <(tdx completion bash)
# Zsh: after autoload -Uz compinit && compinit
source <(tdx completion zsh)
# Fish: create its completions directory first if needed
tdx completion fish > ~/.config/fish/completions/tdx.fish
```

Completion generation is static and never reads your tasks or history.

## Installation

### Homebrew (macOS/Linux)

```bash
brew install niklas-heer/tap/tdx
```

To upgrade an existing Homebrew installation, run `brew update` and `brew upgrade niklas-heer/tap/tdx`.

### Quick Install Script

```bash
curl -fsSL https://niklas-heer.github.io/tdx/install.sh | bash
```

The installer verifies the selected executable against the release's `SHA256SUMS` before replacing an existing installation. It requires `sha256sum` or `shasum`, uses a temporary download directory, and installs to `~/.local/bin` by default. Set `TDX_INSTALL_DIR` or `TDX_VERSION` to choose the destination or release. Releases predating checksum manifests cannot be installed with the verified script; use the Homebrew formula or download that historical binary directly from its release page. The script never silently bypasses verification.

New releases also publish a signed provenance bundle. With GitHub CLI installed, verify a downloaded executable using `gh attestation verify <binary> --repo niklas-heer/tdx`. Checksum verification detects mismatched bytes; attestation verification separately checks the build's provenance.

The installer writes to `~/.local/bin` by default. Add that directory to `PATH`, or set `TDX_INSTALL_DIR` when running the script. To pin a release, use `curl -fsSL https://niklas-heer.github.io/tdx/install.sh | TDX_VERSION=0.14.0 bash`. Check `command -v tdx` if an older installation exists elsewhere.

### Download Binary

Download the latest binary for your platform from [Releases](https://github.com/niklas-heer/tdx/releases):

- `tdx-darwin-arm64` - macOS Apple Silicon
- `tdx-darwin-amd64` - macOS Intel
- `tdx-linux-amd64` - Linux x64
- `tdx-linux-arm64` - Linux ARM64
- `tdx-windows-amd64.exe` - Windows x64

### From Source

Requires [mise](https://mise.jdx.dev), which installs the pinned Go toolchain:

```bash
git clone https://github.com/niklas-heer/tdx.git
cd tdx
mise trust
mise install
mise run build
mise run install
```

### Nix

```bash
# Try it without installing
nix run github:niklas-heer/tdx

# Install to profile
nix profile install github:niklas-heer/tdx
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
| `gg` | Go to first item |
| `G` | Go to last item |
| `Space` / `Enter` | Toggle completion |
| `n` | New todo after cursor |
| `N` | New todo at end of file |
| `e` | Edit todo |
| `d` | Delete todo |
| `c` | Copy to clipboard |
| `m` | Move mode |
| `Tab` | Indent (nest under previous) |
| `Shift+Tab` | Outdent (move up one level) |
| `/` | Fuzzy search |
| `t` | Tag filter |
| `p` | Priority filter |
| `D` | Due date filter |
| `s` / `S` | Section overview / show all sections |
| `r` | Recent files |
| `:` | Command palette |
| `u` | Undo |
| `?` | Help menu |
| `Esc` | Quit |
| `Cmd+V` / `Ctrl+Y` | Paste (in edit mode) |

**Command Palette (`:`):**

Press `:` to open the command palette with fuzzy search. Available commands:

| Command | Description |
|---------|-------------|
| `check-all` | Mark all todos as complete |
| `uncheck-all` | Mark all todos as incomplete |
| `sort-done` | Sort todos by completion (incomplete first) |
| `sort-priority` | Sort todos by priority (p1 first, then p2, etc.) |
| `sort-due` | Sort todos by due date (earliest first) |
| `filter-done` | Toggle showing/hiding completed todos |
| `filter-due` | Toggle showing only todos with due dates |
| `filter-overdue` | Toggle showing only overdue todos |
| `filter-today` | Toggle showing only todos due today |
| `filter-week` | Toggle showing only todos due this week |
| `clear-done` | Delete all completed todos |
| `manual-save` | Toggle automatic saving (`read-only` remains an alias) |
| `save-view` / `views` | Save the current filters / open a saved view |
| `delete-view` / `clear-view` | Delete a saved view / clear filters and active view |
| `save` | Save current state to file |
| `force-save` | Force save even if file was modified externally |
| `reload` | Reload file from disk (discards unsaved changes) |
| `versions` | Browse, compare, and restore file version history |
| `diff` | Inspect a pending save conflict |
| `sections` / `all-sections` | Open section overview / clear section focus and folds |
| `wrap` | Toggle word wrap for long lines |
| `line-numbers` | Toggle relative line numbers |
| `set-max-visible` | Set max visible items for this session |
| `show-headings` | Toggle displaying markdown headings between tasks |

**Manual Save (compatible with read-only mode):**

Start tdx with `--manual-save`, `-r`, or `--read-only` for workflows where you want temporary checklist edits without automatic saving:

```bash
tdx -r checklist.md
```

The status bar displays `MANUAL SAVE`. Use `:save` to save when ready, or `:manual-save` (legacy alias `:read-only`) to enable automatic saving. This mode permits temporary task edits; CLI mutation commands are rejected when either flag is present. The preferred global setting is `[defaults] manual_save = true`; an explicitly set `manual_save` overrides the legacy `read_only` setting.

**Vim-style navigation:**
- `5j` - Move down 5 lines
- `3k` - Move up 3 lines
- `gg` - Jump to first item
- `G` - Jump to last item

**Fuzzy Search:**
Press `/` to enter search mode. Type to filter todos with live highlighting. Press `Enter` to select or `Esc` to cancel.

**Nested Tasks:**

Organize your todos hierarchically using `Tab` and `Shift+Tab`:

```markdown
- [ ] Main project
  - [ ] Subtask 1
  - [ ] Subtask 2
    - [ ] Sub-subtask
- [ ] Another task
```

- Press `Tab` to indent a task under its previous sibling
- Press `Shift+Tab` to outdent (move up one level)
- Deleting a parent task promotes its children to the parent's level
- New tasks (`n`) are created at the same nesting level as the cursor

**Tags & Filtering:**

Add hashtags to your todos for organization:

```markdown
- [ ] Fix authentication #urgent #backend
- [ ] Update docs #docs
- [ ] Add dark mode #feature #frontend
```

Press `t` to open tag filter mode:
- Navigate with `↑/↓` or `j/k`
- Toggle tags with `Space` or `Enter`
- Clear all filters with `c`
- Press `Esc` when done

Active tag filters are shown in the status bar. Todos are automatically filtered to show only matching items.

**Priorities:**

Add priority markers to your todos using `!p1`, `!p2`, `!p3`, etc.:

```markdown
- [ ] Fix critical security bug !p1
- [ ] Update dependencies !p2
- [ ] Write documentation !p3
- [ ] Refactor code !p2
- [ ] Add nice-to-have feature
```

Priority levels:
- `!p1` - Critical/Urgent (displayed in red)
- `!p2` - High priority (displayed in orange)
- `!p3` - Medium priority (displayed in yellow)
- `!p4+` - Lower priorities (displayed dimmed)

Use the `:sort-priority` command to sort todos by priority (p1 first, then p2, etc.). Tasks without a priority marker are placed at the end. You can combine priorities with tags: `Fix bug !p1 #backend #urgent`

**Priority Filtering:**

Press `p` to open priority filter mode:
- Navigate with `↑/↓` or `j/k`
- Toggle priorities with `Space` or `Enter`
- Clear all filters with `c`
- Press `Esc` when done

Active priority filters are shown in the status bar (e.g., `⚡ p1 p2`). You can combine priority and tag filters to narrow down your view.

**Due Dates:**

Add due dates to your todos using `@due(YYYY-MM-DD)`:

```markdown
- [ ] Submit quarterly report @due(2025-12-01)
- [ ] Review pull request @due(2025-11-30) #code-review
- [ ] Fix critical bug !p1 @due(2025-11-29) #urgent
- [ ] Plan team meeting @due(2025-12-15)
```

Due date display colors based on urgency:
- **Overdue** - Red (past the due date)
- **Due today** - Orange
- **Due soon** - Yellow (within 3 days)
- **Future** - Dimmed

Use the `:sort-due` command to sort todos by due date (earliest first). Tasks without a due date are placed at the end. You can combine due dates with priorities and tags.

**Due Date Filtering:**

Press `D` (capital D) to open due date filter mode:
- **Overdue** - Show only overdue tasks
- **Today** - Show tasks due today
- **This Week** - Show tasks due within 7 days
- **Has Due Date** - Show all tasks with any due date

Navigate with `↑/↓` or `j/k`, select with `Space` or `Enter`, clear with `c`, and press `Esc` when done.

Active due date filters are shown in the status bar (e.g., `📅 overdue`). You can combine due date filters with priority and tag filters.

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

# Open most recent file
tdx last

# Use custom file
tdx ~/notes/work.md list
tdx project.md add "Task"
```

### Clipboard

Copy uses `pbcopy` on macOS, `wl-copy`/`wl-paste` in Wayland sessions, `xclip` or `xsel` in X11 sessions, and PowerShell on Windows. Clipboard commands run outside the input loop with a bounded timeout. Successful copies show confirmation; missing helpers, desktop connection failures and timeouts show an error. Terminal paste remains available in input fields, including over SSH; a headless remote process may have no desktop clipboard to copy into.

### Recent Files

tdx automatically tracks recently opened files and restores your cursor position when you reopen them.

**TUI Mode:**

Press `r` in the TUI to open the recent files overlay:
- Type to filter files by path (fuzzy search)
- Navigate with `↑/↓` or `j/k`
- Press `Enter` to open a file
- Press `Esc` or `r` to close

**CLI Commands:**

```bash
# Open the most recently used file
tdx last

# List recently opened files (sorted by frequency and recency)
tdx recent

# Open a specific recent file by number
tdx recent 1

# Clear recent files history
tdx recent clear
```

**Features:**
- **Smart Sorting**: Files are ranked by both frequency (how often you open them) and recency (when you last accessed them)
- **Cursor Restoration**: When you reopen a file, tdx automatically restores your cursor to the last position
- **Change Detection**: If the file content has changed since your last visit, the cursor resets to the first item for safety
- **Configurable Limit**: Set maximum recent files in your config (default: 20)

**Configuration:**

In `~/.config/tdx/config.toml`:

```toml
[recent]
max_files = 20  # Maximum number of recent files to track
```

Recent files are stored in `~/.config/tdx/recent.json` and include:
- File path
- Last access time
- Access count (frequency)
- Last cursor position
- Content hash (for change detection)

### Version History

tdx automatically stores content-addressed snapshots when a file is opened or successfully changed. Open the command palette and run `:versions` to compare the current file with earlier versions and restore one safely.

- Navigate versions with `↑`/`↓` or `j`/`k`
- Scroll the diff with `PgUp`/`PgDn`
- Press `Enter`, then `y`, to confirm a restore
- Press `Esc` to close without changing the file

By default, tdx retains the latest 100 versions per file. Configure the limit in `~/.config/tdx/config.toml`; set it to `0` for unlimited history:

```toml
[versioning]
max_versions = 100
```

## File Format

Todos are stored in `todo.md` using standard Markdown:

```markdown
# Todos

- [x] Completed task
- [ ] Incomplete task
- [ ] Another task
```

Checkbox-only changes to a freshly loaded document preserve every surrounding byte, including frontmatter, HTML, fenced examples and line endings. Adding, editing, moving or removing tasks uses the AST serializer and may normalize formatting. HTML blocks, table syntax and ordinary paragraph line breaks are retained, but complex multiline task bodies and reference definitions do not yet have a general lossless round-trip guarantee. Keep rich documents under version control.

### Configuration

tdx supports three levels of configuration with the following priority:

**Priority Order:** CLI flags > Frontmatter > Global config > Defaults

#### Global Configuration

Create `~/.config/tdx/config.toml` (or `$XDG_CONFIG_HOME/tdx/config.toml`) to set defaults:

```toml
[theme]
name = "tokyo-night"

[display]
check_symbol = "✓"
select_marker = "➜"

[defaults]
file = "todo.md"      # default file (use ~/path for central file)
max_visible = 0       # 0 = unlimited
word_wrap = true
show_headings = false
read_only = false
filter_done = false

[recent]
max_files = 20

[versioning]
max_versions = 100  # 0 = unlimited
```

You only need to include the settings you want to change from the defaults.

**Available options:**

| Section | Option | Type | Default | Description |
|---------|--------|------|---------|-------------|
| `[theme]` | `name` | string | "tokyo-night" | Theme to use |
| `[display]` | `check_symbol` | string | "✓" | Symbol for completed items |
| `[display]` | `select_marker` | string | "➜" | Symbol for selected item |
| `[defaults]` | `file` | string | "todo.md" | Default file path (use `~/path` for central file) |
| `[defaults]` | `max_visible` | number | 0 | Limit visible tasks (0 = unlimited) |
| `[defaults]` | `word_wrap` | boolean | true | Enable word wrapping for long lines |
| `[defaults]` | `show_headings` | boolean | false | Show markdown headings between tasks |
| `[defaults]` | `manual_save` | boolean | unset | Preferred setting; explicitly overrides `read_only` |
| `[defaults]` | `read_only` | boolean | false | Legacy setting: disable automatic TUI saves; reject CLI mutations |
| `[defaults]` | `filter_done` | boolean | false | Hide completed tasks by default |
| `[recent]` | `max_files` | number | 20 | Maximum recent files to track |
| `[versioning]` | `max_versions` | number | 100 | Versions retained per file (0 = unlimited) |
| `[views]` | `restore` | boolean | false | Restore the last saved/selected view when opening its file |

#### Per-File Configuration

Add YAML frontmatter to customize behavior for specific files:

```markdown
---
read-only: false
max-visible: 10
show-headings: true
---
# Todos

- [ ] Task one
```

**Examples:**

Read-only checklist:
```markdown
---
read-only: true
---
# Shopping List
- [ ] Milk
```

Project tracker with headings:
```markdown
---
show-headings: true
max-visible: 15
filter-done: true
---
# Project Tasks

## Backend
- [ ] API endpoints

## Frontend
- [ ] UI components
```

#### Configuration Priority

Settings are applied in this order (highest to lowest priority):

1. **CLI flags** - `tdx -r --show-headings todo.md`
2. **Frontmatter** - YAML at top of individual todo files
3. **Global config** - `~/.config/tdx/config.toml`
4. **Defaults** - Built-in defaults (word_wrap: true, others: false/0)

**Example:**
```bash
# config.toml sets word_wrap = false
# Frontmatter sets read-only: true
# CLI flag: --show-headings
# Result: word_wrap=false, read_only=true, show_headings=true
tdx --show-headings todo.md
```

## Architecture

### Editing and Markdown preservation

[Goldmark](https://github.com/yuin/goldmark) identifies tasks and headings using task-list, table and strikethrough extensions. The CLI and TUI apply shared document actions through `internal/editor`; each application instance owns its configuration, styles and persistence callbacks.

A checkbox edit uses the AST's exact source location and updates cached checked state without extracting all task metadata again. Text and structural edits patch affected source ranges, validate the resulting task structure, and reparse before committing. Unrelated source bytes—including reference definitions, HTML, tables and line endings—stay intact. Moving or sorting a parent carries its subtree and attached body. Unsupported boundaries, including structural rearrangement inside blockquotes or tab-indented continuations, return an error instead of rewriting the document. Checkbox toggles and supported localized edits remain available where their source ranges can be identified.

Undo retains up to 100 committed snapshots, with provisional input stored separately until confirmed. File saves compare the loaded disk revision, acquire a file lock and atomically replace the target. Watcher reloads defer during pending input so a concurrent editor's changes cannot silently become the revision used by a later save. Explicit conflict recovery and version history remain available in the TUI.

### Performance and sustained-use testing

Checkbox nodes and headings are cached. Search is debounced for 50 ms, while immediate Enter and navigation use the current query. The [measured 100-hour simulated campaign](experiments/usage/README.md) covers 36,000 actions, real disk saves, conflicts, cancellation and reload, with separate executable and PTY contracts.

The report includes loaded-document checkbox benchmarks, complete action latency, allocation profiles, exact source revisions and reproduction commands. `mise run test:usage-structural` exercises rich documents and nested structural actions; failed campaigns retain a replayable prefix, reduced trace and reduction diagnostic. Weekly CI rotates recorded seeds and retains replay/profile artifacts. These measurements are workload-specific; they do not establish an application-wide comparison with other tools. Simulated hours are accelerated actions rather than wall-clock endurance. See the [Rust evaluation](experiments/rust-eval/README.md) for the narrower parser experiment and its limitations.

### Project Structure

```
tdx/
├── .dagger/             # Portable CI and release pipeline (Go)
├── mise.toml           # Pinned tools and development tasks
├── cmd/tdx-usage/       # Developer replay runner (not shipped)
├── cmd/tdx/             # Main application
│   ├── main.go          # Entry point, CLI routing
│   ├── config.go        # Build-time configuration
│   ├── userconfig.go    # User configuration (themes, settings)
│   └── *_test.go        # Comprehensive test suite
├── internal/
│   ├── markdown/        # AST-based markdown engine
│   │   ├── parser.go    # Markdown → AST
│   │   ├── ast.go       # AST data structures
│   │   └── serializer.go # AST → Markdown
│   ├── tui/             # Terminal UI (Bubble Tea)
│   │   ├── model.go     # Application state
│   │   ├── update.go    # Event handling
│   │   ├── view.go      # Rendering
│   │   ├── commands.go  # Command palette
│   │   ├── render.go    # Display logic
│   │   └── *_test.go    # Unit & benchmark tests
│   ├── usage/           # Independent oracle and session replay
│   ├── versioning/      # SQLite-backed file history
│   ├── editor/          # Shared editing actions and bounded undo
│   ├── cmd/             # CLI command handlers
│   │   └── cli.go       # List, add, toggle, etc.
│   ├── config/          # Configuration handling
│   │   ├── config.go    # Legacy YAML config (deprecated)
│   │   └── recent.go    # Recent files tracking
│   └── util/            # Utilities
│       ├── text.go      # Text processing, fuzzy search
│       ├── clipboard.go # Clipboard operations
│       └── text_test.go # Unit & benchmark tests
└── scripts/             # Development & release tools
```

## Development

### Prerequisites

- Go 1.27.1 (installed by mise)
- [mise](https://mise.jdx.dev) (tool versions and tasks)
- Bash (use Git Bash on Windows)

After cloning, run `mise trust` and `mise run setup`. Docker is required only for Dagger tasks (`ci` and `release-artifacts`); normal build, test, and lint tasks run locally.

- [Dagger 0.21.9](https://docs.dagger.io/install/) and a Docker-compatible container runtime for the portable CI pipeline

The CLI and TUI share document actions in `internal/editor`. Configuration, styles, recent files, and Markdown history callbacks belong to each application instance, making isolated integration tests straightforward.

For the optional Rust parser experiment, run `mise run rust:check` and `mise run rust-eval`. See the [measured comparison and limitations](experiments/rust-eval/README.md); Rust is not part of the shipped application.

### Sustained-use testing

`mise run test:usage` replays 100 simulated hours (36,000 actions) against an independent task oracle, real TUI updates and disk saves. Use `mise run test:usage-cli` for the executable contract and `mise run test:terminal` for actual terminal sessions. See the [harness, measured improvements and rewrite limitations](experiments/usage/README.md) for replay instructions and the distinction between simulated time and wall-clock endurance.

### Building

```bash
# Build binary
mise run build

# Build for all platforms
mise run build-all

# Install to ~/.local/bin
mise run install
```

### Commands

```bash
mise run build        # Build binary
mise run build-all    # Build all release targets with Dagger
mise run install      # Install to ~/.local/bin
mise run tui          # Run TUI
mise run demo         # Try a disposable project with isolated config/history
mise run coverage     # Generate dist/coverage/index.html
mise run list         # List todos
mise run add "X"      # Add todo
mise run toggle 1     # Toggle todo
mise run check        # Run local quality checks
mise run fmt          # Format code
mise run ci           # Run the same portable checks used by GitHub CI
mise run ci-lint      # Run the pinned linter through Dagger
mise run ci-test      # Run race-enabled tests through Dagger
mise run ci-workflows # Validate GitHub workflow syntax locally
mise run clean        # Clean artifacts
```

For a fast feedback loop, pass Go test arguments directly:

```bash
mise run test -- ./internal/cmd -run TestList -count=1
mise run test:race -- ./internal/markdown
mise run coverage
```

Without arguments, both test tasks run every application package. `mise run demo` opens a temporary copy of `examples/project-tracker.md`, with its own configuration and history. Edits disappear on exit; personal settings and the source example are untouched. The command tests likewise use temporary configuration, and their subprocess binary/config directories are cleaned up after the suite finishes.

The Dagger pipeline is pinned in `dagger.json` and implements CI in Go. GitHub still runs native macOS and Windows filesystem tests because those platform semantics cannot be reproduced by Linux containers.

## Theme Customization

### Theme Picker

Press `:` to open the command palette and select `theme` to open the theme picker:
- Navigate with `↑/↓` or `j/k` to preview themes in real-time
- Press `Enter` to apply and save the theme
- Press `Esc` to cancel and restore the previous theme

The selected theme is automatically saved to your config file.

### Theme Config File

Set your theme in `~/.config/tdx/config.toml`:

```toml
[theme]
name = "tokyo-night"  # or any builtin/custom theme
```

See the [Global Configuration](#global-configuration) section for all available settings.

### Builtin Themes

- `tokyo-night` (default)
- `catppuccin-latte` (light)
- `catppuccin-frappe`
- `catppuccin-macchiato`
- `catppuccin-mocha`
- `dracula`
- `github-dark`
- `gruvbox-dark`
- `monokai`
- `nord`
- `one-dark`
- `rose-pine`
- `solarized-dark`

### Custom Themes

Create your own themes by adding `.toml` files to `~/.config/tdx/themes/`:

```toml
# ~/.config/tdx/themes/my-theme.toml
[theme]
name = "my-theme"
author = "Your Name"

[colors]
# Core colors
Base = "#c0caf5"       # default foreground
Dim = "#565f89"        # muted text
Accent = "#7aa2f7"     # highlights, selections
Success = "#9ece6a"    # completed items, matches
Warning = "#e0af68"    # move mode
Important = "#bb9af7"  # checked items
AlertError = "#f7768e" # errors

# Tags (hashtags like #urgent)
Tag = "#e0af68"

# Priorities (!p1, !p2, !p3+)
PriorityHigh = "#f7768e"    # !p1 - critical
PriorityMedium = "#bb9af7"  # !p2 - high
PriorityLow = "#565f89"     # !p3+ - medium/low

# Due dates (@due(YYYY-MM-DD))
DueUrgent = "#7dcfff"  # overdue or due today
DueSoon = "#7aa2f7"    # due within 3 days
DueFuture = "#565f89"  # due later
```

Custom themes appear in the theme picker alongside builtin themes. You can also override builtin themes by creating a file with the same theme name.

**Note:** The Tag, Priority*, and Due* colors are optional. If omitted, sensible defaults are used based on the core colors.

### Custom File Path

```bash
tdx ~/notes/work.md           # Use specific file
tdx project.md add "Task"     # All commands work
```

### Build Configuration

Build metadata in `tdx.toml`:
```toml
version = "0.6.0"
description = "your todos, in markdown, done fast"
```

## License

MIT - see [LICENSE](LICENSE)

The application and build containers use Go 1.27.1. Dagger's SDK module targets Go 1.26.7, the newest toolchain supported by Dagger 0.21.9's generator. Its OpenTelemetry logging packages retain compatibility pins because the current Dagger adapter does not support the newer logging API.
