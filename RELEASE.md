# Releasing tdx

Update `version` in `tdx.toml` in a reviewed pull request. Merge the release changes and wait for CI to pass before publishing.

```bash
git switch main
git pull --ff-only
mise run release
```

The release task verifies a clean checkout of `main`, checks that local and remote commits match, runs local validation, and asks before pushing the single version tag. It uses the version already configured in `tdx.toml` and does not increment it automatically.

The tag workflow checks the version, builds five platform binaries with Dagger, generates release notes, creates the GitHub release, and updates the Homebrew tap. The application and Dagger module use the same pinned Go version.

To inspect release artifacts without publishing:

```bash
mise run release-artifacts
ls dist/
```

For a specific existing release, users can set `TDX_VERSION=1.0.0` when running the install script. `TDX_INSTALL_DIR` overrides its default `~/.local/bin` destination.
