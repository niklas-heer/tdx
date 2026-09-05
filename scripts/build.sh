#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
version=$(sed -n 's/^version = "\(.*\)"/\1/p' tdx.toml)
description=$(sed -n 's/^description = "\(.*\)"/\1/p' tdx.toml)
go build -trimpath -ldflags "-X main.Version=$version -X 'main.Description=$description'" -o tdx ./cmd/tdx
printf 'Built tdx v%s\n' "$version"
