#!/bin/bash
set -e

REPO="niklas-heer/tdx"
INSTALL_DIR="/usr/local/bin"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    darwin) OS="darwin" ;;
    linux) OS="linux" ;;
    mingw*|msys*|cygwin*) OS="windows" ;;
    *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Build artifact name
if [ "$OS" = "windows" ]; then
    ARTIFACT="tdx-${OS}-${ARCH}.exe"
    BINARY="tdx.exe"
else
    ARTIFACT="tdx-${OS}-${ARCH}"
    BINARY="tdx"
fi

echo "Detected: ${OS}-${ARCH}"
echo "Downloading ${ARTIFACT}..."

# Get latest release URL
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${ARTIFACT}"

# Download
curl -fsSL "$DOWNLOAD_URL" -o "$BINARY"
chmod +x "$BINARY"

# Install
if [ -w "$INSTALL_DIR" ]; then
    mv "$BINARY" "$INSTALL_DIR/$BINARY"
else
    echo "Installing to $INSTALL_DIR (requires sudo)..."
    sudo mv "$BINARY" "$INSTALL_DIR/$BINARY"
fi

echo "Installed tdx to $INSTALL_DIR/$BINARY"
echo "Run 'tdx' to get started!"
