## Why

The current CI and release logic is spread across GitHub Actions YAML and shell scripts, which makes most of the pipeline difficult to execute and validate locally. The release also uses action versions backed by the deprecated Node.js 20 runtime.

Moving portable checks and builds into a Dagger pipeline written in Go gives developers and CI one implementation while preserving the native macOS and Windows filesystem coverage required by tdx's saving guarantees.

## What Changes

- Add a version-pinned Dagger module written in Go for formatting, vet, lint, race tests, coverage, workflow linting, normal builds, and release artifact generation.
- Expose the portable CI and release-build operations through `just` commands that run identically on a developer machine and in GitHub Actions.
- Replace duplicated portable GitHub Actions logic with thin Dagger invocations.
- Retain native macOS and Windows jobs for filesystem, locking, and binary-execution coverage that Linux containers cannot reproduce.
- Simplify release artifact generation to one Dagger cross-build while leaving GitHub release publication, release-note secrets, and Homebrew repository writes in GitHub Actions.
- Upgrade remaining first-party GitHub Actions to Node.js 24-compatible major versions and pin the Dagger CLI/action versions.

## Impact

- Affected specs: new `ci-pipeline` capability
- Affected code: Dagger module, `Justfile`, `.github/workflows/ci.yml`, `.github/workflows/release.yml`, `.github/workflows/fix-vendor-hash.yml`
- New development dependency: Dagger CLI and a Docker-compatible container runtime
- Release behavior: artifact contents and names remain unchanged; publication still occurs on `v*` tags and still updates the Homebrew tap
- CI behavior: portable checks become locally reproducible while native platform validation remains on GitHub-hosted runners
