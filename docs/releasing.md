# Releasing

Everything for shipping builds: uploads, tracks, staged rollouts, promotions, and store listing metadata.

## One-command release

```bash
# Creates edit, uploads, updates track, commits — in one step
gplay release --package com.example.app --track internal --bundle app.aab

# With release notes and staged rollout
gplay release --package com.example.app --track production --bundle app.aab \
  --release-notes @notes.json --rollout 0.1

# With metadata and screenshots
gplay release --package com.example.app --track production --bundle app.aab \
  --listings-dir ./metadata --screenshots-dir ./screenshots

# Dry-run any command (intercepts write operations)
gplay --dry-run release --package com.example.app --track internal --bundle app.aab

# Generate release notes from git history
gplay release-notes generate
```

## Promote & rollout

```bash
# Promote between tracks (internal → beta → production)
gplay promote --package com.example.app --from internal --to beta

# Manage staged rollout
gplay rollout update --package com.example.app --track production --rollout 0.5
gplay rollout halt --package com.example.app --track production
gplay rollout resume --package com.example.app --track production
gplay rollout complete --package com.example.app --track production
```

## Low-level edit lifecycle

```bash
gplay edits create --package com.example.app
gplay edits list --package com.example.app
gplay edits validate --package com.example.app --edit <id>
gplay edits commit --package com.example.app --edit <id>

# Upload artifacts
gplay bundles upload --package com.example.app --edit <id> --file app.aab
gplay apks upload --package com.example.app --edit <id> --file app.apk

# Manage tracks
gplay tracks list --package com.example.app --edit <id>
gplay tracks get --package com.example.app --edit <id> --track production
gplay tracks update --package com.example.app --edit <id> --track internal --json @release.json
```

## Store listing

```bash
# Listings
gplay listings list --package com.example.app --edit <id>
gplay listings get --package com.example.app --edit <id> --locale en-US
gplay listings update --package com.example.app --edit <id> --locale en-US --json @listing.json
gplay listings locales      # available locales with validation

# Images
gplay images list --package com.example.app --edit <id> --locale en-US --type phoneScreenshots
gplay images upload --package com.example.app --edit <id> --locale en-US --type phoneScreenshots --file screenshot.png

# App details
gplay details get --package com.example.app --edit <id>
gplay details update --package com.example.app --edit <id> --contact-email dev@example.com
```

## Fastlane interop

```bash
# Export metadata to Fastlane format
gplay sync export-listings --package com.example.app --dir ./fastlane/metadata/android

# Import metadata from Fastlane format
gplay sync import-listings --package com.example.app --dir ./fastlane/metadata/android

# Compare local metadata with Play Store
gplay sync diff-listings --package com.example.app --dir ./fastlane/metadata/android

# One-shot migration from Fastlane
gplay migrate fastlane

# Validate before upload
gplay validate listing --dir ./fastlane/metadata/android --locale en-US
gplay validate screenshots --dir ./fastlane/metadata/android/en-US/images
gplay validate bundle --file app.aab
```

## Testing & distribution

```bash
# Manage testers on closed tracks
gplay testers list --package com.example.app --edit <id> --track internal
gplay testers update --package com.example.app --edit <id> --track internal --emails user@example.com

# Internal app sharing (quick sharing without review)
gplay internal-sharing upload-bundle --package com.example.app --file app.aab
gplay internal-sharing upload-apk --package com.example.app --file app.apk
```

## Pre-submission checks

```bash
# Offline compliance scan on an AAB or APK — no API calls, no credentials
gplay preflight --file app.aab
gplay preflight --file app.aab --max-size 100M --fail-on warning   # CI gate
gplay preflight --file app.aab --listings-dir ./metadata           # also check the listing
gplay preflight --file app.aab --output json --pretty | jq .
gplay preflight --list-scanners

# Bundle size analysis (offline)
gplay bundles analyze --file app.aab --top-files 20
gplay bundles compare --base old.aab --candidate new.aab --threshold 2M
```

### What preflight scans

`preflight` decodes `AndroidManifest.xml` for real — binary AXML for APKs, aapt2
protobuf for App Bundles — so checks read typed attribute values instead of
guessing from substrings. Nine scanners run by default; select them with
`--only` or `--skip`.

| Scanner | Catches |
|---|---|
| `manifest` | `debuggable`, `testOnly`, exported components missing `android:exported` (install failure on Android 12+), exported providers granting URI permissions, foreground service types without their matching permission (Android 14 `SecurityException`), cleartext traffic, `allowBackup`, package/version sanity |
| `permissions` | Restricted permissions that need a Play declaration form, sensitive permissions that need a Data safety disclosure, legacy storage permissions on modern targets, duplicates, deprecated permissions |
| `native_libs` | Missing `arm64-v8a`, **16 KB memory page alignment** (read from real ELF program headers — required by Play for `targetSdk` 35+), unstripped debug symbols, `extractNativeLibs="true"` |
| `metadata` | Listing title/description/release-note lengths and **actual screenshot pixel dimensions**, aspect ratio, count, and icon/feature-graphic sizes. Requires `--listings-dir` |
| `secrets` | Private keys, AWS/Stripe/GitHub/Slack/SendGrid/OpenAI/Anthropic tokens, service-account JSON, shipped keystores and `.pem` files, `.git`/`.env` leakage. Also scans dex string data |
| `billing` | Third-party payment processors alongside (or instead of) Play Billing, `com.android.vending.BILLING` declared with no implementation |
| `privacy` | 40+ analytics/attribution/ads SDKs, and `AD_ID` permission consistency against `targetSdk` 33+ |
| `policy` | Target API level floor (override with `--min-target-sdk`), restricted services (accessibility, VPN, device admin, notification listener), APK-vs-AAB upload format |
| `size` | Download size budget, dex fragmentation, payload breakdown by bucket |

Severity is `error`, `warning`, or `info`; `--fail-on` picks the CI gate
threshold. Findings are grouped per scanner in text output and carry Play policy
documentation links where one applies.

> **Note:** `policy` compares `targetSdkVersion` against a constant that Google
> raises roughly every August. Pass `--min-target-sdk` to override it without
> waiting for a new gplay release.

## Google Checks (privacy & compliance)

```bash
# Upload an AAB/APK to Google Checks for privacy & policy analysis (async)
# Requires a Checks account: pass --account, or set GPLAY_CHECKS_ACCOUNT / checks_account.
gplay checks apps list --account <account-id>
gplay checks analyze --account <account-id> --app <app-id> --binary app.aab

# CI/CD gate: analyze, wait for the report, exit non-zero on high-priority failures
gplay checks analyze --account <account-id> --app <app-id> --binary app.aab \
  --code-ref "$GIT_SHA" --severity-threshold PRIORITY --checks-filter "state = FAILED"

# Browse compliance reports (what the app *does* vs. what data-safety *declares*)
gplay checks reports list --account <account-id> --app <app-id> --checks-filter "state = FAILED"
gplay checks reports get --account <account-id> --app <app-id> --report <report-id>

# AI safety content classification (moderate app/generated text against policies)
gplay checks classify --text @content.txt --policies all --severity-threshold
```
