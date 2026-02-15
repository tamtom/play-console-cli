package vitals

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/cli/vitals/errors"
)

// VitalsCommand returns the vitals command group.
func VitalsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("vitals", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "vitals",
		ShortUsage: "gplay vitals <subcommand> [flags]",
		ShortHelp:  "App vitals: crashes, performance, and error reports.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			errors.ErrorsCommand(),
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
