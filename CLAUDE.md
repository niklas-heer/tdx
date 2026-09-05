# CLAUDE.md

## Project Overview

tdx is a fast, single-binary CLI todo manager with vim-style navigation and an interactive TUI. Todos are stored in plain markdown files.

## Build & Test Commands

```bash
mise run build          # Build binary to ./tdx
mise run install        # Install to ~/.local/bin
mise run test           # Run all tests
mise run ci-lint        # Run the pinned linter through Dagger
mise run test:usage     # 100 simulated hours; replay artifacts in dist/usage
mise run test:terminal  # Real PTY contracts (macOS/Linux)
go test ./...          # Run tests directly
go test -v ./cmd/tdx -run "TestName"  # Run specific test
```

## Architecture

### Directory Structure

- `cmd/tdx/` - Main application entry point, CLI routing, user config, themes
- `internal/tui/` - Bubble Tea TUI (model, update, view, commands, render)
- `internal/markdown/` - AST-based markdown parser/serializer using Goldmark
- `internal/config/` - Recent files tracking (legacy YAML config deprecated)
- `internal/editor/` - Shared actions and bounded undo with provisional snapshots
- `internal/usage/` - Independent task oracle and deterministic replay
- `internal/versioning/` - SQLite history and safe restores
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
- Config, styles and store callbacks are injected per application instance through `tui.Runtime` and `tui.New`

**Markdown handling**: AST-based, not regex
- Goldmark parses markdown into AST
- Checkbox-only changes preserve source bytes; structural edits use a serializer that can normalize formatting
- CLI and TUI apply document operations through `internal/editor`

### Testing

**Testing is critical for this project. Always add tests for new features and bug fixes.**

- Test files use `runPiped(t, filePath, keystrokes)` helper to simulate TUI interaction
- Prefer per-instance stores and temporary directories; subprocess tests isolate `XDG_CONFIG_HOME`
- Bounded model replay checks task state and disk contents; PTY tests exercise the real event loop
- See `experiments/usage/README.md` for replay, profiling and rewrite-contract limitations

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
