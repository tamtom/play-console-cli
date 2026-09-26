# 1.0.0 audit resolution — 2026-09-26

All seven P1 findings have fixes and passing local regressions. R8–R16 and the
required release-process improvements are implemented. Changes remain in the
working tree; no commit, tag, push, or release was created during this work.
Native Linux/Windows execution remains a release-CI requirement.

The original [audit](release-readiness-1.0.0-2026-09-26.md),
[API audit](api-audit-1.0.0-2026-09-26.md), and isolated reproductions are
preserved as historical evidence. Their original failure descriptions are not
the current implementation status.

| Finding | Resolution |
|---|---|
| R1: dry-run writes | Global and local dry-run reach HTTP, subprocess, local configuration, notification, and updater write paths; tests assert absence of side effects. |
| R2: late output rejection | Root output validation runs before command execution, including writes and uploads; explicit output flags take precedence over environment defaults. |
| R3: unloadable init config | Init writes loader-compatible JSON through the shared serializer; legacy YAML produces actionable migration guidance. |
| R4: invalid rollout payload | Completion omits `userFraction`; fraction 1 completes; invalid fractions and decreasing percentages fail. Updating a halted release requires explicit resume. |
| R5: outdated SDK policy | Dated application-type rules select API 36 for ordinary submissions, with specialized-app rules and an explicit reported override. |
| R6: retained secrets | Audit arguments and errors, webhook results, and dry-run logs redact sensitive values. |
| R7: unverified downloads | Self-update and both installers require a matching checksum before replacement; missing, malformed, duplicate and mismatched entries fail closed. |
| R8: disconnected notifier | Successful interactive commands now check for newer stable releases with bounded requests, successful-result caching, opt-out and stdout preservation. |
| R9–R11: API contracts | Unsupported archive is an actionable deprecated stub; purchase examples execute through the real SDK serialization path; a narrow App Store adapter preserves SDK-missing fields. |
| R12–R14: help/configuration/RTDN | Global flags and errors are visible, all help paths exit successfully, canonical package environment defaults work, and existing RTDN topics proceed to IAM repair. |
| R15–R16: timeouts/builds | Verification uses configured timeouts and retry behavior; make always invokes incremental Go compilation for source/version changes. |

Billing checks now distinguish a known supported version, a known expired
version, and an unknown version. Debug documentation describes the behavior
actually implemented. Play Grouping remains optional scope and was not added.

## Verification

- `make test`: passed with race detection, 94 packages and 2,220 test/subtest passes.
- Real `golangci-lint` v2.13.2: zero issues; formatting, generated docs and
  `git diff --check` pass.
- Original isolated audit reproductions: all pass unchanged.
- Shell and PowerShell installer contracts, source/version rebuild behavior,
  and SDK-field checker fixtures: four Python test groups pass, including
  PowerShell executed locally rather than skipped.
- Discovery comparison: 218 endpoints and 568 types across eight APIs; SDK
  coverage checks 1,574 fields and three reviewed adapter fields. Current live
  discovery drift check passes.
- Go 1.27.1 vulnerability scan: no reachable vulnerabilities.
- All five shipped platform binaries cross-compile. Both macOS binaries execute
  and report 1.0.0. Published 0.10.0 upgrades to the candidate on macOS arm64.
- Independent `autoreview` completed with one P2 rollout finding. Its regression
  reproduced the unsafe commit before the fix and passes afterward; the final
  full suite and lint include this correction.

The release workflow now requires checks against the tagged commit, native
installer/updater and previous-stable upgrade checks on each of the five shipped
platforms, embedded-version verification, and exactly five expected assets
before publishing. That remote workflow has not been triggered here. No live
Google Play publishing, purchases, refunds or account-specific API eligibility
checks were performed.

## Test audit and incident

The installed OpenClaw test-audit skill was applied as a focused audit. Five
existing low-value tests were removed, stronger observable behavior tests were
retained/added, and newly introduced production test hooks were removed. See
the [candidate evidence and review record](release-1.0.0-test-audit.md).

During test development, an insufficiently isolated auth-init regression wrote
an empty template to `~/.gplay/config.json`. Prior contents were not captured;
no adjacent backup or local Time Machine snapshot was found, so restoration
could not be established. This was disclosed immediately. The test now uses a
local fixture and the root helper isolates HOME and USERPROFILE. This incident
is separate from the release-validation results above.
