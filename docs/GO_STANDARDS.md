# Go Standards

## Error Handling

Wrap errors with context:
```go
return fmt.Errorf("listing tracks: %w", err)
```

## CLI Conventions

- **Explicit flags**: `--package`, `--output`, `--edit` (no single-letter aliases)
- **JSON-first output**: minified JSON by default, `--output table` or `--output markdown` for humans
- **No interactive prompts**: use `--confirm` for destructive operations
- **Pagination**: `--paginate` fetches all pages automatically

## Command Structure

Commands use [ffcli](https://github.com/peterbourgon/ff):

```go
func MyCommand() *ffcli.Command {
    fs := flag.NewFlagSet("mycommand", flag.ExitOnError)
    // register flags...
    return &ffcli.Command{
        Name:       "mycommand",
        ShortUsage: "gplay mycommand [flags]",
        ShortHelp:  "Brief description.",
        FlagSet:    fs,
        UsageFunc:  shared.DefaultUsageFunc,
        Exec:       execFunc,
    }
}
```

## Naming Conventions

- Command packages: `internal/cli/<command>/`
- Exported command constructor: `FooCommand() *ffcli.Command`
- Test files: `<name>_test.go` in the same package
- Integration tests: `<name>_integration_test.go` with build tag

## Code Organization

- `internal/cli/` — command implementations
- `internal/cli/shared/` — shared utilities (flags, timeouts, output)
- `internal/playclient/` — Google Play API client wrapper
- `internal/output/` — output formatting (JSON, table, markdown)
- `internal/config/` — configuration management
