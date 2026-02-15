// Package errors implements the `gplay vitals errors` command group for
// searching error issues and individual error reports via the Google Play
// Developer Reporting API.
package errors

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// ErrorsCommand returns the errors command group.
func ErrorsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("errors", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "errors",
		ShortUsage: "gplay vitals errors <subcommand> [flags]",
		ShortHelp:  "Search error issues and reports.",
		LongHelp: `Search grouped error issues and individual error reports.

Uses the Google Play Developer Reporting API to retrieve crash,
ANR, and non-fatal error data for your application.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			IssuesCommand(),
			ReportsCommand(),
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
