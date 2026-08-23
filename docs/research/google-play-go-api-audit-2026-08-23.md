# Google Play official Go API and CLI coverage audit (2026-08-23)

## Executive summary

The repository is already unusually broad, but it is not fully current:

- `google.golang.org/api` is pinned to `v0.290.0`; the latest tagged first-party release is `v0.293.0` (published 2026-08-11). The repository should upgrade to `v0.293.0` and run `go mod tidy`.
- `google.golang.org/protobuf` is pinned to `v1.36.11`; `v1.36.12` is current. `golang.org/x/oauth2 v0.36.0` is already current.
- The checked-in Android Publisher discovery snapshot is revision `20260212` with 136 methods. The live official v3 discovery document is revision `20260820` with 145 methods: 12 additions and 3 removed legacy subscription methods.
- Across the first-party API families the CLI already imports, static public-call coverage is 177 of 214 methods (82.7%), excluding Cloud Storage because it is only an implementation detail for downloading reports. The important gaps are two Orders methods, 15 Play Developer Reporting methods, four Checks repository-scan methods, and six destructive Play Games bulk-reset methods.
- The current tagged Go module does not yet contain the two new self-hosted Play App Signing methods or the four new memory-reporting methods. Google’s `google-api-go-client` `main` branch does contain generated clients for all six, but no tagged module release contains them yet. Do not pin an untagged pseudo-version merely to obtain them; either use a small documented-REST adapter with contract fixtures or wait for the next tag.
- Eight Android Publisher methods absent from the CLI are for registered third-party app stores, not ordinary Play developers. They are official, but should live behind an explicit `third-party-store` namespace and prerequisite checks rather than appear as normal app-publishing commands.
- Two adjacent, current first-party APIs are worth adding: Play Integrity (three methods) and Android Developer ID Status (one read-only method). The latter is especially useful as a release/preflight check for the new Android developer-verification requirements.

No private Play Console RPCs, browser endpoints, account credentials, or live Google account calls were used in this audit.

## Implementation status in this branch

The audited gaps were subsequently implemented in this branch: dependencies
were upgraded, Android Publisher discovery was refreshed to revision 20260820,
Orders uses the real batch endpoint and exposes refund review, all Reporting and
Checks methods have CLI paths, all Games Management bulk resets are guarded and
available, and the two adjacent APIs (Play Integrity and Android Developer ID
Status) are integrated. The third-party-store and enterprise self-hosted-KMS
surfaces are separately scope-gated. A reviewed 218-method manifest and weekly
credential-free drift workflow now keep this inventory current.

## Primary sources and reproducibility

The canonical method inventories were fetched from Google’s documented Discovery endpoints on 2026-08-23:

