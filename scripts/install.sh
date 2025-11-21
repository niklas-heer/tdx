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
    x86_64|amd64) ARCH="x64" ;;
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
echo "Downloading ${ARTIFACT}.zip..."

# Get latest release URL
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${ARTIFACT}.zip"

# Download and extract
curl -fsSL "$DOWNLOAD_URL" -o tdx.zip
unzip -q tdx.zip
rm tdx.zip

chmod +x "$ARTIFACT"

# Install
if [ -w "$INSTALL_DIR" ]; then
    mv "$ARTIFACT" "$INSTALL_DIR/$BINARY"
    mv yoga.wasm "$INSTALL_DIR/yoga.wasm"
else
    echo "Installing to $INSTALL_DIR (requires sudo)..."
    sudo mv "$ARTIFACT" "$INSTALL_DIR/$BINARY"
    sudo mv yoga.wasm "$INSTALL_DIR/yoga.wasm"
fi

echo "Installed tdx to $INSTALL_DIR/$BINARY"
echo "Run 'tdx' to get started!"
