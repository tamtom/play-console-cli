// Package auditcmd implements the `gplay audit` command family.
package auditcmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/audit"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// AuditCommand builds the root `gplay audit` command.
func AuditCommand() *ffcli.Command {
	fs := flag.NewFlagSet("audit", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "audit",
		ShortUsage: "gplay audit <subcommand> [flags]",
		ShortHelp:  "Query and manage the local command audit log.",
		LongHelp: `Query and manage the local command audit log.

Every gplay command invocation appends a JSON entry to ~/.gplay/audit.log.
Disable with GPLAY_AUDIT=0. Override the path with GPLAY_AUDIT_LOG.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			listCommand(),
			searchCommand(),
			clearCommand(),
			pathCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", args[0])
			return flag.ErrHelp
		},
	}
}

func listCommand() *ffcli.Command {
	fs := flag.NewFlagSet("audit list", flag.ExitOnError)
	limit := fs.Int("limit", 50, "Maximum number of entries to show (0 = all)")
	since := fs.String("since", "", "Only include entries newer than this (RFC3339 or duration like 24h)")
	status := fs.String("status", "", "Filter by status (ok, error, started)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay audit list [flags]",
		ShortHelp:  "List recent audit entries (newest first).",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			q := audit.Query{Limit: *limit, Status: strings.TrimSpace(*status)}
			if s := strings.TrimSpace(*since); s != "" {
				t, err := parseSince(s)
				if err != nil {
					return fmt.Errorf("--since: %w", err)
				}
				q.Since = t
			}
			entries, err := audit.Read(q)
			if err != nil {
				return err
			}
			result := struct {
				Count   int            `json:"count"`
				Entries []audit.Entry  `json:"entries"`
				Query   map[string]any `json:"query,omitempty"`
			}{
				Count:   len(entries),
				Entries: entries,
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}

func searchCommand() *ffcli.Command {
	fs := flag.NewFlagSet("audit search", flag.ExitOnError)
	command := fs.String("command", "", "Substring to match against command name")
	status := fs.String("status", "", "Filter by status")
	limit := fs.Int("limit", 100, "Maximum number of results")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "search",
		ShortUsage: "gplay audit search --command <substr> [flags]",
		ShortHelp:  "Search audit entries by command or status.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*command) == "" && strings.TrimSpace(*status) == "" {
				return fmt.Errorf("at least one of --command or --status is required")
			}
			entries, err := audit.Read(audit.Query{
				Command: strings.TrimSpace(*command),
				Status:  strings.TrimSpace(*status),
				Limit:   *limit,
			})
			if err != nil {
				return err
			}
			return shared.PrintOutput(struct {
				Count   int           `json:"count"`
				Entries []audit.Entry `json:"entries"`
			}{Count: len(entries), Entries: entries}, *outputFlag, *pretty)
		},
	}
}

func clearCommand() *ffcli.Command {
	fs := flag.NewFlagSet("audit clear", flag.ExitOnError)
	confirm := fs.Bool("confirm", false, "Required to actually truncate the log")

	return &ffcli.Command{
		Name:       "clear",
		ShortUsage: "gplay audit clear --confirm",
		ShortHelp:  "Truncate the audit log.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if !*confirm {
				return fmt.Errorf("--confirm is required")
			}
			if err := audit.Clear(); err != nil {
				return err
			}
			path, _ := audit.Path()
			return shared.PrintOutput(struct {
				Cleared bool   `json:"cleared"`
				Path    string `json:"path"`
			}{Cleared: true, Path: path}, "json", false)
		},
	}
}

func pathCommand() *ffcli.Command {
	return &ffcli.Command{
		Name:       "path",
		ShortUsage: "gplay audit path",
		ShortHelp:  "Print the active audit log path.",
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			path, err := audit.Path()
			if err != nil {
				return err
			}
			fmt.Println(path)
			return nil
		},
	}
}

// parseSince accepts RFC3339 timestamps or relative durations like "24h", "7d".
func parseSince(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	// Support "d" suffix for days (time.ParseDuration doesn't).
	if strings.HasSuffix(s, "d") {
		days, err := parseInt(strings.TrimSuffix(s, "d"))
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid days value: %s", s)
		}
		return time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour), nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return time.Time{}, fmt.Errorf("expected RFC3339 or duration, got %q", s)
	}
	return time.Now().UTC().Add(-d), nil
}

func parseInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number: %s", s)
		}
		n = n*10 + int(c-'0')
	}
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	return n, nil
}
