#!/usr/bin/env bash
# Verify release-note selection offline, without credentials or dependencies.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
exec python3 -m unittest discover -s .github/scripts -p 'test_*.py'
