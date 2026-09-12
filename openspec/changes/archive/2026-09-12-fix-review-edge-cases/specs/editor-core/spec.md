## MODIFIED Requirements

### Requirement: Safe structural Markdown editing
Task and heading edits SHALL retain unrelated Markdown content, including reference definitions, HTML, tables, multiline bodies and line-ending boundaries. Unsupported operations SHALL fail without changing document content.

#### Scenario: A task is renamed near a reference link
- **WHEN** a task is renamed near a reference link
- **THEN** the reference definition and unrelated content remain intact through saving and undo

#### Scenario: Replace a task paragraph with a continuation
- **WHEN** task text is replaced and its checkbox paragraph contains continuation lines
- **THEN** those continuation lines SHALL be replaced with the edited text
- **AND** separate body blocks and unrelated reference definitions SHALL remain intact

#### Scenario: Reorder a nested task without filters
- **WHEN** an unfiltered move would cross the boundary of the selected task's parent
- **THEN** the task and its descendants SHALL retain their position and nesting
- **AND** moving among siblings SHALL move the complete subtree without changing its depth
