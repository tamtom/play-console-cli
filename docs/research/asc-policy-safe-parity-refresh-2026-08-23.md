# ASC 4.9.0 policy-safe parity refresh

Date: 2026-08-23
Gplay source baseline: `3d4cea4245a33e814f440fe9e99bc6325e072611` plus the uncommitted policy-safe `capabilities` and `bootstrap plan` work
ASC source baseline: `4.9.0`, commit `563ee9e0ba2652348a8defdd71ccb454c3aa7b60`

## Relationship to the full audit

This note supplements, and does not replace, [the full ASC audit](./asc-cli-audit-2026-08-23.md). It refreshes the next-work ranking after `gplay capabilities` and `gplay bootstrap plan` were added and applies the account-safety requirement more strictly.

The latest official GitHub release is still [ASC 4.9.0](https://github.com/rorkai/App-Store-Connect-CLI/releases/tag/4.9.0), published on 2026-08-23. Its annotated tag resolves to [`563ee9e0`](https://github.com/rorkai/App-Store-Connect-CLI/commit/563ee9e0ba2652348a8defdd71ccb454c3aa7b60).

No Google account, credentials, Play Console session, or production resource was accessed during this refresh. The only live Google input was the public Android Publisher discovery document.

## Implementation status (2026-08-24)

The policy-safe roadmap in this note is implemented on the parity branch as one
mergeable delivery. This includes test/account isolation, the lazy command
catalog and local search, embedded schema inspection and drift checks, injected
runtime/IO/clock/filesystem/audit seams, rooted symlink-safe writes, workflow
resilience, local insights, the current official API coverage refresh, and the
remaining ASC-inspired work:

- deterministic metadata/image plan, apply, receipt, reconciliation, and resume;
- a pinned per-tree-verified transactional `gplay install-skills` command;
- canonical offline submission readiness plus strict app-content inventory;
- direct-exec Gradle, signing, and screenshot helpers; and
- a truthful store-listing experiment boundary with official-API-only winner application.

No implementation uses private Play Console RPCs, cookies, authenticated
browser automation, or automated legal acceptance. Console-only lifecycle and
results remain explicit manual handoffs.

## Immediate safety correction

The first task is not a new feature. It is to close two testing paths that conflict with the owner's explicit requirement that personal or production accounts must never be used for testing.

1. Gplay's integration workflow currently runs every Monday, enables `GPLAY_MUTATING_INTEGRATION_TEST=1`, and targets `com.itdeveapps.stepsshare`. [Gplay integration workflow](https://github.com/tamtom/play-console-cli/blob/3d4cea4245a33e814f440fe9e99bc6325e072611/.github/workflows/integration.yml#L1-L39)
2. That suite creates a one-time product and only logs a cleanup failure, so a failed cleanup can leave account state behind. Other integration tests create and delete edits without requiring the separate mutating opt-in. [Monetization integration test](https://github.com/tamtom/play-console-cli/blob/3d4cea4245a33e814f440fe9e99bc6325e072611/internal/cli/monetizationpricing/integration_test.go#L18-L83), [Play client integration test](https://github.com/tamtom/play-console-cli/blob/3d4cea4245a33e814f440fe9e99bc6325e072611/internal/playclient/integration_test.go#L12-L73)
3. Black-box command tests isolate `HOME` and the config path, but child processes inherit the rest of the developer environment through `cmd.Environ()`. They do not centrally clear `GPLAY_SERVICE_ACCOUNT`, `GPLAY_SERVICE_ACCOUNT_JSON`, package defaults, or integration flags. [Gplay cmdtest runner](https://github.com/tamtom/play-console-cli/blob/3d4cea4245a33e814f440fe9e99bc6325e072611/internal/cmdtest/cmdtest.go#L22-L36), [Gplay cmdtest setup](https://github.com/tamtom/play-console-cli/blob/3d4cea4245a33e814f440fe9e99bc6325e072611/internal/cmdtest/test_main_test.go#L11-L25)

ASC provides the safer reference point here: its live integration workflow is manual-only, build-tagged, and environment-gated, while its repository instructions require a disposable app for necessary mutations. [ASC integration workflow](https://github.com/rorkai/App-Store-Connect-CLI/blob/563ee9e0ba2652348a8defdd71ccb454c3aa7b60/.github/workflows/integration.yml), [ASC testing instruction](https://github.com/rorkai/App-Store-Connect-CLI/blob/563ee9e0ba2652348a8defdd71ccb454c3aa7b60/AGENTS.md#L103-L107)

Recommended acceptance boundary:

- Remove the scheduled live run and never enable mutation by default.
- Scrub all credential, package, project, and integration variables from unit and black-box child environments; tests may add an explicit mock value afterward.
- Add a test-only authentication bypass that fails closed if a test unexpectedly requests real credentials.
- Keep normal CI completely offline with `httptest` or injected transports.
- If live tests are ever restored, require a manual dispatch and a clearly identified disposable app in a separate non-personal developer account. Production and personal apps remain forbidden.

## Ranked policy-safe parity opportunities

| Rank | Workstream | Value | Scope |
|---:|---|---|---|
| 0 | Test/account isolation | Prevents accidental personal or production access | Small |
| 1 | Lazy command catalog plus `gplay search` | Makes the large CLI discoverable to humans and agents | Medium |
| 2 | `gplay schema` plus API drift CI | Keeps official endpoint coverage current and inspectable | Medium |
| 3 | Finish runtime and I/O injection | Makes all command paths safely mockable | Large, incremental |
| 4 | Rooted, symlink-safe filesystem module | Protects local metadata, reports, workflows, and secrets | Medium |
| 5 | Generic plan/apply/receipt/resume contract | Adds reviewable and tamper-evident mutation safety | Large |
| 6 | Official endpoint refresh | Delivers immediate API-backed features without Console automation | Small to medium |
| 7 | Workflow timeout/retry/resume hardening | Avoids unsafe replay after ambiguous failures | Medium |
| 8 | `gplay insights` | Turns existing official data into useful trends | Medium |
| 9 | Verified skills installer and local Android helpers | Improves onboarding and end-to-end ergonomics | Medium |

### 1. Lazy catalog and local command search

ASC's root registry stores lightweight name/help/factory records, materializes only the selected command, and exposes the same catalog to search and completion. Its `search` command locally indexes command paths, summaries, usages, examples, and flags, then returns deterministic ranked JSON. [ASC lazy registry](https://github.com/rorkai/App-Store-Connect-CLI/blob/563ee9e0ba2652348a8defdd71ccb454c3aa7b60/internal/cli/registry/registry.go#L104-L246), [ASC search implementation](https://github.com/rorkai/App-Store-Connect-CLI/blob/563ee9e0ba2652348a8defdd71ccb454c3aa7b60/internal/cli/search/search.go#L29-L168)

Gplay still eagerly constructs every root command in one slice. [Gplay registry](https://github.com/tamtom/play-console-cli/blob/3d4cea4245a33e814f440fe9e99bc6325e072611/internal/cli/registry/registry.go#L82-L164)

Recommended slice:

- Introduce one `CommandSpec` catalog containing path, summary, canonical intents, stability, provider, and factory.
- Implement `gplay search <intent>` with JSON-first deterministic output and no network access.
- Generate root help, docs, and completions from that same catalog so they cannot drift.
- Join capability records to command records by stable intent ID rather than duplicating strings.

### 2. Runtime schema inspection and discovery drift

ASC embeds a generated endpoint schema index and exposes it through `asc schema`; its documentation check regenerates both the path and schema indexes and fails when they drift. [ASC schema command](https://github.com/rorkai/App-Store-Connect-CLI/blob/563ee9e0ba2652348a8defdd71ccb454c3aa7b60/internal/cli/schema/schema.go), [ASC schema generator](https://github.com/rorkai/App-Store-Connect-CLI/blob/563ee9e0ba2652348a8defdd71ccb454c3aa7b60/scripts/generate-schema-index.py), [ASC generated-artifact checks](https://github.com/rorkai/App-Store-Connect-CLI/blob/563ee9e0ba2652348a8defdd71ccb454c3aa7b60/Makefile#L178-L228)

Gplay already checks in Google's discovery document and a 136-method endpoint index, but refresh is manual. The official live discovery document, revision `20260820`, contains 145 methods. Compared with the checked-in snapshot, it adds 12 method IDs and removes three deprecated subscription methods. [Gplay checked-in discovery snapshot](https://github.com/tamtom/play-console-cli/blob/3d4cea4245a33e814f440fe9e99bc6325e072611/docs/api/discovery.json), [official live discovery document](https://androidpublisher.googleapis.com/$discovery/rest?version=v3)

Recommended slice:

- Add `gplay schema` to query method ID, HTTP path, parameters, request type, response type, and scopes locally.
- Add a generator `--check` mode and CI failure on checked-in discovery/index inconsistency.
- Add a separate scheduled read-only drift report that opens no credentials and never updates code automatically.
- Surface newly added, removed, implemented, and unsupported methods through `gplay capabilities`.

### 3. Complete runtime and output injection

The current Gplay runtime migration is incomplete: this refresh found 42 production CLI files that still call `playclient.NewService` directly and 266 production references to process-global output functions or streams. That makes it harder to prove a test cannot escape to real credentials or the network.

Recommended slice:

- Inject API factories, transport, stdout, stderr, clock, filesystem, and audit sink through one runtime.
- Keep command packages as flag/output adapters; put orchestration in deep domain modules.
- Make the default production runtime the only place that resolves credentials.
- Give tests a fail-closed runtime whose transport errors on every unregistered request.

ASC is not perfect dependency injection, but it centralizes normal client creation and supports custom HTTP clients for API and upload tests. [ASC client constructors](https://github.com/rorkai/App-Store-Connect-CLI/blob/563ee9e0ba2652348a8defdd71ccb454c3aa7b60/internal/asc/client_core.go#L624-L688), [ASC upload transport option](https://github.com/rorkai/App-Store-Connect-CLI/blob/563ee9e0ba2652348a8defdd71ccb454c3aa7b60/internal/asc/upload.go#L55-L61)

### 4. Rooted, symlink-safe filesystem operations

ASC centralizes operations on caller-selected trees behind a rooted filesystem abstraction that validates relative paths and rejects traversal and unsafe symlink resolution. It backs this with attack-oriented tests. [ASC rooted filesystem](https://github.com/rorkai/App-Store-Connect-CLI/blob/563ee9e0ba2652348a8defdd71ccb454c3aa7b60/internal/rootfs/rootfs.go), [ASC rooted filesystem tests](https://github.com/rorkai/App-Store-Connect-CLI/tree/563ee9e0ba2652348a8defdd71ccb454c3aa7b60/internal/rootfs)

Gplay should introduce the same boundary for metadata trees, screenshots, report downloads, migration output, workflow state, and plan/receipt files. Pair it with create-only or atomic replacement policies, restrictive permissions for sensitive state, and tests for `..`, absolute paths, symlink swaps, destination races, and cross-platform behavior.

### 5. Reviewable mutation plans and resumable receipts

Gplay already has global dry-run, workflows, release validation, and the new non-executing bootstrap plan. The next parity step is not another workflow language; it is a shared mutation contract.

ASC has a deterministic non-mutating diff surface, while its distribution workflow persists a canonical SHA-256-sealed plan, verifies it before use, records exact effects, and supports stateful resume without duplicating completed work. [ASC diff surface](https://github.com/rorkai/App-Store-Connect-CLI/blob/563ee9e0ba2652348a8defdd71ccb454c3aa7b60/internal/cli/diffcmd/diff.go), [ASC plan sealing and verification](https://github.com/rorkai/App-Store-Connect-CLI/blob/563ee9e0ba2652348a8defdd71ccb454c3aa7b60/internal/cli/distribute/orchestration_state.go#L431-L499), [ASC resume idempotency test](https://github.com/rorkai/App-Store-Connect-CLI/blob/563ee9e0ba2652348a8defdd71ccb454c3aa7b60/internal/cli/distribute/orchestration_lease_test.go#L150-L256)

Recommended first domain: metadata and listing images, because current `sync diff-listings` already supplies much of the comparison logic. Then extend the contract to monetization and releases.

Every plan should include provider=`official-api`, package, normalized effects, destructive count, artifact/content hashes, expiry, capability IDs, and a canonical plan hash. `apply` must reject changed inputs or remote preconditions and emit a receipt. It must never fall back to private Console automation.

### 6. Close current official API gaps

These are policy-safe because they use the documented Android Publisher API:

- Replace `gplay orders batch-get`'s one-request-per-order loop with the real `orders.batchget` method. The current comment incorrectly says the API has no true batch operation; Google documents up to 1,000 distinct order IDs per call. [Current Gplay implementation](https://github.com/tamtom/play-console-cli/blob/3d4cea4245a33e814f440fe9e99bc6325e072611/internal/cli/orders/orders.go#L83-L130), [official `orders.batchget`](https://developers.google.com/android-publisher/api-ref/rest/v3/orders/batchget)
- Add `gplay orders review-refund` for chargeback evidence and refund preference, with `--confirm`, explicit structured input, dry-run coverage, and no automated decision-making. [Official `orders.reviewrefund`](https://developers.google.com/android-publisher/api-ref/rest/v3/orders/reviewrefund)
- Add enterprise-only Play App Signing enrollment/key rotation only if clearly namespaced and guarded. Google explicitly warns that these methods are for self-hosted Cloud KMS keys and must not be used for standard Play App Signing enrollment. [Official app-signing resource](https://developers.google.com/android-publisher/api-ref/rest/v3/appsigning), [official enrollment warning](https://developers.google.com/android-publisher/api-ref/rest/v3/appsigning/enrollApp)
- Do not reimplement direct release listing: `gplay tracks releases` already uses the new `applications.tracks.releases.list` official API.

The new registered-third-party-store review/catalog resources are lower priority unless Gplay deliberately expands beyond operating a developer's own Play-published apps.

### 7. Workflow resilience

Gplay workflows already validate and dry-run multi-step definitions. The next safety increment is bounded per-step timeouts, explicit opt-in retry policy, a definition fingerprint, persisted attempt diagnostics, and resume that never assumes a timed-out mutation is safe to replay. ASC documents this ambiguity explicitly and tests the runner contract. [ASC workflow timeout/retry design](https://github.com/rorkai/App-Store-Connect-CLI/blob/563ee9e0ba2652348a8defdd71ccb454c3aa7b60/docs/design/workflow-step-retry-timeout.md), [ASC workflow module](https://github.com/rorkai/App-Store-Connect-CLI/tree/563ee9e0ba2652348a8defdd71ccb454c3aa7b60/internal/workflow)

### 8. Consolidated insights

ASC's `insights` command produces deterministic weekly/daily comparisons and marks metrics unavailable when the source cannot support them instead of inventing a value. [ASC insights implementation](https://github.com/rorkai/App-Store-Connect-CLI/blob/563ee9e0ba2652348a8defdd71ccb454c3aa7b60/internal/cli/insights/insights.go#L28-L255)

Gplay can implement the same product idea over capabilities it already has: downloaded statistics/financial reports, vitals, reviews, and release history. Start with pure parsers over local downloaded report files, then add explicitly requested official read-only fetches. Do not scrape acquisition dashboards or claim metrics that the documented sources cannot provide.

### 9. Lower-priority parity

- A pinned `gplay install-skills` flow with commit/checksum verification, preview, no overwrite by default, and rollback. ASC exposes skill installation as a first-class command. [ASC install command registration](https://github.com/rorkai/App-Store-Connect-CLI/blob/563ee9e0ba2652348a8defdd71ccb454c3aa7b60/internal/cli/registry/registry.go#L113-L122)
- Optional Gradle build/sign/package helpers and local screenshot capture/framing. These remain local tooling and must not become required for normal API commands.
- Continue expanding offline validation rules inside the canonical `gplay validate` surface: URL/contact syntax, placeholder and minimum-quality checks, stable check IDs, remediation, advisory severity, and strict-mode escalation. ASC 4.9.0 added this class of push-time metadata rejection checks offline. [ASC 4.9.0 metadata-validation change](https://github.com/rorkai/App-Store-Connect-CLI/pull/2086), [ASC validation module](https://github.com/rorkai/App-Store-Connect-CLI/tree/563ee9e0ba2652348a8defdd71ccb454c3aa7b60/internal/validation)
- Preserve Gplay's stronger artifact preflight and race-detector CI; ASC's command count and raw test volume are not goals by themselves.

## Suggested delivery order

1. **Safety PR:** remove scheduled/mutating live integration defaults; scrub test environments; add fail-closed auth/network test seams.
2. **Discovery PR:** lazy command catalog plus `gplay search`.
3. **Schema PR:** `gplay schema`, discovery refresh, generator checks, read-only drift reporting.
4. **Architecture series:** migrate two or three command families at a time to injected runtime and I/O.
5. **Filesystem PR:** root-bound, symlink-safe, atomic local writes with attack-oriented tests.
6. **Mutation-safety vertical slice:** metadata/image plan, apply, receipt, and resume using only official APIs.
7. **Endpoint PR:** true order batch get and carefully guarded refund-review support.
8. **Workflow PR:** explicit timeout/retry policy and ambiguity-safe resume.
9. **Product PR:** local-first weekly insights.

All tests for these workstreams should be offline and mocked. No implementation step requires a Google Account browser session, private Play Console RPC, cookie import, authenticated UI automation, or automated legal acceptance.
