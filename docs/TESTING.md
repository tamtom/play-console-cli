# Testing Guide

## Running Tests

```bash
make test                  # Unit tests with race detector
make test-integration      # Integration tests (requires credentials)
make test-coverage         # Unit tests with coverage report
```

## Test Types

### Unit Tests
Standard `_test.go` files. Run offline, no credentials needed. Use table-driven tests.

### Integration Tests
Files with `//go:build integration` tag. Require real API credentials. Also call `testutil.SkipUnlessIntegration(t)` as a safety net.

### CLI Tests
Test command-line behavior by verifying flag parsing, output format, and error messages. Use `bytes.Buffer` for capturing stdout/stderr.

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
