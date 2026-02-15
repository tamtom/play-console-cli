# AGENTS.md

A fast, lightweight, AI-agent-friendly CLI for Google Play Console. Built in Go with [ffcli](https://github.com/peterbourgon/ff).

## Core Principles

- **Explicit flags**: Always `--package` not `-p`, `--output` not `-o`
- **JSON-first**: Minified JSON by default (saves tokens), `--output table/markdown` for humans
- **No interactive prompts**: Use `--confirm` flags for destructive operations
- **Pagination**: `--paginate` fetches all pages automatically
- **Dry run**: `--dry-run` intercepts write HTTP methods and logs to stderr

## Discovering Commands

**Use `--help` to discover commands and flags.** The CLI is self-documenting:

```bash
gplay --help                    # List all commands
gplay tracks --help             # List tracks subcommands
gplay tracks list --help        # Show all flags for a command
```

Do not memorize commands. Always check `--help` for the current interface.

### Command Reference

```bash
gplay apps list                 # List apps accessible by service account
gplay init                      # Initialize project configuration
gplay docs generate             # Generate markdown command reference
gplay vitals crashes            # Crash clusters and reports
gplay vitals performance        # Performance metrics (startup, rendering, battery)
gplay vitals errors             # Error issues and reports
gplay users                     # Manage developer account users (list, create, update, delete)
gplay grants                    # Manage per-app permission grants (create, update, delete)
gplay update                    # Self-update the CLI binary
gplay notify send               # Send webhook notifications (Slack, Discord, generic)
gplay migrate fastlane          # Migrate from Fastlane metadata
gplay reports financial         # Financial reports (list/download)
gplay reports stats             # Statistics reports (list/download)
gplay listings locales          # List available locales with validation
```

### Notable Flags

- `--dry-run` (global): Intercepts write HTTP methods, logs requests to stderr without executing
- `--video`: Attach video URL when updating listings
- `--fix` / `--confirm`: Auto-fix issues found by `auth doctor`
- `--listings-dir`, `--screenshots-dir`, `--skip-metadata`, `--skip-screenshots`: Control release metadata bundling
- Plain text release notes (no JSON): Auto-assigned to `en-US`

## Documentation

For Google Play Android Publisher API documentation:
- **API Overview**: https://developers.google.com/android-publisher
- **REST Reference**: https://developers.google.com/android-publisher/api-ref/rest

## Build & Test

```bash
make build      # Build binary
make test       # Run tests (always run before committing)
make lint       # Lint code
make format     # Format code
make dev        # format + lint + test + build
```

## Testing Discipline

- Use TDD for everything: bugs, refactors, and new features.
- Start with a failing test that captures the expected behavior and edge cases.
- For new features, begin with CLI-level tests (flags, output, errors) and add unit tests for core logic.
- Verify the test fails for the right reason before implementing; keep tests green incrementally.

## CLI Implementation Checklist

- Register new commands in `internal/cli/registry/registry.go`.
- Always set `UsageFunc: shared.DefaultUsageFunc` for command groups and subcommands.
- For outbound HTTP, use `shared.ContextWithTimeout` (or `shared.ContextWithUploadTimeout`) so `GPLAY_TIMEOUT` applies.
- Validate required flags and assert stderr error messages in tests (not just `flag.ErrHelp`).
- Use the existing patterns in other command files as reference.

## Authentication

Service accounts are required for the Google Play Android Developer API. Create one in Google Cloud Console and grant access in Play Console. Credentials are stored in config with file path reference. Never commit service account JSON files.

```bash
# Login with service account
gplay auth login --service-account /path/to/key.json

# Validate setup
gplay auth doctor

# Auto-fix issues (with confirmation)
gplay auth doctor --fix --confirm
```

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `GPLAY_SERVICE_ACCOUNT` | Path to service account JSON |
| `GPLAY_PACKAGE` | Default package name |
| `GPLAY_PROFILE` | Active profile name |
| `GPLAY_TIMEOUT` | Request timeout (e.g., `90s`, `2m`) |
| `GPLAY_TIMEOUT_SECONDS` | Timeout in seconds (alternative) |
| `GPLAY_UPLOAD_TIMEOUT` | Upload timeout (e.g., `5m`, `10m`) |
| `GPLAY_DEBUG` | Enable debug logging (set to `api` for HTTP requests) |
| `GPLAY_NO_UPDATE` | Disable update checks |
| `GPLAY_MAX_RETRIES` | Max retries for failed requests (default: 3) |
| `GPLAY_RETRY_DELAY` | Base delay between retries (default: `1s`) |
| `GPLAY_DEFAULT_OUTPUT` | Default output format (`json`, `table`, `markdown`) |

## Config File

Global config: `~/.gplay/config.yaml`
Local config: `./.gplay/config.yaml` (takes precedence)

```yaml
default_package: com.example.app
timeout: 120s
upload_timeout: 5m
max_retries: 3
debug: false
```

## Project Structure

```
play-console-cli/
├── cmd/                    # Main entry point
├── internal/
│   ├── cli/               # Command implementations
│   │   ├── apps/          # App listing commands
│   │   ├── docs/          # Documentation generation
│   │   ├── initcmd/       # Project initialization
│   │   ├── migrate/       # Fastlane migration
│   │   ├── notify/        # Webhook notifications
│   │   ├── registry/      # Command registration
│   │   ├── reports/       # Financial & stats reports
│   │   ├── shared/        # Shared utilities
│   │   ├── updatecmd/     # Self-update command
│   │   ├── users/         # User management
│   │   ├── vitals/        # Vitals command group
│   │   │   ├── errors/    # ANR and error reports
│   │   │   └── performance/ # Startup, rendering, battery
│   │   └── */             # Other command packages
│   ├── cmdtest/           # Black-box CLI testing
│   ├── config/            # Configuration management
│   ├── output/            # Output formatting (table, ANSI colors)
│   ├── playclient/        # Google Play API client
│   ├── reportingclient/   # Play Developer Reporting API client
│   ├── testutil/          # Shared test helpers
│   ├── update/            # Self-update functionality
│   └── version/           # Version information
├── .github/workflows/     # CI/CD workflows
├── Makefile              # Build automation
└── install.sh            # Installation script
```

## Agent Skills

For detailed workflow guidance, install the gplay skills:

```bash
npx skills add tamtom/gplay-cli-skills
```

Available skills:

- `gplay-cli-skills` - Core CLI workflows
- `gplay-vitals-monitoring` - Crash, ANR, and performance monitoring
- `gplay-user-management` - Developer account user and grant management
- `gplay-migrate-fastlane` - Fastlane metadata migration
- `gplay-submission-checks` - Pre-submission validation
- `gplay-screenshot-automation` - Screenshot management workflows
- `gplay-subscription-localization` - Subscription and in-app product localization

Skills repository: https://github.com/tamtom/gplay-cli-skills
