# Release handover — 2026-08-24

## Immediate state

- Repository: `tamtom/play-console-cli`
- Working branch: `codex/release-metadata-hotfix`
- Base: `0db97553b01c2e507ebd59e20809975c4e2372fc` (`v0.9.0`, merged PR #262)
- Do **not** discard the uncommitted changes on this branch.
- Goal: finish and publish immutable hotfix `v0.9.1`.

Expected working-tree changes:

```text
M  .github/workflows/release.yml
M  CHANGELOG.md
M  docs/api/google-play-api-manifest.json
M  internal/apischema/schema-index.json
M  main.go
?? release_metadata_test.go
?? docs/research/developer-id-status-api-drift-2026-08-24.md
?? docs/research/release-handover-2026-08-24.md
```

## Why v0.9.1 is required

Release `v0.9.0` is published and its five binaries match the published checksums, but a downloaded macOS ARM64 binary reports:

```text
dev (commit: unknown, date: unknown)
```

The release build injected `internal/version.Version`, `Commit`, and `BuildDate`, while `main.go` printed unrelated variables in package `main`. This also makes update/version comparisons unreliable in the released binary.

Do not delete, move, or rewrite the `v0.9.0` tag. Publish `v0.9.1`, then mark the `v0.9.0` release as superseded.

## Hotfix already implemented (uncommitted)

1. `main.go` now prints `internal/version.Version`, `Commit`, and `BuildDate` without changing the output format.
2. `release_metadata_test.go` builds and executes the real CLI with linker metadata and asserts the exact result. It failed before the fix and passes after it.
3. `.github/workflows/release.yml` executes the Linux AMD64 artifact before publication and fails unless `--version` exactly matches the tag, commit, and build date.
4. `CHANGELOG.md` contains the `0.9.1` hotfix entry and explains that it supersedes `0.9.0`.
5. Official Android Developer ID Status discovery revisions were synchronized from `20260820` to `20260823` in the manifest and embedded schema index.

Developer ID Status drift is revision-only. The live API still has one identical GET method and an identical response schema; `google.golang.org/api v0.293.0` exposes the full live contract. No client or command change is needed.

The independent primary-source comparison is recorded in `docs/research/developer-id-status-api-drift-2026-08-24.md`.

## Validation already completed

- Regression test: pass
- `go test -race ./... -count=1`: pass before the final revision-only manifest refresh
- `go vet ./...`: pass
- golangci-lint v2.13.1: 0 issues
- `make format-check`: pass
- `make check-docs`: pass
- `make check-api-drift`: pass after manifest refresh
- Developer ID/schema targeted tests: pass
- `make build-all VERSION=v0.9.1`: all five platforms built
- Local executable reports: `v0.9.1 (commit: localtest, date: 2026-08-24T18:30:00Z)`
- `govulncheck`: 0 reachable vulnerabilities

Before committing, rerun the complete final gate because the Developer ID revision refresh happened after the full race run:

```bash
GOTOOLCHAIN=go1.25.13 go test -race ./... -count=1
GOTOOLCHAIN=go1.25.13 go vet ./...
GOTOOLCHAIN=auto go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.1 run
GOTOOLCHAIN=go1.25.13 make format-check
GOTOOLCHAIN=go1.25.13 make check-docs
GOTOOLCHAIN=go1.25.13 make check-api-drift
GOTOOLCHAIN=go1.25.13 make build-all VERSION=v0.9.1
GOTOOLCHAIN=go1.25.13 go run golang.org/x/vuln/cmd/govulncheck@latest ./...
git diff --check
```

## Exact release continuation

1. Inspect `git diff` and confirm only the files above changed.
2. Run the final gate.
3. Commit on `codex/release-metadata-hotfix`, suggested message: `fix: embed release version metadata`.
4. Push the branch and open one PR to `main`; do not split the work.
5. Wait for all PR checks, merge, then wait for main-branch and CodeQL checks.
6. Optionally run the official live integration workflow once on the merged commit.
7. Tag and push `v0.9.1`.
8. Wait for the release workflow and Homebrew tap update.
9. Download every release asset and verify `checksums.txt`.
10. Execute the native binary and assert it reports the exact `v0.9.1` metadata.
11. Confirm the `v0.9.1` release body uses the changelog entry.
12. Add a warning at the top of the `v0.9.0` release body: superseded by `v0.9.1` because `v0.9.0` binaries reported development metadata. Keep all `v0.9.0` assets and its tag immutable.

## Completed v0.9.0 work

- PR #262: <https://github.com/tamtom/play-console-cli/pull/262>
- Release: <https://github.com/tamtom/play-console-cli/releases/tag/v0.9.0>
- All six PR checks, main CI, CodeQL, release packaging, checksums, and Homebrew dispatch passed.
- Manual live integration on merged `main` passed: <https://github.com/tamtom/play-console-cli/actions/runs/32762203804>
- Five release binaries and checksums were downloaded and verified.

The release contains the policy-safe ASC parity work and official Google API coverage. Dependencies are `google.golang.org/api v0.293.0`, protobuf `v1.36.12`, and Go `1.25.13`. The embedded catalog covers 218 methods and 566 types across eight Play-specific APIs.

## Policy boundary

All implemented and live-tested Google integrations use official APIs. There are no private Play Console RPCs, cookie/session reuse, authenticated browser automation, or automated legal acceptance.

Keep these manual unless Google publishes an official API:

- initial public Play app creation
- first upload/onboarding steps that the official API cannot perform
- ordinary Play App Signing enrollment and legal agreements
- experiment creation/lifecycle/results

The live integration app is `com.itdeveapps.stepsshare`, authorized by the user as a test app. Never mutate a production track, issue real refunds, delete users/grants, accept agreements, or exercise third-party app-store/KMS features in live CI without dedicated disposable infrastructure.

## Current live integration coverage and gaps

`.github/workflows/integration.yml` runs official Android Publisher API tests weekly and manually. It currently covers edit create/delete, tracks list/get, listings list/get, reviews list, invalid identifiers, regional-price conversion, and creation/cleanup of a unique one-time product. It does not upload an AAB, publish a track, call unofficial APIs, or cover every CLI command.

Known workflow gap: four auth integration tests check `GPLAY_SERVICE_ACCOUNT`, while the workflow supplies `GPLAY_SERVICE_ACCOUNT_JSON`, so those tests skip.

Recommended next test work after `v0.9.1`:

1. Invoke the built `gplay` binary black-box in live CI, not only Go clients.
2. Fix the auth environment mismatch and isolate `HOME`.
3. Add a command-coverage manifest: every command must be classified as live-tested, mocked-contract-tested, offline-tested, or manual/scope-gated.
4. Add safe read-only CLI smoke coverage for apps, tracks, listings, reviews, reporting, and vitals where data exists.
5. Exercise listing/image mutations inside a disposable edit and delete it without commit.
6. With a dedicated fixture app and monotonically increasing AAB, upload and publish only to the internal track, then verify readback.
7. Expand disposable monetization coverage where cleanup is safe.
8. Add concurrency locking, unique run IDs, an exact package allowlist, resource ledger, and always-run cleanup/janitor.

## Broader ASC parity follow-up

The latest audited ASC CLI was `4.9.0` (2026-08-23). High-value, policy-safe follow-up work:

- rooted and symlink-safe filesystem operations
- richer offline metadata validation with stable check IDs and strict mode
- deterministic structured metadata diffing
- local command search
- bounded workflow timeouts/retries with safe resume and ambiguity handling
- exact-decimal subscription regional-price derivation
- risk-focused CLI black-box coverage

Explicitly exclude ASC web sessions, cookie authentication, private endpoints, browser-driven legal acceptance, and other unofficial surfaces.

## Useful release identifiers

- `v0.9.0` merge/tag commit: `0db97553b01c2e507ebd59e20809975c4e2372fc`
- Main Branch run: `32762015271`
- CodeQL run: `32762015184`
- Live integration run: `32762203804`
- Release run: `32762371025`
- Homebrew tap run: `32762549088`
