## Decisions
Use a small shared Go state machine to order preparation, lock acquisition, revision validation, preimage capture, replacement, directory sync, postimage capture and unlock. Native and simulated drivers execute the same phases. Errors retain their identity; the native adapter wraps post-replacement failures as PostCommitError and advances the accepted revision immediately after replacement.

Use a fixed seeded generator, virtual milliseconds and cooperative writers for replayable scheduling. Independently track visible/durable bytes, lock ownership, preimage capture and actual replacement, rather than trusting the protocol flags. Deliberately bypass validation, directory sync or replacement in negative controls. Retain native subprocess crash tests and the existing Go parser/history regressions.

Validate the logical path while holding the target lock. A changed target conflicts even during force-save once preparation has selected a target. Reject incomplete temporary writes before syncing/replacement. Advisory locks cannot protect against uncooperative writes after final validation; simulation does not claim to emulate actual disks, SQLite internals or all filesystem semantics.

Move the shared parser corpus into Go testdata before retiring the experiment directories. Historical benchmarks and the Rust implementation remain recoverable from Git; active documentation identifies Go as the sole maintained application.
