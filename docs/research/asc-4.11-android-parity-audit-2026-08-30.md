# ASC 4.11.0 Android parity audit

Date: 2026-08-30  
Gplay baseline: `v0.9.2`, commit `5f12cc2`  
ASC baseline: `4.11.0`, commit `5dc8846bfd709a604808610a8a617ee5b19ea2bc`

## Executive conclusion

Gplay already implements most useful Google Play equivalents through documented
Google APIs, plus several ASC-inspired foundations: capabilities, command search,
embedded schemas, API drift checks, rooted file access, dry-run, workflows,
plan/apply/receipt metadata sync, local preflight, insights, verified skills, and
a policy-safe live test app.

The next release should not begin by adding another broad command family. It
should first repair five correctness gaps in existing advertised behavior:

1. `release` accepts metadata and screenshot flags but deliberately ignores them.
2. `testers --emails` parses input but cannot send it; the official resource only
   supports Google Group email addresses.
3. retry/backoff configuration exists but is not connected to the HTTP transport.
4. paginated loops have neither repeated-token detection nor a maximum page limit.
5. upload paths do not consistently reject empty files before creating edits or
   making upload calls.

After those repairs, the highest-value ASC parity is a single coherent quality
batch: extend sealed plan/apply/resume semantics from `sync` to `publish`; deepen
offline submission validation; make manual Play Console handoffs more useful;
improve query, pagination, patch, and receipt semantics; then strengthen
cross-platform and contract testing.

This audit does **not** recommend private Play Console RPCs, Google cookie/session
storage, authenticated browser automation, DOM scripting, or automated legal
acceptance. Gplay should continue using only documented Google APIs and explicit
human handoffs for Console-only work.

## Implementation status

The five P0 findings were reproduced with deterministic CLI/HTTP regression
tests and repaired in the same 2026-08-30 working batch:

- release now applies requested listing metadata and screenshots inside the
  release edit instead of discarding the options; sparse locale data is patched
  without clearing omitted fields, explicit empty files still clear fields, and
  screenshots are decoded and dimension-checked before the edit is created;
- the compatibility-retained `--emails` tester flag fails before credentials
  with guidance to use the only official field, `--google-groups`; full tester
  replacement now requires `--confirm` and rejects unknown JSON fields;
- documented retry settings now install bounded exponential backoff on official
  authenticated Google clients, while mutation/upload requests are never
  automatically replayed; an explicit config value of `max_retries: 0` is
  preserved and disables retries;
- manual and generated-client pagination loops now stop on repeated tokens or
  the shared 1,000-page ceiling; and
- direct artifact/media uploads now reject empty and non-regular files through
  one checked-open path before API setup wherever the command owns that setup.
  Release screenshots are uploaded from the same descriptors that passed
  validation, eliminating the changed-path race.

The batch also upgrades `google.golang.org/api` from `v0.293.0` to `v0.295.0`
and refreshes the official discovery revisions. The reviewed method inventory
remains 218 methods; the tagged client now contains the App Signing and memory
metric generated services that previously required narrow official-REST
adapters.

Update 2026-09-26: The batch merged as one PR for each fix:

- #287: the retry transport.
- #288: the pagination guard.
- #289: the check for empty and non-regular upload files.
- #290: the `testers` changes.
- #291: the release listings and screenshots.

#291 is different from the working batch. A release skips a local screenshot
when an image with the same SHA-256 is already on Play, and it never deletes a
screenshot. It checks the limit of 8 screenshots for each type on the final
count, before the bundle upload. A live check on the test app confirmed that
`Image.sha256` is the lowercase hex SHA-256 of the uploaded bytes.

Dependabot updated `google.golang.org/api` after the batch. #292 refreshed
the official discovery files to revision 20260924.

## Scope and method

