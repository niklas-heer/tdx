## Why

The project is one stable Go release behind, several direct dependencies have compatible updates, and the current `justfile` duplicates task documentation in a format that is not readable on GitHub. The untested CLI command layer is also the largest straightforward coverage gap.

## What Changes

- Update the application and Dagger modules to Go 1.26.4 and keep the pinned Dagger build image synchronized.
- Update compatible direct application dependencies and accept only changes that pass the portable and native-focused test suites.
- Replace `justfile` with a documented `maskfile.md` while preserving the existing build, development, CI, release, maintenance, and shortcut commands.
- Update README, release, contributor, and Nix development-environment references from Just to Mask.
- Add command-layer tests for list, add, toggle, edit, delete, and command dispatch behavior.

## Impact

- Affected specs: `ci-pipeline`, new `development-tooling`
- Affected code: `go.mod`, `go.sum`, `.dagger/`, `maskfile.md`, `justfile`, `flake.nix`, developer documentation, `internal/cmd`
