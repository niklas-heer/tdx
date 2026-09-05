# Go / Rust evaluation

A runnable [Rust CLI/TUI prototype](#runnable-rust-prototype) now extends this historical parser evaluation. The original measurements below remain unchanged.

**Recommendation: keep tdx in Go for now.** Rust wins this parser experiment, but the measured Go editor operation is about 3 ms for 1,000 tasks. A rewrite would still need to reproduce the TUI, metadata, undo, version history, and safe-save behavior. The application remains Go and pre-1.0.

The new `internal/editor` package gives CLI and TUI a shared action boundary. Configuration, styles, recent-file storage, and Markdown history callbacks are supplied per instance. This makes engine changes and compatibility tests easier without requiring a language migration.

## Reproduce

```sh
mise run rust:check # Rust formatting, Clippy, unit tests and shared fixtures
mise run test -- ./experiments/rust-eval/go-probe ./internal/editor ./internal/markdown
mise run rust-eval  # correctness, timings, build sizes, CPU/RSS and profiles
```

Run the evaluation on an otherwise idle macOS or Linux machine. mise supplies Rust 1.98.1 and Python 3.14.7 for the task; the project uses Go 1.27.1. Both probe dependency sets are locked. `mise run setup` installs task tools as well. Docker is unnecessary for this experiment.

Generated files go to ignored `dist/rust-eval/`: `results.json`, the two binaries, Go CPU/heap profiles, `go-cpu-top.txt`, `go-alloc-top.txt`, and build logs. On macOS, the runner also samples a separate symbolized Rust release build after it signals entry into the workload; inspect `rust-sample.txt`. This profiling build is excluded from timing and size comparisons. Linux records CPU/RSS but does not collect Rust stack samples. Windows measurements are not implemented.

The checked-in [baseline.json](baseline.json) contains the raw measurement snapshot, ranges, individual trials, toolchains, host, and source fingerprint. A run on an uncommitted working tree records that fact; its Git parent alone does not identify the measured code. Timings are observations, not CI thresholds. The existing Dagger CI runs the correctness corpus for both languages, plus Rust formatting and Clippy.

## What was measured

- **Matched scan:** parse Markdown and extract checkbox state and list depth. Go uses Goldmark's AST; Rust uses pulldown-cmark events and offsets.
- **Matched patch:** perform the scan and return a copy with one checkbox byte changed. Neither probe writes its input file.
- **Production Go context:** parse metadata and the application document model, apply the shared toggle action, and serialize the document. This includes tag/priority/due-date extraction, but excludes rendering, undo capture, history, locking, and disk I/O. It is not equivalent to the smaller Rust workload.

Before timing, 17 fixtures validate parsing and 30 individual edits. They cover Unicode, frontmatter, CRLF, links, fences, HTML, escaped text, nesting, ordered lists, and blockquotes. Both minimal probes must agree on every single-byte patch and reject an invalid index. Production Go must preserve task states and depths through its parse/edit/serialize path. The corpus exposed dropped ordered-list numbers and lost quote prefixes in the Go serializer; both were fixed and received regression coverage.

At the historical baseline revision, production serialization was **not byte-preserving**: the report records differences from minimal patches separately. The semantic gate does not check preservation of non-task blocks: the baseline production serializer could drop HTML blocks and alter multiline paragraphs. The [0.14.0 follow-up](../usage/README.md) adds source-preserving checkbox edits and fixes those structural cases; arbitrary structural Markdown still has fidelity limitations. The corpus does not establish full Markdown round-trip fidelity or exhaustively compare the parsers. A one-byte patch preserves surrounding content in either language; that benefit does not require Rust.

## Measurements

September 5, 2026; Apple M2 Pro, macOS 15.7.9, arm64. Seven trials per workload, alternating implementation order. Values below are median milliseconds per operation, measured inside the process. Iteration counts are calibrated using the slower implementation; the complete ranges and process CPU measurements are in the baseline.

| Tasks | Go scan | Rust scan | Go patch | Rust patch | Production Go context |
| ---: | ---: | ---: | ---: | ---: | ---: |
| 100 | 0.121 | 0.011 | 0.121 | 0.011 | 0.322 |
| 1,000 | 1.164 | 0.107 | 1.184 | 0.109 | 3.020 |
| 10,000 | 12.271 | 1.200 | 16.495 | 1.501 | 41.243 |

Rust is roughly 10–11× faster on the matched operations on this host. Different parser designs, enabled extensions, allocation patterns, and libraries contribute to these ratios; they are not isolated language effects.

| Measurement | Go probe | Rust probe |
| --- | ---: | ---: |
| Stripped release binary | 3.75 MiB | 0.63 MiB |
| Empty-file process wall time, median of 9 | 8.12 ms | 5.12 ms |
| Empty-file peak RSS, median | 6.48 MiB | 1.39 MiB |
| 1,000-task scan peak RSS, median | 12.92 MiB | 3.88 MiB |
| 10,000-task scan peak RSS, median | 30.05 MiB | 10.94 MiB |
| Fresh build cache, one observation | 3.29 s | 6.58 s |
| No-op build, one observation | 0.120 s | 0.033 s |

Go's probe binary also contains the production-reference path; these are not full tdx binary sizes or minimal language-runtime sizes. Builds exclude dependency downloads. Go's fresh cache also requires standard-library compilation, while Rust ships a precompiled standard library. Rust uses release optimization, thin LTO, and one codegen unit; both binaries are stripped. RSS measures the whole process, not live heap size. Go's CPU time can exceed wall time because work runs on multiple threads.

## Profiles and decision

The separate Go benchmark for 1,000 tasks recorded **3.18 ms/op, 2.98 MB allocated/op, and 39,221 allocations/op**. Its allocation profile attributed approximately 43% of cumulative allocated bytes to todo extraction, including descendants. Goldmark nodes and regular-expression extraction are useful optimization targets. The macOS CPU profile contained substantial runtime/system samples, so it is not a clean ranking of application costs.

Rust's one-second stack sample collected 771 main-thread samples. About 73% included parser construction, with the block-parsing first pass accounting for about 70% (overlapping, not additive). This locates work inside the parser; it is not a Rust allocation profile or evidence about a complete TUI. Sampling perturbs execution and uses a separately symbolized build.

The next useful performance work is to measure real editing sessions, reduce repeated extraction/allocation, and evaluate source-preserving Go edits through the new editor boundary. At 10,000 tasks, the roughly 41 ms parse/edit/serialize cost warrants attention, but it does not predict total interactive latency.

Revisit a rewrite only if realistic documents remain too slow after those changes, and a broader Rust spike demonstrates equivalent metadata, structural edits, cancellation/undo, conflict detection, atomic replacement, recovery/history, and terminal behavior across macOS, Linux, and Windows. This experiment implements none of those Rust application guarantees. There is no Rust FFI integration or production dependency.

Implementation references: [pulldown-cmark offset iterator](https://docs.rs/pulldown-cmark/0.13.4/pulldown_cmark/struct.Parser.html#method.into_offset_iter), [Cargo profiles](https://doc.rust-lang.org/cargo/reference/profiles.html), and [Go diagnostics](https://go.dev/doc/diagnostics).

## Follow-up implementation

The [sustained-use harness and Go follow-up](../usage/README.md) implements the recommendations above, including source-preserving checkbox edits, cached checked-state updates, reproducible simulated sessions and executable contracts for future rewrite candidates. Its new measurements are separate from this historical baseline; the workload definitions differ.


## Runnable Rust prototype

The follow-up lives in [experiments/rust-rewrite](../rust-rewrite/). It is a standalone Rust executable with a basic interactive terminal editor, not a production language migration. Both CLI and TUI share its document editor and guarded saves.

### Try and reproduce

```sh
mise run rust-rewrite -- --file /tmp/tdx-rust-demo.md
mise run rust-rewrite -- --file /tmp/tdx-rust-demo.md list --json
mise run rust-rewrite:check       # formatting, Clippy, unit and save tests
mise run rust-rewrite:contracts   # Go/Rust executable, replay and basic PTY checks
mise run rust-rewrite:eval        # full comparison, including fresh build caches
```

A missing file opens as an empty document and is created on the first edit. In the TUI, use `j/k` or arrows to navigate, space to toggle, `a` to append, `e` to edit, `d` to delete, `u` to undo, `r` to reload and `q` to quit. Enter saves input; Escape cancels; Ctrl-U clears it. Unicode typing, backspace and bracketed paste work at the end of the input. Quitting or reloading with unsaved changes requires a second keypress. This prototype has no version history.

CLI commands support `--file`, `--read-only`, `list --json`, `--status all|open|done`, repeated `--tag`, `add TEXT`, `edit INDEX TEXT`, `toggle INDEX` and `delete INDEX`. Indexes are one-based. Use `--` before literal text beginning with a dash. Only the documented options are implemented; user TOML settings and theme preferences are not loaded.

Checkbox changes preserve source bytes, including ordered and quoted nesting, CRLF, HTML and unknown frontmatter. Structural edits require single-line task items without children or additional blocks; unsupported changes fail before disk writes. Multiline task queries and TUI documents containing them fail explicitly. JSON query compatibility is established by the corpus, not for every possible inline Markdown construction.

Saves preserve Unix permissions, resolve symlinks, synchronize a same-directory replacement and its parent directory, and use the same canonical-path advisory lock as Go. The loaded contents and canonical target are checked under that lock. Conflicts preserve external disk bytes and retain the TUI's unsaved local candidate until explicit reload or quit. Non-cooperating writes after final validation remain outside the guarantee. Windows writes are explicitly disabled in the prototype; Linux behavior still needs its own native run. No crash/power-loss guarantee follows from these tests.

### Correctness and implementation scope

The final run checks the existing 17-fixture parser corpus and exact checkbox patches. Full CLI JSON matches Go on the 16 supported query fixtures; multiline task bodies are explicitly rejected. Two identical seeded CLI traces pass 200 actions per executable, checking persisted content and the full JSON task metadata against the existing independent oracle. Shared-lock tests make both binaries reject a held Go-compatible lock without changing the file.

The basic PTY contract covers Unicode edit, cancellation, append/delete/undo, symlinks, read-only mode, external edits during pending input, explicit reload and terminal restoration. Each executable also performs 30 immediate resize/toggle/undo cycles. These checks use semantic actions with each interface's keys: Go `N` and Rust `a` append; Go `:reload` and Rust `r` reload. They do not claim keyboard compatibility or passage of Go's advanced history/restore terminal suite.

The experiment exposed two compatibility differences retained deliberately: Go currently accepts empty CLI edits as a no-op while Rust rejects them; Rust enforces frontmatter `read-only: true` for CLI writes while Go's current CLI toggle does not. Command-line `--read-only` is enforced by both. Production behavior is unchanged in this branch.

An intermittent simultaneous resize/input stall appeared with Crossterm's default Mio backend on this host. The prototype uses its poll-based `use-dev-tty` backend and retains the repeated PTY regression. JSON output is buffered so serialization does not issue a stream of tiny writes. These findings are reasons to test complete executables, not evidence that language choice alone determines performance.

The prototype contains approximately 1,200 Rust lines including tests, plus roughly 400 Python lines for comparison. It uses 100 bounded source snapshots for undo and reparses after mutations. This establishes feasibility for a subset; it is not an estimate of the effort to reach full parity. The substantial remaining work includes multiline/structural Markdown, task moves, sections, search/filter pickers, Unicode cursor movement, configuration/themes, file watching, version history/recovery, cross-platform save-fault/race testing and release packaging.

### Application measurements

The [raw runnable-prototype baseline](../rust-rewrite/baseline.json) records every trial, process CPU/RSS, toolchains, build observations, binary hashes, source fingerprint and correctness reports. The Git parent alone does not identify the measured uncommitted source. Generated binaries, full traces and terminal transcripts live under ignored `dist/rust-rewrite/`.

September 5, 2026; Apple M2 Pro, macOS 15.7.9 arm64; Go 1.27.1 and Rust 1.98.1. Nine alternating CLI trials per workload. All values below are medians.

| Tasks | Go list JSON (ms) | Rust list JSON (ms) | Go CLI toggle/save (ms) | Rust CLI toggle/save (ms) |
| ---: | ---: | ---: | ---: | ---: |
| 100 | 8.73 | 3.70 | 22.18 | 14.02 |
| 1,000 | 11.46 | 4.93 | 24.06 | 16.06 |
| 10,000 | 40.72 | 19.44 | 45.43 | 37.63 |

| Tasks | Go observed TUI save (ms) | Rust observed TUI save (ms) | Go TUI RSS after 20 toggles (MiB) | Rust TUI RSS after 20 toggles (MiB) |
| ---: | ---: | ---: | ---: | ---: |
| 100 | 7.26 | 8.05 | 30.72 | 5.03 |
| 1,000 | 13.20 | 9.88 | 58.14 | 12.84 |
| 10,000 | 35.49 | 24.13 | 382.44 | 63.30 |

| Measurement | Go | Rust prototype |
| --- | ---: | ---: |
| Stripped executable (MiB) | 10.39 | 2.29 |
| 1,000-task JSON query peak RSS (MiB) | 16.77 | 3.64 |
| Fresh-cache build (s) | 6.907 | 12.997 |
| No-op build (s) | 0.209 | 0.083 |

At 1,000 tasks, Rust JSON queries were about twice as fast, while both observed interactive saves were around 10 ms. Resource usage is the stronger result: several times less retained TUI memory and a much smaller executable. The larger 10,000-task gap remains a useful investigation target.


Both binaries use stripped release builds; Rust additionally uses thin LTO and one codegen unit. CLI trials alternate execution order after two warmups per binary, using identical disposable documents. JSON output is compared before accepting each query measurement. CLI latency includes startup and output capture. Peak RSS is the whole child process, not heap size. Go still loads configuration and styles for queries.

Writes and persistent TUI sessions are **not equivalent service workloads**: Go includes SQLite version history, while Rust does not. PTY timing runs three sessions of 20 toggles per document size and measures key send to observed file replacement, including parent polling/read overhead. It does not measure completed directory synchronization or the next rendered frame. TUI memory is process RSS after 20 toggles, with undo and other application state retained; it is not a leak measurement.

Fresh-cache builds exclude downloads. Go builds its standard library while Rust ships a precompiled one. Build timings are single observations. Results are local macOS observations rather than cross-platform claims or CI performance thresholds. The existing parser and loaded-Go-checkbox benchmarks measure different operations and must not be divided into these timings.

### Decision

Keep Go as the production implementation. The runnable Rust prototype is useful for experiments and shows substantial resource savings, but everyday interactive saves are already quick in Go and the prototype omits important services. At 10,000 tasks, the memory difference deserves investigation; profile Go's retained undo/document state and compare an equivalent bounded source-snapshot approach before attributing the gap to the language.

Pursue another Rust milestone only if lower memory or large-document responsiveness is a concrete product goal. That milestone should cover history/recovery, representative structural edits, search/moves/sections and native Linux/Windows behavior, then repeat the same contracts with equivalent services enabled. The current evidence supports keeping a runnable experiment; it does not yet justify committing to a full rewrite.

Library references: [Ratatui application lifecycle](https://docs.rs/ratatui/0.30.2/), [Crossterm event handling](https://docs.rs/crossterm/0.29.0/crossterm/event/index.html), and [Rust file synchronization and locking](https://doc.rust-lang.org/std/fs/struct.File.html).
