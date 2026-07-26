# Using gplay with AI Agents

`gplay` is designed to be driven by AI coding agents — Claude Code, Codex, Cursor, Gemini CLI, Windsurf, or anything that can run shell commands.

## Why it works well with agents

- **Minified JSON by default** — machine-parseable output that saves tokens
- **Self-documenting** — every command supports `--help`; agents discover the interface instead of hallucinating it
- **Explicit long flags only** (`--package`, never `-p`) — unambiguous for generation and review
- **No interactive prompts** — destructive operations use `--confirm` flags, so agents never hang on stdin
- **`--dry-run` everywhere** — agents can show you exactly what would change before executing
- **`GPLAY.md`** — a complete auto-generated command reference an agent can read in one shot

## Point your agent at the reference

Add this to your project's `CLAUDE.md` / `AGENTS.md` / rules file:

```markdown
Use the `gplay` CLI for all Google Play Console operations.
Discover commands with `gplay --help` and `gplay <command> --help`.
Full reference: https://github.com/tamtom/play-console-cli/blob/main/GPLAY.md
```

## Install the Agent Skills

Pre-built workflow skills compatible with any agent that supports the [Agent Skills](https://github.com/anthropics/agent-skills) format:

```bash
npx skills add tamtom/gplay-cli-skills
```

| Skill | Description |
|-------|-------------|
| `gplay-cli-usage` | Guidance for running gplay commands (flags, pagination, output, auth) |
| `gplay-release-flow` | End-to-end release workflows for internal, beta, and production tracks |
| `gplay-gradle-build` | Build, sign, and package Android apps with Gradle before uploading |
| `gplay-metadata-sync` | Metadata and localization sync (including Fastlane format) |
| `gplay-rollout-management` | Staged rollout orchestration and monitoring |
| `gplay-review-management` | Review monitoring, filtering, and automated responses |
| `gplay-iap-setup` | In-app products, subscriptions, base plans, and offers |
| `gplay-ppp-pricing` | Purchasing-power-parity regional pricing |
| `gplay-purchase-verification` | Server-side purchase verification |
| `gplay-testers-orchestration` | Beta testing groups and tester management |
| `gplay-signing-setup` | Android app signing, keystores, and Play App Signing |
| `gplay-vitals-monitoring` | App vitals monitoring for crashes, errors, and performance |
| `gplay-user-management` | Developer account user and permission grant management |
| `gplay-migrate-fastlane` | Migration from Fastlane metadata to gplay format |
| `gplay-reports-download` | Financial and statistics report listing/downloading from GCS |
| `gplay-preflight` | Offline AAB/APK scanning: nine scanners, manifest decoding, CI gating |
| `gplay-submission-checks` | Pre-submission validation |
| `gplay-screenshot-automation` | Screenshot management workflows |
| `gplay-subscription-localization` | Subscription and in-app product localization |

Skills repository: [github.com/tamtom/gplay-cli-skills](https://github.com/tamtom/gplay-cli-skills)

## Example agent prompts

Things you can ask your agent once `gplay` is installed and authenticated:

- *"Release the AAB in `app/build/outputs` to the internal track with release notes from git history."*
- *"Create a monthly and yearly subscription with a 7-day free trial, priced at $4.99/$29.99, then convert prices for all regions."*
- *"Summarize this week's crash clusters and reply to every 1-star review mentioning crashes."*
- *"Set up the Play products, then import them into RevenueCat and attach them to a `premium` entitlement."*
- *"Download last month's earnings report and tell me revenue by country."*
