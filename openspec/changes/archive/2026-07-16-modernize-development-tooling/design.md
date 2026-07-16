## Context

The application module and isolated Dagger module currently target Go 1.25.4. Local tasks live in a Justfile, while Mask 0.11.7 can expose the same commands from a Markdown file that also serves as readable developer documentation. Repository coverage is 66.7%, with the command package at 0% despite controlling persistent todo mutations.

## Goals / Non-Goals

- Goals: synchronize the supported Go toolchain, update compatible dependencies, preserve task behavior in Mask, and cover the successful CLI command paths.
- Non-Goals: redesign command error handling, change user-facing CLI syntax, update Dagger-generated SDK dependencies independently, or set an arbitrary repository coverage threshold.

## Decisions

- Use Go 1.26.4 in both modules and the pinned Dagger image because it is the newest release supported by Dagger 0.21.7's Go SDK generator.
- Treat the Dagger module dependency graph as SDK-managed; only the application dependencies receive general upgrades.
- Preserve task names where Mask supports them and use `$MASK` for task chaining so commands remain location-independent.
- Use optional positional strings for development command arguments; common documented invocations remain unchanged.
- Test command success paths against temporary Markdown files without refactoring the existing `os.Exit` error contract.

## Risks / Trade-offs

- A compatible-version dependency update may still alter terminal rendering or parsing behavior. Full tests, linting, and native filesystem tests mitigate this.
- Mask does not implement Just dependencies directly. Explicit `$MASK build` calls make sequencing visible and preserve failure propagation.
- The command tests mutate process-wide output and style hooks. Tests remain serial and restore global state after capture.

## Migration Plan

1. Update and verify the Go toolchain and direct dependencies.
2. Add and validate `maskfile.md`, then remove `justfile` and update all maintained references.
3. Add command-layer coverage and run the complete local and hosted verification matrix.
