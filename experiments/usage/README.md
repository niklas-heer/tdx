# Sustained-use harness and Go follow-up

The [Rust evaluation](../rust-eval/README.md) recommended measuring actual editing, reducing repeated extraction, and trying source-preserving Go edits before a rewrite. This harness implements that follow-up. **Keep Go for now.** The Rust parser probe still implements none of the editor, save/conflict, history or terminal contracts exercised here.

## Run and replay

```sh
mise run test:usage                         # 100 simulated hours; dist/usage
mise run test:usage -- -seed 500             # explore another set of sessions
mise run test:usage -- -sessions 1 -steps 120 # short investigation
mise run test:usage -- -replay dist/usage/tui-100-failure.json -output dist/usage/replay
mise run test:usage-cli                     # 200 real CLI actions plus JSON queries
mise exec -- go run ./cmd/tdx-usage -driver structural -sessions 8 -steps 500
mise run test:terminal                      # actual PTYs, macOS/Linux
mise run profile:usage                      # CPU/heap profiles and checkbox benchmarks
```

The default campaign runs twenty independent five-hour sessions, seeds 100–119, with 1,800 actions each: **36,000 deliberate actions at six actions per minute**. Ten seconds of think time per action is accounted for without sleeping. An action can include navigation, multiple keys, a save, and validation. This is a stated workload assumption, not telemetry about how people use tdx.

Generated traces, reports, failure prefixes and terminal transcripts stay in ignored `dist/usage/`. A failure exits nonzero and writes the trace through the failing action. The runner also writes a `-minimized.json` trace and `-reduction.json` diagnostic. Chunk reduction tries up to 64 candidate traces while retaining the same failure category; flat candidates regenerate expectations through the independent oracle. A budget-limited trace is smaller when possible, not guaranteed globally minimal. Failures outside reproducible step invariants retain the original prefix with an explicit reason. Reports include the canonical trace SHA-256, counts by action, simulated duration, actual wall time, per-action latency percentiles, and total allocated bytes. Reuse a failure trace with `-replay`; changing the generator later cannot change that trace's actions or expectations.

## Contracts and rewrite candidates

The Go driver sends real Bubble Tea `Update` messages, renders after each action at widths 24/80/120 and heights 8/24/40, and uses real conditional disk saves in a temporary directory. A separate value-slice oracle checks order, task text, checked state, picker metadata, selection bounds and saved task lines. It never calls the production parser or serializer to calculate expected results.

The campaign mixes add/edit/delete/toggle, move/undo, cancelled edit/add/move, empty input, navigation, immediate search selection, reopen, external edits and conflicting saves followed by reload. External changes use real filesystem writes; conflict cases verify that disk content remains authoritative before resolving the conflict. Fixtures start with 24 flat tasks containing Unicode, tags, priorities and due dates, and growth is bounded around 65 tasks. Dedicated regressions cover full undo capacity, nested/ordered/rich Markdown, exact bytes, dirty snapshots and external changes while input is unfinished.

The `structural` driver starts with nested tasks, ordered lists, multiline bodies, fenced examples, HTML, tables and reference definitions, alternating LF and CRLF by seed. It mixes text, section, sort, move, indentation, bulk and undo operations with real conditional saves and reopen. It checks task identity conservation, protected source bytes, rejection atomicity, exact undo and parser/cache agreement. It supplements the independent flat oracle; it does not claim an independent full Markdown parser or drive terminal keys. The checked-in [heading-cache regression](heading-cache-regression.json) is a one-step replay reduced from the first structural campaign.

The model driver executes only the explicit reload command. It does not run the full asynchronous event loop. The separate PTY suite launches the actual application and exercises terminal initialization, resize, Unicode paste/edit/undo, read-only saves, canonical history lookup through a symlink, and a real watcher tick during an unfinished edit. All config, SQLite history and files are isolated. The terminal suite checks observable text and files; it is not a complete terminal screen emulator or a screenshot comparison.

A future executable can use the same CLI and PTY contracts:

```sh
mise exec -- go run ./cmd/tdx-usage -driver cli -binary /absolute/path/to/candidate \
  -sessions 2 -steps 100 -output dist/usage/candidate
python3 scripts/usage-pty.py --binary /absolute/path/to/candidate \
  --output dist/usage/candidate-pty
```

The CLI adapter checks each mutation, then the full `list --json` task representation: one-based indexes, text, checked state, depth, parent, tags, priority and due date. Each subprocess has a ten-second timeout. Its supported actions are add, edit, delete, toggle and reopen/query. TUI-only actions fail explicitly when given to the CLI adapter; they are never silently counted as passing. The existing Rust parser probe is **not** a compatible candidate executable.

