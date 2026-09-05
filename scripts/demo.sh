#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
demo_dir=$(mktemp -d "${TMPDIR:-/tmp}/tdx-demo.XXXXXX")
trap 'rm -rf "$demo_dir"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
mkdir -p "$demo_dir/config/tdx"
# A real (empty) config prevents fallback to personal settings.
: > "$demo_dir/config/tdx/config.toml"
cp examples/project-tracker.md "$demo_dir/todo.md"
printf 'Disposable demo: edits and history will be discarded on exit.\n' >&2
XDG_CONFIG_HOME="$demo_dir/config" ./tdx --file "$demo_dir/todo.md" "$@"
