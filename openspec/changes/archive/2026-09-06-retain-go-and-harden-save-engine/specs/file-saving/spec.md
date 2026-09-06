## ADDED Requirements
### Requirement: Shared deterministic Go save protocol
Native Go saves and deterministic simulation SHALL execute the same ordered save protocol with driver-supplied I/O results and monotonic time. Errors after replacement SHALL retain committed classification while directory-sync, history and unlock failures remain observable and cleanup continues.

#### Scenario: Reproduce faults and recovery
- **WHEN** a seeded campaign schedules delayed I/O, lock contention, storage/history failures, external revisions, process crashes and power loss
- **THEN** repeating its seed and simulator schema produces the same ordered trace and digest
- **AND** independent checks reject stale overwrites, partial targets, missing overwrite history, false commit reports and acknowledged non-durable saves
- **AND** writers recover after faults stop

#### Scenario: Validate the test oracle
- **WHEN** negative controls bypass validation, syncing or replacement
- **THEN** the campaign detects each broken guarantee and reports a replayable seed
- **AND** native platform tests independently exercise real filesystem replacement, history and crash recovery

### Requirement: Complete preparation and logical-path validation
Go saves SHALL reject short temporary writes and changes to the resolved logical target observed between preparation and final validation, including force-save. File loading SHALL inspect regular-file status before reading content.

#### Scenario: Short write without an I/O error
- **WHEN** the temporary writer reports fewer bytes than requested without an error
- **THEN** the save returns a short-write error, cleans up the temporary file and leaves the target unchanged

#### Scenario: Path retargeted during preparation
- **WHEN** the logical path resolves to a different target before final validation
- **THEN** the save conflicts and changes neither the prepared target nor the newly resolved target

#### Scenario: Unsupported input type
- **WHEN** the input path is a FIFO or another non-regular file
- **THEN** loading rejects it before attempting a potentially blocking content read
