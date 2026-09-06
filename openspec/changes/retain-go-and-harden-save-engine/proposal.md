## Why
The user has decided to abandon the Rust rewrite because the Go interface is more polished and the rewrite offers insufficient user benefit. The useful engine hardening should run in Go.

## What Changes
- Retire active Rust experiments, dependencies, tasks and CI; retain historical work in Git and archived OpenSpec changes.
- Run Go native saves and a deterministic fault simulator through one effect-by-effect save protocol.
- Preserve committed-error classification, bounded locking, overwrite-history capture and cleanup after later failures.
- Reject non-regular inputs before reading, short temporary writes and logical-path retargeting observed before replacement.
- Add replayable delayed-I/O, contention, storage/history outage, process-crash and power-loss campaigns, independent negative controls, and expanded native crash checks.
- Preserve the current Go CLI/TUI and rebuild the Go executable.

## Authorization
The user explicitly requested abandoning Rust, returning to Go, and applying the engine improvements already implemented in Rust. This proposal records that authorized scope.

## Impact
- Affected specs: file-saving, development-tooling, tdx-cli
- Affected code: internal/markdown, shared Go save protocol/simulator, developer CLI, mise and CI
- No UI redesign or new application dependency. Simulation models I/O outcomes; native filesystem and SQLite behavior still require native checks. There is no network service to harden.
