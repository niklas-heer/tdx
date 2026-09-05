# Project Context

## Purpose
tdx is a standalone terminal todo manager. Markdown files remain the user's source of truth; no account or service is required.

## Stack and structure
- Go application entry point: cmd/tdx/main.go.
- Bubble Tea v2 and Lip Gloss v2: internal/tui.
- Goldmark AST parsing and guarded atomic saves: internal/markdown.
- CLI commands: internal/cmd; user configuration: cmd/tdx/userconfig.go.
- Local SQLite version history: internal/versioning.
- mise.toml pins development tools and defines tasks; .dagger contains the separate portable CI module.

## Conventions
Use conventional commits. Keep user documentation in README.md and docs; use OpenSpec for implementation specifications. Preserve unrelated Markdown content and reject stale writes. New interactions must work with read-only files, filters, Unicode input, empty documents, and external file changes.

## Validation
Run mise run check for local checks, mise run test:race for race detection, and mise run ci for the portable container pipeline. GitHub also validates native macOS and Windows behavior. Interactive tests use the piped harness with isolated configuration; terminal input changes additionally need a real PTY smoke test.

## Toolchain constraints
The application and build container use Go 1.27.1. Dagger 0.21.9's generator supports Go 1.26.7 for its isolated module. The current Dagger telemetry adapter requires OpenTelemetry logging v0.16 compatibility pins. These exceptions are intentional until upstream supports the newer APIs.
