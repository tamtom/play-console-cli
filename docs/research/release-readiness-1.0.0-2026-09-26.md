# gplay 1.0.0 release audit — 2026-09-26

**Recommendation: hold 1.0.0 until the core fixes below are complete.**

Audited the current repository at `5de0f33` (`v0.10.0-24-g5de0f33`), rather than
only changes since the last tag. The working tree was initially clean. This was
an audit: production code, credentials, release tags, and Google accounts were
not changed. Findings combine source inspection, the existing test suite,
isolated regressions, and current official Google documentation.

The API surface is broad: all **218 endpoint IDs across the eight advertised API
families** have implementation paths. Endpoint counts do not establish correct
payloads or usable deprecated operations. See the separate
[official API audit](api-audit-1.0.0-2026-09-26.md) for the complete inventory,
latest revisions, dependency comparison, citations, and restricted API findings.

**Validation evidence**

| Check | Result |
|---|---|
| `make test` / `go test -v -race ./...` | Passed: 93 packages, 2,091 test/subtest passes, no race failures; local Go 1.26.0 |
| `make lint` | Passed its `go vet ./...` fallback; golangci-lint was not installed locally |
| `make format-check`, `make check-docs` | Passed |
| `make check-api-schema` | Passed: 218 endpoints, 568 types |
| Fresh CLI build | Passed; used an isolated binary in `/tmp` |
| Actual `--help` invocations | 381 paths checked; all printed usage, but `snitch` and `snitch flush` exited 2 instead of 0 |
| Current official API comparison | No missing method IDs in the eight tracked APIs; Checks has revision-only drift |
| `govulncheck` using release Go 1.27.1 | No reachable vulnerabilities; three findings in required modules were outside called/imported vulnerable paths |
| `govulncheck` using local Go 1.26.0 | 26 reachable standard-library findings; this is not the result for the configured release toolchain |
| New isolated audit regressions | Intentionally fail on the audited code; reproduce gaps that the existing suite misses |

Run the preserved regressions with
`python3 docs/research/release-1.0.0-repros/run.py`.
They use temporary Go overlays, fake Google responses, fake gcloud, and local
HTTP servers. They do not alter the ordinary test suite or replace a binary.
The harness has 12 failing test groups at the audited commit.

**Core fixes before 1.0.0**

1. **R1 / P1 — Global dry-run can perform real writes.**
   `internal/cli/notify/send.go:90` posts through `http.DefaultClient` without
   consulting dry-run or using its transport. `internal/cli/rtdn/rtdn.go:90`
   copies only the command-local flag into `SetupOptions`, ignoring the root
   context. The tests observed one POST for `--dry-run notify send`, and both
   topic creation and IAM binding calls for `--dry-run rtdn setup`. The local
   `rtdn setup --dry-run` spelling follows a different path. Make both forms
   honor the same setting before crossing a write boundary. Apply the same
   review to `setup`/`auth setup`, which also read only their local flag, and to
   self-update/local write operations. Test zero outbound writes, not merely
   the presence of a dry-run message.

2. **R2 / P1 — Invalid output can be rejected after a successful mutation.**
   `internal/cli/shared/shared.go:172` validates only the pretty/format
   combination; it accepts `xml`. The recursive format validator in
   `internal/cli/shared/output_validation.go:14` has no production caller.
   A fake-API invocation of `rollout halt --output xml` created an edit, updated
   the track, validated, and committed; only then did it exit 1 with
   `unsupported format: xml`. An automation retry can repeat a completed write.
   Validate supported formats before creating clients or executing commands,
   while preserving intentionally supported `text` output in setup/RTDN.
   Cover at least one write command and one upload with this invariant.

