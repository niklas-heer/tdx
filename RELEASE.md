# Releasing tdx

tdx remains pre-1.0. The current release is **[0.14.0](https://github.com/niklas-heer/tdx/releases/tag/v0.14.0)**. A 1.0 release requires a separate stabilization decision.

Update `version` in `tdx.toml` and write reviewed user-facing notes in `docs/releases/<version>.md` in a pull request. Update README release highlights, website feature descriptions and upgrade instructions at the same time. Merge the release changes and wait for CI to pass before publishing.

```bash
git switch main
git pull --ff-only
mise run release
```

The release task verifies a clean checkout of `main`, checks that local and remote commits match, runs local validation, and asks before pushing the single version tag. It uses the version already configured in `tdx.toml` and does not increment it automatically.

The tag workflow first validates the exact tagged commit through the shared portable and native matrix, independently of the local helper. Only after all checks succeed does it verify the version, build five platform binaries with Dagger, publish SHA256SUMS and signed build provenance, use the reviewed release notes (or generate notes when none exist), create the GitHub release, and attempt to update the Homebrew tap. The application and Dagger module use the compatible toolchains documented in README.md.

To inspect release artifacts without publishing:

```bash
mise run release-artifacts
ls dist/
```

For a release containing SHA256SUMS, users can set `TDX_VERSION` when running the install script. The script fails closed if its checksum manifest is missing or invalid; historical releases such as v0.14.0 require Homebrew or a direct release-asset download. `TDX_INSTALL_DIR` overrides the default `~/.local/bin` destination. `gh attestation verify <binary> --repo niklas-heer/tdx` verifies release provenance separately.

The website is GitHub Pages, published from `main:/docs`; merging documentation updates triggers deployment. Verify the Pages build, the live release link and installer after publishing. Keep `docs/install.sh` byte-identical to `scripts/install.sh`.

Before calling a release complete, verify that GitHub marks it as the latest release, all five binaries plus SHA256SUMS and provenance.json are present, downloaded binaries report the intended version, and the Homebrew formula references the new version and checksums. Read the generated notes before announcing the release. The reviewed-note path requires no OpenRouter credentials; the legacy generated-note path does.

Verify release-note selection offline with `mise run test:release-notes` (also included in `mise run check` and portable CI).

## Homebrew pull request requirement

The tap protects `main` with a pull request requirement. The current release workflow attempts a direct push, which was rejected for v0.14.0. Complete the formula update through a PR:

1. Clone `niklas-heer/homebrew-tap` and create a release branch.
2. In the tdx checkout, run `TAP_REPO=/path/to/homebrew-tap mise run update-homebrew -- 0.14.0`, substituting the release version.
3. Check all four formula checksums against the published release assets. Commit with `chore: update tdx to v0.14.0`, push the branch, create the tap PR, and merge after review.
4. Re-run only the failed Homebrew job. Once the formula matches, it completes without another push. Do not re-run the successful release-creation job for an existing release.

The workflow needs a future change to create a tap PR automatically; keep the repository protection enabled. For v0.14.0, see the [formula update PR](https://github.com/niklas-heer/homebrew-tap/pull/3).
