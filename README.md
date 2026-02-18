# Google Play Console CLI (gplay)

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/License-MIT-yellow?style=for-the-badge" alt="License">
  <img src="https://img.shields.io/badge/Homebrew-compatible-blue?style=for-the-badge" alt="Homebrew">
</p>

A **fast**, **lightweight**, and **scriptable** CLI for Google Play Console. Automate your Android app workflows from your terminal.

## Why gplay?

Stop clicking through Play Console. Ship your Android apps with a single command.

**Publish & Release**
- One-command releases: upload, configure track, and go live in a single step
- Staged rollouts with pause, resume, and percentage control
- Promote builds between tracks (internal → beta → production)
- Generate release notes automatically from your git history
- Upload bundles (AAB) or APKs, manage edits, and commit changes

**Store Presence**
- Update store listings, screenshots, and app details across all locales
- Manage images: phone screenshots, tablet screenshots, feature graphics, and more
- Sync metadata with your local directory — export, import, and diff
- Migrate from Fastlane metadata format with a single command
- Validate listings, screenshots, and bundles before you submit
- Manage data safety declarations

**Monetization**
- In-app products: create, update, and batch-manage managed products
- One-time products for single purchases
- Subscriptions with base plans and promotional offers
- Price conversion across regions
- External transaction reporting (EU compliance)

**Purchases & Orders**
- Verify in-app purchases and subscription tokens server-side
- Look up and refund orders
- Acknowledge purchases programmatically

**Monitor Your App**
- Crash clusters and detailed crash reports
- ANR and error issue tracking
- Performance metrics: startup time, rendering jank, and battery drain
- Read and reply to user reviews without opening a browser

**Testing & Distribution**
- Manage testers for closed testing tracks
- Internal app sharing for quick testing without review
- Check country availability for your tracks
- Download device-specific APKs generated from your app bundle
- Upload deobfuscation files (ProGuard/R8 mapping) for readable crash reports
- System APK creation and expansion file (OBB) management
- App recovery actions

**Team & Permissions**
- Manage developer account users: invite, update roles, or remove
- Fine-grained per-app permission grants
- Multiple profiles for different accounts or apps

**Reports & Notifications**
- Download financial reports (earnings, sales, payouts) from Google Cloud Storage
- Download statistics reports (installs, ratings, crashes, store performance)
- Send release notifications to Slack, Discord, or any webhook

**Built for Automation**
- Works in any CI/CD pipeline — GitHub Actions, GitLab CI, Jenkins, and more
- JSON output by default — pipe to `jq`, scripts, or dashboards
- Table and Markdown output for human-friendly views
- Dry-run mode to preview changes before they go live
- Shell completions for Bash, Zsh, Fish, and PowerShell
- Self-updating: checks for new versions and upgrades in place
- Instant startup: single binary, no dependencies, no runtime
- Project initialization and auth diagnostics (`init`, `auth doctor`)
- Auto-generated command documentation (`docs generate`)
- Device tier configuration management

## Table of Contents

