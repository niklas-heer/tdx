# Go / Rust evaluation

The [Ratatui UI completion](#ratatui-ui-completion) extends the full-featured application comparison with explicit presentation checks. The [full-featured application comparison](#full-featured-application-comparison) supersedes the earlier parser, basic prototype and history-only milestones below. Historical measurements remain unchanged and describe the capabilities at their original revisions.

**Recommendation: pursue the complete Rust candidate while retaining Go as the production default for evaluation.** The rewrite implements the complete CLI/TUI feature matrix with a polished Ratatui interface. The latest full-application comparison shows lower memory use, a smaller executable and faster large-document editing; fresh Rust builds take longer. The [updated decision](#updated-decision) supersedes the earlier milestones below.

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

A missing file opens as an empty document and is created on the first edit. In the TUI, use `j/k` or arrows to navigate, space to toggle, `a` to append, `e` to edit, `d` to delete, `u` to undo, `r` to reload and `q` to quit. Enter saves input; Escape cancels; Ctrl-U clears it. Unicode typing, backspace and bracketed paste work at the end of the input. Quitting or reloading with unsaved changes requires a second keypress. Use `v` or `:versions` to browse saved snapshots; Enter then `y` restores the selected version. `:reload` discards local changes explicitly; `:force-save` captures the overwritten disk revision before saving a conflicting local edit. Idle external changes reload every 500 ms, deferred while input, the version browser or unsaved changes are active.

CLI commands support `--file`, `--read-only`, `list --json`, `--status all|open|done`, repeated `--tag`, `add TEXT`, `edit INDEX TEXT`, `toggle INDEX` and `delete INDEX`. History commands are `versions [--json]`, `show-version ID` (exact saved source), and `restore ID`. Indexes are one-based. Use `--` before literal text beginning with a dash. The prototype reads only `[versioning].max_versions` from `config.toml` (default 100, nonpositive values unlimited); other TOML settings and theme preferences are not applied. Invalid configuration produces an error. History shares Go’s `versions.sqlite` under `$XDG_CONFIG_HOME/tdx` or `~/.config/tdx`; ordinary list/help/version do not open history. For isolated experiments, set `XDG_CONFIG_HOME` to a disposable directory.

Checkbox changes preserve source bytes, including ordered and quoted nesting, CRLF, HTML and unknown frontmatter. Single-line task labels, including parents with children, can be edited. Deletion requires an item without children or additional blocks; unsupported changes fail before disk writes. Multiline task queries and TUI documents containing them fail explicitly. JSON query compatibility is established by the corpus, not for every possible inline Markdown construction.

Saves preserve Unix permissions, resolve symlinks, synchronize a same-directory replacement and its parent directory, and use the same canonical-path advisory lock as Go. The loaded contents and canonical target are checked under that lock. Conflicts preserve external disk bytes and retain the TUI's unsaved local candidate until explicit reload or quit. Non-cooperating writes after final validation remain outside the guarantee. Windows writes are explicitly disabled in the prototype; the history milestone adds native Linux checks described below. No crash/power-loss guarantee follows from these tests.

### Correctness and implementation scope

The final run checks the existing 17-fixture parser corpus and exact checkbox patches. Full CLI JSON matches Go on the 16 supported query fixtures; multiline task bodies are explicitly rejected. Two identical seeded CLI traces pass 200 actions per executable, checking persisted content and the full JSON task metadata against the existing independent oracle. Shared-lock tests make both binaries reject a held Go-compatible lock without changing the file.

The basic PTY contract covers Unicode edit, cancellation, append/delete/undo, symlinks, read-only mode, external edits during pending input, explicit reload and terminal restoration. Each executable also performs 30 immediate resize/toggle/undo cycles. These checks use semantic actions with each interface's keys: Go `N` and Rust `a` append; Go `:reload` and Rust `r` reload. The history milestone also passes the shared terminal suite for ordered parent edits, Unicode, history browsing, deferred external changes and edits after reload; it adds Rust restore/force-save checks. This does not establish keyboard or full UI parity.

The experiment exposed two compatibility differences retained deliberately: Go currently accepts empty CLI edits as a no-op while Rust rejects them; Rust enforces frontmatter `read-only: true` for CLI writes while Go's current CLI toggle does not. Command-line `--read-only` is enforced by both. Production behavior is unchanged in this branch.

An intermittent simultaneous resize/input stall appeared with Crossterm's default Mio backend on this host. The prototype uses its poll-based `use-dev-tty` backend and retains the repeated PTY regression. JSON output is buffered so serialization does not issue a stream of tiny writes. These findings are reasons to test complete executables, not evidence that language choice alone determines performance.

It uses 100 bounded source snapshots for undo and reparses after mutations. This establishes feasibility for a subset; it is not an estimate of the effort to reach full parity. The substantial remaining work includes multiline/structural Markdown, task moves, sections, search/filter pickers, Unicode cursor movement, configuration/themes, character-level history diffs, Windows behavior, broader save-fault/race testing and release packaging.

### Application measurements (initial milestone without Rust history)

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

### Initial decision

Keep Go as the production implementation. The runnable Rust prototype is useful for experiments and shows substantial resource savings, but everyday interactive saves are already quick in Go and the prototype omits important services. At 10,000 tasks, the memory difference deserves investigation; profile Go's retained undo/document state and compare an equivalent bounded source-snapshot approach before attributing the gap to the language.

Pursue another Rust milestone only if lower memory or large-document responsiveness is a concrete product goal. That milestone should cover history/recovery, representative structural edits, search/moves/sections and native Linux/Windows behavior, then repeat the same contracts with equivalent services enabled. The current evidence supports keeping a runnable experiment; it does not yet justify committing to a full rewrite.

Library references: [Ratatui application lifecycle](https://docs.rs/ratatui/0.30.2/), [Crossterm event handling](https://docs.rs/crossterm/0.29.0/crossterm/event/index.html), and [Rust file synchronization and locking](https://doc.rust-lang.org/std/fs/struct.File.html).


## History-enabled comparison

This milestone adds Go-compatible SQLite/zstd history, snapshot browsing and conditional restore, force-save recovery, idle external reload, and single-line parent-label edits. It keeps the basic terminal interface and standalone experimental executable. The production Go application is unchanged.

The [history baseline](../rust-rewrite/history-baseline.json) preserves individual measurements, binary hashes and the measured source fingerprint. The [Linux correctness report](../rust-rewrite/history-linux-check.json) records the additional native execution checks. This historical runner wrote `dist/rust-rewrite/history-results.json`. The current `mise run rust-rewrite:eval` runs the full-featured comparison below and writes `dist/rust-rewrite/full-parity-results.json`.

Both writers now capture opened and committed content, identify files by canonical path, hash uncompressed bytes with SHA-256, compress with zstd, deduplicate content, retain 100 versions by default, and use SQLite WAL/NORMAL with a five-second busy timeout and a shutdown checkpoint. Rust uses bundled native SQLite and zstd C libraries; Go uses pure-Go implementations. This tests matching history behavior, not identical libraries or full application parity. Snapshot recovery in Rust is limited to 64 MiB of decompressed UTF-8 content; larger or corrupt snapshots fail without replacing the active document.

The correctness gates pass 19 Rust tests, the supported CLI corpus, 200 identical replay actions per executable, shared advisory locking, Go-to-Rust and Rust-to-Go snapshot reads, configurable retention and deduplication, and real terminal recovery. Failure tests cover bounded SQLite contention, a busy shutdown checkpoint, corrupt snapshots, conflicting restore, post-commit history failure and refusal to force-save when the overwritten revision cannot be captured. PTY checks cover confirmed/cancelled restore, read-only restore refusal, conflict preservation, deferred input, idle reload and recovery of overwritten disk content. The shared Go terminal suite also passes against Rust, including parent-label edits that preserve children. Test assertions use saved revisions and subsequent edits when incremental terminal redraws reuse text from an earlier frame.

Native Linux aarch64 checks passed in Docker using Rust 1.97.1, Go 1.27.1 Linux binaries and Python 3.11.2. The exact Rust 1.98.1 Docker image was unavailable, so this is explicitly a different compiler from the macOS measurements. The Linux run checks correctness, not performance. Windows remains unvalidated and Rust writes there remain disabled.

### Measurements with history on both sides

September 5, 2026; Apple M2 Pro, macOS 15.7.9 arm64; Go 1.27.1, Rust 1.98.1 and Python 3.14.7. Nine alternating CLI trials follow two warmups. Terminal results are medians of three independent sessions per engine, operation and document size, with 20 actions in each session.

| Tasks | Go JSON (ms) | Rust JSON (ms) | Go CLI edit/save (ms) | Rust CLI edit/save (ms) |
| ---: | ---: | ---: | ---: | ---: |
| 100 | 8.71 | 4.02 | 21.70 | 15.22 |
| 1,000 | 11.78 | 5.49 | 23.51 | 17.72 |
| 10,000 | 41.12 | 19.74 | 56.76 | 43.21 |

| Tasks | Go TUI toggle/save (ms) | Rust TUI toggle/save (ms) | Go TUI distinct edit/save (ms) | Rust TUI distinct edit/save (ms) |
| ---: | ---: | ---: | ---: | ---: |
| 100 | 6.78 | 6.51 | 8.00 | 7.71 |
| 1,000 | 9.34 | 9.19 | 11.15 | 10.83 |
| 10,000 | 31.77 | 21.88 | 44.32 | 33.84 |

| Tasks, after 20 distinct edits | Go RSS (MiB) | Rust RSS (MiB) | Go session CPU (ms) | Rust session CPU (ms) | Go history DB (KiB) | Rust history DB (KiB) |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 100 | 34.19 | 7.47 | 130.15 | 67.54 | 36.00 | 36.00 |
| 1,000 | 67.45 | 16.22 | 206.11 | 154.05 | 56.00 | 56.00 |
| 10,000 | 431.38 | 74.83 | 1134.91 | 770.79 | 156.00 | 188.00 |

| Measurement | Go | Rust |
| --- | ---: | ---: |
| Stripped executable (MiB) | 10.39 | 4.53 |
| Fresh-cache build (s) | 7.93 | 35.01 |
| No-op build (s) | 0.441 | 0.144 |

Each toggle session must retain exactly two content hashes; each distinct-edit session must retain 21. Both implementations pass those gates. CLI edits also use distinct text for every trial. Database sizes are measured after shutdown/checkpoint, and raw results include compressed snapshot byte counts. At 10,000 tasks, Rust's compressed snapshots are about 5% larger and its database is about 21% larger, reflecting codec output and database page allocation.

Observed TUI save time ends when the parent sees the expected atomic file replacement. It includes input processing and parent polling/read overhead, but excludes some post-replacement history capture, synchronization and rendering. Whole-session child CPU includes startup, all actions, history work and shutdown; it excludes the parent harness. CLI wall time includes process completion and history shutdown. Neither metric measures key-to-frame latency. RSS is whole-process resident memory with retained undo state, not a heap or leak measurement.

These are application comparisons with matching tested history behavior. Go still has a richer document/TUI model, themes and other services; Rust reparses into a simpler model and retains source snapshots for undo. The workload uses repetitive generated documents and 20 actions per session, not hours of real use. Builds exclude downloads and are single observations; Rust includes native SQLite/zstd compilation and a precompiled standard library, while Go's fresh cache includes standard-library compilation. Timing and memory ranges are retained in the raw baseline. The earlier run used different code and conditions, so changes between baselines are not a controlled measurement of history overhead alone.

### Updated recommendation

Continue the Rust prototype if lower memory use or large-document responsiveness is a product goal. The resource advantage remains after adding history: about **76% less resident memory after 20 distinct edits at 1,000 tasks**, and **83% less at 10,000 tasks**, with a **56% smaller executable**. JSON queries are about twice as fast. Interactive saves at 1,000 tasks are effectively similar on this measurement; at 10,000 tasks Rust's distinct edits are about 24% faster and consume about 32% less whole-session CPU.

Keep Go as the production implementation while pursuing a bounded next milestone. A full rewrite still lacks evidence for complete structural Markdown, search/moves/sections, input cursor movement, settings/themes, history diffs and Windows behavior. The fresh Rust build took about 4.4 times as long, and native C dependencies add cross-compilation work. Before choosing a migration, compare a bounded source-snapshot undo strategy in Go against the same workloads and run representative user documents through both editors. The current result supports further Rust development for resource savings; it does not establish that a complete rewrite will preserve these ratios.

Implementation references: [rusqlite connection and checkpoint APIs](https://docs.rs/rusqlite/0.40.2/rusqlite/struct.Connection.html), [zstd reusable compressor](https://docs.rs/zstd/0.13.3/zstd/bulk/struct.Compressor.html), and [TOML configuration parsing](https://docs.rs/toml/1.1.4/toml/).

## Full-featured application comparison

The Rust candidate now implements all nine groups in the [feature matrix](../rust-rewrite/feature-parity.json): CLI, structural Markdown, terminal editing, search/filters, sections, display/configuration, recent files, history/recovery, and native persistence. It runs independently of Go. Its CLI and TUI use the same Rust editor; the Go adapters are development-only differential test tools. Go remains the installed and released implementation.

The terminal supports navigation/counts, Unicode cursor editing and clipboard paste, subtree moves, bounded undo, search and metadata filters, section creation/renaming/folding, all bundled and custom themes, settings and command completion, recent-file switching, and version browsing with inline diffs and confirmed restore. Both implementations honor matching configuration precedence and share recent-file and SQLite/zstd history formats. Read-only checklist edits remain in memory until an explicit save. Guarded saves, force-save recovery, deferred external reload, and platform-specific atomic replacement are enabled.

Behavioral parity means matching commands, controls, information, document semantics and persisted state. Ratatui and Bubble Tea can lay out and redraw cells differently; this is not a pixel-identical rendering claim. Structural edits may normalize Markdown whitespace, while checkbox-only changes preserve surrounding bytes. The corpus does not prove equivalence for every possible Markdown document or terminal.

### Run the complete candidate

```sh
mise run rust-rewrite -- --file /path/to/tasks.md          # interactive TUI
mise run rust-rewrite -- --file /path/to/tasks.md list --json
mise run rust-rewrite:check                               # format, Clippy, Rust tests
mise run rust-rewrite:contracts                           # both executables and PTY contracts
mise run rust-rewrite:eval --trials 9                     # fresh comparable measurements
mise run check                                           # complete Go/project checks
```

Evaluation runs on macOS/Linux and writes ignored `dist/rust-rewrite/full-parity-results.json`. The separate `Rust feature parity` CI workflow executes native builds, regression tests, differential actions, application workflows, shared-lock checks and actual PTY/Windows ConPTY interactions on macOS, Linux and Windows. The candidate is available for native Windows builds through Cargo; it has no Go runtime dependency.

### Correctness and bug handling

The differential gate covers 1,154 cases and 2,993 action states, including fixed-seed action sequences, inline/multiline Markdown, nested and ordinary lists, malformed/empty inputs, rejection behavior, headings, and preservation of unrelated prose, tables and HTML. Another 58 executable workflows compare tasks, headings and saved cursor positions, and exercise settings, themes, recent-file switching and Unicode clipboard handling. Additional CLI replay, history interoperability, contention/corruption, restore, conflict, resize and terminal-shutdown contracts gate the timing run. Rust has 32 application unit tests on Unix and 30 on Windows (two Unix symlink tests are platform-specific), plus three document tests exercised again through the independent parity adapter; the adapter does not add three unique tests.

Differential testing corrected Go defects instead of reproducing them in Rust. Corrections include sorting whole subtrees, promoting every child list when deleting a parent, preventing cyclic moves, retaining paragraph/list order, inserting after a subtree, preserving unrelated configuration keys during theme saves, honoring settings precedence, resetting settings and cursor ownership when switching files, Unicode fuzzy matching and clipboard support, and deferring reload while browsing history or retaining read-only edits. Focused Go regressions and the shared contracts cover these decisions; the matrix lists them explicitly. Native Windows testing also caught a Rust input defect: Crossterm 0.29 paired UTF-16 surrogate key-down and key-up records incorrectly, dropping emoji. A Windows console reader now decodes key-down text separately, handles repeated/navigation/control keys and bracketed paste, and has dedicated regressions.

Native CI passed on **macOS arm64, Linux x86_64 and Windows x86_64** with Go 1.27.1, Rust 1.98.1 and Python 3.14.7. Each platform passed the 1,154-case/2,993-state action corpus, 58 application workflows, cross-engine advisory locking and real terminal resize, Unicode typing/paste, moves, undo, conflict/reload and shutdown. The [native evidence](../rust-rewrite/full-parity-native.json) and [successful CI run](https://github.com/niklas-heer/tdx/actions/runs/33982922599) identify the same measured source files and contents. Windows pathlib orders filenames differently; its recorded hash was independently reproduced from the benchmark source using that recorded order. Windows additionally uses PyWinpty 3.0.2 with ConPTY. Snapshot interoperability and the more extensive restore/fault PTY suite are recorded in the macOS benchmark gates; all platforms run native history/storage unit tests. Native correctness does not imply matching performance across these hosts.

### Measurement method

Both applications enable themes/configuration, recent-file handling, bounded undo and the same tested history policy: canonical-path SHA-256 identity, zstd snapshots, retention of 100 versions, SQLite WAL/NORMAL and shutdown checkpoint. Each terminal toggle session must produce two unique versions; each distinct-edit session must produce 21. Rust uses bundled native SQLite/zstd libraries and Go uses pure-Go libraries, so this is an application comparison, not a controlled language-runtime experiment.

Nine CLI trials alternate engine order after two warmups, using matching disposable documents and checking JSON output. Three persistent terminal sessions per engine, operation and document size each perform 20 actions. Table values use median session RSS/CPU and pooled medians of the 60 observed-save samples. The raw baseline retains individual trials, ranges, source identity, compiler versions and binary hashes.

Observed terminal save latency runs from sending input to observing the expected file replacement. It includes parent polling/read overhead and excludes some post-replacement synchronization, history capture and rendering; it is neither key-to-frame latency nor completed durable-save latency. Session CPU includes the entire child lifetime. RSS is whole-process resident memory with undo state retained, not live heap size or evidence of a leak. The repetitive generated documents and short sessions are reproducible workloads, not a substitute for long-term use with personal documents.

Both executables are stripped release builds; Rust uses thin LTO and one codegen unit. Fresh-cache build timings exclude downloads and are single observations. Go compiles its standard library while Rust ships one precompiled and builds bundled C dependencies. Measurements describe one local Apple M2 Pro/macOS host, not cross-platform performance or CI thresholds. Other host activity and run-to-run variation are uncontrolled; native CI separately validates correctness.

### Full-application measurements before UI completion

September 5, 2026; Apple M2 Pro, macOS 15.7.9 arm64; Go 1.27.1, Rust 1.98.1 and Python 3.14.7. The [raw full-parity baseline](../rust-rewrite/full-parity-baseline.json) records revision `c4084c4` and source fingerprint `c1a9a7c552ce93aab346468a719e38e4d20ac2d105bc2a988eb0a509bf1f5ad8`. Source remained unchanged throughout measurement; the recorded dirty working tree contained report/matrix updates.

| Tasks | Go JSON (ms) | Rust JSON (ms) | Go CLI edit/save (ms) | Rust CLI edit/save (ms) |
| ---: | ---: | ---: | ---: | ---: |
| 100 | 9.32 | 5.86 | 21.81 | 17.14 |
| 1,000 | 12.51 | 7.03 | 27.93 | 20.34 |
| 10,000 | 42.74 | 22.64 | 82.32 | 50.28 |

| Tasks | Go TUI toggle/save (ms) | Rust TUI toggle/save (ms) | Go TUI distinct edit/save (ms) | Rust TUI distinct edit/save (ms) |
| ---: | ---: | ---: | ---: | ---: |
| 100 | 6.36 | 6.55 | 8.04 | 7.64 |
| 1,000 | 8.61 | 6.98 | 11.25 | 16.09 |
| 10,000 | 28.42 | 19.26 | 43.21 | 103.47 |

| Tasks, after 20 distinct edits | Go RSS (MiB) | Rust RSS (MiB) | Go session CPU (ms) | Rust session CPU (ms) |
| ---: | ---: | ---: | ---: | ---: |
| 100 | 37.25 | 10.23 | 153.60 | 76.58 |
| 1,000 | 90.20 | 23.05 | 268.70 | 334.07 |
| 10,000 | 629.02 | 95.12 | 1804.66 | 2932.43 |

| Measurement | Go | Rust |
| --- | ---: | ---: |
| Stripped executable (MiB) | 10.39 | 5.01 |
| Fresh-cache build (s) | 7.511 | 40.469 |
| No-op build (s) | 0.289 | 0.130 |

### Decision before UI completion (historical)

Rust retains a strong resource advantage with the full services enabled: about **74% less RSS after 20 distinct edits at 1,000 tasks**, **85% less at 10,000 tasks**, and a **52% smaller executable**. JSON queries and CLI edits are faster on this host. Both retain the required snapshot counts; the 10,000-task edit history occupies 156 KiB in Go and 188 KiB in Rust after checkpoint.

The interactive tradeoff is substantial: Rust distinct edits take about **1.43× as long at 1,000 tasks** and **2.39× as long at 10,000 tasks** on the observed-save metric. At 10,000 tasks they consume about **63% more whole-session CPU**, despite faster checkbox toggles. Fresh Rust builds take about **5.4× as long** in this observation. The full-featured result reverses the earlier, simpler prototype's text-edit advantage; its earlier performance ratios should not be used to justify migration.

The rewrite is worth pursuing when memory or executable size is a concrete product requirement, but it is not a clear overall replacement for Go. Keep Go as the default while profiling Rust text input/edit/render work and comparing a bounded source-snapshot undo strategy in Go. Those are follow-up optimization experiments, not missing application features. The current measurements do not identify a single proven cause for the edit slowdown or isolate language effects from parser, renderer, undo and database-library choices. Try the complete candidate with representative documents before making a production migration decision.


## Ratatui UI completion

The standalone rewrite already uses [Ratatui 0.30.2](https://ratatui.rs/). Ratatui provides widgets, styling and responsive layout; it does not automatically reproduce Go's presentation. A further audit after the behavioral parity milestone found missing clickable terminal links, insufficient context indicators, long-input clipping and a hard-coded help scroll limit. Those were presentation gaps beyond what the earlier action/application corpus established.

This revision completes those behaviors and improves the interface with theme-aware rounded panels, a stronger selection indicator, compact pickers over the task list, task counts and section/filter/settings context, Unicode input that keeps the cursor visible, and history panes that stack on narrow terminals. Help and wrapped diffs scroll through their full content. Markdown link labels are styled and emitted with OSC 8 hyperlinks; the renderer also clears stale hyperlink metadata after filtering or changing modes. Inline code and metadata retain their styles across wrapping. Only the visible window of task candidates is shaped, avoiding text layout for thousands of off-screen tasks. Recent-file search matches Go’s case-insensitive substring behavior and preserves frecency order; navigation wraps, and rows retain the filename and access count. Theme and due-filter pickers show their active state and explanations, with positions and empty-state messages throughout the pickers.

All 24 Go command names are checked automatically against the Rust registry. The prior document, CLI, TUI, configuration, recent-file, history and native-save feature matrix remains implemented. New tests assert rendered information, long-input visibility, clickable-link positions and complete scroll boundaries. The native terminal gate verifies OSC 8 output as well as resize, Unicode typing/paste, moves, undo, conflict/reload and shutdown. The gallery export is an explicitly invoked development utility, separate from the acceptance tests.

Generate and inspect actual Ratatui screens with `mise run rust-rewrite:gallery`. It exports six real renderer buffers with sample Markdown to `dist/rust-rewrite/gallery/`, including `index.html`, `tasks.svg`, `commands.svg`, `sections.svg`, `history.svg`, `narrow.svg` and `input.svg`. These are generated from the Rust renderer, not design mockups. The sample terminal background uses the Tokyo Night palette; the application continues to respect the user's terminal background and selected theme. Run the complete CLI/TUI with `mise run rust-rewrite -- --file /path/to/tasks.md`.

“Full parity” refers to all documented Go application features and passing compatibility contracts. The two renderers can place cells differently, and finite tests cannot prove bug-free equivalence for every possible input or terminal.

### Measurements after UI completion

September 5, 2026; Apple M2 Pro, macOS 15.7.9 arm64; Go 1.27.1, Rust 1.98.1 and Python 3.14.7. The [Ratatui baseline](../rust-rewrite/ratatui-baseline.json) records revision `4bff503` and source fingerprint `3cf0aeaa15500106e0aa1a01a781d4855591a9687e6b55079b015030b83efe8f`. All measured source stayed unchanged; the dirty working tree contained report/matrix updates. The method and limitations above still apply: nine alternating CLI trials and three terminal sessions per engine/workload with 20 actions each, using full history, configuration and recent-file services.

| Tasks | Go JSON (ms) | Rust JSON (ms) | Go CLI edit/save (ms) | Rust CLI edit/save (ms) |
| ---: | ---: | ---: | ---: | ---: |
| 100 | 9.90 | 6.08 | 20.19 | 15.79 |
| 1,000 | 12.27 | 7.26 | 25.68 | 18.22 |
| 10,000 | 42.12 | 22.88 | 79.63 | 47.77 |

| Tasks | Go TUI toggle/save (ms) | Rust TUI toggle/save (ms) | Go TUI distinct edit/save (ms) | Rust TUI distinct edit/save (ms) |
| ---: | ---: | ---: | ---: | ---: |
| 100 | 6.66 | 6.68 | 9.00 | 8.82 |
| 1,000 | 9.22 | 8.19 | 12.91 | 9.05 |
| 10,000 | 30.74 | 22.32 | 43.27 | 20.21 |

| Tasks, after 20 distinct edits | Go RSS (MiB) | Rust RSS (MiB) | Go session CPU (ms) | Rust session CPU (ms) |
| ---: | ---: | ---: | ---: | ---: |
| 100 | 38.38 | 10.08 | 188.42 | 99.31 |
| 1,000 | 86.89 | 17.12 | 353.94 | 120.93 |
| 10,000 | 605.97 | 102.23 | 1745.14 | 397.34 |

| Measurement | Go | Rust |
| --- | ---: | ---: |
| Stripped executable (MiB) | 10.39 | 5.07 |
| Fresh-cache build (s) | 7.271 | 37.199 |
| No-op build (s) | 0.191 | 0.155 |

The full comparison passes 1,154 differential cases with 2,993 action states, 58 application workflows, CLI fixtures and replay, cross-language history, restore/conflict recovery, and the full PTY suite. The UI adds seven application tests over the prior milestone: **39 pass on Unix, 37 on Windows**, including two presentation helper tests in those totals. The three document tests also run through the parity adapter and are not extra unique tests. The ignored gallery exporter is run explicitly to produce the inspected previews.

The first final Windows run exposed a polling race in the test harness: a read during atomic replacement raised `PermissionError`. The native polling now retries within its original deadline. Three Python regressions verify transient recovery, bounded permanent failure, and propagation of unrelated exceptions; these run locally and in native CI.

Final native CI passed on **macOS arm64, Linux x86_64 and Windows x86_64** at revision `4bff503`. Every platform passed the complete action/application corpus, shared locks, and PTY/ConPTY resize, Unicode typing/paste, moves, undo, conflict/reload and clean shutdown. Both engines emitted clickable OSC 8 links in every platform’s retained transcript. The [native evidence](../rust-rewrite/ratatui-native.json) and [successful CI run](https://github.com/niklas-heer/tdx/actions/runs/33988052015) identify the same source files and contents as the measured baseline; Windows path-order differences were verified independently. Rust formatting/Clippy/tests, the three polling regressions, Go/project checks and workflow validation also passed. Native correctness does not establish native performance on those CI hosts.

### Updated decision

**The complete Rust candidate is now worth pursuing for both resource use and large-document interaction.** At 10,000 tasks it uses about **83% less RSS** after 20 distinct edits, takes about **53% less observed edit/save time**, and consumes about **77% less whole-session CPU** than Go. The executable is **51% smaller**. At 1,000 tasks, edit/save time is about 30% lower and RSS about 80% lower. At 100 tasks the interactive timings are effectively similar in this sample.

The earlier text-edit slowdown is no longer present after the rendering changes, which restrict task layout to the visible window. This is a comparison of completed applications using different parsers, renderers and database libraries; it does not isolate a Rust language advantage. Observed save time is not key-to-frame or completed durable-save latency. Fresh Rust builds still take about **5.1× longer** in this run (37.2 versus 7.3 seconds). The generated workload and short sessions support further use with representative documents, not an immediate production migration decision. Go remains the default while the complete standalone Rust candidate is available for that comparison.
