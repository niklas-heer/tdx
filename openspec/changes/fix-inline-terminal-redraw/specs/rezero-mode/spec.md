## MODIFIED Requirements
### Requirement: Inline context and temporary view
Rezero SHALL retain the normal inline terminal list, render selected dots separately from checkboxes, dim completed tasks, bypass filters and section folds during the round and restore the normal view on exit. It SHALL not reorder the source for review. All nested open tasks SHALL be reviewed individually. Sort and unrelated structural commands SHALL require leaving the mode. Closing or filtering a taller overlay SHALL remove its old rows while retaining preceding shell context and the list's rendering origin.

#### Scenario: Start with hidden tasks
- **WHEN** completed-task, tag, priority, due-date or section filters are active
- **THEN** Rezero shows the full document task list and completed context
- **AND** leaving restores the user's filters

#### Scenario: Execute Rezero from the command palette
- **WHEN** the command palette is taller than the resulting Rezero list
- **THEN** the final terminal screen contains no palette border, command entries or duplicate task rows
- **AND** prior shell output remains above the list through repeated transitions
