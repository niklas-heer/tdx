## Why
Final PR review identified edge cases in saved section references, recent-file cursor restoration and structural replay coverage, plus ambiguous spec examples.

## What Changes
- Correct verified state-restoration and nesting edge cases with regression tests.
- Strengthen structural ownership checks and clarify editable task text semantics.
- Clarify opt-in view restoration and CLI read-only scenarios; initialize centralized view defaults explicitly.

## Impact
- Affected specs: saved-views and usage-replay; existing editor contracts remain authoritative.
- Scope authorized by the user request to finish, test and shepherd PR #24 to completion.
