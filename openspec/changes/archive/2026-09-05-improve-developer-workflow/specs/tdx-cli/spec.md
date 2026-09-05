## ADDED Requirements
### Requirement: Scriptable task queries
The CLI SHALL support `list --json`, `--status all|open|done`, and exact `--tag` filters. JSON SHALL be an array with documented lowercase fields, original one-based task indexes, text, completion, depth, parent index, tags, priority, and optional due date. Filters SHALL compose without renumbering tasks. Listing SHALL not require a writable configuration/history directory or create missing files.

#### Scenario: Script reads a filtered project
- **WHEN** a developer runs `tdx --file tasks.md list --json --status open --tag backend`
- **THEN** stdout SHALL contain only a JSON array of matching tasks with original indexes
- **AND** an empty result SHALL be `[]`
- **AND** no history or recent-file data SHALL be written

### Requirement: Predictable command arguments and errors
The CLI SHALL support `--file`/`-f`, retain positional Markdown paths, honor `--` to delimit literal arguments, reject unknown options and unsupported list options before side effects, send errors to stderr with nonzero exit status, and reject CLI mutations in read-only mode. Command functions SHALL return errors instead of terminating the process.

#### Scenario: Task text resembles a flag
- **WHEN** a developer runs `tdx add -- --read-only`
- **THEN** the literal text `--read-only` SHALL be added

#### Scenario: Read-only script tries to mutate a file
- **WHEN** a developer runs a mutating command with read-only mode enabled
- **THEN** the command SHALL fail on stderr without changing the task file or opening history storage
