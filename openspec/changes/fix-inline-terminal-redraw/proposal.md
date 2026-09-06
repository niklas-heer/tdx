## Why
Closing the command palette leaves stale rows above the Rezero list. The pinned Ultraviolet renderer resizes its old buffer before moving the physical cursor, which clamps away its real position during an inline frame shrink.

## What Changes
- Pin the last compatible pre-regression Ultraviolet renderer and Lip Gloss 2.0.5; keep Bubble Tea 2.0.9. Avoid a private renderer fork or clearing shell scrollback.
- Add screen-emulated PTY regression checks for palette activation/filtering/cancellation, Rezero transitions and repeated grow/shrink cycles, retaining shell context.
- Run these checks alongside the existing terminal contracts locally and on Linux/macOS CI, with pinned test-only Python dependencies.

## Authorization
The user reported the incorrect rendering with a screenshot and asked for it to be corrected. This restores the approved inline UI behavior.

## Impact
- go.mod/go.sum, terminal tests, mise task and CI terminal step.
- The older renderer does not include later general fullscreen/grapheme rendering fixes. Validate the inline application’s Unicode and resize contracts before adopting the pin. The pin should be revisited once upstream preserves cursor coordinates during shrink.
