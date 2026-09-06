## Why
Users with small lists need a lightweight way to choose work they are ready to start. Rezero supports a complete bottom-to-top review followed by a selected work batch, while retaining tdx's inline terminal context.

## What Changes
- Add optional :rezero review/work/round-complete states, separate readiness dots, a compact status line and inline continuation input.
- Review all open tasks in file order, bypassing filters/folds temporarily, retaining completed entries dimmed. New tasks and continuations join the next round.
- Add source-preserving append and subtree continuation actions: retain notes, nested tasks, frontmatter and surrounding bytes. Retire the old subtree and append its continuation in one guarded save and undo entry.
- Re-entry promotes a nested task to a root task at the end of the file; its descendants retain relative nesting and completion states. Existing sections are preserved; the appended task belongs to the last section.
- Reset ephemeral readiness state on document reload or structural changes outside the mode. Prevent sorting/filter changes from silently changing an active round. Support read-only review, cancellation, undo and explicit conflict recovery.
- Document the method, controls and boundaries; add unit, integration and actual terminal tests.

## Approval
The user approved the concrete Rezero interaction described in the preceding response with “Okay then let's implement all of that”. This change records that approved scope; the broader backlog (raw Markdown editor, general Vim editing and other rendering features) remains separate.

## Impact
- Affected specs: rezero-mode (new).
- Affected code: internal/markdown, internal/editor, internal/tui, terminal contracts and README.
