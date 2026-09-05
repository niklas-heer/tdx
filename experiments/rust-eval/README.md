# Go / Rust evaluation

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

Production serialization is **not byte-preserving**: the report records differences from minimal patches separately. The semantic gate does not check preservation of non-task blocks: production serialization can drop HTML blocks and alter multiline paragraphs. This remains a Go fidelity limitation. The corpus does not establish full Markdown round-trip fidelity or exhaustively compare the parsers. A one-byte patch preserves surrounding content in either language; that benefit does not require Rust.

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
