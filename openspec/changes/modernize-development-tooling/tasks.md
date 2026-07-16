## 1. Toolchain and dependencies

- [ ] 1.1 Update both Go module directives and the pinned Dagger build image to Go 1.26.5.
- [ ] 1.2 Update compatible direct application dependencies and tidy the application module.
- [ ] 1.3 Verify the Dagger SDK-managed module remains reproducible without unrelated dependency upgrades.

## 2. Mask migration

- [ ] 2.1 Add a root `maskfile.md` preserving all maintained Just tasks and shortcuts.
- [ ] 2.2 Remove `justfile` and replace Just with Mask in the Nix development shell.
- [ ] 2.3 Update README, release, contributor, and build-comment references with Mask installation and commands.
- [ ] 2.4 Validate Mask parsing, help output, argument handling, build, test, and Dagger task entry points.

## 3. Coverage

- [ ] 3.1 Add tests for successful list, add, toggle, edit, and delete behavior.
- [ ] 3.2 Add tests for successful command dispatch and empty-list output.
- [ ] 3.3 Measure and report package and repository coverage improvement.

## 4. Verification

- [ ] 4.1 Run formatting, vet, lint, race tests, coverage generation, and workflow linting through Dagger.
- [ ] 4.2 Export release artifacts and verify the Linux executable metadata.
- [ ] 4.3 Run native filesystem tests on the host.
- [ ] 4.4 Confirm hosted Linux, macOS, Windows, and CodeRabbit checks pass.
