package reports

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// ReportsCommand returns the top-level reports command group.
func ReportsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("reports", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "reports",
		ShortUsage: "gplay reports <subcommand> [flags]",
		ShortHelp:  "Download and manage Play Console reports.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			FinancialCommand(),
			StatsCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}
