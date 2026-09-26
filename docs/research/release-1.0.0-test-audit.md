# Focused test audit for 1.0.0 fixes

Applied the installed OpenClaw `test-audit` authoring gate to the release fixes.
This is a focused audit, not a repository-wide deletion campaign. This Go
repository uses its AGENTS.md validation commands rather than OpenClaw's Node runners.

## Candidate evidence before replacement

- Candidate: `TestNormalCommandUpdateSuggestion`, `cmd/update_notification_test.go`.
- Actual proof: selection of an injected checker and an injected interactive boolean;
  it cannot catch broken terminal detection or disconnected real release lookup.
- Non-test callers: none for the newly added `Runtime.WithUpdateChecker` seam.
  `Runtime.Interactive` and `Runtime.CheckForUpdate` merely wrap that seam for startup.
- Stronger owner: run the command in a real pseudo-terminal, intercept only the
  external GitHub HTTP boundary, and verify stdout, advice, exclusions and failure behavior.
- History: all three runtime methods and this test were introduced in these
  uncommitted release fixes; no compatibility obligation. Existing runtime service
  factories predate this work and are outside this candidate.
- Deletion: remove the checker/terminal fields, methods and runtime imports.
- Risk/proof: terminal behavior differs by platform. Use the platform's `script`
  utility on macOS/Linux; skip only the PTY cases where unavailable. Run
  `go test ./cmd -run TestNormalCommandUpdateSuggestion` and prove the disconnected
  notifier fails with a pre-fix source overlay.

## Retention decisions

Keep real HTTP payload tests (independent Google protocol contracts), configuration
round trips/migration tests (persistent format contract), checksum failure tests
(security and installation contract), and offline policy tests (dated eligibility
rules). Do not replace these with mock-call assertions or source-text inventories.

## Updater metadata assertions

- Candidates in `internal/cli/updatecmd/update_test.go`:
  `TestUpdateCommandName`, `TestUpdateCommandShortHelp`,
  `TestUpdateCommandFlags`, `TestUpdateCommandUsageFunc`.
- Actual failures: changed constructor metadata, flag declarations/defaults,
  or a nil function pointer. They do not prove that checking avoids installation,
  forcing installs an otherwise current release, or help actually works.
- Production callers: the command catalog invokes `UpdateCommand`; no production
  consumer needs these test-specific metadata assertions.
- Stronger proof: real installer subprocess cases for `--check`, default current
  version, and `--force`; root command/help and documentation catalog contracts.
- History: introduced with the updater command in `b92fcc9`; later changes
  (`478cdff`, `3aa4ced`) concern platform fixes and formatting.
- Deletion unlocked: four declaration-only tests. The duplicate installation
  detector has already been consolidated in `internal/update`; retain its
  installation-layout cases because they protect correct operator instructions.
- Risk/validation: loss of metadata-only assertions is low risk; run
  `go test ./internal/cli/updatecmd ./cmd ./internal/cmdtest` after replacement.

## Obsolete init renderer

- Removed candidate: `TestGenerateConfig`, `internal/cli/initcmd/init_test.go`.
- Actual proof: only that a private YAML renderer returned a nonempty string.
- Production caller: formerly `InitCommand`; it now uses the actual JSON config
  serializer/loader. The private YAML renderer is obsolete and removed.
- Stronger proof: `TestInitCreatesLoadableConfiguration` runs the root command
  and loads its saved package, credentials reference, and timeout with the real loader.
- History/reason: old initializer predated the loader-compatible JSON fix.
- Risk/validation: this intentionally changes the format contract; existing-file,
  force, legacy-migration and dry-run tests remain. Focused init/config/root tests pass.

## Validation and implementation notes

- Final full race-enabled suite: 94 packages, 2,220 test/subtest passes.
  Real golangci-lint reports zero issues; format/docs/schema checks pass.
- The terminal regression fails on the original `cmd/run.go` overlay with zero
  release-lookup calls, and passes against the connected production startup path.
- Five existing low-value tests were removed (the four updater metadata tests
  and private YAML nonempty-string test). The updater check/force contracts now
  execute a copied binary and inspect the installed bytes, download behavior,
  and the replacement executable's version.
- Keep the generic command-usage architecture guard; it covers every command
  required by repository policy, unlike the redundant per-command pointer check.
- Target SDK tests distinguish the CLI's reported rule/flag wiring from actual
  acceptance/rejection at the policy engine with decoded manifests.
- Existing dry-run upsert coverage caught over-redaction of harmless query
  parameters. That test was retained and the logging implementation repaired.
- Full-suite testing caught a race in the new installer fixture's server URL
  capture. The fixture now uses the incoming HTTP host, and the focused race
  rerun passes. No production hook was added for the test.
- Self-update installation has been verified over published 0.10.0 on macOS
  arm64. Other native platforms are mandatory jobs in the tagged release gate.
- The original isolated audit reproductions now pass unchanged.
- Independent `autoreview` completed and reported one P2: `rollout update`
  could resume a halted release. A root-command HTTP regression first reproduced
  a track write and commit at both 50% and 100%. Update now requires an active
  staged release and directs halted releases to `rollout resume`; the same
  regression passes with no track update or commit. Explicit resume and
  completion payload cases still pass.
- A final SDK-policy fixture expansion initially failed to compile because its
  table used `int` rather than the manifest encoder's `int32`. The corrected
  table passes, and an overlay restoring the original SDK floor fails for the
  expected ordinary, TV, XR, and private-app cases.
- The dry-run updater cache assertion now checks the home directory actually
  used by the isolated root-command helper.
- Focused diff size, including new implementation/test files and excluding
  research reports: production Go +1,012/-408 lines in 38 files; tests
  +1,455/-62 in 18 files. Additional documentation and build tooling are
  separate. The additions protect external CLI, protocol, persistence,
  installation and policy contracts; there are no new runtime-only test hooks.
- A test-isolation mistake during this work wrote an empty global config
  template at `~/.gplay/config.json`. Its prior contents were not captured and
  no adjacent backup or local Time Machine snapshot was found. This was
  disclosed immediately. The new root test helper now isolates HOME and
  USERPROFILE as well as command I/O and audit sinks.
