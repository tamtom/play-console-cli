# gplay — Google Play Console CLI for AI Agents

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/License-MIT-yellow?style=for-the-badge" alt="License">
  <img src="https://img.shields.io/badge/Homebrew-compatible-blue?style=for-the-badge" alt="Homebrew">
</p>

**gplay** is a fast, single-binary CLI for **Google Play Console**, built for **AI coding agents** — Claude Code, Codex, Cursor, Gemini CLI — and for humans who script. Release Android apps, create **subscriptions, in-app products, and one-time purchases**, verify purchases server-side, monitor crashes and reviews — all from the terminal, no Play Console clicking.

It **keeps you out of the web browser console** for day-to-day work. Most deployment tools only handle standard AAB uploads — gplay supports **250+ endpoints across 6 Google APIs** (Android Publisher, Play Developer Reporting, Checks, Play Games, Managed Google Play, and Cloud Storage): complete console management, staged rollouts, store listings, screenshots, localization, vitals, and private app publishing. And the API-backed commands are **lightweight with no runtime to install** — no Node.js, no Python, no JVM, just one static binary. The optional `gplay web` commands are the exception: they drive a real local Google Chrome and are macOS-only today.

## Table of Contents

- [Quick Start](#quick-start)
  - [Install](#install)
  - [Authentication](#authentication)
- [Highlights](#highlights)
- [Use it with your AI agent](#use-it-with-your-ai-agent)
- [How gplay compares](#how-gplay-compares)
- [Documentation](#documentation)
- [Contributing](#contributing)
- [License](#license)

## Quick Start

### Install

```bash
# Homebrew (recommended)
brew tap tamtom/tap
brew install tamtom/tap/gplay

# Standalone binary — macOS/Linux
curl -fsSL https://raw.githubusercontent.com/tamtom/play-console-cli/main/install.sh | bash

# Standalone binary — Windows (PowerShell)
irm https://raw.githubusercontent.com/tamtom/play-console-cli/main/install.ps1 | iex
```

**Update:** `gplay update` self-updates in place. It also checks for new versions on startup (disable with `GPLAY_NO_UPDATE=1`).

### Authentication

`gplay` authenticates to Google Play with a **service account**.

#### One-command setup (recommended)

```bash
gplay setup --auto
```

That's it. This will:
- Install `gcloud` if needed (via Homebrew on macOS, or curl on Linux)
- Log you into Google Cloud
- Create a service account and download credentials
- Open Play Console for the one manual step (granting access)
- Configure everything automatically

Pass `--project <id>` if `gcloud` has no default project, `--dry-run` to preview the commands, or `--no-browser` for CI/agent runs. Then verify with `gplay auth doctor`.

Already have a service account key?

```bash
gplay auth login --service-account /path/to/service-account.json
```

#### Manual setup

<details>
<summary>Set up the service account by hand (full control)</summary>

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

</details>

Profiles for multiple accounts and troubleshooting: **[docs/authentication.md](docs/authentication.md)**

Once authenticated, you're ready to go:

```bash
# Ship a release in one command
gplay release --package com.example.app --track production --bundle app.aab --rollout 0.1

# Create a subscription with base plans and offers
gplay subscriptions create --package com.example.app --json @subscription.json

# Everything outputs minified JSON — built for agents and pipes
gplay reviews list --package com.example.app | jq '.reviews[0]'
```

## Highlights

- **250+ commands across 6 Google APIs** — not just AAB uploads like most deployment tools. gplay covers 98% of the Google Play Developer API v3 (134/137 endpoints) plus Play Developer Reporting, Checks, Play Games, Managed Google Play, and Cloud Storage — complete console management from the terminal, so you drive Play Console from the terminal instead of clicking through it.
- **Managed Google Play** — publish private (custom) apps to specific organizations with `gplay custom-apps create`, straight from the terminal.
- **Console-only features via `gplay web`** — the gaps the official API leaves: create apps, change category, country availability, paid-app pricing, App content declarations, policy status, distribution, promo codes, content rating, and send-for-review / publish. These drive a real local Google Chrome (macOS only today) against an unofficial console interface; session setup in [docs/authentication.md](docs/authentication.md#web-sessions-browser-cookies).
- **Complete releases & rollouts** — upload, track assignment, staged rollout with pause/resume/percentage control, promote between tracks, release notes from git history.
- **Store listings, screenshots & localization** — manage listing text, images, and metadata across every locale; sync with a local directory or Fastlane.
- **Full monetization stack** — subscriptions, base plans, promotional offers, in-app products, one-time purchases, regional price conversion. Pairs with [RevenueCat](docs/monetization.md#using-gplay-with-revenuecat): create products in Play with `gplay`, import them into RevenueCat.
- **Purchase verification** — verify tokens, acknowledge purchases, refund orders, decode RTDN webhooks.
- **Offline preflight** — `gplay preflight --file app.aab` fully decodes `AndroidManifest.xml` (binary AXML for APKs, aapt2 protobuf for App Bundles) and runs nine scanners with no API calls and no credentials: manifest flags, restricted permissions, 64-bit and **16 KB page alignment**, listing text and real screenshot dimensions, secrets, billing, privacy SDKs, target API floor, and size. Catches the rejections *before* you upload.
- **App health** — crash clusters, ANRs, performance vitals, review replies, financial & statistics reports.
- **AI-agent native** — minified JSON output, explicit flags, `--help` everywhere, `--dry-run` for every write, no interactive prompts, [Agent Skills](docs/ai-agents.md) included.
- **Lightweight, no runtime to install** — a single compiled Go binary with instant startup. No Node.js, no Python, no JVM, no Gradle project required. Only the `gplay web` commands above need anything extra: a local Google Chrome.

## Use it with your AI agent

Every command is discoverable via `--help`, outputs token-efficient JSON, and never blocks on a prompt — so agents can drive the whole Play workflow. Install the ready-made skills:

```bash
npx skills add tamtom/gplay-cli-skills
```

Then ask your agent things like *"release this AAB to internal"*, *"create monthly + yearly subscriptions with a 7-day trial and convert prices for all regions"*, or *"summarize this week's crash clusters"*. Full guide: **[docs/ai-agents.md](docs/ai-agents.md)**

## How gplay compares

Versus [Fastlane `supply`](https://docs.fastlane.tools/actions/supply/) and [gradle-play-publisher](https://github.com/Triple-T/gradle-play-publisher) (all three cover AAB/APK upload, tracks, rollouts, and listings equivalently):

| Capability | gplay | Fastlane supply | gradle-play-publisher |
|---|---|---|---|
| Subscriptions, base plans, offers, one-time purchases | ✅ Full | ❌ | ⚠️ Basic (no base plans/offers) |
| Purchase verification, orders, refunds | ✅ | ❌ | ❌ |
| Vitals: crashes, ANRs, performance | ✅ | ❌ | ❌ |
| Reviews: read + reply | ✅ | ❌ | ❌ |
| Financial & statistics reports | ✅ | ❌ | ❌ |
| Users & permission grants | ✅ | ❌ | ❌ |
| Offline preflight (manifest decode, 16 KB alignment, secrets) | ✅ 9 scanners | ❌ | ❌ |
| Managed Google Play (private/custom apps) | ✅ | ❌ | ❌ |
| Google Checks compliance gate | ✅ | ❌ | ❌ |
| AI-agent-friendly output | ✅ JSON default | ❌ Human logs | ❌ Gradle logs |
| Runtime | Single Go binary | Ruby + gems | JVM + Gradle project |

## Documentation

| Guide | What's inside |
|-------|---------------|
| **[Command reference (GPLAY.md)](GPLAY.md)** | Every command and flag, auto-generated |
| [Authentication](docs/authentication.md) | Service account setup, profiles, `auth doctor` |
| [Releasing](docs/releasing.md) | Releases, rollouts, listings, testers, pre-submission checks, Fastlane interop |
| [Monetization](docs/monetization.md) | Subscriptions, IAP, one-time purchases, pricing, purchase verification, RevenueCat |
| [Monitoring & operations](docs/monitoring.md) | Vitals, reviews, reports, users, notifications, diagnostics |
| [AI agents](docs/ai-agents.md) | Agent setup, skills, example prompts |
| [Configuration](docs/configuration.md) | Config file, env vars, output formats, shell completion |
| [CI/CD](docs/ci-cd.md) | GitHub Actions, GitLab CI, release gates |
| [Contributing](CONTRIBUTING.md) | Development setup, testing, PR guidelines |

## Contributing

Contributions welcome — see [CONTRIBUTING.md](CONTRIBUTING.md). Quick local check: `make dev` (format + lint + test + build).

## License

MIT — see [LICENSE](LICENSE).

---

<p align="center">
  Built with Go and <a href="https://github.com/peterbourgon/ff">ffcli</a>
</p>
