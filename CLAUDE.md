# CLAUDE.md

## Project Overview

tdx is a fast, single-binary CLI todo manager with vim-style navigation and an interactive TUI. Todos are stored in plain markdown files.

## Build & Test Commands

```bash
just build          # Build binary to ./tdx
just install        # Install to /usr/local/bin
just test           # Run all tests
just lint           # Run golangci-lint
go test ./...       # Run tests directly
go test -v ./cmd/tdx -run "TestName"  # Run specific test
```

## Architecture

### Directory Structure

- `cmd/tdx/` - Main application entry point, CLI routing, user config, themes
- `internal/tui/` - Bubble Tea TUI (model, update, view, commands, render)
- `internal/markdown/` - AST-based markdown parser/serializer using Goldmark
- `internal/config/` - Recent files tracking (legacy YAML config deprecated)
- `internal/cmd/` - CLI command handlers (list, add, toggle, edit, delete)
- `internal/util/` - Text utilities, fuzzy search, clipboard

### Key Patterns

**Configuration hierarchy** (highest to lowest priority):
1. CLI flags (`-r`, `--show-headings`, `-m`)
2. YAML frontmatter in todo files
3. Global config (`~/.config/tdx/config.toml`)
4. Built-in defaults

**TUI architecture**: Uses Bubble Tea framework
- `model.go` - State management
- `update.go` - Event/message handling
- `view.go` - Rendering
- Config and styles are injected via package-level globals (`tui.Config`, `tui.StyleFuncs`)

**Markdown handling**: AST-based, not regex
- Goldmark parses markdown into AST
- Custom serializer reconstructs markdown preserving formatting
- Operations (toggle, add, delete, swap) manipulate AST nodes directly

### Testing

**Testing is critical for this project. Always add tests for new features and bug fixes.**

- Test files use `runPiped(t, filePath, keystrokes)` helper to simulate TUI interaction
- Use `config.SetConfigDirForTesting(tmpDir)` to isolate config in tests
- TUI tests send key sequences and verify output contains expected strings

**TUI tests are especially important** - they verify the interactive behavior users experience. Add TUI tests when:
- Adding new keybindings or commands
- Modifying navigation or filtering behavior
- Changing how content is displayed
- Adding new overlays or modes

## Code Style

- No emojis in code/comments unless user requests
- Prefer editing existing files over creating new ones
- Use conventional commits: `feat:`, `fix:`, `test:`, `docs:`, `chore:`
- Keep changes minimal and focused

## Common Tasks

**Adding a new CLI command**: Edit `cmd/tdx/main.go` switch statement, add handler function

**Adding a new TUI command**: Edit `internal/tui/commands.go`, add to `commands` slice

**Adding a new theme**: Create `cmd/tdx/themes/<name>.toml` with `[theme]` and `[colors]` sections

**Modifying config**: Update `UserConfig` struct in `cmd/tdx/userconfig.go`, update `LoadConfig()` and `DefaultConfig()`
