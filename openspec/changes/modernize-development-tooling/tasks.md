## 1. Toolchain and dependencies

- [x] 1.1 Update both Go module directives and the pinned Dagger build image to Go 1.26.4.
- [x] 1.2 Update compatible direct application dependencies and tidy the application module.
- [x] 1.3 Verify the Dagger SDK-managed module remains reproducible without unrelated dependency upgrades.
- [x] 1.4 Update the Nix package set and vendor hash, then verify the Nix build with Go 1.26.4.

## 2. Mask migration

- [x] 2.1 Add a root `maskfile.md` preserving all maintained Just tasks and shortcuts.
- [x] 2.2 Remove `justfile` and replace Just with Mask in the Nix development shell.
- [x] 2.3 Update README, release, contributor, and build-comment references with Mask installation and commands.
- [x] 2.4 Validate Mask parsing, help output, argument handling, build, test, and Dagger task entry points.

## 3. Coverage

- [x] 3.1 Add tests for successful list, add, toggle, edit, and delete behavior.
- [x] 3.2 Add tests for successful command dispatch and empty-list output.
- [x] 3.3 Measure and report package and repository coverage improvement.

## 4. Verification

- [x] 4.1 Run formatting, vet, lint, race tests, coverage generation, and workflow linting through Dagger.
- [x] 4.2 Export release artifacts and verify the Linux executable metadata.
- [x] 4.3 Run native filesystem tests on the host.
- [ ] 4.4 Confirm hosted Linux, macOS, Windows, and CodeRabbit checks pass.