- Pinned the latest ASC release, [`4.11.0`](https://github.com/rorkai/App-Store-Connect-CLI/releases/tag/4.11.0), published 2026-08-29, and audited its exact tag rather than the moving default branch.
- Reviewed the complete generated [ASC command reference](https://github.com/rorkai/App-Store-Connect-CLI/blob/4.11.0/docs/COMMANDS.md), architecture, public API clients, private web client, validation, workflows, tests, CI, and release automation.
- Reviewed the changes from [ASC 4.9.0 through 4.11.0](https://github.com/rorkai/App-Store-Connect-CLI/compare/4.9.0...4.11.0), including the 4.10 correctness work and 4.11 deep submission validation.
- Audited Gplay `v0.9.2` from its root catalog, help output, capability registry,
  API manifest, coverage manifest, clients, tests, workflows, and release files.
- Compared missing behavior only against documented Google surfaces, primarily
  the [Android Publisher REST API](https://developers.google.com/android-publisher/api-ref/rest),
  [Edits guide](https://developers.google.com/android-publisher/edits), and
  [Play Developer Reporting API](https://developers.google.com/play/developer/reporting/reference/rest).

No Google credentials, private endpoint, browser session, personal app, or
production app was used in this audit.

## Current scale and test snapshot

| Metric | ASC 4.11.0 | Gplay 0.9.2 |
|---|---:|---:|
| Top-level command families | 81 | 294 leaf command paths |
| `_test.go` files | 1,234 | 191 under `internal` |
| `Test*` functions | 10,355 | Not used as a parity target |
| Benchmarks | 15 | Not used as a parity target |
| Files using `httptest.NewServer` | 138 | Contract tests exist but are uneven |
| Race detector in normal CI | No | Yes |
| Command coverage classes | No equivalent manifest | 272 offline, 20 live, 2 sandbox |
| Documented API catalog | Apple public API plus private web providers | 218 methods across 8 Google Play-related APIs |

ASC's raw size is evidence of breadth, not a target. Gplay's race CI, scheduled
live smoke, offline artifact preflight, capability truthfulness, and explicit
test-class manifest should be preserved.

## Complete ASC feature-family inventory

These are all 81 user-facing root families in ASC 4.11.0, grouped according to
its generated command reference.

### Getting started and web

- `auth`, `doctor`, `install-skills`, `init`, `docs`
- `web`

`web` contains authenticated Apple private-web workflows for areas such as
agreements, API keys, sandbox users, initial app creation, privacy, review,
subscription gaps, analytics dashboards, and Xcode Cloud. Its safety separation
is instructive; its private protocol and session model are not suitable for
Gplay.

### Analytics, finance, and growth

- `analytics`, `ads`, `optimize`, `insights`, `finance`, `performance`

The surface includes analytics requests/reports/segments, daily and weekly
insights, financial reports, diagnostics, keyword/search optimization, and a
large Apple Ads command tree.

### App management

- `apps`, `app-setup`, `app-tags`, `versions`, `localizations`, `metadata`
- `screenshots`, `video-previews`, `background-assets`, `product-pages`
- `routing-coverage`, `pricing`, `pre-orders`, `categories`, `age-rating`
- `accessibility`, `encryption`, `eula`, `agreements`, `app-clips`
- `android-ios-mapping`, `marketplace`, `alternative-distribution`, `nominations`
- `game-center`

This includes deterministic metadata plan/approve/apply workflows; image capture,
framing, review, upload, and receipts; product pages and experiments; app
availability and pricing; declarations; alternative distribution; nominations;
and the extensive Game Center resource tree.

### Builds and beta distribution

- `testflight`, `builds`, `build-bundles`, `build-localizations`, `xcode`
- `distribute`, `sandbox`

The TestFlight/build surface covers groups, testers, distribution, recruitment,
feedback, crashes, metrics, processing waits, expiry, encryption, dSYMs, build
notes, and local Xcode archive/export/upload operations.

### Review and release

- `release`, `status`, `release-notes`, `review`, `reviews`, `submit`, `validate`
- `publish`

It includes staged high-level releases, changed-state status watching, submission
health, review items/attachments/history, localized release notes, customer
review response workflows, and deep validation.

### Monetization

- `iap`, `storekit`, `app-events`, `subscriptions`

The underlying trees include IAP versions/localizations/pricing/availability and
review media; subscription groups, products, base versions, prices, offers,
offer codes, and win-back offers; and StoreKit server operations.

### Signing and local distribution

- `signing`, `bundle-ids`, `certificates`, `profiles`, `merchant-ids`
- `pass-type-ids`, `notarization`

### Account and team access

- `account`, `users`, `actors`, `devices`

### Automation

- `workflow`, `webhooks`, `xcode-cloud`, `notify`, `migrate`

### Utilities

- `system-status`, `diff`, `capabilities`, `search`, `snitch`, `version`
- `completion`, `schema`, `telemetry`

## What changed after the previous ASC 4.9.0 audit

ASC 4.10.0 and 4.11.0 mostly improve correctness and operational quality rather
than introduce a new major domain.

Transferable changes include:

- deeper submission, metadata, content, agreement, and optional URL validation;
- structured resolution metadata for validation findings;
- better `latest`, status, version, and query filters;
- repeated pagination-token protection and improved pagination coverage;
- sparse/null/explicit-value serialization correctness;
- preflight validation for zero-length uploads;
- more robust partial upload failure recovery and ordering;
- stable typed errors and exact notification failure handling;
- safer destructive confirmation behavior;
- cold-cache documentation and command-surface contract tests; and
- a reusable HTTP handler test helper for assertions originating in handlers.

ASC 4.11's deepest validation can combine the public API with cached Apple web
session data. Gplay should copy the evidence model—`passed`, `blocked`,
`unverified`, `not-applicable`, remediation, and source—not the private session.

## Architecture comparison

| Architecture concern | ASC 4.11 | Gplay 0.9.2 | Decision |
|---|---|---|---|
| Command catalog | Lazy metadata catalog | Lazy registry and local search already present | Keep; improve intent aliases and root discovery text |
| Provider boundary | Public API and private web clients are separate | Official/manual/unsupported capability registry | Gplay's boundary is safer; preserve it |
| API clients | Centralized typed clients and test injection | Official generated clients with runtime seams | Add shared retry/pagination behavior, not bespoke APIs |
| Filesystem | Rooted/safe file operations | `rootfs` and `secureopen` already present | Already matched |
| Workflows | Retry, timeout, fingerprint, resume, ambiguity handling | Equivalent core behavior now exists | Do not build another workflow engine |
| Mutation plans | Strong in distribution and metadata domains | Strong in `sync`, enterprise signing, and selected commands | Extend to `publish` and ordinary media mutation paths |
| Validation | Large shared model with evidence/remediation | Strong artifact preflight but fragmented metadata/release checks | Unify and deepen offline validation |
| Patch semantics | Explicit sparse/null/false handling | Tri-state bool helper, ordinary strings remain ambiguous | Add field-presence and explicit-clear semantics |
| Status/watch | Bounded, changed-state output in key flows | Watch exists | Add max polls and changed-only output consistently |
| Web support | Private HTTP session and Apple cookies | Safe URL launcher/manual handoff | Expand navigation only; never add Google sessions/private RPCs |
| Telemetry | Present | Absent | Do not add solely for parity |

## Feature parity matrix

| ASC domain | Current Android/Gplay position | Next useful work |
|---|---|---|
| Auth, doctor, init, docs, skills | Matched | Better root-help onboarding only |
| Apps and initial setup | App inventory plus deterministic manual bootstrap plan | Richer handoff/status checklist; app creation and first upload remain manual |
| Versions, builds, distribution | Strong bundles/APKs/tracks/publish/promote/rollout/release | Sealed publish plan/apply/resume and fix ignored release media flags |
| Metadata/localizations | Strong listings, metadata, sync, Fastlane migration | Central validation, explicit clear/null, receipts for ordinary push paths |
| Images/screenshots/video | Upload/sync/capture and YouTube URL support | Per-file receipts, upload preflight, safer remote media fetches |
| Product pages/experiments | Play experiments represented truthfully as manual/limited | Deterministic experiment manifest and Console handoff, no scraping |
| Pricing/availability | Pricing helpers and supported catalog APIs | Keep within documented methods |
| App content/declarations | Data Safety API plus offline/manual inventory | Evidence-based unified submission validator and exact Console destinations |
| TestFlight | Testing tracks, internal sharing, testers | Fix tester contract; improve group orchestration and query filters |
| Reviews/review health | List/get/reply plus checks/validate | Optional local summaries and batch response plans; manual review-status handoff |
| Monetization | Broad IAP, subscriptions, plans, offers, orders, purchases, RTDN | Mostly parity-complete; focus on shared correctness contracts |
| Finance/analytics/performance | Reports, insights, vitals, errors, crashes, ANRs | Add local compare/coverage only where official data exists |
| Users/devices/access | Users, grants, device tiers | Mostly parity-complete |
| Game Center | Partial Android equivalent through Play Games | Expand only for documented, valuable Play Games operations |
| Apple signing/notarization/Xcode | Android build/signing helpers where meaningful | Apple-only items are not parity gaps |
| Apple Ads | Separate Google Ads API can provide an official analogue | Optional scope-gated `gplay ads`; not part of core Play auth |
| Workflow/diff/capabilities/search/schema | Already implemented | Improve semantics and tests, do not duplicate families |
| System status | No current command | Optional links/read-only aggregation from official Google status sources |

## Already implemented: do not build twice

The previous ASC work is now present in Gplay and should be evolved in place:

- `gplay capabilities` with official/manual/unsupported boundaries;
- lazy command registry, `gplay search`, generated docs, and completion;
- embedded API schema inspection and API drift checks;
- `gplay bootstrap plan` for Console-only first-app actions;
- symlink-safe rooted file handling and secure writes;
- global dry-run, timeouts, workflows, fingerprints, retries at the workflow
  layer, receipts, resume, and ambiguity-aware mutation handling;
- metadata/image `sync plan`, `apply`, `run`, `export`, and `import`;
- offline `preflight`, submission `validate`, and app-content inventory;
- daily/weekly insights, vitals, reports, and reviews;
- pinned/verified skill installation;
- local Gradle, signing-inspection, and screenshot helpers;
- official API manifest for 218 methods across eight APIs; and
- race CI, command coverage classification, sandbox tests, scheduled live smoke,
  and mutation-ledger cleanup.

## Confirmed correctness blockers in Gplay 0.9.2

### P0: advertised release inputs are ignored

[`internal/cli/release/run.go`](../../internal/cli/release/run.go) validates and
accepts `--listings-dir`, `--screenshots-dir`, `--skip-metadata`, and
`--skip-screenshots`, then explicitly discards all four. A command can report a
successful release without applying the requested metadata/media. Wire them into
the edit transaction or fail before authentication with a clear unsupported
usage error.

### P0: `testers --emails` cannot work

[`internal/cli/testers/testers.go`](../../internal/cli/testers/testers.go) parses
the emails and then throws the list away. More importantly, the official
`androidpublisher.Testers` schema contains only `googleGroups`; it does not have
an individual-email field. Remove or reject `--emails`, document Google Groups,
and add an outbound request test. Never reinterpret individual emails as groups.

### P0: retry settings do not affect HTTP

`max_retries` and `retry_delay` are configured and surfaced in diagnostics, but
[`internal/playclient/service.go`](../../internal/playclient/service.go) installs
only the OAuth client, with no retrying transport. Implement bounded exponential
backoff with jitter, `Retry-After`, context cancellation, replayable-body checks,
and an explicit safe-method/idempotency policy. Do not blindly replay ambiguous
mutations.

### P0: pagination loops can cycle forever

Several command packages independently follow `nextPageToken` until empty. They
do not detect a repeated token, cap the number of pages, or share consistent
filter/token preservation. Add one shared paginator with seen-token detection,
maximum pages, context checks, stable aggregation, and cycle tests.

### P0: empty upload artifacts reach mutation setup

Bundles, APKs, release, images, deobfuscation, internal sharing, Checks, custom
apps, and app-store uploads do not consistently reject zero-byte/non-regular
files before network work. The release path can create an edit before discovering
an invalid artifact. Add a reusable upload preflight and verify zero API calls on
failure.

### P1: patch presence and clearing are incomplete

Some commands use a good tri-state boolean helper, but ordinary string flags do
not preserve “omitted” versus “explicitly empty.” Generated `omitempty` behavior
means promises such as clearing `--video ""` are unreliable. Add explicit field
presence, `ForceSendFields`/`NullFields` where supported, and serialized-body
tests for omitted, empty, null, true, and false.

### P1: validation is fragmented

Metadata validation currently focuses mainly on Play text lengths, and direct
metadata push does not consistently invoke the canonical validator before
creating an edit. Centralize metadata, release-note, media, content, and optional
URL checks and execute them before auth or edit creation.

### P1: usage errors and mutation confirmations are inconsistent

Typed usage errors exist but many commands return plain errors. In addition,
tester replacement can overwrite the entire tester resource without a consistent
confirmation contract. Adopt typed usage errors and apply the repository's
explicit-confirmation rule to destructive replacement semantics.

### P1: ordinary media commands lack resumable receipts

The `sync` transaction engine is strong, while direct image/metadata mutations
do not consistently get per-file receipts, reconciliation, or resume. Route
these through shared transaction primitives. Harden remote media downloads with
scheme/host policy, redirects, timeouts, content limits, and tests.

## Recommended one-batch implementation order

The user preference is one fast, test-gated delivery rather than a bureaucracy
of many PRs. The order below is an implementation sequence inside that one batch,
not a PR split.

1. **Truth and data-loss blockers:** reject/remove tester emails; make release
   metadata/screenshot flags real or fail explicitly; add confirmation for full
   tester replacement.
2. **Transport correctness:** shared safe retry transport and shared guarded
   paginator.
3. **Input correctness:** zero-byte/regular-file upload preflight, repeated CSV
   flags, typed usage errors, and presence-aware patch semantics.
4. **Unified validation:** one offline engine for metadata, release notes, media,
   app-content inventory, optional SSRF-safe URL checks, and evidence states.
5. **Release orchestration:** extend existing sealed transaction primitives to
   `publish plan`, `apply`, `resume`, and receipt/status. Do not replace `sync` or
   `workflow`; compose them.
6. **Media durability:** per-file hashes/receipts, deterministic ordering,
   partial-failure reconciliation, and safe remote downloads.
7. **Read/query ergonomics:** `tracks releases list --latest`, optional status and
   version filters, repeated multi-value flags, consistent pagination output,
   and bounded changed-only watchers.
8. **Manual web handoffs:** expand `gplay web` to named Play Console destinations,
   preconditions, next actions, and copyable checklists. It remains a URL opener,
   never an authenticated automation client.
9. **Testing/release gates:** stronger handler fixtures, manifest-to-real-test
   enforcement, representative populated responses, native macOS/Windows smoke,
   and post-download checksum/version verification.
10. **Optional growth module:** scope-gated Google Ads App Campaign support only
    after core correctness is green.

## Optional official Google Ads analogue

Apple Ads is not intrinsically “not applicable.” Google exposes documented
[App Campaigns](https://developers.google.com/google-ads/api/docs/app-campaigns/overview),
[campaign creation](https://developers.google.com/google-ads/api/docs/app-campaigns/create-campaign),
and [reporting](https://developers.google.com/google-ads/api/docs/app-campaigns/reporting)
through the Google Ads API.

This should be an optional, clearly isolated provider because it requires a
different OAuth/developer-token/customer-ID model from Android Publisher:

- `gplay ads auth status`
- `gplay ads campaigns list|create|pause|resume`
- `gplay ads assets ...`
- `gplay ads reports app-performance`
- `gplay optimize app-campaign plan`

It is P2: useful official parity, but it must not delay correctness of the core
Play release path or silently broaden the credentials/scopes used by normal
commands.

## Safe `gplay web` scope

Recommended additions are navigation and handoff only:

- named destinations for app creation, first release, Play App Signing,
  app-content declarations, content rating, target audience, app access,
  managed publishing, experiments, pre-registration, policy/review pages, and
  pre-launch reports;
- structured JSON describing `status=manual`, provider, destination, reason,
  prerequisites, next actions, and the official command to run afterward;
- `--print-only` as the deterministic default in non-interactive environments;
- package/account confirmation in the generated handoff text; and
- no credential resolution or Google request for handoff generation.

Explicitly excluded:

- private Play Console RPCs or undocumented endpoints;
- Google Account passwords, cookies, session import, or profile retention;
- browser DOM automation or remote-controlled authenticated profiles;
- automatic agreement/declaration acceptance;
- scraping policy, acquisition, review, or experiment dashboards; and
- pretending that a manual-only state was verified by an API.

## Testing strategy for the batch

### Hermetic command and contract tests

Every affected command should cover:

- successful representative populated output;
- validation failure before credential or network resolution;
- official API error decoding and stable exit classification;
- empty results and multi-page results;
- repeated-token and maximum-page termination;
- `Retry-After`, backoff exhaustion, context cancellation, and unsafe-mutation
  no-replay behavior;
- zero-byte, directory, symlink, and changed-after-plan artifacts;
- dry-run and required confirmation behavior;
- omitted/empty/null/true/false request bodies; and
- deterministic JSON, table, and markdown rendering where supported.

The coverage manifest gate should prove that a registered test exercises each
command class. Today it proves only that a command has a label, and the generator
defaults new commands to `offline`.

### Sandbox

Keep `GPLAY_API_BASE_URL` loopback-only. Expand discovery-backed handlers beyond
the current small subset, validate upload bodies, simulate filters and page
tokens, and make unimplemented known routes fail explicitly. Sandbox tests remain
credential-free and use only official API shapes.

### Live integration

Preserve the existing dedicated compile-time package guard, minimal environment,
single-edit mutation budget, cleanup ledger, weekly smoke, daily janitor, and
optional internal-track fixture release. These tests use documented Google APIs
against the dedicated Steps Share test app; they do not require a personal or
production app.

Add compiled-CLI coverage for the most important official flows, but do not add
private web behavior. A live test should be skipped unless the dedicated package,
credentials, explicit mutation opt-in, and disposable fixture are all present.

### Release verification

Retain race, lint, docs, schema-drift, and vulnerability checks. Add native or
emulated smoke for each released platform, download the produced artifacts,
verify checksums, execute supported binaries, and rehearse the release job before
tagging.

Suggested final gate:

```text
go test -race ./...
go vet ./...
make lint
make format-check
make check-docs
make check-api-drift
make check-api-schema
go run ./tools/covmanifest
git diff --check
make build-all
```

## Manual, unsupported, and platform-specific boundaries

Keep these manual unless Google publishes a documented API:

- initial standard Play app creation;
- first artifact upload that establishes the app;
- ordinary Google-managed Play App Signing enrollment;
- current legal agreements and policy declarations;
- content-rating questionnaire and other human attestations;
- managed publishing and published/unpublished state where unavailable;
- complete Play review/policy issue state;
- pre-launch reports, custom listing experiments, promotions, and some
  pre-registration operations.

Apple-only features—App Clips, TestFlight-specific feedback, StoreKit-specific
operations, Xcode/Xcode Cloud, Apple certificates/profiles, merchant/pass IDs,
notarization, routing coverage, and Apple legal constructs—are not Gplay gaps.

Registered third-party app-store review/catalog methods and enterprise
self-hosted-KMS signing remain official but scope-gated. They must not inflate
ordinary Play developer parity claims.

## Final recommendation

Gplay is feature-rich enough that reliability now creates more parity value than
another long list of resource wrappers. Fix the confirmed false-success paths,
install shared transport and pagination guarantees, make validation evidence
explicit, then extend the existing transaction engine to the end-to-end publish
path. Once those are green, a richer manual Console navigator and optional
Google Ads module are the defensible ASC-inspired additions.
