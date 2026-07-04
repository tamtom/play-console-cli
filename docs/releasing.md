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
# Offline compliance scan on an AAB or APK (no API calls)
# Checks: manifest, bundle size, native ABIs, dex, debuggable, testOnly,
# cleartext traffic, dangerous permissions, secret scan, misplaced files.
gplay preflight --file app.aab
gplay preflight --file app.aab --max-size 100M --fail-on warning   # CI gate

# Bundle size analysis (offline)
gplay bundles analyze --file app.aab --top-files 20
gplay bundles compare --base old.aab --candidate new.aab --threshold 2M
```

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
