# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

> Release notes for **0.5.0 – 0.7.1** were auto-generated and live in
> [GitHub Releases](https://github.com/tamtom/play-console-cli/releases).


## [0.8.1] - 2026-08-06

### Fixed

- `gplay deobfuscation upload --type nativeCode` no longer fails with HTTP 400.
  The type was lowercased for validation and the lowercased value was then reused
  to build the request, so the API received `nativecode` and rejected it. `--type`
  is still matched case-insensitively, but the value is now sent in the exact
  casing the API defines. `--type proguard` was unaffected. ([#253](https://github.com/tamtom/play-console-cli/issues/253))

## [0.8.0] - 2026-07-26

### Added

#### Offline preflight engine — `gplay preflight`

`preflight` now decodes `AndroidManifest.xml` for real instead of matching
substrings, and runs nine independently selectable scanners with no API calls
and no credentials.

- **Manifest decoding** — binary AXML (`internal/preflight/axml.go`) for APKs and
  aapt2 protobuf `XmlNode` (`internal/preflight/protoxml.go`) for App Bundles,
  normalized into one typed model (`internal/preflight/manifest.go`). Optional
  attributes are tri-state, so *absent*, *explicitly false*, and *not statically
  determinable* stay distinguishable.
- **`manifest`** — `debuggable`, `testOnly`, exported components missing
  `android:exported` on `targetSdk` 31+, exported providers granting URI
  permissions, foreground service types without their matching Android 14
  permission, cleartext traffic, `allowBackup`, package/version sanity.
- **`permissions`** — 23 restricted permissions requiring a Play declaration,
  18 sensitive permissions requiring a Data safety disclosure, legacy storage
  permissions on modern targets, duplicates, deprecated permissions. Findings
  carry Play policy documentation links.
- **`native_libs`** — missing `arm64-v8a`, **16 KB memory page alignment** read
  from real ELF program headers via `debug/elf`, unstripped debug symbols,
  `extractNativeLibs="true"`.
- **`metadata`** — listing text limits plus **real screenshot pixel dimensions**,
  aspect ratio, count, and icon/feature-graphic sizes. Enabled by
  `--listings-dir`.
- **`secrets`** — 14 credential patterns, shipped keystores and `.pem` files,
  `.git`/`.env` leakage; also scans dex string data with bounded chunked reads.
- **`billing`** — Play Billing vs. third-party payment processors,
  `com.android.vending.BILLING` declared without an implementation.
- **`privacy`** — 40+ analytics/attribution/ads SDK signatures and `AD_ID`
  permission consistency against `targetSdk` 33+.
- **`policy`** — target API level floor, restricted services (accessibility,
  VPN, device admin, notification listener), APK-vs-AAB upload format.
- **`size`** — download size budget, dex fragmentation, payload breakdown.

#### New flags

- `--listings-dir` — validate a Fastlane-style listings directory in the same pass
- `--only` / `--skip` — comma-separated scanner selection
- `--list-scanners` — print the available scanner IDs and exit
- `--min-target-sdk` — override the target API level floor without a rebuild

### Changed

- `preflight` text output now groups findings per scanner, reporting `ok`,
  `skipped (reason)`, or the finding count for every scanner that ran.
- The `google_api_key` secret pattern is now a **warning** rather than an error.
  Android apps legitimately embed Maps and Firebase `AIza…` keys; the defence is
  key restriction, not absence. Hard credentials (private keys, service-account
  JSON, `sk_live_`, GitHub/Slack tokens) remain errors.

## [Unreleased]

### Added

#### New Commands
- `gplay web auth login` — Store browser cookies as a Play Console web session
- `gplay web auth status` — List stored web sessions, `--check` validates them
- `gplay web auth logout` — Delete stored web sessions
- `gplay web apps list` — List every app in the developer account, including brand-new ones
- `gplay web apps create` — Create a new app by driving the Play Console UI
- `gplay web apps update` — Update an existing package's App category
- `gplay web apps status` — Show publishing readiness: setup checklist and pending changes
- `gplay web apps availability` — Read or set track country availability
- `gplay web apps pricing` — Read or set the paid app's price
- `gplay web apps review` — Send pending changes for review from the Publishing overview
- `gplay web apps rollout` — Roll out the production draft release: preview, confirm, send for review
- `gplay web apps declarations` — Read or set App content declarations, including questionnaires and data-safety CSV import
- `gplay web apps policy` — Read the app's Play policy status and reported issues
- `gplay web apps publish` — Publish approved changes, or toggle managed publishing
- `gplay web apps distribution` — Read or add app form factors (Android TV, Wear OS, and more)
- `gplay web apps promo-codes` — List promo-code campaigns or create paid-app codes
- `gplay web apps rating` — Read the app's IARC content-rating state

**Requirements and caveats:** the `web` commands require a locally installed Google Chrome and are macOS-only today. They authenticate with browser cookies rather than a service account (see `docs/authentication.md`), and they drive Play Console's unofficial web interface, which Google can change without notice.

## [0.4.5] - 2026-02-26

### Added

#### New Commands
- `gplay purchase-options batch-update-states` — Batch activate/deactivate one-time product purchase options
- `gplay purchase-options batch-delete` — Batch delete one-time product purchase options
- `gplay otp-offers list` — List offers for a purchase option
- `gplay otp-offers activate` — Activate an OTP offer
- `gplay otp-offers deactivate` — Deactivate an OTP offer
- `gplay otp-offers cancel` — Cancel an OTP offer
- `gplay otp-offers batch-get` — Get multiple OTP offers
- `gplay otp-offers batch-update` — Batch update OTP offers
- `gplay otp-offers batch-update-states` — Batch activate/deactivate/cancel OTP offers
- `gplay otp-offers batch-delete` — Batch delete OTP offers
- `gplay subscriptions batch-get` — Get multiple subscriptions by product IDs
- `gplay subscriptions batch-update` — Batch update multiple subscriptions

### Fixed
- `onetimeproducts create` now sets `AllowMissing(true)` on the Patch call (required for creation) and supports `--regions-version`
- `onetimeproducts patch` now supports `--regions-version` and `--allow-missing` flags

## [0.4.4] - 2026-02-18

### Added

#### New Commands
- `gplay release-notes generate` — Generate release notes from git history with `--since-tag`/`--since-ref`, auto-truncation to Google Play's 500-char limit

#### New Packages & Utilities
- **Spinner** (`internal/cli/shared/spinner.go`) — Braille spinner on stderr during API calls; TTY-gated, disabled when `GPLAY_DEBUG` or `GPLAY_SPINNER_DISABLED` is set
- **JUnit CI reports** (`internal/cli/shared/junit_report.go`) — `--report junit --report-file results.xml` flags for CI integration
- **Error classification** (`internal/cli/shared/errfmt/`) — `Classify(err)` auto-detects 401/403/404/timeout errors with actionable hints
- **SanitizeTerminal** (`internal/output/sanitize.go`) — Strips ANSI escapes and control chars from table output
- **SecureOpen** (`internal/secureopen/`) — Path-validated file opening with symlink resolution and directory boundary checks
- **OptionalBool** (`internal/cli/shared/optionalbool.go`) — Tri-state boolean flag type (`unset`/`true`/`false`) implementing `flag.Value`

#### Documentation & Tooling
- `llms.txt` — LLM-friendly project summary at repo root
- `.golangci.yml` — Curated linter config with `govet`, `staticcheck`, `unused`, `ineffassign`, `misspell`, `unparam`, `errorlint`

### Fixed
- Use `errors.Is`/`errors.As` throughout codebase (errorlint compliance)
- Octal literal format (`0o644`) for Go 1.13+ compatibility

## [0.4.3] - 2026-02-15

### Changed

#### Reports — Real GCS-Based Implementation
- `gplay reports financial list/download` now fetches real report files from Google Cloud Storage bucket `pubsite_prod_rev_<developer_id>`
- `gplay reports stats list/download` now fetches real statistics CSVs from GCS
- Added `--developer` flag (required) to stats commands for bucket name construction
- `--package` is now optional for `stats list` (filters results) and required for `stats download`
- Date range filtering via `--from`/`--to` extracts YYYYMM from filenames
- Download writes files to `--dir` with JSON summary output
- New `internal/gcsclient` package — thin GCS client reusing the same credential resolution pattern as `playclient` and `reportingclient`
- GCS mock tests using `httptest` for listing, filtering, and download verification

## [0.4.2] - 2026-02-15

### Added

#### New Commands
- `gplay apps list` - List apps accessible by service account
- `gplay init` - Initialize project configuration with `.gplay/config.yaml`
- `gplay docs generate` - Generate markdown command reference
- `gplay vitals crashes` - View crash clusters and reports (Play Developer Reporting API)
- `gplay vitals performance` - View performance metrics (startup, rendering, battery)
- `gplay vitals errors` - View error issues and reports
- `gplay users` - Manage developer account users (list, create, update, delete)
- `gplay grants` - Manage per-app permission grants (create, update, delete)
- `gplay update` - Self-update the CLI binary
- `gplay notify send` - Send webhook notifications (Slack, Discord, generic)
- `gplay migrate fastlane` - Migrate from Fastlane metadata format
- `gplay reports financial` - Financial reports (list/download)
- `gplay reports stats` - Statistics reports (list/download)
- `gplay listings locales` - List available locales with validation

#### Enhancements
- Real table output using `text/tabwriter` for `--output table`
- ANSI color utilities with `NO_COLOR` environment variable support
- "Did you mean?" command suggestions for typos
- `GPLAY_DEFAULT_OUTPUT` environment variable support
- `--dry-run` flag for all commands (intercepts write HTTP methods)
- `--video` flag for listings with YouTube URL validation
- `--fix` and `--confirm` flags for `auth doctor`
- Progress indicators for file uploads
- Enhanced `release` command with `--listings-dir`, `--screenshots-dir`, and plain text release notes
- Full locale validation for listings commands

#### Testing & Developer Experience
- `testutil` package with shared test helpers and fixtures
- `cmdtest` package for black-box CLI testing
- `httptest`-based API mocking for unit tests
- Comprehensive test coverage across all CLI commands (500+ tests)
- Integration test build tags (`-tags integration`)
- Pre-commit git hooks (`make install-hooks`)
- `GPLAY.md` auto-generated command reference

#### CI/CD
- Main branch CI workflow for post-merge checks
- Security scanning workflow with gosec

#### Documentation
- `docs/API_NOTES.md` - Google Play API quirks and gotchas
- `docs/api/discovery.json` - API spec tracking with endpoint index
- `docs/GO_STANDARDS.md` and `docs/TESTING.md` guides

## [0.4.0] - 2025-02-05

### Removed
- Browser OAuth login flow (requires custom OAuth client which is not available)
- `--client-id`, `--client-secret`, `--timeout` flags from `auth login`

### Changed
- `auth login` now requires `--service-account` flag
- Updated Agents.md documentation to reflect service-account-only authentication

## [0.3.1] - 2025-02-05

### Changed
- Updated README with detailed step-by-step service account setup instructions
- Clarified that service accounts are required (browser OAuth requires custom client)

## [0.3.0] - 2025-02-05

### Added
- `apks addexternallyhosted` - Add externally hosted APKs without uploading
- `tracks create` - Create custom release tracks
- `purchases productsv2 get` - Get purchase details using v2 API
- `onetimeproducts` - Full one-time products management (list, get, create, patch, delete, batch operations)

### Improved
- API coverage increased to 89% (32/36 resources)

## [0.2.0] - 2025-02-05

### Added
- Browser-based OAuth login - `gplay auth login` now opens your browser for authentication
- `Agents.md` documentation for AI coding agents
- Agent Skills section in README with link to [gplay-cli-skills](https://github.com/tamtom/gplay-cli-skills)

### Changed
- `gplay auth login` defaults to browser OAuth flow
- Service account auth moved to `gplay auth login --service-account <path>`
- `CLAUDE.md` now references `Agents.md`

## [0.1.0] - 2025-02-05

### Added
- Initial release
- Authentication: service account and OAuth profiles
- Edit lifecycle: create, get, validate, commit, delete
- Bundle and APK upload/list
- Track management: list, get, update, patch
- Store listings: CRUD operations by locale
- Images: upload, delete, list by type
- Reviews: list, get, reply
- High-level commands: release, promote, rollout
- Monetization: in-app products, subscriptions, base plans, offers
- Purchase management: orders, product/subscription verification
- FastLane integration: sync, import/export, validate
- Shell completion: bash, zsh, fish, powershell
- Self-update mechanism
- Homebrew tap distribution
