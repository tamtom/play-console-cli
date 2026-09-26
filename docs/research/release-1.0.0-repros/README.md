# 1.0.0 audit reproductions

Run from the repository root:

```sh
python3 docs/research/release-1.0.0-repros/run.py
```

At audited commit `5de0f33`, this intentionally fails 12 test groups. The tests
describe the expected corrected behavior and reproduce the findings in the
[release audit](../release-readiness-1.0.0-2026-09-26.md).

The runner adds temporary tests through Go's overlay flag. It does not change
production files or add failures to the normal `go test ./...` suite. The `.txt`
files contain Go source so the normal package loader ignores them.

Requests use local `httptest` servers and an injected Publisher service. RTDN
uses a fake `gcloud` executable. Configuration and download checks use temporary
files. No Google account is contacted, no webhook is sent to a real recipient,
and the currently installed CLI is never replaced.

The checks cover rollout wire payloads, validation before writes, global
dry-run, sensitive-argument redaction, invalid update downloads, root help,
initialization/configuration compatibility, the documented package environment
variable, existing RTDN topics, the current target SDK floor, and SemVer ordering.

These are audit evidence, not a replacement for production regression tests.
When fixing a finding, move its behavioral assertion into the appropriate
maintained test suite, verify it fails first, and then implement the fix.
