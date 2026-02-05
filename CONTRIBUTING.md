# Contributing

Thanks for your interest in contributing to gplay (Google Play Console CLI).

## Development Setup

Requirements:
- Go 1.21+

Clone and build:
```bash
git clone https://github.com/tamtom/play-console-cli.git
cd play-console-cli
make build
```

Run tests:
```bash
go test ./...
```

Optional tooling:
```bash
make tools   # installs gofumpt + golangci-lint
make lint    # uses golangci-lint if installed, else go vet
make format  # go fmt + gofumpt
make dev     # format + lint + test + build
```

## Integration Tests (Opt-in)

Integration tests hit the real Google Play API and are skipped by default.
Set credentials in your environment and run:

```bash
export GPLAY_SERVICE_ACCOUNT="/path/to/service-account.json"
export GPLAY_PACKAGE="com.example.app"

make test-integration
```

## Local API Testing (Optional)

If you have Google Play Console API credentials, you can run real API calls locally:

```bash
export GPLAY_SERVICE_ACCOUNT="/path/to/service-account.json"
export GPLAY_PACKAGE="com.example.app"

gplay tracks list --package "$GPLAY_PACKAGE"
gplay reviews list --package "$GPLAY_PACKAGE"
```

Credentials are stored in config with file path reference only.
Do not commit service account JSON files.

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

## Adding a New Command

1. Create a new package under `internal/cli/`:
   ```bash
   mkdir internal/cli/yourcommand
   ```

2. Follow the existing pattern (see other command files for reference):
   ```go
   package yourcommand

   import (
       "context"
       "flag"

       "github.com/peterbourgon/ff/v3/ffcli"
       "github.com/tamtom/play-console-cli/internal/cli/shared"
       "github.com/tamtom/play-console-cli/internal/playclient"
   )

   func YourCommand() *ffcli.Command {
       fs := flag.NewFlagSet("yourcommand", flag.ExitOnError)
       return &ffcli.Command{
           Name:       "yourcommand",
           ShortUsage: "gplay yourcommand <subcommand> [flags]",
           ShortHelp:  "Brief description.",
           FlagSet:    fs,
           UsageFunc:  shared.DefaultUsageFunc,
           Subcommands: []*ffcli.Command{
               // Add subcommands here
           },
           Exec: func(ctx context.Context, args []string) error {
               return flag.ErrHelp
           },
       }
   }
   ```

3. Register the command in `internal/cli/registry/registry.go`:
   ```go
   import "github.com/tamtom/play-console-cli/internal/cli/yourcommand"

   // In Subcommands():
   yourcommand.YourCommand(),
   ```

## CLI Implementation Checklist

- Register new commands in `internal/cli/registry/registry.go`
- Always set `UsageFunc: shared.DefaultUsageFunc` for command groups and subcommands
- For outbound HTTP, use `shared.ContextWithTimeout` so `GPLAY_TIMEOUT` applies
- Validate required flags and return clear error messages
- Use `shared.PrintOutput()` for consistent output formatting

## Common Patterns

**Flag handling:**
```go
packageName := fs.String("package", "", "Package name (applicationId)")
outputFlag := fs.String("output", "json", "Output format: json, table, markdown")
pretty := fs.Bool("pretty", false, "Pretty-print JSON output")
```

**Package name resolution:**
```go
pkg := shared.ResolvePackageName(*packageName, service.Cfg)
if strings.TrimSpace(pkg) == "" {
    return fmt.Errorf("--package is required")
}
```

**Context with timeout:**
```go
ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
defer cancel()
```

**Output formatting:**
```go
return shared.PrintOutput(result, *outputFlag, *pretty)
```

## Pull Request Guidelines

- Keep PRs small and focused
- Add or update tests for new behavior
- Update `README.md` if behavior or scope changes
- Avoid committing any credentials or service account files
- Run `make dev` before submitting

## Security

If you find a security issue, please report it responsibly by opening a private issue
or contacting the maintainer directly.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
