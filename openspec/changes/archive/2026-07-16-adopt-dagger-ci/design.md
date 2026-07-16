## Context

tdx currently implements linting, tests, coverage, cross-platform builds, release packaging, and Homebrew updates directly in GitHub Actions YAML. This worked, but it duplicates setup and build logic and prevents developers from exercising the complete portable path locally.

The saving subsystem intentionally depends on native filesystem semantics. A container-only pipeline cannot validate Windows replacement behavior or macOS filesystem behavior, so portability must not come at the cost of platform coverage.

## Goals / Non-Goals

**Goals:**
- Make portable CI and release builds executable locally with one command.
- Implement pipeline logic in Go and keep GitHub Actions as a thin orchestrator.
- Preserve race, coverage, lint, release artifact, and native filesystem coverage.
- Remove Node.js 20 action-runtime warnings from maintained workflows.
- Pin tool versions for reproducible local and hosted execution.

**Non-Goals:**
- Emulate macOS or Windows inside Linux containers.
- Move GitHub release creation or cross-repository Homebrew pushes into Dagger.
- Require Dagger Cloud or add a new hosted CI provider.
- Rewrite the release-note generator or Nix vendor-hash workflow in Go.

## Decisions

### Use a separate Go module for Dagger

The Dagger code SHALL live in its generated module rather than adding Dagger SDK dependencies to the application `go.mod`. This keeps the shipped dependency graph and Nix vendor hash independent from CI implementation dependencies while still using Go for both.

The Dagger engine and CLI SHALL be pinned to `v0.21.7`. GitHub SHALL invoke it through `dagger/dagger-for-github@v8.4.1` with the same explicit version.

### Make Dagger the portable execution boundary

The Dagger module SHALL expose focused functions for linting, testing, coverage, normal builds, and release artifacts, plus a single CI function that evaluates all portable checks. Source inputs SHALL exclude `.git`, local build outputs, and other irrelevant paths so cache keys remain stable.

The toolchain SHALL use Go 1.25.x as declared by the application module and `golangci-lint` 2.6.2. Container images SHALL be pinned rather than tracking floating `latest` tags.

### Retain native platform verification

GitHub-hosted macOS and Windows jobs SHALL continue to build the application, run `internal/markdown` and `internal/versioning` tests, and execute the native binary. These jobs SHALL use Node.js 24-compatible first-party actions. Linux behavior, race detection, vet, formatting, lint, and coverage SHALL run through Dagger.

### Cross-build release artifacts in one Dagger operation

Dagger SHALL generate the existing five release asset names:

- `tdx-darwin-amd64`
- `tdx-darwin-arm64`
- `tdx-linux-amd64`
- `tdx-linux-arm64`
- `tdx-windows-amd64.exe`

The version and description SHALL still come from `tdx.toml` and be injected with the existing linker variables. GitHub Actions SHALL export the returned Dagger directory and pass it to the release action. This makes release compilation locally testable without pretending that GitHub token operations are portable.

### Keep publishing operations GitHub-specific

Release creation, release-note secret handling, coverage badge commits, and Homebrew tap commits require GitHub credentials and event context. They SHALL remain small, explicit workflow steps outside Dagger. Dagger functions SHALL not receive repository write tokens.

### Validate workflow YAML locally

The portable CI function SHALL run `actionlint` against maintained workflow files. This checks workflow syntax and expressions locally; it does not claim to emulate GitHub-hosted runners or third-party actions.

## Risks / Trade-offs

- Dagger adds a Docker-compatible runtime requirement for full local CI. Mitigation: individual Go commands remain usable, and GitHub native jobs do not depend on local developer setup.
- A separate Go module adds generated SDK files. Mitigation: keep it isolated and verify generated-code drift in CI.
- A single Linux cross-build no longer compiles release binaries on native hosts. Mitigation: the project is pure Go for release targets, and native CI jobs still compile and execute macOS and Windows binaries.
- Dagger cache behavior differs between local Docker and ephemeral GitHub runners. Mitigation: correctness never depends on cache availability.

## Migration Plan

1. Add and locally validate the pinned Dagger Go module.
2. Add `just` entry points and Dagger module tests.
3. Replace portable CI jobs while retaining native macOS and Windows jobs.
4. Replace release compilation with Dagger and update action majors.
5. Run the complete Dagger pipeline locally and on the pull request.
6. Exercise release artifact generation locally and compare names/version metadata with the existing release contract.

Rollback is limited to restoring the previous workflow YAML because application and release artifact formats do not change.

## Open Questions

None. Dagger Cloud remains optional and is deliberately excluded from this change.
