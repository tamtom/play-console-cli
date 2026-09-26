# gplay 1.0.0 official API audit — 2026-09-26

## Release assessment

No missing endpoint was found within the eight API families already claimed by
gplay: all **218 current method IDs** have implementation paths. That is endpoint
coverage, not complete request-field fidelity or a guarantee that every command
works against Google. Two subscription help examples are invalid, an unsupported
subscription-archive operation is presented as usable, and the third-party-store
adapter silently loses newly documented fields. These deserve correction before
declaring the existing surface stable.

The current Google API Go dependency is one release behind, but upgrading alone
does **not** fix these findings: the eight relevant generated API descriptions are
identical in `v0.298.0` and `v0.299.0`.

This audit used public first-party discovery documents, official reference and
deprecation pages, local source inspection, existing hermetic tests, and offline
serialization reproductions. No authenticated Google requests or account
mutations were made. Production code and reviewed API snapshots were not changed.

## Current inventories and sources

Each discovery document below was fetched three times using the repository's
existing fetch policy; the greatest observed revision was compared with the
reviewed manifest and embedded schema. Dates below are observed discovery
revision identifiers, not inferred launch dates.

| API and official discovery source | Reviewed revision | Observed revision | Methods | Types |
|---|---:|---:|---:|---:|
| [Android Publisher v3](https://androidpublisher.googleapis.com/$discovery/rest?version=v3) | 20260924 | 20260924 | 145 | 395 |
| [Play Developer Reporting v1beta1](https://playdeveloperreporting.googleapis.com/$discovery/rest?version=v1beta1) | 20260924 | 20260924 | 25 | 58 |
| [Checks v1alpha](https://checks.googleapis.com/$discovery/rest?version=v1alpha) | 20260923 | 20260924 | 15 | 64 |
| [Games Configuration v1configuration](https://gamesconfiguration.googleapis.com/$discovery/rest?version=v1configuration) | 20260907 | 20260907 | 10 | 10 |
| [Games Management v1management](https://gamesmanagement.googleapis.com/$discovery/rest?version=v1management) | 20260907 | 20260907 | 18 | 13 |
| [Custom App Publishing v1](https://playcustomapp.googleapis.com/$discovery/rest?version=v1) | 20260924 | 20260924 | 1 | 2 |
| [Play Integrity v1](https://playintegrity.googleapis.com/$discovery/rest?version=v1) | 20260924 | 20260924 | 3 | 25 |
| [Android Developer ID Status v1](https://androiddeveloperidstatus.googleapis.com/$discovery/rest?version=v1) | 20260924 | 20260924 | 1 | 1 |
| Total | | | **218** | **568** |

All method IDs match, and every observed endpoint representation and type
definition matches the embedded index when processed through the repository's
schema generator. The sole inventory discrepancy is the Checks revision number;
its methods and schemas are unchanged. Review and refresh that revision before
the release gate. The actual type count is 568; [the coverage document](../api/official-api-coverage.md#L23)
and [README](../../README.md#L133) still say 566.

A static resource/method-chain scan found 201 direct generated-client call paths.
The remaining 17 are accounted for by the two documented REST App Signing
adapters in [client.go](../../internal/appsigningclient/client.go#L86) and the
generic Reporting metric-set/release-filter implementation in
[metricsets.go](../../internal/cli/vitals/metricsets.go#L22). The latter exposes
all ten metric-set descriptor/query pairs plus release filters. This does not
prove every request parameter has a dedicated CLI flag.

The [official API directory](https://www.googleapis.com/discovery/v1/apis)
still marks Android Publisher v3 and Reporting v1beta1 as their preferred
versions. Google documents discovery as the machine-readable API contract in the
[Publisher REST overview](https://developers.google.com/android-publisher/api-ref/rest).

## Findings to resolve before stabilizing the affected commands

### API-1 — Subscription archive is advertised despite being unsupported

**Priority: P2; confirmed reference/implementation mismatch.**

[subscriptions.go:465](../../internal/cli/subscriptions/subscriptions.go#L465)
promises archiving without deletion, and
[line 488](../../internal/cli/subscriptions/subscriptions.go#L488) calls
`Monetization.Subscriptions.Archive`. Google's
[archive reference](https://developers.google.com/android-publisher/api-ref/rest/v3/monetization.subscriptions/archive)
explicitly states: “Deprecated: subscription archiving is not supported.”
The page reports an update date of 2025-05-21; this is not a newly announced
shutdown.

Remove the operation from the supported command surface, or retain a clearly
deprecated compatibility command that fails locally with an actionable message.
Do not imply deactivating base plans or deleting a subscription has identical
semantics. Add a CLI test asserting the guidance and absence of an outbound
archive request. If the command is removed, refresh command coverage and generated
help. The discovery manifest can continue to record Google's deprecated method
without treating its existence as evidence that it is usable.

### API-2 — Both subscription deferral examples are broken

**Priority: P2; reproduced offline.**

The v2 help uses `"deferDuration": "P7D"` in
[purchases.go:542](../../internal/cli/purchases/purchases.go#L542). The
[official v2 contract](https://developers.google.com/android-publisher/api-ref/rest/v3/purchases.subscriptionsv2/defer)
requires a protobuf duration expressed as seconds ending in `s`. Seven days is
`"604800s"`. The command passes the invalid string through to Google without
validating it.

The legacy help in
[purchases.go:767](../../internal/cli/purchases/purchases.go#L767) uses bare numbers
for the two expiry fields. The generated Go request uses `int64,string` JSON
fields, so copying this example fails during local decoding. It also says the
new expiry must be before the current billing period ends; the
[legacy API reference](https://developers.google.com/android-publisher/api-ref/rest/v3/purchases.subscriptions/defer)
requires it to be greater than the current expiry.

Observed reproduction output using the pinned SDK and protobuf package:

```text
legacy documented defer JSON: json: invalid use of ,string struct tag, trying to unmarshal unquoted value into int64
v2 documented defer duration: proto: (line 1:1): invalid google.protobuf.Duration value "P7D"
valid seven-day duration: <nil>
```

Correct the examples, quote legacy timestamps, and validate required v2 context,
etag, and duration before making a request. Exercise the actual help examples in
contract tests. Prefer the already implemented v2 command for new integrations.

### API-3 — App Store Review drops fields present in the current API schema

**Priority: P2 within the explicitly gated third-party-store namespace.**

Google added `versionCode` and `alreadyPublishedOnPlay` to `AppStoreAppActiveApkSet`.
The latter requires a version code when true. They appear in the current
[official request reference](https://developers.google.com/android-publisher/api-ref/rest/v3/appstoreappsreview/updateappstorehostedapp)
and live discovery, but neither generated dependency version examined includes
them. [appstores.go:126](../../internal/cli/appstores/appstores.go#L126)
decodes `--json` into the older generated type with ordinary `json.Unmarshal`,
which silently discards both fields.

Offline request round-trip:

```json
{"activeApks":{"activeApkSets":[{"baseApkId":"base.apk","versionCode":"42","alreadyPublishedOnPlay":true}]},"packageName":"com.example.app"}
```

becomes:

```json
{"activeApks":{"activeApkSets":[{"baseApkId":"base.apk"}]},"packageName":"com.example.app"}
```

The current discovery also defines `UpdateAppStoreHostedAppResponse.updateId`.
Both generated tags decode a response containing it to `{}`, and
[appstores.go:139](../../internal/cli/appstores/appstores.go#L139) discards the
response anyway. Return that identifier if supporting the new contract. The HTML
reference still says the response is empty, so this particular response-field
finding is based on the newer
[official discovery document](https://androidpublisher.googleapis.com/$discovery/rest?version=v3),
not on a claim that the prose documentation is synchronized.

A narrow documented-REST adapter can preserve the current request/response, or
the CLI can explicitly reject unsupported new fields until a compatible client
is available. A dependency bump to `v0.299.0` is insufficient. Add a wire-level
contract fixture that sends both optional fields and returns `updateId`.

Also observed: `booleanResponse:{"value":false}` becomes `booleanResponse:{}`
through the same typed round-trip. Google's
[App Store Review guide](https://developers.google.com/android-publisher/app-store-review)
uses explicit false answers and the reference labels the field required. Preserve
the user's explicit scalar values in the adapter and test them. The serialization
loss is confirmed; no live API rejection or distinct server treatment of an
omitted default-valued boolean was established in this audit.

These APIs target registered third-party stores, as the guide states. This is
not a missing capability for ordinary Play publishing, but the affected commands
should be corrected or their limitations stated before claiming complete support.

### API-4 — Verification status ignores the shared timeout and retry settings

**Priority: P2; confirmed by call-path inspection.**

[verification.go:58](../../internal/cli/verification/verification.go#L58) creates
the service and passes the original context directly to `Check`.
[developeridclient/service.go:34](../../internal/developeridclient/service.go#L34)
creates the Google client directly with an API key; it does not load gplay config
or attach the shared retry transport. Therefore configured `GPLAY_TIMEOUT`,
`GPLAY_TIMEOUT_SECONDS`, `GPLAY_MAX_RETRIES`, and `GPLAY_RETRY_DELAY` do not apply
to this GET path. A caller-supplied context deadline still applies.

This differs from
[shared.ContextWithTimeout](../../internal/cli/shared/shared.go#L135) and the
[Reporting client retry setup](../../internal/reportingclient/service.go#L79).
Apply the normal config/deadline/retry policy while retaining this API's API-key
authentication. Add a slow fake-server deadline test and a transient-GET retry
test. The current verification tests check path, fingerprint, key selection, and
API error propagation, but not those settings.

## Deprecations and migration planning

The [official Play Developer API deprecation schedule](https://developer.android.com/google/play/billing/play-developer-apis-deprecations),
updated 2026-09-01, provides concrete dates:

| Surface | Current repository use | Official timing | Release action |
|---|---|---|---|
| Legacy subscription `get`, `refund`, `revoke`; `SubscriptionPurchaseV2.latestOrderId` | Compatibility `get`/`revoke` use v2; no generated calls to the removed methods found | Removed from client libraries after 2026-07-01; API shutdown 2027-08-31 | Keep the existing migration; do not restore old endpoints |
| Legacy subscription `cancel`, `defer` | Still called at purchases.go lines 738 and 811; v2 alternatives already exist | Deprecated 2026-05-19; removed from client libraries after 2027-07-01; shutdown 2028-08-31 | Mark legacy help deprecated and direct new callers to v2; schedule compatibility migration |
| `Order.lineItems.subscriptionDetails.offer_phase` | No application logic using the legacy generated field found | Same 2026–2028 window | Prefer `offer_phase_details` in examples and future code |

These dates do not make legacy cancel/defer a current service-shutdown blocker.
They do make it unwise to present them as the preferred stable interface.

Cancellation semantics must be preserved deliberately during migration. The
[legacy cancel reference](https://developers.google.com/android-publisher/api-ref/rest/v3/purchases.subscriptions/cancel)
says its unspecified cancellation type defaults to developer-requested stopping
of payments, which prevents restoration and cancels remaining installment
payments. User-requested stopping of renewals has different behavior. Existing
[legacy cancel help](../../internal/cli/purchases/purchases.go#L707) does not
explain that distinction. Do not silently switch its behavior when routing it to
v2; expose/document the cancellation type and add a request-body test.

## Dependency freshness and nonblocking drift

The pin is [google.golang.org/api v0.298.0](../../go.mod#L11).
`go list -m -json google.golang.org/api@latest` resolved `v0.299.0` with timestamp
`2026-09-21T17:48:13Z`. This agrees with Google's
[v0.299.0 release](https://github.com/googleapis/google-api-go-client/releases/tag/v0.299.0)
and tagged commit `38da4c87c2f63f47f7167d3b43bc6ec6423c19a8`.

The browser's cached `/releases/latest` initially returned v0.297.0; the explicit
tag page and module resolution were used to avoid reporting that stale result.
No future release date was assumed.

Comparing each package's tagged `*-api.json` found no differences at all between
v0.298.0 and v0.299.0 for the eight audited APIs. Bumping the dependency is
reasonable maintenance, not an API blocker fix. Comparing the pin with live
discovery, excluding descriptions, found only the App Store Review fields above
and a deobfuscation media-upload maximum increasing from 1,677,721,600 to
2,097,152,000 bytes. The CLI upload path does not enforce the older maximum, so
no blocked supported upload was established. The
[official upload reference](https://developers.google.com/android-publisher/api-ref/rest/v3/edits.deobfuscationfiles/upload)
and live discovery own the current limit.

## Missing APIs outside the declared surface

The discovery directory also exposes the
[Play Grouping v1alpha1 API](https://playgrouping.googleapis.com/$discovery/rest?version=v1alpha1),
revision 20260924, with `apps.tokens.verify` and
`apps.tokens.tags.createOrUpdate`. No gplay command covers it. Its described
workflow verifies an app/persona token and manipulates tags for the represented
user; it is not a Play publishing/catalog/reporting workflow. Treat this as an
optional product-scope decision, not a 1.0 publishing blocker or evidence that the
claimed eight-family inventory is incomplete. No public launch date or general
eligibility conclusion was established.

The directory also lists the runtime Games API and Android Enterprise/Management
APIs. Full runtime player operations and enterprise device management are separate
product areas; adding all of them is not required for this release. Custom App
Publishing remains a one-method API; its
[official guide](https://developers.google.com/android/work/play/custom-app-api)
describes permanently private enterprise apps. Its presence does not supply
initial public Play app creation.

## Improvements to API maintenance

1. **Check request and response contracts, not just method counts.**
   [check-api-drift.py:83](../../scripts/check-api-drift.py#L83) compares method IDs
   and revisions. It flags revision-only changes, which is useful, but does not
   explain schema differences or prove the SDK supports the newly reviewed
   fields. Add a structural schema comparison and an explicit record of fields
   requiring adapters. The App Store Review gap is a concrete example of why a
   refreshed embedded schema is not the same as a refreshed implementation.
2. **Preserve or reject unsupported JSON fields deliberately.**
   [LoadJSONArg](../../internal/cli/shared/json.go#L12) silently drops unknown
   fields when its destination is a typed SDK request. Help claiming that
   `--json` automatically allows new Google fields should be narrowed or backed
   by a raw/presence-preserving adapter. Tests should compare final HTTP JSON,
   especially for explicit false, zero, empty collections, and newly added fields.
3. **Validate runnable help examples against the actual request contract.**
   The two deferral examples demonstrate failures that ordinary happy-path
   command tests missed. For generic metric queries, provide a complete dated
   timeline example and document page-token handling; the current generic query
   has no `--paginate` flag despite paginated API responses.
4. **Maintain precise coverage claims.** Count current types automatically and
   distinguish supported ordinary publishing, deprecated entries, restricted
   namespaces, and optional API families. Keep the weekly
   [credential-free drift workflow](../../.github/workflows/api-drift.yml#L3),
   and review its findings before updating snapshots.

## Validation performed

- Fetched the eight official discovery documents and the Google API directory.
- Compared all 218 method IDs, normalized endpoint definitions, and 568 complete
  schema definitions with the reviewed files.
- Compared relevant package discovery files in the pinned and latest tagged Go
  modules, then compared the pin with live discovery.
- Verified endpoint call paths statically, including the custom REST adapters.
- Reproduced the two invalid deferral examples and third-party-store JSON
  round-trip loss locally with the pinned dependencies.
- `python3 scripts/gen-schema-index.py --check`: passed, 218 endpoints/568 types.
- `go test ./internal/cli/purchases ./internal/cli/subscriptions ./internal/cli/appstores ./internal/cli/verification ./internal/developeridclient ./internal/cli/vitals ./internal/appsigningclient`:
  passed; `internal/developeridclient` has no tests of its own.

Passing existing tests does not negate the contract findings. Live behavior,
permission prerequisites, and destructive-operation semantics were not exercised
against Google. Release/update, rollout, global dry-run, audit-log redaction, and
general command-help findings are covered by the separate repository audit.
