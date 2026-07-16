# Release Process

## Creating a Release

### 1. Run the release script

```bash
mask release
```

This interactive script will:
- Show current version from `tdx.toml`
- Ask you to choose: Major, Minor, or Patch
- Calculate the next version number
- Update `tdx.toml`
- Commit and tag the release
- Push to GitHub

### 2. That's it! 🎉

Everything else is **fully automated**. GitHub Actions will:

#### Build Workflow (`release.yml`)
- ✅ Build binaries for all platforms (macOS, Linux, Windows)
- ✅ Generate AI-powered release notes with Claude Haiku 4.5
- ✅ Create GitHub Release with all binaries attached

#### Homebrew Tap Workflow (`update-homebrew.yml`)
- ✅ Download all release binaries
- ✅ Calculate SHA256 checksums
- ✅ Update `homebrew-tap/Formula/tdx.rb` automatically
- ✅ Commit and push to tap repository

#### CI Workflow (`ci.yml`)
- ✅ Run tests and linting on all PRs
- ✅ Verify builds on all platforms

### Monitoring the Release

Watch the workflows at:
- https://github.com/niklas-heer/tdx/actions

Within minutes, users can install with:
```bash
brew upgrade niklas-heer/tap/tdx
```

## Conventional Commits

Use these prefixes for better AI-generated release notes:

- `feat:` - New features → **✨ Features** section
- `fix:` - Bug fixes → **🐛 Bug Fixes** section
- `docs:` - Documentation → **📚 Documentation** section
- `chore:` - Maintenance → **⚙️ Maintenance** section
- `refactor:` - Code improvements → **🔧 Improvements** section

**Pro tip:** Add detailed commit bodies! The AI uses them to write richer release notes.

Examples:
```bash
# Good - with detailed body
git commit -m "feat: add dark mode support" -m "- Toggle with :dark-mode command
- Persists user preference
- Works with all color schemes"

# Also good - simple commit
git commit -m "fix: resolve crash on startup"
git commit -m "docs: update installation guide"
```

## Build Artifacts

The release workflow builds these binaries:

| Platform | Architecture | Artifact |
|----------|--------------|----------|
| macOS | Apple Silicon | `tdx-darwin-arm64` |
| macOS | Intel | `tdx-darwin-amd64` |
| Linux | x64 | `tdx-linux-amd64` |
| Linux | ARM64 | `tdx-linux-arm64` |
| Windows | x64 | `tdx-windows-amd64.exe` |

## Local Build

To build locally for all platforms:

```bash
mask build-all
```

Binaries will be in the `dist/` directory.
