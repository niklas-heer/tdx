#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
version=$(sed -n 's/^version = "\(.*\)"/\1/p' tdx.toml)
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo 'Invalid version in tdx.toml' >&2; exit 1; }
[[ -z "$(git status --porcelain)" ]] || { echo 'Commit or stash work before releasing.' >&2; exit 1; }
[[ "$(git branch --show-current)" == "main" ]] || { echo 'Release from main after CI passes.' >&2; exit 1; }
git fetch origin main --tags
[[ "$(git rev-parse HEAD)" == "$(git rev-parse origin/main)" ]] || { echo 'Local main must match origin/main.' >&2; exit 1; }
if git rev-parse "refs/tags/v$version" >/dev/null 2>&1; then
    echo "Tag v$version already exists." >&2
    exit 1
fi
mise run check
printf 'Publish v%s from %s? [y/N] ' "$version" "$(git rev-parse --short HEAD)"
read -r confirm
[[ "$confirm" == "y" || "$confirm" == "Y" ]] || exit 0
git tag -a "v$version" -m "Release v$version"
git push origin "refs/tags/v$version"
printf 'Tag v%s pushed. GitHub Actions will build and publish the release.\n' "$version"
