#!/usr/bin/env bash
set -euo pipefail

REPO="niklas-heer/tdx"
INSTALL_DIR="${TDX_INSTALL_DIR:-$HOME/.local/bin}"
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    darwin|linux) ;;
    mingw*|msys*|cygwin*) OS="windows" ;;
    *) printf 'Unsupported OS: %s\n' "$OS" >&2; exit 1 ;;
esac
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) printf 'Unsupported architecture: %s\n' "$ARCH" >&2; exit 1 ;;
esac
if [[ "$OS-$ARCH" == "windows-arm64" ]]; then
    echo 'Windows ARM64 binaries are not available; use the amd64 binary with emulation.' >&2
    exit 1
fi
BINARY="tdx"
[[ "$OS" != "windows" ]] || BINARY="tdx.exe"
ARTIFACT="tdx-${OS}-${ARCH}"
[[ "$OS" != "windows" ]] || ARTIFACT+=".exe"
release_path="latest/download"
if [[ -n "${TDX_VERSION:-}" ]]; then
    [[ "$TDX_VERSION" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo 'TDX_VERSION must be a release version such as 1.0.0.' >&2; exit 1; }
    release_path="download/v${TDX_VERSION#v}"
fi
tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT
printf 'Downloading %s...\n' "$ARTIFACT"
curl --fail --show-error --silent --location --retry 3 \
    "https://github.com/${REPO}/releases/${release_path}/${ARTIFACT}" -o "$tmp_dir/$BINARY"
mkdir -p "$INSTALL_DIR"
install -m 755 "$tmp_dir/$BINARY" "$INSTALL_DIR/$BINARY"
printf 'Installed %s/%s\n' "$INSTALL_DIR" "$BINARY"
case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *) printf 'Add %s to your PATH to run tdx.\n' "$INSTALL_DIR" ;;
esac
printf 'Run tdx to get started; press ? for help and s for sections.\n'
