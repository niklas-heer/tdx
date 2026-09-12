## Why
The approved review identified remaining Markdown preservation risks, platform clipboard failures, repeated filtering effort, unsafe positional automation, and gaps between local checks and release validation.

## What Changes
- Preserve unrelated document content during edits and reject unsupported unsafe structural changes.
- Add saved per-file views and clearer manual-save terminology with compatible aliases.
- Support platform clipboards with truthful feedback.
- Extend CLI queries, idempotent completion, revision-guarded writes, shell completion and compatibility documentation.
- Expand structural replay, failure reduction, native terminal checks and scheduled campaigns.
- Gate tagged releases on portable/native checks and verify checksummed artifacts during installation.

## Impact
- Affected specs: editor-core, tdx-cli, saved-views, clipboard, usage-replay, ci-pipeline.
- Affected code: internal/markdown, internal/editor, internal/tui, internal/cmd, cmd/tdx, scripts, .github, .dagger.
- User approved implementation of all six review recommendations in the current conversation. Go and existing Markdown/configuration compatibility are retained.
