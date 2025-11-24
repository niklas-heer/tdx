# tdx Justfile - Command runner for development tasks
# Usage: just [command]
# Run 'just --list' to see all available commands

set shell := ["bash", "-c"]

# Display available commands
@default:
    just --list

# ============================================================================
# BUILDING
# ============================================================================

# Build the binary
build:
    #!/bin/bash
    VERSION=$(grep '^version' tdx.toml | cut -d'"' -f2)
    DESCRIPTION=$(grep '^description' tdx.toml | cut -d'"' -f2)
    go build -ldflags "-X main.Version=$VERSION -X 'main.Description=$DESCRIPTION'" -o tdx ./cmd/tdx
    echo "✓ Built tdx v$VERSION"

# Build for all platforms
build-all:
    #!/bin/bash
    VERSION=$(grep '^version' tdx.toml | cut -d'"' -f2)
    DESCRIPTION=$(grep '^description' tdx.toml | cut -d'"' -f2)
    LDFLAGS="-X main.Version=$VERSION -X 'main.Description=$DESCRIPTION'"

    mkdir -p dist

    echo "Building for all platforms..."
    GOOS=darwin GOARCH=arm64 go build -ldflags "$LDFLAGS" -o dist/tdx-darwin-arm64 ./cmd/tdx
    GOOS=darwin GOARCH=amd64 go build -ldflags "$LDFLAGS" -o dist/tdx-darwin-amd64 ./cmd/tdx
    GOOS=linux GOARCH=amd64 go build -ldflags "$LDFLAGS" -o dist/tdx-linux-amd64 ./cmd/tdx
    GOOS=linux GOARCH=arm64 go build -ldflags "$LDFLAGS" -o dist/tdx-linux-arm64 ./cmd/tdx
    GOOS=windows GOARCH=amd64 go build -ldflags "$LDFLAGS" -o dist/tdx-windows-amd64.exe ./cmd/tdx

    echo "✓ Built binaries in dist/"
    ls -lh dist/

# Install binary to /usr/local/bin (requires sudo)
install: build
    #!/bin/bash
    sudo mv tdx /usr/local/bin/tdx
    sudo chmod +x /usr/local/bin/tdx
    echo "✓ Installed tdx to /usr/local/bin"

# Uninstall binary from /usr/local/bin
uninstall:
    #!/bin/bash
    sudo rm -f /usr/local/bin/tdx
    echo "✓ Uninstalled tdx"

# ============================================================================
# DEVELOPMENT & RUNNING
# ============================================================================

# Run in development mode (pass args after --)
dev *ARGS: build
    ./tdx {{ARGS}}

# Launch interactive TUI (pass flags after --)
tui *ARGS: build
    ./tdx {{ARGS}}

# Show help message
help: build
    ./tdx help

# ============================================================================
# TODO COMMANDS
# ============================================================================

# List all todos
list: build
    ./tdx list

# Add a new todo - usage: just add "Your todo text"
add TEXT: build
    ./tdx add "{{TEXT}}"

# Toggle a todo by index - usage: just toggle 1
toggle INDEX: build
    ./tdx toggle {{INDEX}}

# Edit a todo - usage: just edit 1 "New text"
edit INDEX TEXT: build
    ./tdx edit {{INDEX}} "{{TEXT}}"

# Delete a todo - usage: just delete 1
delete INDEX: build
    ./tdx delete {{INDEX}}

# Display todo.md file content
show-file:
    #!/bin/bash
    if [ -f todo.md ]; then
        cat todo.md
    else
        echo "No todo.md file yet"
    fi

# ============================================================================
# CODE QUALITY
# ============================================================================

# Run go vet
check:
    go vet ./...

# Run tests
test:
    go test ./...

# Format code
fmt:
    go fmt ./...

# ============================================================================
# MAINTENANCE
# ============================================================================

# Clean build artifacts
clean:
    rm -f tdx
    rm -rf dist/

# Show git status
status:
    @git status --short

# Git commit with conventional commit - usage: just commit "feat: description"
commit MESSAGE:
    git add -A && git commit -m "{{MESSAGE}}"

# Create a new release
release:
    @./scripts/release.sh

# Update Homebrew formula after release - usage: just update-homebrew 0.2.0
update-homebrew VERSION:
    @./scripts/update-homebrew.sh {{VERSION}}

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

# Shortcut: just d 1 = just delete 1
d INDEX:
    @just delete {{INDEX}}
