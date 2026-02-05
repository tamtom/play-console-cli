# Google Play Console CLI (gplay)

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/License-MIT-yellow?style=for-the-badge" alt="License">
  <img src="https://img.shields.io/badge/Homebrew-compatible-blue?style=for-the-badge" alt="Homebrew">
</p>

A **fast**, **lightweight**, and **scriptable** CLI for Google Play Console. Automate your Android app workflows from your terminal.

## Why gplay?

| Problem | Solution |
|---------|----------|
| Manual Play Console work | Automate everything from CLI |
| Slow, heavy tooling | Single Go binary, instant startup |
| Poor scripting support | JSON output, explicit flags, clean exit codes |
| Complex release workflows | High-level commands like `release`, `promote`, `rollout` |

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
  - [FastLane Integration](#fastlane-integration)
- [Output Formats](#output-formats)
- [Design Philosophy](#design-philosophy)
- [Installation](#installation)
- [Authentication](#authentication)
- [Configuration](#configuration)
- [CI/CD Integration](#cicd-integration)
- [Security](#security)
- [Contributing](#contributing)
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

### Authenticate

```bash
# Using service account (recommended for CI/CD)
gplay auth login --service-account /path/to/service-account.json

# Add named profile
gplay auth add-profile production --service-account /path/to/prod-sa.json

# Switch profiles
gplay auth switch --name production

# Use profile for single command
gplay --profile production tracks list --package com.example.app

# Validate setup
gplay auth doctor
```

Generate service accounts at: https://console.cloud.google.com/iam-admin/serviceaccounts

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

### Service Account (Recommended)

1. Create a service account in [Google Cloud Console](https://console.cloud.google.com/iam-admin/serviceaccounts)
2. Grant the service account access in [Play Console](https://play.google.com/console) → Users and permissions
3. Download the JSON key file

```bash
# Via environment variable
export GPLAY_SERVICE_ACCOUNT=/path/to/service-account.json

# Via flag
gplay --service-account /path/to/service-account.json <command>

# Via profile
gplay auth add-profile production --service-account /path/to/service-account.json
gplay auth use-profile production
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
# Add profiles
gplay auth add-profile work --service-account /path/to/work-sa.json
gplay auth add-profile personal --service-account /path/to/personal-sa.json

# Switch profiles
gplay auth switch --name work
gplay auth current-profile

# List profiles
gplay auth list-profiles

# Use profile for single command
gplay --profile personal apps list
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

- [CLAUDE.md](CLAUDE.md) - Development guidelines for AI assistants
- [CONTRIBUTING.md](CONTRIBUTING.md) - Contribution guidelines

## License

MIT License - see [LICENSE](LICENSE) for details.

---

<p align="center">
  Built with Go and the <a href="https://github.com/peterbourgon/ff">ffcli</a> framework
</p>
