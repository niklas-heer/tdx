## Why
The user requested publishing the completed 0.14.0 work and updating all documentation and the GitHub Pages website. Current pages still describe an unreleased version and overstate Markdown fidelity and performance.

## What Changes
- Publish reviewed release notes for 0.14.0, keeping the project pre-1.0.
- Refresh README, contributor/release guidance, experiment status, presentation claims and website features, examples, links and limitations.
- Prefer checked-in release notes over generated notes so this release publishes the reviewed content without relying on an external text-generation service.
- Merge validated changes, publish the configured version, and verify release binaries, Homebrew and GitHub Pages.

## Impact
- Affected spec: development-tooling. Affected code: release-note selection and its tests; website and documentation.
- Uses the existing GitHub Pages and tag-triggered release workflows. No production application or version-major changes.
- Publishing is explicitly authorized by the user's request.
