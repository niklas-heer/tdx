## Why
Oversized inline frames leave old headings in terminal scrollback when task wrapping and section transitions change frame height, as reported in the user screenshot.

## What Changes
- Budget by rendered rows, preserve the active task/cursor, and constrain every terminal frame.
- Add layout regressions and verify section transitions and resizing in a real PTY.

## Impact
- Affected spec: tdx-cli.
- Affected code: internal/tui rendering and tests.
- Approved scope: the user explicitly requested fixing these visual defects and testing them.
