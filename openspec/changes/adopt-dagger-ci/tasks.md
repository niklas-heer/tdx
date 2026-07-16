## 1. Dagger module

- [x] 1.1 Initialize a Go Dagger module pinned to engine `v0.21.7` without changing the application module dependencies.
- [x] 1.2 Implement cached Go toolchain containers and filtered source inputs.
- [x] 1.3 Implement formatting, vet, lint, race-test, coverage, and actionlint functions.
- [x] 1.4 Implement normal build and five-target release artifact functions with linker metadata.
- [x] 1.5 Add Go unit tests for pure pipeline logic such as target naming, version parsing, and coverage badge thresholds.

## 2. Local developer interface

- [x] 2.1 Add `just ci` and focused Dagger check commands.
- [x] 2.2 Add a local release-artifact command that exports the same files used by GitHub releases.
- [x] 2.3 Document the Dagger and container-runtime prerequisites in the existing developer documentation.

## 3. GitHub Actions integration

- [x] 3.1 Replace portable CI implementation with pinned Dagger invocation.
- [x] 3.2 Retain native macOS and Windows filesystem and executable tests with Node.js 24-compatible actions.
- [x] 3.3 Preserve coverage badge generation and safe main-branch commit behavior.
- [x] 3.4 Replace the release build matrix with Dagger artifact generation while preserving release names and metadata.
- [x] 3.5 Keep GitHub release publication and Homebrew tap updates explicit and upgrade maintained first-party actions to Node.js 24-compatible versions.
- [x] 3.6 Update the Nix maintenance workflow's maintained first-party action versions without changing its behavior.

## 4. Verification

- [x] 4.1 Run Dagger module unit tests and the full portable CI pipeline locally.
- [x] 4.2 Export release artifacts locally and verify all five names and embedded version metadata where executable.
- [x] 4.3 Run native Go tests on the host and validate all workflow files with actionlint.
- [x] 4.4 Confirm the pull request passes Dagger, macOS, Windows, and CodeRabbit review.
- [x] 4.5 Confirm no maintained workflow uses a Node.js 20 first-party action generation.
