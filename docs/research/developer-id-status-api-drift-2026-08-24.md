# Android Developer ID Status API discovery drift review — 2026-08-24

## Conclusion

Android Developer ID Status API `v1` drift from revision `20260820` to
`20260823` is a revision-only metadata change. There are no method, endpoint,
parameter, response, or schema changes. No `gplay` production-code, dependency,
or drift-driven test changes are required.

The repository follow-up is only to advance this API's revision in the reviewed
manifest and embedded schema index together to `20260823`. This investigation
does not update either artifact.

## Evidence

Primary sources compared:

- Google's [live Android Developer ID Status `v1` discovery document](https://androiddeveloperidstatus.googleapis.com/$discovery/rest?version=v1),
  which reported revision `20260823` during this review.
- The repository's committed `main` baseline at `0db97553`: the reviewed
  [API manifest](https://github.com/tamtom/play-console-cli/blob/0db97553b01c2e507ebd59e20809975c4e2372fc/docs/api/google-play-api-manifest.json#L3-L9)
  and embedded [schema index metadata](https://github.com/tamtom/play-console-cli/blob/0db97553b01c2e507ebd59e20809975c4e2372fc/internal/apischema/schema-index.json#L3-L9),
  both at revision `20260820`, plus the indexed
  [endpoint](https://github.com/tamtom/play-console-cli/blob/0db97553b01c2e507ebd59e20809975c4e2372fc/internal/apischema/schema-index.json#L68-L95)
  and [response schema](https://github.com/tamtom/play-console-cli/blob/0db97553b01c2e507ebd59e20809975c4e2372fc/internal/apischema/schema-index.json#L7934-L7970).
- Google's official `google-api-go-client` `v0.293.0`
  [discovery artifact](https://github.com/googleapis/google-api-go-client/blob/v0.293.0/androiddeveloperidstatus/v1/androiddeveloperidstatus-api.json)
  and [generated Go source](https://github.com/googleapis/google-api-go-client/blob/v0.293.0/androiddeveloperidstatus/v1/androiddeveloperidstatus-gen.go).
  The repository pins that version in [`go.mod`](../../go.mod).

Exact comparison results:

| Surface | Reviewed `20260820` | Live `20260823` | Difference |
| --- | --- | --- | --- |
| Revision | `20260820` | `20260823` | Revision string only |
| Method set | One method: `androiddeveloperidstatus.packages.packageRegistrationStatus.check` | Same | None |
| Base/service path | `https://androiddeveloperidstatus.googleapis.com/`; empty service path | Same | None |
| HTTP endpoint | `GET v1/{+name}:check` | Same | None |
| Flat path | `v1/packages/{packagesId}/packageRegistrationStatus:check` | Same | None |
| Required parameter | Path `name`, pattern `^packages/[^/]+/packageRegistrationStatus$` | Same | None |
| Optional parameter | Query string `certificateFingerprint` | Same | None |
| Response | `PackageRegistrationStatus` | Same | None |
| Schemas | One type with three properties | Same full definition | None |

`PackageRegistrationStatus` remains unchanged: `name` is a string;
`certificateFingerprint` is a read-only string; and `state` is a read-only
string with the same four values: `REGISTRATION_STATE_UNSPECIFIED`, `REGISTERED`,
`NOT_REGISTERED`, and `REGISTERED_WITH_ANOTHER_CERTIFICATE_FINGERPRINT`.
Descriptions, enum descriptions, read-only markers, parameter order, and the
response reference also match exactly.

As an independent completeness check, Google's `v0.293.0` discovery artifact
reports the older revision `20260804`, but its complete discovery JSON becomes
identical to live `20260823` after removing only the top-level `revision`
property. Canonical JSON produced with `jq -cS` has the same SHA-256 for both:
`65a0e23feb7ec4fb2369c8150a1157904e79e4a50977b93c6c8cd5d8df76f797`.
The generated source already exposes `Check(name)`, the optional
`CertificateFingerprint` setter, the `GET v1/{+name}:check` request, and all
three response fields. A `google.golang.org/api` upgrade is therefore not
needed for this drift.

## Code and test impact

The existing `gplay verification status` implementation already builds the
required resource name, converts package-name dots to hyphens, calls the
generated `Check` method, and conditionally sends `certificateFingerprint`.
Existing endpoint tests cover the required API key, fingerprint validation,
official path, optional query parameter, and API-error propagation. No behavior
or fixture change is required by this revision bump.

One optional, unrelated hardening improvement remains: the unit test's success
body currently uses an unknown `packageName` property and does not assert
decoding of `name`, `certificateFingerprint`, or `state`. Replacing it with a
schema-valid response and asserting emitted JSON would improve response-contract
coverage, but it is not necessary to accept this drift.

## Verification

The relevant implementation, generated-client boundary, embedded schema, and
production-root CLI tests remain green:

```text
go test ./internal/cli/verification ./internal/developeridclient ./internal/apischema
go test ./internal/cmdtest -run 'TestOfficialParityFamilies_RunThroughProductionRoot/developer_verification|TestParityCommands_ProductionRootFailurePaths/verification' -count=1
python3 scripts/gen-schema-index.py --check
```

This review added only this research note; it did not modify the manifest,
schema index, dependency, production code, or tests.
