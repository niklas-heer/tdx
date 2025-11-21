# tdx Justfile - Command runner for development tasks
# Usage: just [command]
# Run 'just --list' to see all available commands

set shell := ["bash", "-c"]

# Display available commands
@default:
    just --list

# ============================================================================
# DEPENDENCIES & SETUP
# ============================================================================

# Install dependencies
install:
    bun install

# Development setup - install and verify
dev-setup: install check
    @echo "Development environment ready!"
    @echo ""
    @echo "Next steps:"
    @echo "  just dev          - Run CLI in development mode"
    @echo "  just build        - Build standalone binary"
    @echo "  just tui          - Launch interactive todo manager"

# ============================================================================
# DEVELOPMENT & RUNNING
# ============================================================================

# Run CLI in development mode (pass args after --)
dev *ARGS:
    bun run src/cli.ts {{ARGS}}

# Launch interactive TUI
tui:
    bun run src/cli.ts

# Show help message
help:
    bun run src/cli.ts help

# ============================================================================
# TODO COMMANDS
# ============================================================================

# List all todos
list:
    bun run src/cli.ts list

# Add a new todo - usage: just add "Your todo text"
add TEXT:
    bun run src/cli.ts add "{{TEXT}}"

# Toggle a todo by index - usage: just toggle 1
toggle INDEX:
    bun run src/cli.ts toggle {{INDEX}}

# Edit a todo - usage: just edit 1 "New text"
edit INDEX TEXT:
    bun run src/cli.ts edit {{INDEX}} "{{TEXT}}"

# Display todo.md file content
show-file:
    #!/bin/bash
    if [ -f todo.md ]; then
        echo "Current todo.md:"
        cat todo.md
    else
        echo "No todo.md file yet"
    fi

# ============================================================================
# BUILDING & INSTALLATION
# ============================================================================

# Build standalone binary
build:
    #!/bin/bash
    echo "Building tdx binary..."
    bun build --compile --minify src/cli.ts --outfile tdx
    if [ -f tdx ]; then
        echo "✓ Binary built successfully"
        ls -lh tdx
    fi

# Install binary to /usr/local/bin (requires sudo)
install-bin: build
    #!/bin/bash
    echo "Installing tdx to /usr/local/bin..."
    sudo mv tdx /usr/local/bin/tdx
    sudo chmod +x /usr/local/bin/tdx
    echo "✓ Installed! Run 'tdx' from anywhere"

# Uninstall binary from /usr/local/bin
uninstall-bin:
    #!/bin/bash
    echo "Removing tdx from /usr/local/bin..."
    sudo rm -f /usr/local/bin/tdx
    echo "✓ Uninstalled"

# ============================================================================
# CODE QUALITY
# ============================================================================

# Type check with TypeScript compiler
check:
    bunx tsc --noEmit

# All quality checks
check-all: check
    #!/bin/bash
    echo "Running quality checks..."
    echo "✓ TypeScript check passed"
    openspec validate add-bun-tdx-cli && echo "✓ OpenSpec validation passed"

# ============================================================================
# OPENSPEC WORKFLOWS
# ============================================================================

# Show OpenSpec change details
spec-show:
    openspec show add-bun-tdx-cli

# Validate OpenSpec change
spec-validate:
    openspec validate add-bun-tdx-cli

# Archive completed OpenSpec change
spec-archive:
    openspec archive add-bun-tdx-cli

# ============================================================================
# MAINTENANCE
# ============================================================================

# Clean build artifacts
clean:
    #!/bin/bash
    echo "Cleaning build artifacts..."
    rm -f tdx
    rm -rf dist/
    echo "✓ Cleaned"

# Show git status
status:
    @git status --short || echo "Not a git repository"

# Git commit with conventional commit - usage: just commit "feat: description"
commit MESSAGE:
    git add -A && git commit -m "{{MESSAGE}}"

# Create a new release (interactive version bump, tag, and push)
release:
    @./scripts/release.sh

# Update Homebrew formula after release - usage: just update-homebrew 0.2.0
update-homebrew VERSION:
    @./scripts/update-homebrew.sh {{VERSION}}

# ============================================================================
# WORKFLOWS & DEMOS
# ============================================================================

# Show common development commands
workflow:
    #!/bin/bash
    echo "Common Development Workflows"
    echo ""
    echo "GETTING STARTED:"
    echo "  just install          Install dependencies"
    echo "  just dev-setup        Setup development environment"
    echo "  just check            Type check code"
    echo ""
    echo "RUNNING:"
    echo "  just dev              Run CLI in dev mode"
    echo "  just tui              Launch interactive todo manager"
    echo "  just help             Show CLI help"
    echo ""
    echo "MANAGING TODOS:"
    echo "  just list             List all todos"
    echo "  just add \"Task\"       Add new todo"
    echo "  just toggle 1         Toggle first todo"
    echo "  just edit 1 \"New\"     Edit first todo"
    echo ""
    echo "BUILDING:"
    echo "  just build            Build standalone binary"
    echo "  just install-bin      Install binary to PATH"
    echo "  just uninstall-bin    Remove from PATH"
    echo "  just clean            Remove build artifacts"
    echo ""
    echo "OPENSPEC:"
    echo "  just spec-show        Show change details"
    echo "  just spec-validate    Validate change"
    echo "  just spec-archive     Archive change"
    echo ""
    echo "QUALITY:"
    echo "  just check            Type check"
    echo "  just check-all        Run all checks"
    echo ""
    echo "VERSION CONTROL:"
    echo "  just status           Show git status"
    echo "  just commit \"msg\"     Commit with message"
    echo ""
    echo "SHORTCUTS:"
    echo "  just l                List todos (l = list)"
    echo "  just a \"Task\"         Add todo (a = add)"
    echo "  just t 1              Toggle todo (t = toggle)"
    echo "  just e 1 \"New\"        Edit todo (e = edit)"
    echo ""
    echo "DEMO:"
    echo "  just demo             Run demo workflow"

# Demo workflow - add some todos and show them
demo:
    #!/bin/bash
    echo "Running demo: Adding todos"
    echo ""
    just add "Buy groceries"
    sleep 0.5
    just add "Walk the dog"
    sleep 0.5
    just add "Review PR 42"
    echo ""
    echo "Current todos:"
    just list

# ============================================================================
# ALIASES & SHORTCUTS
# ============================================================================

# Shortcut: just l = just list
l: list

# Shortcut: just a "text" = just add "text"
a TEXT:
    @just add "{{TEXT}}"

# Shortcut: just t 1 = just toggle 1
t INDEX:
    @just toggle {{INDEX}}

# Shortcut: just e 1 "text" = just edit 1 "text"
e INDEX TEXT:
    @just edit {{INDEX}} "{{TEXT}}"
