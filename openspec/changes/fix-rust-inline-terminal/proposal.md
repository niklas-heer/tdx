## Why
The Rust TUI hides preceding terminal output and puts task input in a detached bottom panel, regressing the Go interaction.

## What Changes
- Restore a compact, normal-buffer TUI that preserves earlier terminal context, including on resize.
- Edit task text in place and create tasks beside the selected task or at the append position.
- Keep overlays and the full Markdown editor available within the managed terminal region.

## Authorization
The user explicitly requested this restoration of the Go experience. This is a parity bug fix within the approved rewrite scope.

## Impact
- Affected specs: tdx-cli
- Affected code: Rust terminal lifecycle, task rendering, native terminal regression checks
