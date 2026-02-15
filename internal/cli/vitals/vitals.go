package vitals

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/cli/vitals/performance"
)

// VitalsCommand returns the "gplay vitals" parent command group.
func VitalsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("vitals", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "vitals",
		ShortUsage: "gplay vitals <subcommand> [flags]",
		ShortHelp:  "App vitals: crashes, performance, and error reports.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			CrashesCommand(),
			performance.PerformanceCommand(),
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

// CrashesCommand returns the "gplay vitals crashes" command group.
func CrashesCommand() *ffcli.Command {
	fs := flag.NewFlagSet("crashes", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "crashes",
		ShortUsage: "gplay vitals crashes <subcommand> [flags]",
		ShortHelp:  "Query crash and ANR metrics.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			CrashesQueryCommand(),
			AnomaliesCommand(),
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
