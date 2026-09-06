## ADDED Requirements
### Requirement: Complete review before work
The TUI SHALL offer an optional Rezero mode that visits every open task from the bottom of the document to the top, separately marking readiness without changing completion. Work SHALL begin only after the review completes and SHALL process marked tasks from bottom to top. An empty selection SHALL permit another review or exit.

#### Scenario: Review an entire list
- **WHEN** the user marks or passes each task
- **THEN** all tasks present at round start are reviewed before work begins
- **AND** new tasks are deferred to the next round

### Requirement: Inline context and temporary view
Rezero SHALL retain the normal inline terminal list, render selected dots separately from checkboxes, dim completed tasks, bypass filters and section folds during the round and restore the normal view on exit. It SHALL not reorder the source for review. All nested open tasks SHALL be reviewed individually. Sort and unrelated structural commands SHALL require leaving the mode.

#### Scenario: Start with hidden tasks
- **WHEN** completed-task, tag, priority, due-date or section filters are active
- **THEN** Rezero shows the full document task list and completed context
- **AND** leaving restores the user's filters

### Requirement: Atomic source-preserving continuation
The user SHALL be able to edit a continuation title inline. Confirming SHALL retire the old task subtree and append an open continuation with its notes and descendant completion states in one guarded save and one undo entry. Nested continuations SHALL become root tasks at file end with relative child nesting preserved. Unrelated source bytes SHALL remain unchanged; ambiguous source structures SHALL fail without mutation. Cancellation SHALL make no document edit.

#### Scenario: Continue partially completed work
- **WHEN** a task with notes and children is continued with a revised title
- **THEN** the original subtree is checked and the continuation is appended for the next round
- **AND** notes, Markdown formatting and child states are retained
- **AND** one undo restores the prior document and round position

#### Scenario: Concurrent external edit
- **WHEN** disk content changes while continuation input is open
- **THEN** save rejects the stale revision and retains the local candidate for explicit conflict recovery
- **AND** the external file remains unchanged

### Requirement: Session safety and recovery
Readiness dots SHALL be session-local. Read-only mode SHALL allow reviewing but reject document mutations. Reload and force-save recovery SHALL invalidate the old round. Failed precommit saves SHALL not advance the work queue; postcommit warnings SHALL acknowledge committed work. Undo SHALL restore a committed document action and its session state using the current disk revision. Status hints SHALL fit narrow terminals.

#### Scenario: Reload a reordered document
- **WHEN** a file reload changes task indexes
- **THEN** the old round and dots are discarded before another task action is accepted

#### Scenario: Read-only session
- **WHEN** the user reviews a read-only document and attempts completion, re-entry or creation
- **THEN** the action is rejected and disk content remains unchanged
