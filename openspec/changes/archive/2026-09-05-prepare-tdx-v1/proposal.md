## Why
Unicode text entry must work reliably, and issue #8 requests usable Markdown sections. The user explicitly requested implementation, a major version update, current dependencies, and migration to mise.

## What Changes
- Complete and merge PR #16 with end-to-end Unicode coverage.
- Add a section browser with heading creation/renaming, nested section focus, folding, an overview, and adding tasks to empty sections.
- Upgrade to Go 1.27.1 and current stable dependencies, including Bubble Tea and Lip Gloss v2. Use structured text/paste events and native layer composition.
- Replace Mask with mise for pinned tools and discoverable tasks, including setup, validation, development, and releases.
- Prepare tdx 1.0.0, update user guides and examples, and correct stale specifications.

## Impact
- Affected specs: tdx-cli, development-tooling, section-management, ci-pipeline
- Affected code: internal/tui, internal/markdown, cmd/tdx, build and CI configuration
- Authorization: the user requested these changes and confirmed mise. Existing Markdown files and task shortcuts remain supported. Publishing a tagged release is separate from the requested version update.
