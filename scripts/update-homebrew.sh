#!/bin/bash
set -e

# Configuration
VERSION="${1:-}"
TAP_REPO="${TAP_REPO:-/Users/nheer/Projects/github.com/niklas-heer/homebrew-tap}"
FORMULA="$TAP_REPO/Formula/tdx.rb"
REPO="niklas-heer/tdx"

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 0.2.0"
    exit 1
fi

echo "Updating Homebrew formula for v${VERSION}..."

# Download binaries and calculate checksums
BASE_URL="https://github.com/${REPO}/releases/download/v${VERSION}"

echo "Downloading binaries and calculating checksums..."

DARWIN_ARM64_SHA=$(curl -fsSL "${BASE_URL}/tdx-darwin-arm64" | shasum -a 256 | cut -d' ' -f1)
echo "  darwin-arm64: $DARWIN_ARM64_SHA"

DARWIN_AMD64_SHA=$(curl -fsSL "${BASE_URL}/tdx-darwin-amd64" | shasum -a 256 | cut -d' ' -f1)
echo "  darwin-amd64: $DARWIN_AMD64_SHA"

LINUX_ARM64_SHA=$(curl -fsSL "${BASE_URL}/tdx-linux-arm64" | shasum -a 256 | cut -d' ' -f1)
echo "  linux-arm64: $LINUX_ARM64_SHA"

LINUX_AMD64_SHA=$(curl -fsSL "${BASE_URL}/tdx-linux-amd64" | shasum -a 256 | cut -d' ' -f1)
echo "  linux-amd64: $LINUX_AMD64_SHA"

# Update formula
echo "Updating formula..."

# Create updated formula using awk for reliable multi-line updates
awk -v version="$VERSION" \
    -v darwin_arm64="$DARWIN_ARM64_SHA" \
    -v darwin_amd64="$DARWIN_AMD64_SHA" \
    -v linux_arm64="$LINUX_ARM64_SHA" \
    -v linux_amd64="$LINUX_AMD64_SHA" '
/version "/ { gsub(/"[0-9]+\.[0-9]+\.[0-9]+"/, "\"" version "\"") }
/tdx-darwin-arm64/ { next_sha = "darwin_arm64" }
/tdx-darwin-amd64/ { next_sha = "darwin_amd64" }
/tdx-linux-arm64/ { next_sha = "linux_arm64" }
/tdx-linux-amd64/ { next_sha = "linux_amd64" }
/sha256/ && next_sha != "" {
    if (next_sha == "darwin_arm64") gsub(/"[a-f0-9]{64}"/, "\"" darwin_arm64 "\"")
    else if (next_sha == "darwin_amd64") gsub(/"[a-f0-9]{64}"/, "\"" darwin_amd64 "\"")
    else if (next_sha == "linux_arm64") gsub(/"[a-f0-9]{64}"/, "\"" linux_arm64 "\"")
    else if (next_sha == "linux_amd64") gsub(/"[a-f0-9]{64}"/, "\"" linux_amd64 "\"")
    next_sha = ""
}
{ print }
' "$FORMULA" > "$FORMULA.tmp" && mv "$FORMULA.tmp" "$FORMULA"

echo "Formula updated!"
echo ""
echo "Next steps:"
echo "  cd $TAP_REPO"
echo "  git add -A && git commit -m 'tdx ${VERSION}'"
echo "  git push"
