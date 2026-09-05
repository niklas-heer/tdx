#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
race=false
if [[ "${1:-}" == "--race" ]]; then
    race=true
    shift
fi
if [[ "$#" -eq 0 ]]; then
    set -- ./...
fi
if [[ "$race" == true ]]; then
    exec go test -race "$@"
fi
exec go test "$@"
