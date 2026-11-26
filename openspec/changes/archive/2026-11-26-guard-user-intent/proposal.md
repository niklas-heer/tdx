## Why
A council review of tdx found four places where it acts against what the user asked for: CLI commands ignore a file's own `read-only: true`, quitting in manual-save mode silently drops unsaved edits, tdx replaces write-protected files, and version history hides a return to earlier content and can prune the snapshot of the current file. Undo also discards section focus and folds even when no heading changed.

## What Changes
- CLI mutations use the same read-only precedence as the TUI: command-line flag, then the file's frontmatter, then `config.toml`.
- Quitting with unsaved changes warns once; a second quit key confirms.
- Saves refuse targets the user cannot write to instead of replacing them.
- Saving content that matches an older version moves that version to the newest position.
- Undo keeps section focus and folds when the headings are unchanged.

## Impact
- Affected specs: saved-views, tdx-cli, file-saving, file-versioning, section-management.
- No JSON output or schema change. Authorized by Niklas on 2026-11-25 after the council review of the 0.15 follow-ups.
