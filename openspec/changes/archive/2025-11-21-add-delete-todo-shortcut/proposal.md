## Why

The tdx workflow already lets power users add, edit, toggle, and script todos without leaving their terminal, yet deleting still requires opening `todo.md` in an editor. That friction makes inbox cleanup tedious, discourages quick pruning of obsolete tasks, and forces users out of the otherwise keyboard-driven loop. Because the project encourages treating `tdx` as a “power tool” with state tracked in git, immediate deletion (without confirmation prompts) matches user expectations and keeps day-to-day task hygiene fast.

## What Changes

- **TUI behavior:** When the user presses `d`, the currently selected todo should disappear instantly, the selection should shift to a valid neighbor (or show the empty state), and the updated list should persist via the existing atomic writer.
- **CLI parity:** Introduce `tdx delete <index>` so scripts and non-interactive users can remove todos by 1-based index with the same safety checks (bounds validation, helpful errors, success messages).
- **Implementation details:** Share the markdown parser/writer so deletion only removes the targeted todo line, preserves all non-todo content byte-for-byte, and keeps indices consistent after removal.
- **Documentation + help:** Update README usage examples, CLI help output, and any shortcut listings so users learn about the `d` shortcut and the new `delete` command.
- **Validation:** Extend the manual/test plan to cover deleting the first, last, and only todo (both via CLI and TUI) to ensure state, selection, and persistence behave as specified.

## Impact

- **Specs affected:** `tdx-cli` capability gains an expanded requirement describing delete interactions for both TUI and CLI (keyboard shortcut, CLI command, scenarios for success/error).
- **Code touched:** `src/tui/App.tsx`, `src/cli.ts`, a new `src/commands/delete.ts`, shared todo model/parser/writer utilities, and help/documentation assets (README, CLI help text).
- **User experience:** Users can prune tasks without leaving `tdx`, automation scripts gain a first-class delete command, and the TUI retains its single-keystroke ergonomics with behavior aligned to the “power tool” philosophy.