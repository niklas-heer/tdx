## Design
TigerBeetle's VOPR substitutes time and I/O around actual engine code. tdx will share a synchronous save state machine between its native driver and deterministic simulator, rather than test only an independently written model. The simulator schedules multiple writers and effect completions using a fixed integer PRNG and virtual clock; reports record seed, schema, ordered events and source identity. Errors before replacement must preserve the target, errors after replacement must report committed state, stale revisions must fail, and recovery must permit another save after faults stop.

The modeled storage has visible and durable versions and whole-file atomic replacement. Interrupted temporary writes cannot alter the target. A process crash releases its lock; power loss additionally discards unflushed replacement state. Native OS operations remain outside the deterministic model and receive real filesystem and process-crash tests. SQLite history remains a separately committed service: the simulation models its success/failure, not SQLite pages or the OS itself. Non-cooperating editors can write after the final revision check; no advisory-lock design can prevent all such races. There is no network protocol, so network partitions are not simulated as if one existed.

The source editor owns a private UTF-8 draft and cursor. No model/file change occurs before explicit save. A conflicted save leaves the draft available and the original editor state intact. Rendered preview uses the existing pulldown-cmark parser and Ratatui styles; raw HTML is text, never executable. Wide screens show source and preview together; narrow screens can switch views. Existing commands and keys retain their behavior.

## Research
- https://tigerbeetle.com/blog/2026-08-20-protocol-aware-dst/
- https://tigerbeetle.com/blog/2023-07-06-simulation-testing-for-liveness/
- https://docs.tigerbeetle.com/single-page/

The method is language-independent (TigerBeetle uses Zig). Rust contributes exhaustive effect handling, ownership of prepared resources, and a reusable deterministic core; passing finite simulations does not prove absence of corruption or superiority over Go.

## Native verification finding
The first Windows native run exposed multiline paste losing newline boundaries because ConPTY strips bracketed-paste markers without virtual-terminal input enabled. The reader now enables virtual input, parses navigation/control sequences and preserves paste text as one event, following Microsoft's console and Win32 OpenSSH guidance. Cleanup restores only the changed console flag so it cannot undo Ratatui's panic-time terminal restoration. The native source test now requires exact mixed LF/CRLF, tabs and Unicode bytes. This is a native input finding, distinct from the save simulation's intentional defect controls.
