# Release Process

## Creating a Release

### 1. Create the release

```bash
just release
```

This will:
- Show current version from `package.json`
- Prompt to select major/minor/patch
- Update `package.json` with new version
- Commit the version bump
- Create and push the git tag

### 2. Wait for GitHub Actions

The release workflow will automatically:
- Build binaries for all platforms (macOS, Linux, Windows)
- Generate changelog from conventional commits
- Create GitHub Release with all binaries attached

### 3. Update Homebrew formula

After the GitHub Actions workflow completes:

```bash
just update-homebrew 0.2.0
```

This downloads all binaries and updates the formula with correct SHA256 checksums.

### 4. Commit and push the tap repo

```bash
cd /Users/nheer/Projects/github.com/niklas-heer/homebrew-tap
git add -A && git commit -m "tdx 0.2.0"
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
| macOS | Intel | `tdx-darwin-x64` |
| Linux | x64 | `tdx-linux-x64` |
| Linux | ARM64 | `tdx-linux-arm64` |
| Windows | x64 | `tdx-windows-x64.exe` |
