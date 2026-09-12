## Context
v0.14.0 supplies a shared editor, guarded saves, source-preserving checkbox updates, a JSON CLI, bounded undo and replay/PTY contracts. Build on those boundaries without a language rewrite.

## Decisions
- Keep unchanged source bytes authoritative; source patches are preferred for localized edits. Reject unsafe unsupported edits without mutating the document.
- Store named views per canonical file outside Markdown, combining section focus/folds and existing filters; restore only explicitly or through an opt-in setting. View commands must be discoverable and cancellable.
- Retain read-only aliases while explaining the TUI behavior as manual save. CLI read-only writes remain rejected.
- Keep existing list JSON arrays stable; opt-in revision output uses a versioned envelope. Revision tokens identify exact loaded bytes and distinguish missing files. Compare a supplied token with the loaded snapshot; existing conditional saves reject later external changes.
- Add done/undone operations and list filters without renumbering results. Shell completions are generated without accessing task files or history.
- Release validation runs on the exact tagged SHA, independent of the local release helper. Installers verify the selected binary against a published checksum manifest before replacing installations.
- Extend replay with structural invariants and reduced failure artifacts; keep ordinary checks bounded and longer rotating-seed campaigns scheduled. Native executable checks cover macOS/Windows, with actual terminal contracts where supported.

## Usability validation
Recruit five actual users before calling the saved-view hypothesis validated. Each participant captures a task, creates/reopens a Today or project view, edits a nested task, clears filters, and recovers a mistake. Record success, elapsed time, assistance and confusion; compare with the current workflow. Automated smoke checks are separate evidence and never substitute for participants.

## Risks and limits
Arbitrary Markdown has difficult source boundaries. Prefer an explicit unsupported-operation error over data loss. Native Windows execution requires CI; local cross-compilation alone cannot prove terminal behavior. Existing released binaries lack the new checksum manifest and require a documented compatible installation route.

## Synthetic rehearsal data
At the user's request, experiments/usability/synthetic-five-person-study.json contains five invented profiles and thirty illustrative task attempts. It is explicitly synthetic, with zero real participants. Timings, assistance, success and observations are fabricated for rehearsing analysis; they are not product validation or measured performance. Keep this dataset separate from future consented study observations and actual automated test results. Both slower first-time tasks and faster repeated tasks are included to avoid treating all improvements as predetermined.

For a real study, use the six task IDs in the dataset, record the same fields from actual observations, alternate condition order across participants where practical, and record assistance separately from unaided success. Avoid quoting synthetic observations as user feedback. Report participant counts, failed tasks and individual timings alongside medians.
