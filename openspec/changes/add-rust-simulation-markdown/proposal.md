## Why
The full Rust rewrite needs reproducible evidence of save safety under failures, beyond Go parity and example tests. Users also want to edit and preview the entire Markdown document without losing the existing guarded-save and undo behavior.

## What Changes
- Extract the production Rust save protocol into explicit deterministic transitions shared by native I/O and a seeded simulator. Simulate delayed I/O, lock contention, failed/torn temporary writes, unavailable storage/history, crashes and recovery; retain replayable traces and test the oracle with deliberate defects.
- Keep native filesystem/SQLite and real-process tests as checks on the simulator's assumptions. Document the modeled fault boundary, including the lack of a network service and the difference between process crashes and power loss.
- Add full-document Unicode source editing and rendered Markdown preview, using guarded explicit saves, conflict rejection, existing history/undo and responsive layouts.
- Retain all existing Go compatibility contracts; Rust-only features are measured and reported separately.

## Authorization and impact
The user explicitly requested implementation of deterministic simulation hardening, and selected full-document editing with rendered preview. This proposal records that approved scope. Affected code: experiments/rust-rewrite, mise tasks and native CI. No new runtime service, network protocol or application dependency is required. Go remains the production default.