3. **R3 / P1 — `gplay init` creates a configuration the CLI cannot load.**
   `internal/cli/initcmd/init.go:32` writes `.gplay/config.yaml` using
   `default_package` and `service_account`; `internal/config/config.go:18` and
   `:331` search for `config.json` and parse JSON. The generated timeout is also
   commented out. An isolated `init --package com.example.audit --timeout 90s`
   succeeded, followed by `configuration not found` from the actual local
   loader. On an existing workstation this can cause fallback to unrelated
   global configuration. Write through the existing configuration model and
   serializer, test loading the result, and correct `docs/configuration.md`,
   root help, and the repository instructions. Define compatibility explicitly
   for previously generated YAML files.

4. **R4 / P1 — Rollout completion and the 100% boundary are wrong.**
   At `internal/cli/rollout/rollout.go:194`, completion explicitly serializes
   `"userFraction":0` using `ForceSendFields`. Google permits that field only
   for `inProgress`/`halted`, with a value strictly between zero and one.
   The documented completion payload omits it.
   [Release contract](https://developers.google.com/android-publisher/api-ref/rest/v3/edits.tracks#Release),
   [rollout workflow](https://developers.google.com/android-publisher/tracks).
   Separately, `rollout update --rollout 1` succeeds and commits the unchanged
   fraction: the reproduction started at 10% and remained at 10%. Reject that
   value with guidance to `rollout complete`, or explicitly complete the release.
   Validate finite fractions, status/fraction combinations, and resume bounds
   before writing; test the actual serialized request.

5. **R5 / P1 — The default preflight policy is outdated and ignores app type.**
   `internal/preflight/scan_policy.go:11` sets a universal target SDK floor of
   35. Its target-SDK check returned no finding for an ordinary target-35 app.
   Since August 31, 2026, ordinary new apps/updates require 36; Wear OS and
   Automotive require 35, while TV/XR require 34. There are extension and private
   app exceptions. A single global 35 both misses invalid ordinary submissions
   and can reject eligible specialized apps. Use a dated policy table and app
   context, retain an explicit override, and surface which rule was applied.
   [Current official requirement, updated September 16, 2026](https://developer.android.com/google/play/requirements/target-sdk).

6. **R6 / P1 — Default audit logging retains secrets.**
   `cmd/run.go:169` redacts only five exact flag names. The reproduction retained
   `--api-key`, `--webhook-url=...`, `-token`, and RTDN `--data` containing a
   purchase token. Audit is enabled by default
   (`internal/audit/audit.go:56`). Redact supported flag spellings and inline
   payloads before persistence; scrub error fields as well, since
   `cmd/run.go:159` stores the raw error. Successful webhook output also returns
   the complete credential-bearing URL (`internal/cli/notify/webhook.go:67`).
   Test fake secret markers against every emitted/audited representation.

7. **R7 / P1 — Self-update installs unverified response bytes.**
   `internal/update/update.go:221` accepts any HTTP 200 body, writes it to a
   temporary file, and returns it for replacement. A local response containing
   `not a gplay executable` was accepted without error. The installer path in
   `internal/cli/updatecmd/update.go:120` immediately applies that file; there
   is no checksum check in between. Release packaging already publishes
   `checksums.txt`. Verify the selected asset before replacing the old binary,
   reject missing/mismatched checksums, handle file-close errors, and test that
   failed downloads leave the executable intact. `install.sh:51` also falls
   back to skipping verification; align the installation paths. Validate
   installation/upgrade on each shipped platform, especially Windows.

8. **R8 / Required product behavior — Automatic update suggestions are disconnected.**
   The manual `gplay update` and `gplay update --check` commands exist.
   `CheckForUpdate` is called only by the update command; `PrintUpdateMessage`
   has no caller. Neither `main.go` nor `cmd/run.go` invokes the checker, and
   production code never reads `GPLAY_NO_UPDATE`. README's claim that startup
   checks for updates is therefore false. Wire the suggestion into normal
   execution and test the top-level command path, rather than only helper
   functions. Detailed expected behavior is below.

**Other fixes and missing help**

| ID | Priority | Finding and required change |
|---|---|---|
| R9 | P2 | **Unsupported subscription archive.** `internal/cli/subscriptions/subscriptions.go:488` calls an operation Google labels unsupported. Retain an actionable deprecated stub or remove it from supported commands. See API-1 in the API audit. |
| R10 | P2 | **Invalid purchase help examples.** V2 defer uses `P7D` instead of `604800s`; legacy defer uses unquoted int64 timestamps and describes the expiry direction incorrectly. Both failures were reproduced with pinned dependencies. Correct and execute the examples as contract fixtures. See API-2. |
| R11 | P2, gated scope | **Third-party-store JSON loses fields.** The generated SDK drops `versionCode`, `alreadyPublishedOnPlay`, and response `updateId`; upgrading to the latest SDK does not repair it. Preserve the current official schema through a narrow adapter or explicitly reject unsupported fields. Explicit false presence also needs a regression; no distinct live-server interpretation of omitted false was established. See API-3. |
| R12 | P2 | **Root help omits every global flag and some errors are silent.** `cmd/root_usage.go:32` never renders its FlagSet. The actual root help omits profile/debug/dry-run/report/report-file/version. `--report junit version` returns 2 with empty stdout/stderr (`cmd/run.go:60`). Bare `setup`, `auth setup`, and RTDN status without gcloud return 1 with no guidance because they use `NewReportedError` without printing anything. Also normalize the two `snitch --help` exit codes. |
| R13 | P2 | **Documented package environment variable is ignored.** Repo instructions and configuration docs advertise `GPLAY_PACKAGE`; the resolver reads only `GPLAY_PACKAGE_NAME` (`internal/cli/shared/shared.go:22`). The regression resolves an empty package from the documented variable. Support/document a canonical name with a compatibility alias and explicit precedence. |
| R14 | P2 | **RTDN setup is not repeatable with an existing topic.** `internal/cli/rtdn/rtdn.go:164` discards gcloud output and checks `err.Error()` for `already exists`; a real process error contains only `exit status 1`. The fake-process reproduction stopped before repairing IAM access. Inspect structured output or describe-before-create, and preserve useful stderr. |
| R15 | P2 | **Verification bypasses timeout/retry settings.** `verification.go:58` and `developeridclient/service.go:34` bypass the normal shared configuration. Add bounded requests and the existing read-only retry policy. See API-4. |
| R16 | P2 | **`make build` can silently retain an old binary.** The `gplay` target depends only on `go.mod` (`Makefile:45`). In an isolated copy with source newer than the binary, `make -n build` printed only “Build complete” and no build command. Make source/version changes invalidate the target, or always invoke incremental `go build`. The release workflow itself calls `go build` directly, so this specific defect concerns developer/local build paths. |

Google also now documents halting a completed rollout. The selector at
`internal/cli/rollout/rollout.go:179` accepts only staged/previously halted
releases; the regression confirms a completed release is rejected. Extend the
high-level halt workflow deliberately, including identifying which eligible
release to halt, or state this limitation and show the lower-level route.
[Official completed-rollout halt workflow](https://developers.google.com/android-publisher/tracks#halt_a_completed_rollout).

**What the update mechanism should do**

Use the existing GitHub release lookup and installation-method detection, with
these acceptance criteria:

- On ordinary interactive commands, suggest the latest stable release on stderr
  when it is newer. Preserve stdout exactly, including minified JSON.
- Keep a bounded check and cache the successful result for the existing 24-hour
  interval. An offline failure must not fail the user's command or masquerade
  as a successful 24-hour check (`update.go:127` currently records failures too).
- Honor `GPLAY_NO_UPDATE=1`; keep help, version, and completion fast. Define
  predictable CI/noninteractive behavior and provide explicit `update --check`.
- Use proper SemVer. The current comparator treats `1.0.0` and `1.0.0-rc.1` as
  equal; a preserved regression demonstrates this. Handle development builds
  and prereleases deliberately.
- Suggest `brew upgrade tamtom/tap/gplay` for Homebrew installations and
  `gplay update` for standalone binaries. Verify any Go-install guidance against
  the executable name it actually produces.
- Verify downloaded assets before replacement and add deterministic tests for
  newer/same/older versions, missing assets, failed checks, opt-out, cache hits,
  integrity failure, stdout stability, and platform installation behavior.

**API additions and maintenance**

The latest `google.golang.org/api` release checked was **v0.299.0**, versus the
repo's v0.298.0. Its eight relevant generated API packages have no changes, so a
dependency bump alone fixes none of the identified contracts.
[Official release](https://github.com/googleapis/google-api-go-client/releases/tag/v0.299.0).

The tracked APIs contain 145 Publisher, 25 Reporting, 15 Checks, 10 Games
Configuration, 18 Games Management, 1 Custom App Publishing, 3 Integrity, and 1
Developer ID Status methods. Checks needs a reviewed revision refresh from
20260923 to 20260924; no method/schema addition was found there. Documentation
should say **568 types**, not 566. See the source-linked inventory in the API
audit for all eight discovery documents.

There is an unimplemented **Play Grouping v1alpha1** surface with two token/tag
methods outside the eight claimed families. Its persona/grouping use case is an
optional product-scope decision, not a required Play-publishing feature.
[Official discovery](https://playgrouping.googleapis.com/$discovery/rest?version=v1alpha1).

Legacy subscription cancel/defer have v2 replacements already implemented. Mark
the old commands accordingly and plan migration; their documented shutdown is
August 31, 2028, with generated-client removal after July 1, 2027. Preserve
cancellation semantics during migration.
[Official deprecation schedule](https://developer.android.com/google/play/billing/play-developer-apis-deprecations).

**Must-have improvements to the release process**

1. Expand tests around actual request bodies and side effects. The manifest
   classifies 294 leaf commands as 272 offline, 20 live, and 2 sandbox. Those
   classes are declarations, not measured behavioral coverage; some behavior
   is covered elsewhere. `internal/update` has no tests, while current
   update-command tests mostly check flags/install detection. Add the reproduced
   invariants to maintained tests using TDD before fixes.
2. Gate the tagged commit before publishing. `.github/workflows/release.yml`
   builds and publishes on a version tag without depending on test, lint, or
   security results for that commit. Require those results, verify all five
   release assets and embedded versions, and test upgrade from the last stable
   release. A successful workflow run alone does not exercise self-update.
3. Use a patched build toolchain consistently. The release workflow already pins
   Go 1.27.1 and its scan is clean for reachable vulnerabilities. The current
   `go 1.26.0` floor allows vulnerable local source builds. Document/enforce the
   patched toolchain for release builds and keep vulnerability checking as a
   release gate. [Go release/security history](https://go.dev/doc/devel/release).
4. Diff schemas and command behavior alongside method inventories. Add a check
   for fields present in discovery but absent from the SDK/adapter, and validate
   help JSON against the actual serialization path. Refreshing the schema index
   alone does not close the third-party-store field gap.
5. Make defaults and advertised diagnostics consistent. With
   `GPLAY_DEFAULT_OUTPUT=table`, `schema --list` renders a table while
   `experiments support` still emits JSON. `--debug`/`GPLAY_DEBUG=api` currently
   have no HTTP logging consumer; the only production reader suppresses the
   spinner. Either implement these advertised behaviors consistently or narrow
   their help. Keep JSON-first behavior and explicit flag precedence stable.
6. Maintain policy dates, not just presence heuristics. The billing scanner
   detects the library but does not validate its version. Billing v7's normal
   submission deadline passed August 31, 2026, with extensions to November 1.
   Version-aware checks should distinguish reliable evidence from an unknown
   version. [Official Billing version deadlines](https://developer.android.com/google/play/billing/deprecation-faq).

**Limits of this audit**

The 218 endpoint paths and all help paths were inventoried, but every possible
argument/permission/account state was not exercised. No live publishing,
purchases, refunds, real webhook delivery, or platform binary replacement was
performed. Windows behavior, account-specific API eligibility, and real Play
rollout responses still need controlled release validation. Existing tests
passing and an endpoint appearing in discovery are not evidence that those
remaining paths work.
