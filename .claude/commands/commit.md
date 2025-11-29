---
description: Create a conventional commit from staged changes.
---
Analyze all changes and create well-organized commits, grouping related changes together.

**Steps**

1. Run `git status` to see all changes (staged, unstaged, untracked)
2. Run `git diff` and `git diff --cached` to understand what changed
3. Analyze the changes and group them logically (see grouping rules below)
4. For each logical group:
   - Stage only the files belonging to that group with `git add <files>`
   - Run validation checks based on file types (see validation rules below)
   - If validation fails, fix the issues and re-stage
   - Run `git log --oneline -10` to understand commit style (first iteration only)
   - Create a conventional commit for that group
5. Repeat until all changes are committed

**Validation Rules**

Before committing any group, run validation based on the file types in that group:

**Go files (*.go)** - must match CI exactly:
1. Run `gofmt -l .` to check formatting, fix with `gofmt -w .` if needed
2. Run `go vet ./...` to catch common mistakes
3. Run `golangci-lint run --timeout=5m` for linting
4. Run `go test -v -race ./...` to ensure tests pass
5. If any issues found:
   - Fix formatting and lint issues automatically
   - Fix test failures
   - Re-stage the fixed files
   - Re-run validation to confirm fixes work

**Other languages** (extend as needed):
- **TypeScript/JavaScript**: Run `npm test`, `npm run lint` if package.json exists
- **Python**: Run `pytest`, `ruff` or `flake8` if available
- **Rust**: Run `cargo test`, `cargo clippy`

**Validation Failure Handling**:
- Formatting issues: Fix automatically and re-stage
- Lint warnings: Fix automatically where possible
- Test failures: Analyze and fix the root cause
- If a fix is complex or unclear, ask the user before proceeding

**Grouping Rules**

Group changes that belong together semantically:

- **Feature + Tests**: Implementation files with their corresponding test files
- **Refactor**: Related refactoring changes across multiple files
- **Config changes**: Build configs, CI files, dependencies (go.mod, package.json, etc.)
- **Documentation**: README, docs/, comments-only changes
- **Bug fix**: The fix and any related test additions
- **Single responsibility**: Each commit should do ONE thing

**Examples of Good Grouping**

- `feat(parser): add support for nested lists` → parser.go + parser_test.go
- `chore: update dependencies` → go.mod + go.sum
- `docs: add examples to README` → README.md
- `fix(ui): resolve cursor positioning bug` → ui.go + ui_test.go
- `refactor: extract validation logic` → multiple files touched by the refactor

**Examples of Bad Grouping**

- Mixing a bug fix with an unrelated new feature
- Combining dependency updates with code changes
- Docs changes mixed with implementation changes

**Safety**

- Before adding, check for files that should NOT be committed (secrets, .env files, large binaries)
- If suspicious files are found, warn the user and exclude them
- Respect .gitignore
- If unsure whether changes belong together, ask the user

**Conventional Commit Format**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `style`: Formatting, no code change
- `refactor`: Code restructuring without behavior change
- `test`: Adding/updating tests
- `chore`: Maintenance tasks
- `ci`: CI/CD changes
- `perf`: Performance improvements

Include breaking change footer (`BREAKING CHANGE:`) if applicable.
