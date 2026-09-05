## Why
Developers need tdx to compose with scripts and editors, and contributors need a fast, safe way to test changes. Current CLI parsing consumes flag-like task text, listing opens a writable history store, and the project prematurely claims version 1.0.0.

## What Changes
- Continue pre-1.0 development as 0.14.0, following published 0.13.1.
- Add explicit file selection, a literal argument delimiter, filtered lists, and documented JSON output with original task indexes and metadata.
- Return command errors to the entry point, report diagnostics on stderr, and enforce read-only mode for CLI writes.
- Provide an isolated demo, focused-test argument forwarding, and browsable coverage reports through mise.

## Impact
- Affected specs: tdx-cli, development-tooling, ci-pipeline.
- Affected code: cmd/tdx, internal/cmd, mise.toml, scripts, version configuration, README.
- Authorized by the user's request to select and implement developer improvements while retaining a pre-1.0 version.
