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

## Console-only publish gates (web session)

A few Play Console capabilities have no official API. The `gplay web apps` commands cover them by driving the console UI in a managed Chrome, using the browser session from `gplay web auth login` (separate from service-account auth):

- `gplay web apps status --package <id>` — read-only readiness: setup checklist with pending tasks, pending changes, `canSendForReview` and the console's own blocked reason
- `gplay web apps availability --package <id> [--track <id>] --countries "<a,b>" --confirm` — exact-match country targeting per track (`production` or a numeric track ID)
- `gplay web apps pricing --package <id> --price <amount> --confirm` — set a paid app's price in the merchant home currency
- `gplay web apps review --package <id> --confirm` — send pending changes for review
- `gplay web apps distribution --package <id> [--add "Android TV" --confirm]` — read or add exact-match form-factor opt-ins; legal review steps stay in Play Console
- `gplay web apps promo-codes --package <id> [--create ... --confirm]` — list campaigns or create generated paid-app codes; Terms acceptance stays in Play Console
- `gplay web apps rating --package <id>` — read the submitted IARC result, regional ratings, and unfinished draft state

These write commands verify the form before saving and re-read after saving, and they still need user confirmation like any outward action.

### Playbook: "Remove <country> to make your app paid"

When a pricing save is blocked, the app targets a country where paid distribution is not allowed (e.g. Sudan, or the "Rest of world" pseudo-country). The console names one country at a time. With the user's confirmation:

1. Read `gplay web apps status --package <id>` for the setup state.
2. Read `gplay web apps availability --package <id>` on production and on each testing track (`--track <numeric-id>` from the console's Manage track URL) to find where the country is targeted.
3. Set each track's list to itself minus the named country: `gplay web apps availability --package <id> [--track <id>] --countries "<current minus country>" --confirm`.
4. Set the price: `gplay web apps pricing --package <id> --price <amount> --confirm`.
5. Repeat if the console names another country.

## Example agent prompts

Things you can ask your agent once `gplay` is installed and authenticated:

- *"Release the AAB in `app/build/outputs` to the internal track with release notes from git history."*
- *"Create a monthly and yearly subscription with a 7-day free trial, priced at $4.99/$29.99, then convert prices for all regions."*
- *"Summarize this week's crash clusters and reply to every 1-star review mentioning crashes."*
- *"Set up the Play products, then import them into RevenueCat and attach them to a `premium` entitlement."*
- *"Download last month's earnings report and tell me revenue by country."*
