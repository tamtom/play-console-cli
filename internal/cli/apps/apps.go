package apps

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v3/ffcli"
	cliruntime "github.com/tamtom/play-console-cli/internal/cli/runtime"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// AppsCommand returns the apps command group.
func AppsCommand(rt *cliruntime.Runtime) *ffcli.Command {
	fs := flag.NewFlagSet("apps", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "apps",
		ShortUsage: "gplay apps <subcommand> [flags]",
		ShortHelp:  "List and manage apps accessible by the service account.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			ListCommand(rt),
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
