## Why
The Rust evaluation recommends realistic editing measurements and reducing Go extraction costs before considering a rewrite. Existing tests lack sustained, replayable sessions and checkbox saves can normalize unrelated Markdown.

## What Changes
- Add deterministic JSON usage traces, an independent task oracle, real TUI update/persistence replay, and an executable adapter for future implementation comparisons.
- Run at least 100 accelerated simulated hours with explicit think-time assumptions, action counts, seeds, failure prefixes, and actual elapsed measurements.
- Preserve source bytes for checkbox edits and undo snapshots; avoid unnecessary metadata extraction on checkbox changes.
- Fix regressions exposed by the harness and add permanent bounded CI cases, including full undo history and actual terminal use.

## Impact
- Specs: editor-core, usage-replay. Code: markdown, editor, tui, developer harness and mise tasks.
- Keeps Go, version 0.14.0 unreleased, and existing safe-save behavior. The user has explicitly authorized implementing these report recommendations and the 100-hour experiment.
