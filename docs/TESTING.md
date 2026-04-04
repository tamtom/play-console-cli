# Testing Guide

## General Principles

- Write tests for exported functions and user-facing command paths.
- Use table-driven tests when covering repeated endpoint or flag cases.
- Mock external API calls instead of relying on live credentials unless the test is explicitly integration-only.
- Test error cases, not just happy paths.
- Prefer a small number of high-signal tests over broad repetitive matrices.

## Running Tests

```bash
make test                  # Unit tests with race detector
make test-integration      # Integration tests (requires credentials)
make test-coverage         # Unit tests with coverage report
```

## Coverage Requirements

For each API-backed endpoint or client method, cover:
1. Success path.
2. Validation errors or rejected flag combinations.
3. API error responses.

When consolidating repetitive client tests:
- Keep grouped/table-driven coverage for repeated request wiring.
- Preserve at least one representative non-empty response assertion per response family.
- Do not replace list tests with `{"data":[]}` smoke checks only; assert decoded fields for at least one realistic payload.
- For paginated endpoints, verify `--paginate` and `--next` behavior where relevant.

For user-facing CLI commands, cover:
- Required flag validation.
- Help and usage output.
- Stdout versus stderr behavior.
- Output formatting for `json`, `table`, or `markdown` when the command supports them.
- At least one representative success case with real decoded data.

For renderer/output helpers, assert:
- Column or header structure, not just token presence.
- A representative non-empty payload.
- Stable formatting for the supported output mode.

For validation and readiness commands, cover:
- A passing case.
- A failing case with actionable error messages.
- Boundary values at documented limits.
- Any summary or status fields exposed to users.

## Test Types

### Unit Tests
Standard `_test.go` files. Run offline, no credentials needed. Use table-driven tests.

### Integration Tests
Files with `//go:build integration` tag. Require real API credentials. Also call `testutil.SkipUnlessIntegration(t)` as a safety net.

### CLI Tests
Test command-line behavior by verifying flag parsing, output format, and error messages. Use `bytes.Buffer` for capturing stdout/stderr. Help and usage output should be asserted from stderr, and JSON output should be decoded for representative success cases instead of relying on `strings.Contains` checks alone.

## Test Helpers (`internal/testutil`)

- `SkipUnlessIntegration(t)` — skip unless `GPLAY_INTEGRATION_TEST=1`
- `IsolateConfig(t)` — isolated config directory for tests
- `MockServiceAccount(t)` — fake service account JSON file
- `RequireEnv(t, key)` — skip if env var not set

## Writing New Tests

1. Use `t.Helper()` in all helper functions
2. Use `t.Cleanup()` instead of manual defer
3. Use `t.Setenv()` for environment variable changes (auto-restored)
4. Follow table-driven test pattern for multiple cases
5. Test error messages written to stderr, not just error types
6. Assert the decoded response shape for at least one realistic non-empty payload in each new client family
7. When adding a new command, include a black-box CLI test for its primary happy path and one failure path
