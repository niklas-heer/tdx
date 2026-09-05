## Why
The user requests completion and hardening of the full Rust rewrite using the supplied review guidance, excluding Nix/devenv, and prefers the latest nightly. The application parity milestone is implemented; development enforcement must now expose and prevent additional defects.

## What Changes
- Pin the verified latest nightly in rust-toolchain.toml with formatter, Clippy, Rust Analyzer, sources and Miri; retain a tested stable compatibility floor.
- Audit pedantic/nursery and strict safety lints, fix substantive findings, enforce compatible restrictions with narrow justified exceptions.
- Complete all-target/all-feature checks, tests and documentation validation; adopt nextest and optional Bacon feedback without adding runtime dependencies.
- Run focused Miri checks where the native SQLite/terminal boundary permits, preserve full Go/Rust compatibility gates and native CI.
- Document every adopted or omitted recommendation and identify benchmark evidence by its original toolchain.

## Impact
- Affected spec: development-tooling.
- Affected code: Rust rewrite, shared Rust developer configuration, mise tasks and native CI.
- Authorization: this implements the user's explicit request and attached guidance. Nightly preference overrides the attachment's general stable default; Nix/devenv is excluded as requested.