- [Why gplay?](#why-gplay)
- [Quick Start](#quick-start)
- [Commands](#commands)
  - [Scripting Tips](#scripting-tips)
  - [Publishing](#publishing)
  - [High-Level Workflow](#high-level-workflow)
  - [Store Listing](#store-listing)
  - [Monetization](#monetization)
  - [Purchase Management](#purchase-management)
  - [Reviews](#reviews)
  - [Testing](#testing)
  - [App Management](#app-management)
  - [Vitals & Quality](#vitals--quality)
  - [User & Permission Management](#user--permission-management)
  - [Reports](#reports)
  - [Notifications](#notifications)
  - [FastLane Integration](#fastlane-integration)
- [Output Formats](#output-formats)
- [Design Philosophy](#design-philosophy)
- [Installation](#installation)
- [Authentication](#authentication)
- [Configuration](#configuration)
- [CI/CD Integration](#cicd-integration)
- [Security](#security)
- [Contributing](#contributing)
- [Agent Skills](#agent-skills)
- [License](#license)

## Quick Start

### Install

```bash
# Via Homebrew (recommended)
brew tap tamtom/tap
brew install tamtom/tap/gplay

# Install script (macOS/Linux)
curl -fsSL https://raw.githubusercontent.com/tamtom/play-console-cli/main/install.sh | bash

# Build from source
git clone https://github.com/tamtom/play-console-cli.git
cd play-console-cli
make build
./gplay --help
```

### Updates

`gplay` checks for updates on startup and shows upgrade hints. Disable with `--no-update` or `GPLAY_NO_UPDATE=1`.

### Authenticate (Service Account)

**Step 1: Create a Google Cloud Project**
1. Go to [Google Cloud Console](https://console.cloud.google.com)
2. Create a new project or select an existing one
3. Note your project ID

**Step 2: Enable the API**
1. Go to **APIs & Services > Library**
2. Search for "Google Play Android Developer API"
3. Click **Enable**

**Step 3: Create a Service Account**
1. Go to **IAM & Admin > Service Accounts**
2. Click **Create Service Account**
3. Give it a name (e.g., "gplay-cli")
4. Click **Create and Continue**, then **Done**
5. Click on the created service account
6. Go to **Keys > Add Key > Create new key > JSON**
7. Save the downloaded JSON file securely

**Step 4: Grant Access in Play Console**
1. Go to [Google Play Console](https://play.google.com/console)
2. Go to **Users and permissions > Invite new users**
3. Enter the service account email (from the JSON file, looks like `name@project.iam.gserviceaccount.com`)
4. Set permissions (Admin or specific app access)
5. Click **Invite user**

**Step 5: Login with gplay**
```bash
gplay auth login --service-account /path/to/service-account.json

# Verify it works
gplay auth doctor
```

## Commands

### Scripting Tips

- JSON output is default for easy parsing; add `--pretty` when debugging
- Use `--paginate` to automatically fetch all pages
- Sort with `--sort` (prefix `-` for descending): `--sort -uploadedDate`
- Use `--limit` + `--next` for manual pagination control

### Publishing

```bash
# Edit lifecycle
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

### High-Level Workflow

```bash
# One-command release (creates edit, uploads, updates track, commits)
gplay release --package com.example.app --track internal --bundle app.aab

# With release notes and staged rollout
gplay release --package com.example.app --track production --bundle app.aab \
  --release-notes @notes.json --rollout 10

# Promote between tracks
gplay promote --package com.example.app --from internal --to beta

# Manage staged rollout
gplay rollout update --package com.example.app --track production --rollout 50
gplay rollout halt --package com.example.app --track production
gplay rollout resume --package com.example.app --track production
gplay rollout complete --package com.example.app --track production

# Release with metadata and screenshots
gplay release --package com.example.app --track production --bundle app.aab \
  --listings-dir ./metadata --screenshots-dir ./screenshots

# Dry-run any command (intercepts write operations)
gplay --dry-run release --package com.example.app --track internal --bundle app.aab
```

### App Management

```bash
# List apps accessible by your service account
gplay apps list

# Initialize project configuration
gplay init
gplay init --package com.example.app --service-account /path/to/sa.json
```

### Vitals & Quality

```bash
# Crash reports
gplay vitals crashes clusters --package com.example.app
gplay vitals crashes reports --package com.example.app

# Performance metrics
gplay vitals performance startup --package com.example.app
gplay vitals performance rendering --package com.example.app
gplay vitals performance battery --package com.example.app

# Error tracking
gplay vitals errors issues --package com.example.app
gplay vitals errors reports --package com.example.app
```

### User & Permission Management

```bash
# Manage developer account users
gplay users list --developer <id>
gplay users create --developer <id> --email user@example.com --json @permissions.json
gplay users delete --developer <id> --email user@example.com --confirm

# Manage per-app grants
gplay grants create --developer <id> --email user@example.com --package com.example.app --json @grant.json
gplay grants update --developer <id> --email user@example.com --package com.example.app --json @grant.json
gplay grants delete --developer <id> --email user@example.com --package com.example.app --confirm
```

### Reports

Reports are stored as CSV/ZIP files in Google Cloud Storage buckets (`pubsite_prod_rev_<developer_id>`). The service account must have access to the GCS bucket (granted automatically when added to Play Console).

> **Important:** The `--developer` ID for reports is **not** the developer ID in your Play Console URL. To find the correct ID, go to **Play Console > Download reports > Copy Cloud Storage URI**. The URI looks like `gs://pubsite_prod_rev_XXXX/` — the number after `pubsite_prod_rev_` is your developer ID.

```bash
# Financial reports (earnings, sales, payouts)
gplay reports financial list --developer <id>
gplay reports financial list --developer <id> --type earnings --from 2026-01 --to 2026-06
gplay reports financial download --developer <id> --from 2026-01 --type earnings --dir ./reports

# Statistics reports (installs, ratings, crashes, store_performance, subscriptions)
gplay reports stats list --developer <id>
gplay reports stats list --developer <id> --package com.example.app --type installs
gplay reports stats download --developer <id> --package com.example.app --from 2026-01 --type installs --dir ./reports
```

### Notifications

```bash
# Send webhook notifications (Slack, Discord, generic)
gplay notify send --webhook-url https://hooks.slack.com/... --message "Deploy complete" --format slack
gplay notify send --webhook-url https://discord.com/... --message "New release" --format discord
```

### Store Listing

```bash
# Listings
gplay listings list --package com.example.app --edit <id>
gplay listings get --package com.example.app --edit <id> --locale en-US
gplay listings update --package com.example.app --edit <id> --locale en-US --json @listing.json

# Images
gplay images list --package com.example.app --edit <id> --locale en-US --type phoneScreenshots
gplay images upload --package com.example.app --edit <id> --locale en-US --type phoneScreenshots --file screenshot.png

# App details
gplay details get --package com.example.app --edit <id>
gplay details update --package com.example.app --edit <id> --contact-email dev@example.com
```

### Monetization

```bash
# In-app products
gplay iap list --package com.example.app
gplay iap create --package com.example.app --sku premium_upgrade --json @product.json
gplay iap update --package com.example.app --sku premium_upgrade --json @product.json
gplay iap batch-update --package com.example.app --json @products.json

# Subscriptions
gplay subscriptions list --package com.example.app
gplay subscriptions create --package com.example.app --json @subscription.json

# Base plans
gplay baseplans activate --package com.example.app --product-id sub_premium --base-plan monthly
gplay baseplans deactivate --package com.example.app --product-id sub_premium --base-plan monthly

# Offers
gplay offers list --package com.example.app --product-id sub_premium --base-plan monthly
gplay offers create --package com.example.app --product-id sub_premium --base-plan monthly --json @offer.json

# Price conversion
gplay pricing convert --package com.example.app --json @price.json
```

### Purchase Management

```bash
# Verify purchases
gplay purchases products get --package com.example.app --product-id premium --token <token>
gplay purchases products acknowledge --package com.example.app --product-id premium --token <token>
gplay purchases subscriptions get --package com.example.app --token <token>

# Orders
gplay orders get --package com.example.app --order-id <id>
gplay orders refund --package com.example.app --order-id <id> --revoke

# External transactions (EU compliance)
gplay external-transactions create --package com.example.app --json @transaction.json
```

### Reviews

```bash
# List and filter reviews
gplay reviews list --package com.example.app
gplay reviews list --package com.example.app --paginate

# Reply to reviews
gplay reviews get --package com.example.app --review-id <id>
gplay reviews reply --package com.example.app --review-id <id> --text "Thank you!"
```

### Testing

```bash
# Manage testers
gplay testers list --package com.example.app --edit <id> --track internal
gplay testers update --package com.example.app --edit <id> --track internal --emails user@example.com

# Internal app sharing (quick sharing without review)
gplay internal-sharing upload-bundle --package com.example.app --file app.aab
gplay internal-sharing upload-apk --package com.example.app --file app.apk
```

### FastLane Integration

```bash
# Export metadata to FastLane format
gplay sync export-listings --package com.example.app --dir ./fastlane/metadata/android

# Import metadata from FastLane format
gplay sync import-listings --package com.example.app --dir ./fastlane/metadata/android

# Compare local metadata with Play Store
gplay sync diff-listings --package com.example.app --dir ./fastlane/metadata/android

# Validate before upload
gplay validate listing --dir ./fastlane/metadata/android --locale en-US
gplay validate screenshots --dir ./fastlane/metadata/android/en-US/images
gplay validate bundle --file app.aab
```

### Shell Completion

```bash
# Bash
gplay completion bash > /etc/bash_completion.d/gplay

# Zsh
gplay completion zsh > "${fpath[1]}/_gplay"

# Fish
gplay completion fish > ~/.config/fish/completions/gplay.fish

# PowerShell
gplay completion powershell >> $PROFILE
```

## Output Formats

| Format | Flag | Use Case |
|--------|------|----------|
| JSON (minified) | default | Scripting, automation |
| JSON (pretty) | `--pretty` | Debugging |
| Table | `--output table` | Terminal display |
| Markdown | `--output markdown` | Documentation |

```bash
# Parse with jq
gplay tracks list --package com.example.app | jq '.tracks[].track'

# Human-readable
gplay reviews list --package com.example.app --output table
```

## Design Philosophy

### Explicit Over Cryptic

```bash
# Good - self-documenting
gplay reviews list --package com.example.app --output table

# Avoid - cryptic flags (not supported)
# gplay reviews -p com.example.app -o table
```

### JSON-First Output

All commands output minified JSON by default for easy parsing:

```bash
gplay tracks list --package com.example.app | jq '.tracks[] | select(.track == "production")'
```

### No Interactive Prompts

Everything is flag-based for automation:

```bash
# Non-interactive (CI/CD safe)
gplay edits delete --package com.example.app --edit <id> --confirm
```

## Installation

### Homebrew (macOS)

```bash
brew tap tamtom/tap
brew install tamtom/tap/gplay
```

### Install Script (macOS/Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/tamtom/play-console-cli/main/install.sh | bash
```

Specify version:
```bash
GPLAY_VERSION=1.0.0 curl -fsSL https://raw.githubusercontent.com/tamtom/play-console-cli/main/install.sh | bash
```

### From Source

```bash
git clone https://github.com/tamtom/play-console-cli.git
cd play-console-cli
make build
make install  # Installs to /usr/local/bin
```

## Authentication

### Service Account Setup (Required)

Service accounts are required for the Google Play Android Developer API.

#### 1. Create Google Cloud Project & Enable API

```
Google Cloud Console → Create Project → APIs & Services → Library
→ Search "Google Play Android Developer API" → Enable
```

#### 2. Create Service Account & Download Key

```
IAM & Admin → Service Accounts → Create Service Account
→ Name it (e.g., "gplay-cli") → Create → Done
→ Click the account → Keys → Add Key → Create new key → JSON
→ Save the JSON file securely (never commit to git!)
```

#### 3. Grant Access in Play Console

```
Play Console → Users and permissions → Invite new users
→ Paste service account email (from JSON: "client_email" field)
→ Set permissions (Admin, or per-app access)
→ Invite user
```

#### 4. Configure gplay

```bash
# Option A: Login command (saves to profile)
gplay auth login --service-account /path/to/service-account.json

# Option B: Environment variable
export GPLAY_SERVICE_ACCOUNT=/path/to/service-account.json

# Verify setup
gplay auth doctor
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `GPLAY_SERVICE_ACCOUNT` | Path to service account JSON |
| `GPLAY_PACKAGE` | Default package name |
| `GPLAY_PROFILE` | Active profile name |
| `GPLAY_TIMEOUT` | Request timeout (e.g., `90s`, `2m`) |
| `GPLAY_UPLOAD_TIMEOUT` | Upload timeout (e.g., `5m`, `10m`) |
| `GPLAY_NO_UPDATE` | Disable update checks (set to `1`) |
| `GPLAY_DEBUG` | Enable debug logging (`1` or `api`) |
| `GPLAY_MAX_RETRIES` | Max retries for failed requests |
| `GPLAY_RETRY_DELAY` | Base delay between retries |
| `GPLAY_DEFAULT_OUTPUT` | Default output format (`json`, `table`, `markdown`) |

## Configuration

### Config File

Global: `~/.gplay/config.yaml`
Local (takes precedence): `./.gplay/config.yaml`

```yaml
default_package: com.example.app
timeout: 120s
upload_timeout: 5m
max_retries: 3
debug: false
```

### Profiles

```bash
# Add profiles for different accounts/apps
gplay auth login --profile work --service-account /path/to/work-sa.json
gplay auth login --profile personal --service-account /path/to/personal-sa.json

# Switch default profile
gplay auth switch --profile work

# Check current status
gplay auth status

# Use specific profile for a command
GPLAY_PROFILE=personal gplay tracks list --package com.example.app
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Deploy to Play Store

on:
  push:
    tags:
      - 'v*'

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up gplay
        run: |
          curl -fsSL https://raw.githubusercontent.com/tamtom/play-console-cli/main/install.sh | bash
          echo "$HOME/.local/bin" >> $GITHUB_PATH

      - name: Build app
        run: ./gradlew bundleRelease

      - name: Deploy to internal track
        env:
          GPLAY_SERVICE_ACCOUNT: ${{ secrets.PLAY_SERVICE_ACCOUNT }}
        run: |
          gplay release \
            --package com.example.app \
            --track internal \
            --bundle app/build/outputs/bundle/release/app-release.aab
```

### GitLab CI

```yaml
deploy:
  stage: deploy
  image: ubuntu:latest
  before_script:
    - curl -fsSL https://raw.githubusercontent.com/tamtom/play-console-cli/main/install.sh | bash
    - export PATH="$HOME/.local/bin:$PATH"
  script:
    - gplay release --package $PACKAGE_NAME --track internal --bundle app.aab
  variables:
    GPLAY_SERVICE_ACCOUNT: $PLAY_SERVICE_ACCOUNT
```

## Security

- **Never commit service account keys** to version control
- **Use environment variables** or secrets management in CI/CD
- **Limit service account permissions** to only what's needed
- **Rotate keys regularly**
- **Use separate service accounts** for different environments

Credentials are stored in config with file path reference only (not the key content).

## How to test in <10 minutes

```bash
make tools   # installs gofumpt + golangci-lint
make format
make lint
make test
make build
./gplay --help
```

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## Documentation

- [Agents.md](Agents.md) - Guidelines for AI agents (CLI usage, structure, patterns)
- [CONTRIBUTING.md](CONTRIBUTING.md) - Contribution guidelines

## Agent Skills

Use `gplay` with AI coding agents for assisted Android publishing workflows. Compatible with any agent that supports the [Agent Skills](https://github.com/anthropics/agent-skills) format.

### Install Skills

```bash
npx skills add tamtom/gplay-cli-skills
```

### Available Skills

| Skill | Description |
|-------|-------------|
| `gplay-cli-usage` | Guidance for running gplay commands (flags, pagination, output, auth) |
| `gplay-release-flow` | End-to-end release workflows for internal, beta, and production tracks |
| `gplay-gradle-build` | Build, sign, and package Android apps with Gradle before uploading |
| `gplay-metadata-sync` | Metadata and localization sync (including FastLane format) |
| `gplay-rollout-management` | Staged rollout orchestration and monitoring |
| `gplay-review-management` | Review monitoring, filtering, and automated responses |
| `gplay-iap-setup` | In-app products, subscriptions, base plans, and offers |
| `gplay-purchase-verification` | Server-side purchase verification |
| `gplay-testers-orchestration` | Beta testing groups and tester management |
| `gplay-signing-setup` | Android app signing, keystores, and Play App Signing |
| `gplay-vitals-monitoring` | App vitals monitoring for crashes, errors, and performance |
| `gplay-user-management` | Developer account user and permission grant management |
| `gplay-migrate-fastlane` | Migration from Fastlane metadata to gplay format |
| `gplay-reports-download` | Financial and statistics report listing/downloading from GCS |

Skills repository: [github.com/tamtom/gplay-cli-skills](https://github.com/tamtom/gplay-cli-skills)

## License

MIT License - see [LICENSE](LICENSE) for details.

---

<p align="center">
  Built with Go and the <a href="https://github.com/peterbourgon/ff">ffcli</a> framework
</p>
