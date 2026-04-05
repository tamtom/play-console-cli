## Summary
<!-- 2-4 bullet points of what was added or changed. Focus on the user-visible surface. -->

-

## Why
<!-- What problem this solves, what was broken or missing, or what user need this addresses. -->

## What Changed
<!-- Implementation approach: new commands, shared helpers extracted, API client changes, etc. -->

## Alternatives Considered
<!-- What was rejected and why. Remove this section for trivial changes. -->

N/A

## Type of Change
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Refactoring (no functional changes)
- [ ] Documentation update

Closes #<!-- issue number -->

## Implementation Checklist

### Command & Registration
- [ ] Command implementation in `internal/cli/<package>/`
- [ ] Registered in parent command group or `internal/cli/registry/registry.go`
- [ ] `UsageFunc: shared.DefaultUsageFunc` set on command and subcommands
- [ ] Required flags validated with stderr error messages (not just `flag.ErrHelp`)
- [ ] `shared.ContextWithTimeout` (or `shared.ContextWithUploadTimeout`) used for HTTP calls

### Tests
- [ ] CLI black-box tests in `internal/cmdtest/` — flag validation, error messages, output formats
- [ ] Package-level unit tests for non-trivial logic
- [ ] Error paths tested explicitly (missing required flags, invalid inputs, conflicting flags)

### Docs & Help
- [ ] `--help` output is accurate and follows existing conventions
- [ ] Command docs regenerated: `make docs`

## Validation
<!-- Run these in order. Note honestly if any step was skipped and why. -->

```bash
make format
make lint
make test                          # full suite, not just focused tests
make build                         # or: go build -o /tmp/gplay .
/tmp/gplay <new-command> --help    # verify help text renders correctly
/tmp/gplay <new-command> <args>    # smoke test (live or offline)
/tmp/gplay <new-command> <bad-flags>  # error path verification
```

- [ ] `make format` passes
- [ ] `make lint` passes
- [ ] `make test` passes (full suite)
- [ ] `make build` succeeds
- [ ] `--help` verified for new/changed commands
- [ ] Error paths verified (missing flags, bad input)
- [ ] Smoke tested against real API or with `--dry-run`

## Notes
<!-- Honest callouts: what couldn't be tested, known gaps, blocked steps. Remove if none. -->

## Screenshots (if applicable)
<!-- Command output examples, before/after comparisons. -->
