package checks

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// ChecksCommand returns the Checks API command group.
func ChecksCommand() *ffcli.Command {
	fs := flag.NewFlagSet("checks", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "checks",
		ShortUsage: "gplay checks <subcommand> [flags]",
		ShortHelp:  "Analyze app privacy, compliance, and AI safety with the Google Checks API.",
		LongHelp: `Analyze app privacy, compliance, and AI safety with the Google Checks API.

Checks commands use the https://www.googleapis.com/auth/checks OAuth scope and
the https://checks.googleapis.com/v1alpha API surface. App bundle analysis uses
the current media upload endpoint, /upload/v1alpha/{parent=accounts/*/apps/*}/reports:analyzeUpload.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			AppsCommand(),
			ReportsCommand(),
			OperationsCommand(),
			UploadCommand("upload"),
			UploadCommand("analyze"),
			ContentCommand(),
			ClassifyCommand("classify"),
			RepoScansCommand(),
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
