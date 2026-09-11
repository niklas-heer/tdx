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
    [[ "$TDX_VERSION" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo 'TDX_VERSION must be a release version such as 0.13.1.' >&2; exit 1; }
    release_path="download/v${TDX_VERSION#v}"
fi
tmp_dir=$(mktemp -d)
staged=""
trap 'rm -rf "$tmp_dir"; [[ -z "$staged" ]] || rm -f "$staged"' EXIT
printf 'Downloading %s...\n' "$ARTIFACT"
curl --fail --show-error --silent --location --retry 3 --connect-timeout 15 --max-time 120 \
    "https://github.com/${REPO}/releases/${release_path}/${ARTIFACT}" -o "$tmp_dir/$BINARY"
curl --fail --show-error --silent --location --retry 3 --connect-timeout 15 --max-time 120 \
    "https://github.com/${REPO}/releases/${release_path}/SHA256SUMS" -o "$tmp_dir/SHA256SUMS"
expected=$(awk -v artifact="$ARTIFACT" '$2 == artifact { print $1; count++ } END { if (count != 1) exit 1 }' "$tmp_dir/SHA256SUMS") || {
    echo "Checksum manifest must contain exactly one entry for $ARTIFACT." >&2; exit 1;
}
[[ "$expected" =~ ^[[:xdigit:]]{64}$ ]] || { echo 'Invalid SHA-256 checksum.' >&2; exit 1; }
if command -v sha256sum >/dev/null 2>&1; then
    actual=$(sha256sum "$tmp_dir/$BINARY")
elif command -v shasum >/dev/null 2>&1; then
    actual=$(shasum -a 256 "$tmp_dir/$BINARY")
else
    echo 'SHA-256 verification requires sha256sum or shasum.' >&2; exit 1
fi
[[ "${actual%% *}" == "$expected" ]] || { echo 'SHA-256 verification failed; installation unchanged.' >&2; exit 1; }
# Stage on the destination filesystem so replacement is a single rename.
mkdir -p "$INSTALL_DIR"
staged=$(mktemp "$INSTALL_DIR/.tdx-install.XXXXXX")
install -m 755 "$tmp_dir/$BINARY" "$staged"
mv -f "$staged" "$INSTALL_DIR/$BINARY"
staged=""
printf 'Installed %s/%s\n' "$INSTALL_DIR" "$BINARY"
case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *) printf 'Add %s to your PATH to run tdx.\n' "$INSTALL_DIR" ;;
esac
printf 'Run tdx to get started; press ? for help and s for sections.\n'
