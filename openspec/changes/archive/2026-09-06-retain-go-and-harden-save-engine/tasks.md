## Implementation
- [x] Share the production Go save protocol with deterministic simulation; preserve error identities and cleanup.
- [x] Fix and test short writes, logical target changes and failure/recovery boundaries.
- [x] Add replayable campaigns, independent negative controls, a developer command and native CI evidence.
- [x] Retire Rust code/tooling and retain useful correctness fixtures as Go tests.
- [x] Run Go checks, race/native/platform gates, rebuild Go and record evidence.

## Evidence
- Implementation: `34b73b0`; Go-only tooling and retirement: `c6c15ae`.
- [CI run 34028123499](https://github.com/niklas-heer/tdx/actions/runs/34028123499) passed portable CI, native builds, and Go engine/race/history/crash checks on Linux, macOS and Windows.
- Each platform passed 1,000 seeds × 200 scheduled effects with exact replay and 1,000 recoveries. All 1,000 trace digests and event counts matched across the three platform artifacts.
- The campaign exercised 6,957 process crashes, 1,948 power losses, 76,522 busy-lock outcomes and 14,623 injected I/O faults. All three negative controls were detected at seed 0.
- Local `mise run check`, `mise run test:race`, the simulator campaign, 200 real CLI replay actions and real PTY editing/recovery checks passed. The Go application was rebuilt as `./tdx` (v0.14.0).
- Fixed and covered actual short-write handling, failed-preparation temporary-file cleanup, logical-path retargeting during preparation, and non-regular input rejection before reads.
- Rust experiment sources/tooling were retired; historical work remains in Git and archived OpenSpec records. Its shared parser corpus remains in Go testdata. No Go dependency was added.
- Simulation validates the production protocol with modeled I/O results, not actual filesystem/SQLite internals or uncooperative changes after validation. Native checks remain required.
