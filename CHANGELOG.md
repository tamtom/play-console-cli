# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).


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
