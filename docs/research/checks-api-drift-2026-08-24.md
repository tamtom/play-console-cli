# Checks API discovery drift review — 2026-08-24

## Conclusion

The Checks API `v1alpha` drift from revision `20260820` to `20260823` is a
revision-only metadata change. There are no method, endpoint-contract, or schema
changes, so no `gplay` production-code or test-fixture changes are required and
there is no breaking API change to call out for this drift.

The only repository follow-up is to advance the Checks revision in the manifest
and embedded index together to `20260823` so the drift gate records the review.
This investigation intentionally does not update either generated artifact.

## Evidence

Primary sources compared:

- Google's [live Checks `v1alpha` discovery document](https://checks.googleapis.com/$discovery/rest?version=v1alpha), which reported revision `20260823` during this review.
- The repository's reviewed baseline [API manifest](https://github.com/tamtom/play-console-cli/blob/36a9adea4eeebda643dfd06c584de95846e6939b/docs/api/google-play-api-manifest.json) and embedded [schema index](https://github.com/tamtom/play-console-cli/blob/36a9adea4eeebda643dfd06c584de95846e6939b/internal/apischema/schema-index.json), both at Checks revision `20260820`.
- Google's official [`google-api-go-client` `v0.293.0` Checks discovery artifact](https://github.com/googleapis/google-api-go-client/blob/v0.293.0/checks/v1alpha/checks-api.json) and [generated Go source](https://github.com/googleapis/google-api-go-client/blob/v0.293.0/checks/v1alpha/checks-gen.go). The repository pins that module version in [`go.mod`](../../go.mod).

Exact comparison results:

| Surface | Reviewed repository | Live `20260823` | Difference |
| --- | ---: | ---: | --- |
| Discovery revision | `20260820` | `20260823` | Revision string only |
| Method IDs | 15 | 15 | None added or removed |
| Indexed endpoint contracts | 15 | 15 | No differences in HTTP method, path, description, parameters, request/response types, scopes, or media-upload metadata |
| Schemas | 64 | 64 | No types added or removed; every full schema definition is identical |

As an independent completeness check, the full live discovery JSON was compared
with Google's `checks-api.json` shipped in `google.golang.org/api v0.293.0`
(revision `20251112`). Removing only the top-level `revision` property makes the
two complete documents identical; their canonical JSON has the same SHA-256,
`a6f162d1a123508f8816b6939e66443947d63d74141343d9f0d46d14181c8e9d`.
Consequently, `v0.293.0` already exposes the complete live schema—there is no new
schema waiting for a generated-client update.

## Verification

The Checks implementation, client, and embedded-schema packages remain green:

```text
go test ./internal/cli/checks ./internal/checksclient ./internal/apischema
```

No production code, tests, manifest, or index was changed as part of this review.