Trace schema 1 is JSON: `schema`, `seed`, `driver`, `initial` tasks and ordered `steps`. A task contains `text` and `checked`; a step contains `op`, zero-based `index`, optional `text`, positive `seconds`, and the complete expected task array. The driver also recalculates each expectation using the independent oracle, rejecting inconsistent traces. The flat oracle's grammar remains literal task text with whitespace-delimited metadata tokens. Structural traces additionally contain `source`, `protected` source blocks and `action` objects on steps, with `undo` and `reopen` operations. Arbitrary Markdown still needs a language-neutral structural oracle before a full rewrite comparison. See [trace.go](../../internal/usage/trace.go) and [replay.go](../../internal/usage/replay.go).

## Improvements found and implemented

- Checkbox saves and undo could drop HTML, flatten tables/paragraphs, lose unknown frontmatter, change CRLF or add a header/newline. Freshly parsed checkbox edits now patch only the AST-identified checkbox byte, retain raw frontmatter while settings are unchanged, and preserve bytes through snapshots and undo. Fenced examples cannot be mistaken for editable tasks.
- Checkbox updates re-extracted every task's text, tags, priority and due date. They now use indexed checkbox nodes and update cached checked state directly. Source edits reparse and rebuild the index. Metadata extraction also has cheap guards for text without markers.
- Task and section edits now patch the original source rather than render the entire AST, preserving reference definitions, HTML, tables, fences, CRLF and unrelated document bytes. Structural edits move complete task bodies and descendants; sorting preserves sibling groups and sections. Source candidates are reparsed and checked before installation. Unsupported structural edits inside block quotes or tab-indented continuations and moves into a task’s own descendants are rejected without changing the document. Task titles must be single-line; existing multiline bodies remain supported.
- Entering and cancelling edit/add/move at the 100-entry undo limit evicted the oldest committed edit. Provisional snapshots now live outside the bounded stack. Empty input is cancelled too.
- Priority options could remain stale after edits; undo and explicit reload could retain obsolete picker metadata. Task metadata and active filters now refresh at those boundaries.
- The watcher could refresh the document and disk revision while edit/add/move input was pending. Enter could then overwrite another editor's change without a conflict. Watch reloads now defer during unfinished input, preserving the original revision so the save produces a conflict. Both a deterministic regression and the PTY suite exercise this case.
- Snapshot rebuilding now retains pending unsaved reorders. Rejected structural edits preserve source bytes, including invalid insert indexes.

## Measurements and limits

The checked-in [baseline.json](baseline.json) records the exact source revision, complete campaign summaries, executable checks, benchmark observations and host/toolchain context. Raw profiles and full traces are reproducible using the commands above. Timing is diagnostic; CI has no latency threshold.

The verified campaign at commit `70eef6c` completed **36,000 actions / 100 simulated hours in 264.6 wall-clock seconds**, with zero contract failures. It included 1,726 save conflicts, 1,681 external reloads, 1,652 undos and 1,688 searches. Across sessions, action p95 ranged from 10.6–32.3 ms (median session p95 16.0 ms); the slowest action was 290.6 ms. These action timings include navigation and harness checks, and the development host also ran CI work. The separate CLI run passed 200 mutations/queries, and PTY contracts passed on macOS and in Linux Dagger CI. Native Windows Go tests passed; Windows PTYs were not tested.

| Tasks | Before median | After median | Before allocations | After allocations |
| --- | ---: | ---: | ---: | ---: |
| 1,000 | 1.064 ms | 0.0058 ms | 14,027 | 1 |
| 10,000 | 11.354 ms | 0.0250 ms | 140,042 | 1 |

Post-campaign review also hardened the harness: unsupported replay drivers are rejected before constructing output paths, and the PTY conflict scenario waits for observable EDIT mode before the external write. Regression, race, project and terminal checks passed after these changes. The campaign engine and production application were unchanged; the recorded 100-hour measurement remains tied to `70eef6c`.

The loaded-checkbox benchmark measures a previously parsed 1,000- or 10,000-task document, one toggle and serialization. It excludes undo snapshots, rendering, history, locking and disk I/O. It must not be compared directly with the original Rust report's parse/edit/serialize pipeline or presented as an application-wide speedup. The remaining string allocation is the serialized output.

Session allocation totals include the harness oracle, validation and rendering, and count cumulative allocations, **not live memory or leaks**. The sampled CPU profile on macOS is sparse and dominated by system calls; it cannot reliably rank application CPU costs. The allocation profile still identifies parsing/extraction and undo snapshots as future measurement targets. The campaign intentionally keeps filesystem safety checks and synchronous durable saves.

One hundred simulated hours do not certify one hundred wall-clock hours, idle timer behavior, clock advancement or due-date transitions, human usability, crash/power-loss recovery, unbounded documents, network filesystems or exhaustive concurrency. Separate existing save-fault/race tests cover additional failure paths. The PTY suite runs on macOS/Linux; Windows interactive parity and a full nested/structural language-neutral oracle remain requirements for a serious rewrite evaluation.

Bounded seeded and regression tests run in normal Go tests. Dagger CI also runs CLI and Linux PTY contracts. The longer campaign stays an explicit `mise run test:usage` task. Production remains Go. These changes ship in [v0.14.0](https://github.com/niklas-heer/tdx/releases/tag/v0.14.0); the project remains pre-1.0.
