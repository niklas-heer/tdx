# Release Process

## Creating a Release

### 1. Run the release script

```bash
just release
```

This interactive script will:
- Show current version from `tdx.toml`
- Ask you to choose: Major, Minor, or Patch
- Calculate the next version number
- Update `tdx.toml`
- Commit and tag the release
- Push to GitHub

### 2. Wait for GitHub Actions

The release workflow will automatically:
- Build binaries for all platforms (macOS, Linux, Windows)
- Generate changelog from conventional commits
- Create GitHub Release with all binaries attached

### 3. Update Homebrew formula

After the GitHub Actions workflow completes:

```bash
just update-homebrew 0.3.0
```

This downloads all binaries and updates the formula with correct SHA256 checksums.

### 4. Commit and push the tap repo

```bash
cd /path/to/homebrew-tap
git add -A && git commit -m "tdx 0.3.0"
git push
```

## Conventional Commits

Use these prefixes for automatic changelog categorization:

- `feat:` - New features → **Features** section
- `fix:` - Bug fixes → **Bug Fixes** section
- `docs:` - Documentation → **Documentation** section
- `chore:` - Maintenance → **Maintenance** section

Examples:
```bash
git commit -m "feat: add dark mode support"
git commit -m "fix: resolve crash on startup"
git commit -m "docs: update installation guide"
git commit -m "chore: bump dependencies"
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
just build-all
```

Binaries will be in the `dist/` directory.
