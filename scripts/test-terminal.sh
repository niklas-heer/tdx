#!/usr/bin/env bash
set -euo pipefail
python3 -m venv dist/terminal-test-env
dist/terminal-test-env/bin/python -m pip --disable-pip-version-check install -q -r scripts/terminal-requirements.txt
dist/terminal-test-env/bin/python scripts/usage-pty.py "$@"
dist/terminal-test-env/bin/python scripts/terminal-redraw.py "$@"
