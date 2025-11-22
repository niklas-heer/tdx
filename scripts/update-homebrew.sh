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

sed -i.bak "s/version \".*\"/version \"${VERSION}\"/" "$FORMULA"
sed -i.bak "s/PLACEHOLDER_DARWIN_ARM64/${DARWIN_ARM64_SHA}/" "$FORMULA"
sed -i.bak "s/PLACEHOLDER_DARWIN_AMD64/${DARWIN_AMD64_SHA}/" "$FORMULA"
sed -i.bak "s/PLACEHOLDER_LINUX_ARM64/${LINUX_ARM64_SHA}/" "$FORMULA"
sed -i.bak "s/PLACEHOLDER_LINUX_AMD64/${LINUX_AMD64_SHA}/" "$FORMULA"

# Also update existing sha256 values for subsequent releases
sed -i.bak -E "s/sha256 \"[a-f0-9]{64}\"  # darwin-arm64/sha256 \"${DARWIN_ARM64_SHA}\"  # darwin-arm64/" "$FORMULA"
sed -i.bak -E "s/sha256 \"[a-f0-9]{64}\"  # darwin-amd64/sha256 \"${DARWIN_AMD64_SHA}\"  # darwin-amd64/" "$FORMULA"
sed -i.bak -E "s/sha256 \"[a-f0-9]{64}\"  # linux-arm64/sha256 \"${LINUX_ARM64_SHA}\"  # linux-arm64/" "$FORMULA"
sed -i.bak -E "s/sha256 \"[a-f0-9]{64}\"  # linux-amd64/sha256 \"${LINUX_AMD64_SHA}\"  # linux-amd64/" "$FORMULA"

rm -f "$FORMULA.bak"

echo "Formula updated!"
echo ""
echo "Next steps:"
echo "  cd $TAP_REPO"
echo "  git add -A && git commit -m 'tdx ${VERSION}'"
echo "  git push"
