# tdx development tasks

This file is both the task reference and the executable configuration for [Mask](https://github.com/jacobdeichert/mask). Run `mask --help` to list commands or `mask <command> --help` for command-specific usage.

## build

> Build the local tdx binary with metadata from `tdx.toml`.

```bash
set -euo pipefail
version=$(grep '^version' tdx.toml | cut -d'"' -f2)
description=$(grep '^description' tdx.toml | cut -d'"' -f2)
go build -ldflags "-X main.Version=$version -X 'main.Description=$description'" -o tdx ./cmd/tdx
printf 'Built tdx v%s\n' "$version"
```

## build-all

> Build all release targets with Dagger.

```bash
$MASK release-artifacts
```

## install

> Build and install tdx to `/usr/local/bin` using sudo.

```bash
set -euo pipefail
$MASK build
sudo mv tdx /usr/local/bin/tdx
sudo chmod +x /usr/local/bin/tdx
printf 'Installed tdx to /usr/local/bin\n'
```

## uninstall

> Remove tdx from `/usr/local/bin` using sudo.

```bash
set -euo pipefail
sudo rm -f /usr/local/bin/tdx
printf 'Uninstalled tdx\n'
```

## dev [args]

> Build and run tdx in development mode; pass multiple arguments as one quoted string.

```bash
set -euo pipefail
$MASK build
./tdx ${args:-}
```

## tui [args]

> Build and launch the interactive TUI; pass multiple arguments as one quoted string.

```bash
set -euo pipefail
$MASK build
./tdx ${args:-}
```

## help

> Build tdx and show its CLI help.

```bash
set -euo pipefail
$MASK build
./tdx help
```

## list

> Build tdx and list all todos.

```bash
set -euo pipefail
$MASK build
./tdx list
```

## add (text)

> Build tdx and add a todo.

```bash
set -euo pipefail
$MASK build
./tdx add "$text"
```

## toggle (index)

> Build tdx and toggle a todo by index.

```bash
set -euo pipefail
$MASK build
./tdx toggle "$index"
```

## edit (index) (text)

> Build tdx and edit a todo by index.

```bash
set -euo pipefail
$MASK build
./tdx edit "$index" "$text"
```

## delete (index)

> Build tdx and delete a todo by index.

```bash
set -euo pipefail
$MASK build
./tdx delete "$index"
```

## show-file

> Display `todo.md` when it exists.

```bash
if [[ -f todo.md ]]; then
    cat todo.md
else
    printf 'No todo.md file yet\n'
fi
```

## check

> Run `go vet` across the application module.

```bash
go vet ./...
```

## test

> Run all application tests.

```bash
go test ./...
```

## ci

> Run the complete portable CI pipeline with Dagger.

```bash
dagger call ci --source=.
```

## ci-lint

> Run the pinned linter with Dagger.

```bash
dagger call lint --source=.
```

## ci-test

> Run race-enabled tests with Dagger.

```bash
dagger call test --source=.
```

## ci-workflows

> Validate maintained GitHub Actions workflows with Dagger.

```bash
dagger call actionlint --source=.
```

## release-artifacts

> Export the five release binaries to `dist/` with Dagger.

```bash
dagger call release-artifacts --source=. export --path=dist --wipe
```

## fmt

> Format application Go code.

```bash
go fmt ./...
```

## clean

> Remove local build and release artifacts.

```bash
rm -f tdx
rm -rf dist/
```

## status

> Show the concise Git worktree status.

```bash
git status --short
```

## commit (message)

> Stage all changes and create a conventional commit.

```bash
set -euo pipefail
git add -A
git commit -m "$message"
```

## release

> Create a release using the repository release script.

```bash
./scripts/release.sh
```

## update-homebrew (version)

> Update the Homebrew formula for a released version.

```bash
./scripts/update-homebrew.sh "$version"
```

## l

> Shortcut for `mask list`.

```bash
$MASK list
```

## a (text)

> Shortcut for `mask add`.

```bash
$MASK add "$text"
```

## t (index)

> Shortcut for `mask toggle`.

```bash
$MASK toggle "$index"
```

## e (index) (text)

> Shortcut for `mask edit`.

```bash
$MASK edit "$index" "$text"
```

## d (index)

> Shortcut for `mask delete`.

```bash
$MASK delete "$index"
```