- [Android Publisher v3 discovery](https://androidpublisher.googleapis.com/$discovery/rest?version=v3)
- [Play Developer Reporting v1beta1 discovery](https://playdeveloperreporting.googleapis.com/$discovery/rest?version=v1beta1)
- [Checks v1alpha discovery](https://checks.googleapis.com/$discovery/rest?version=v1alpha)
- [Games Configuration discovery](https://gamesconfiguration.googleapis.com/$discovery/rest?version=v1configuration)
- [Games Management discovery](https://gamesmanagement.googleapis.com/$discovery/rest?version=v1management)
- [Custom App Publishing discovery](https://playcustomapp.googleapis.com/$discovery/rest?version=v1)
- [Play Integrity discovery](https://playintegrity.googleapis.com/$discovery/rest?version=v1)
- [Android Developer ID Status discovery](https://androiddeveloperidstatus.googleapis.com/$discovery/rest?version=v1)
- [Google API Discovery index](https://www.googleapis.com/discovery/v1/apis)

Google documents the Android Publisher discovery URL and service endpoint in its [official REST reference](https://developers.google.com/android-publisher/api-ref/rest). Google’s [Go client repository](https://github.com/googleapis/google-api-go-client) states that these libraries are generated from Discovery documents, officially supported, and in maintenance mode.

Dependency versions were resolved reproducibly with:

```bash
go list -m -json google.golang.org/api@latest
go list -m -json golang.org/x/oauth2@latest
go list -m -json google.golang.org/protobuf@latest
```

CLI coverage was computed by walking every method in each Discovery resource tree, mapping it to its generated Go service/method chain, and finding non-test call sites under `internal/`. Composite commands count as coverage when they call the official method. The counts are a static interface audit, not a claim that every parameter or edge case of every method has a first-class flag.

## Dependency and generation status

| Component | Repository | Latest tagged | Action |
|---|---:|---:|---|
| `google.golang.org/api` | `v0.290.0` | `v0.293.0` | Upgrade now |
| `google.golang.org/protobuf` | `v1.36.11` | `v1.36.12` | Upgrade now |
| `golang.org/x/oauth2` | `v0.36.0` | `v0.36.0` | No change |

The safe update is:

```bash
go get google.golang.org/api@v0.293.0 google.golang.org/protobuf@v1.36.12
go mod tidy
```

Do not independently pin newer transitive Google auth/gRPC modules unless a concrete fix requires it. `v0.293.0` itself advances its dependency floor from `cloud.google.com/go/auth v0.20.0` to `v0.23.0`, enterprise-certificate-proxy `v0.3.18` to `v0.3.20`, gRPC `v1.82.0` to `v1.83.0`, and the August 2026 `genproto` revision. Let Minimal Version Selection and `go mod tidy` resolve those together.

### Generated-client lag by relevant package

| API package | Repo `v0.290.0` | Tagged `v0.293.0` | Live discovery | Difference after upgrade |
|---|---:|---:|---:|---|
| Android Publisher v3 | rev `20260717`, 141 methods | rev `20260723`, 143 | rev `20260820`, 145 | App Store Catalog becomes available; two App Signing methods still lag |
| Play Developer Reporting v1beta1 | rev `20260709`, 21 | same | rev `20260820`, 25 | Four memory methods still lag |
| Checks v1alpha | rev `20251112`, 15 | same | rev `20260820`, 15 | No method drift |
| Games Configuration | rev `20251003`, 10 | same | rev `20260813`, 10 | No method drift |
| Games Management | rev `20251003`, 18 | same | rev `20260813`, 18 | No method drift |
| Custom App Publishing | rev `20211022`, 1 | same | rev `20260820`, 1 | No method drift |
| Cloud Storage JSON v1 | rev `20260625`, 81 | same | rev `20260805`, 87 | Six cache/managed-folder methods lag, but are irrelevant to report download |

Only Android Publisher changed among these generated packages between `v0.290.0` and `v0.293.0`. Google’s current `main` branch has Android Publisher revision `20260817` with all 145 current methods and Play Developer Reporting revision `20260813` with all 25 current methods. That proves a generated client exists upstream, but a tagged release remains preferable for this CLI.

## Android Publisher v3

### Discovery drift

Compared with `docs/api/discovery.json` (revision `20260212`, 136 methods), the live official document has these additions:

1. `applications.tracks.releases.list`
2. `orders.reviewrefund`
3. Six `appstoreappsreview` methods
4. Two `appstorecatalog` methods
5. `appsigning.enrollApp`
6. `appsigning.rotateAppSigningKey`

The live document removed three legacy v1 subscription-purchase methods:

- `purchases.subscriptions.get`
- `purchases.subscriptions.refund`
- `purchases.subscriptions.revoke`

The CLI does not call those removed endpoints: its compatibility `purchases subscriptions get` and `revoke` commands already delegate to `subscriptionsv2`. The snapshot should nevertheless be refreshed so future drift reports are truthful.

### Coverage

The CLI calls 133 of 145 current Android Publisher methods. The 12 missing methods are:

| Method | Tagged client? | Intended caller | Recommendation |
|---|---|---|---|
| `orders.batchget` | Yes, including current pin | Ordinary Play developer | Implement immediately; the current command incorrectly loops over `orders.get` |
| `orders.reviewrefund` | Yes, including current pin | Ordinary Play developer | Implement with explicit request JSON, validation, and `--confirm` |
| `appsigning.enrollApp` | Not in a tag; generated on upstream `main` | Enterprise with self-hosted Cloud KMS | Add later under `app-signing kms`; plan + double confirmation; mocked tests only |
| `appsigning.rotateAppSigningKey` | Not in a tag; generated on upstream `main` | Enterprise with self-hosted Cloud KMS | Same; never present as standard key rotation |
| Six `appstoreappsreview.*` methods | Yes | Registered third-party app store | Scope-gated optional namespace only |
| `appstorecatalog.recentUpdateEvents.list` | Yes in `v0.293.0` | Registered third-party app store | Scope-gated optional namespace only |
| `appstorecatalog.recentAppViews.get` | Yes in `v0.293.0` | Registered third-party app store | Scope-gated optional namespace only |

Google’s [Orders reference](https://developers.google.com/android-publisher/api-ref/rest/v3/orders) documents a real batch endpoint with up to 1,000 order IDs per call. The present `gplay orders batch-get` implementation says the API has no batch method and performs one `orders.get` call per ID; this wastes quota and changes failure semantics. Replace it with `service.API.Orders.Batchget(pkg).OrderIds(ids...)`. The same resource now includes the official [refund-review endpoint](https://developers.google.com/android-publisher/api-ref/rest/v3/orders/reviewrefund).

The new App Signing APIs are not general Play App Signing automation. Google’s [enrollment reference](https://developers.google.com/android-publisher/api-ref/rest/v3/appsigning/enrollApp) warns that it is strictly for enterprise organizations required to retain key custody in a self-hosted Google Cloud KMS key; normal Google-managed enrollment still cannot be done via API. The [rotation reference](https://developers.google.com/android-publisher/api-ref/rest/v3/appsigning/rotateAppSigningKey) likewise applies only to apps already using self-hosted KMS. These methods are official but operationally irreversible/high impact, so they should never be live-tested against a personal app.

The eight `appstore*` methods are also official but belong to a different product audience. Google’s [App Store Review guide](https://developers.google.com/android-publisher/app-store-review) says it is for third-party app stores registered through the Third-party app store on Play program. Google’s [Play Catalog guide](https://developers.google.com/android-publisher/play-catalog) says its two methods are for registered third-party app stores and only expose recent eligible-app changes. They should not inflate ordinary Play-developer parity claims.

## Play Developer Reporting v1beta1

The live API has 25 methods; the CLI calls 10. Missing:

- `apps.fetchReleaseFilterOptions`
- all ten metric-set `get` methods (ANR, crash, error count, excessive wakeup, LMK, slow rendering, slow start, stuck background wakelock, anonymous RSS/swap memory, bitmap memory)
- `vitals.errors.counts.query`
- `vitals.lmkrate.query`
- `vitals.anonrssandswapmemoryusage.query`
- `vitals.bitmapmemoryusage.query`

The first 11 gaps above, excluding the two new memory resources and their queries, are already available in the tagged client. A good CLI shape is:

```text
gplay vitals release-filters
gplay vitals describe --metric-set <name>
gplay vitals performance lmk ...
gplay vitals errors counts ...
```

Then add `memory-anon-rss-swap` and `memory-bitmap` once a tagged generated client arrives, or through a narrow official-REST adapter. The [current Reporting reference](https://developers.google.com/play/developer/reporting/reference/rest) lists all 25 methods. The [anonymous RSS/swap resource](https://developers.google.com/play/developer/reporting/reference/rest/v1beta1/vitals.anonrssandswapmemoryusage) and [bitmap-memory resource](https://developers.google.com/play/developer/reporting/reference/rest/v1beta1/vitals.bitmapmemoryusage) were updated on 2026-08-19 and require only the normal read-only app-information permission.

All these commands can be contract-tested against fixtures and `httptest`; no production app is necessary.

## Checks v1alpha

The live API and tagged Go client both have 15 methods. The CLI implements 11. The four missing methods form one coherent resource family:

- `accounts.repos.scans.generate`
- `accounts.repos.scans.get`
- `accounts.repos.scans.list`
- `accounts.repos.operations.get`

Google’s [Checks REST reference](https://developers.google.com/checks/reference/rest) documents these methods, and the [repository-scan resource](https://developers.google.com/checks/reference/rest/v1alpha/accounts.repos.scans) describes source privacy findings and SCM metadata. Add them as `gplay checks repo-scans ...`, but treat `generate` carefully: it uploads selected source/code-attribution content. Require explicit input selection, show an upload manifest in `--dry-run`, reject secrets/binary files, and document exactly what leaves the machine.

## Play Games APIs

The CLI implements all 10 Games Services Publishing (Configuration) methods and 12 of 18 Games Management methods. The six missing methods are:

- `achievements.resetAllForAllPlayers`
- `achievements.resetMultipleForAllPlayers`
- `events.resetAllForAllPlayers`
- `events.resetMultipleForAllPlayers`
- `scores.resetAllForAllPlayers`
- `scores.resetMultipleForAllPlayers`

They are supported by the current tagged Go client but are globally destructive tester-data operations. If completeness is the goal, add them only with `--confirm`, an explicit application ID, a dry-run effect description, and mocked tests. They are lower priority than the read/reporting and Orders gaps.

## Custom App Publishing and Cloud Storage

The CLI covers the sole Custom App Publishing method (`accounts.customApps.create`). Google documents that this API creates permanently private managed-Play apps for enterprise customers in its [Custom App Publishing guide](https://developers.google.com/android/work/play/custom-app-api).

Cloud Storage is not a product surface of `gplay`; it is used with the read-only storage scope to list and download financial/statistics reports. The helper correctly uses `storage.objects.list` and `storage.objects.get`. The other 85 current Storage methods, including six methods newer than the tagged generated client, are intentionally out of scope. Reimplementing `gsutil`/`gcloud storage` would not improve Google Play parity.

## Adjacent official APIs not yet imported

### High-value additions

1. **Android Developer ID Status v1 — one read-only method.** `androiddeveloperidstatus.packages.packageRegistrationStatus.check` is present in `google.golang.org/api v0.293.0`. Google explicitly designed it for IDEs and CI/CD to verify whether a package and optional SHA-256 signing-certificate fingerprint are registered to a verified developer. Add `gplay verification status --package ... [--certificate-fingerprint ...]`, and integrate it into preflight as an opt-in network check. The [official guide](https://developer.android.com/developer-verification/guides/check-registration-status) says it uses an API key and has a default quota of 1,000 requests/day.
2. **Play Integrity v1 — three methods.** The tagged client exposes `decodeIntegrityToken`, `decodePcIntegrityToken`, and beta `deviceRecall.write`. `gplay integrity decode`/`decode-pc` would be useful backend diagnostics. Device Recall should be separately gated because it mutates per-device anti-abuse state and Google restricts its use to security, fraud, and abuse prevention; see the [official Device Recall guide](https://developer.android.com/google/play/integrity/device-recall).

### Scope-gated/excluded

- **Play Grouping v1alpha1 (two methods):** player/persona tagging, requiring a user-specific Play Games token. This is an application-backend concern, not Play Console administration; do not add unless the CLI deliberately broadens into runtime Play Games tooling.
- **Google Play Games Services API v1:** predominantly player-context/runtime methods. Continue limiting `gplay games` to the Configuration and Management admin APIs.
- **Google Play EMM, Android Management, and device provisioning APIs:** enterprise fleet/EMM products, not Play developer-account administration.
- **Android Developer Console API:** Google’s verification guide says this can register package names and keys, but as of this audit it is absent from Google’s public Discovery index and from `google.golang.org/api v0.293.0`. Do not guess endpoints or automate it until Google publishes a canonical REST reference/discovery document. It does not create a Google Play store listing or replace the manual first-app Play Console workflow.

## Ranked implementation plan

### P0 — do next

1. Upgrade `google.golang.org/api` to `v0.293.0`, `protobuf` to `v1.36.12`, tidy, run the full race/lint/build suite, and refresh `docs/api/discovery.json` to the live 145-method document.
2. Replace the fake `orders batch-get` loop with `Orders.Batchget`; add validation for non-empty unique IDs and the official 1,000-ID ceiling.
3. Add `orders review-refund` with typed JSON input, pending-refund-token validation, `--confirm`, dry-run coverage, and mocked HTTP tests.
4. Complete tagged Reporting coverage: release filters, generic metric-set descriptors, LMK queries, and error-count queries.
5. Add read-only Android Developer ID Status to preflight/verification.

### P1 — high-value official coverage

6. Add the two new memory metric queries/descriptors after the next tagged Go API release (or via a deliberately small documented-REST adapter).
7. Add Checks repository scans, with a local upload manifest and aggressive source/secret safety controls.
8. Add Play Integrity decode commands; keep beta Device Recall in a separate guarded command.
9. Add a scheduled, read-only Discovery drift CI job that records method additions/removals and generated-client availability. This job needs no Google account.

### P2 — specialized completeness

10. Add the six Games “all players”/“multiple for all players” resets with explicit destructive-operation safeguards.
11. Add self-hosted-KMS App Signing only as an enterprise namespace with plan/confirm/receipt semantics and no personal-account integration test.
12. Add registered third-party-store commands only if the project intentionally supports that audience; keep auth, docs, capabilities, and command namespace separate from normal Play publishing.

## Testing boundary

All proposed features can start with offline unit/CLI tests and `httptest` contract fixtures generated from the official discovery documents. Live integration coverage should use only documented endpoints and dedicated test resources. Specifically:

- Orders reads may use a dedicated integration fixture; refund-review should remain mocked unless a disposable chargeback-test fixture exists.
- App Signing must never be tested on a personal or ordinary production app.
- Games global resets require a dedicated Play Games test project even though the API is official.
- Checks repository generation must use synthetic source fixtures.
- Reporting, discovery drift, schema generation, help, validation, and output tests require no account at all.
