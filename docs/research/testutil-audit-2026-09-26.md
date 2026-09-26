# Dormant test utility audit

Scope: one cleanup of unused scaffolding in `internal/testutil`. Existing release
work in the checkout is outside this change. Discovery was read-only; the
candidate evidence below was recorded before deleting source or tests.

## Candidates and actual proof

Locations refer to the files before this cleanup.

| Test | Location | Failure it can detect |
| --- | --- | --- |
| `TestNewMockAPI_BaseURL` | `internal/testutil/mockapi_test.go:12` | An empty or non-HTTP URL from the unused mock server. |
| `TestMockAPI_HandlerRouting` | `internal/testutil/mockapi_test.go:23` | Failure to return the status and JSON supplied by the test. |
| `TestMockAPI_UnmatchedRoute_Returns404` | `internal/testutil/mockapi_test.go:48` | The mock's fallback status changes. |
| `TestMockAPI_RequestLog` | `internal/testutil/mockapi_test.go:62` | The mock stops recording the supplied method/path. |
| `TestMockAPI_MultipleRequests` | `internal/testutil/mockapi_test.go:87` | The mock loses or reorders its three supplied paths. |
| `TestHandleError` | `internal/testutil/mockapi_test.go:111` | The mock loses its supplied error status/message. |
| `TestFixtures_NotNil` | `internal/testutil/mockapi_test.go:141` | Unused fixture maps become nil or cannot marshal; no API schema is checked. |
| `TestFixtures_WithHandleJSON` | `internal/testutil/mockapi_test.go:175` | Supplied response statuses or log counts change; fixture bodies are never asserted. |
| `TestSkipUnlessIntegration_Skips` | `internal/testutil/testutil_test.go:9` | None: it unsets an environment variable without calling the helper. |
| `TestSkipUnlessIntegration_Runs` | `internal/testutil/testutil_test.go:17` | A panic/failure on the enabled path; an incorrect skip still passes the suite. |
| `TestSkipUnlessIntegration_RunsTrue` | `internal/testutil/testutil_test.go:23` | The same limited proof for the alternate enabled value. |
| `TestIsolateConfig` | `internal/testutil/testutil_test.go:28` | The unused helper fails to set its returned directory; cleanup is not checked. |
| `TestRequireEnv_Skips` | `internal/testutil/testutil_test.go:64` | The unused helper fails to return a supplied value; skipping is not tested. |

## Callers, history, and deletion unlocked

- `NewMockAPI`, `MockAPI`, `RequestEntry`, `HandleJSON`, `HandleError`, and the
  ten fixture functions have no callers outside `mockapi_test.go` and their
  internal composition. They have no production consumers. Deletion removes
  `mockapi.go`, `fixtures.go`, and `mockapi_test.go` together.
- `SkipUnlessIntegration`, `IsolateConfig`, and `RequireEnv` are used only by
  the five self-tests listed above. They have no production or integration-test
  callers. Delete those helpers and tests, and correct their documentation.
- Commit `6a9a3a1` introduced the integration/config helpers and self-tests on
  February 15, 2026. Commit `e205413` added the mock server, fixtures, and
  self-tests the same day as planned API-test infrastructure. On the current
  branch, subsequent changes to these files were formatting or unrelated
  helper additions; repository-wide reference searches found no consumers.
- Commit `aa5ed94` added the actively used hermetic sandbox on August 24, 2026.
  Removing the earlier unused scaffold does not remove sandbox coverage.

## Remaining proof and retention decisions

No replacement is needed for helpers with no consumers. Actual API behavior
remains covered by `TestSandboxWorkflow_FullEditFlow`,
`TestSandboxWorkflow_ReviewsList`, `TestSandboxWorkflow_ErrorBehavior`,
`TestSandbox_*`, and the `internal/playclient/baseurl_test.go` tests. These
exercise authentication, decoded responses, listing persistence, edit commit,
errors, and the loopback credential boundary.

Keep `MockServiceAccount` and `TestMockServiceAccount`: the dry-run black-box
test `TestOnetimeproductsCreate_DryRunWiresUpsertRequest` consumes that fixture.
Keep `SandboxServiceAccount`, which supplies real local RSA signing to the
sandbox and client tests. Keep the enum helpers and tests: they support checks
against the official schema. No production owner changes are needed.

Integration tests use their own guards in `internal/playclient`,
`internal/cli/auth`, and `internal/cli/monetizationpricing`. The weekly workflow
sets the integration gates and an isolated home. Those guards remain intact.

## Risk and validation

Risk is limited to a missed internal caller or stale documentation. All build
tags were included in reference searches. PR/main CI and `make test` run the
Go suite with the race detector; the weekly workflow adds integration and live
smoke tags. No commands are added or removed, so the coverage manifest does not
change. This Go repository has no OpenClaw/Vitest runners.

Focused validation:

```sh
go test -race ./internal/testutil ./internal/sandbox ./internal/playclient ./internal/cmdtest -count=1
go test -tags=integration -run '^$' ./internal/playclient ./internal/cli/auth ./internal/cli/monetizationpricing
```

The second command compiles integration suites without contacting Play.

## Results

- Before removal, focused self-tests, retained sandbox/client tests, and the
  one-time-product dry-run test passed. This deletion adds no new behavior or
  regression tests.
- After removal, both commands above passed. `make test` passed with the race
  detector: 94 tested packages, 23 packages without tests, and 2,197 test/subtest
  passes. Some unchanged package results came from Go's test cache.
- `make lint` passed using its `go vet ./...` fallback because `golangci-lint`
  was unavailable. `make format-check` and `git diff --check` passed.
- Independent `autoreview` completed with no findings through P2. It reviewed
  an isolated local clone containing exactly this cleanup, with retained enum,
  sandbox, client, and dry-run tests supplied as context. The reviewer relied
  on the recorded repository-wide caller search for files outside that bundle.
- Production/tooling Go: no changes. Test support: 283 lines removed. Tests:
  258 lines removed, covering 13 top-level tests. The testing guide changes
  and this evidence report are separate documentation changes.
- Changes remain uncommitted. No branch, PR, push, or merge was created in the
  working repository; the pre-existing release changes were preserved.

## Follow-up

The separate bundles audit found two registration duplicates:
`TestBundlesSubcommandsIncludeAnalyze` and `TestBundlesCommand_HasSubcommands`.
`TestBundlesCommand_SubcommandNames` already checks all four names and rejects
unexpected names. Leave this for a separate cleanup; command usage-policy and
executable analyze/compare tests remain valuable.
