# CLAUDE.md

A fast, lightweight, AI-agent-friendly CLI for Google Play Console. Built in Go with [ffcli](https://github.com/peterbourgon/ff).

## Core Principles

- **Explicit flags**: Always `--package` not `-p`, `--output` not `-o`
- **JSON-first**: Minified JSON by default (saves tokens), `--output table/markdown` for humans
- **No interactive prompts**: Use `--confirm` flags for destructive operations
- **Pagination**: `--paginate` fetches all pages automatically

## Discovering Commands

**Use `--help` to discover commands and flags.** The CLI is self-documenting:

```bash
gplay --help                    # List all commands
gplay tracks --help             # List tracks subcommands
gplay tracks list --help        # Show all flags for a command
```

Do not memorize commands. Always check `--help` for the current interface.

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

Service accounts are created in Google Cloud Console and granted access in Play Console. Credentials are stored in config with file path reference. Never commit service account JSON files.

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
│   │   ├── registry/      # Command registration
│   │   ├── shared/        # Shared utilities
│   │   └── */             # Individual command packages
│   ├── output/            # Output formatting
│   ├── playclient/        # Google Play API client
│   ├── update/            # Self-update functionality
│   └── version/           # Version information
├── .github/workflows/     # CI/CD workflows
├── Makefile              # Build automation
└── install.sh            # Installation script
```

## References

- **Google Play Android Publisher API**: https://developers.google.com/android-publisher
- **API Reference**: https://developers.google.com/android-publisher/api-ref/rest
