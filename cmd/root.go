package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/registry"
	cliruntime "github.com/tamtom/play-console-cli/internal/cli/runtime"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// RootCommand constructs the root CLI command with all subcommands.
func RootCommand(version string) *ffcli.Command {
	root, _ := constructRootCommand(version)
	return root
}

func constructRootCommand(version string) (*ffcli.Command, *cliruntime.Runtime) {
	rootFS := flag.NewFlagSet("gplay", flag.ExitOnError)
	rt := cliruntime.NewRoot(rootFS)

	var root *ffcli.Command
	root = &ffcli.Command{
		Name:        "gplay",
		ShortUsage:  "gplay <command> [flags]",
		ShortHelp:   "A CLI for Google Play Console.",
		FlagSet:     rootFS,
		Subcommands: registry.SubcommandsWithRuntime(version, rt),
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			var names []string
			for _, sub := range root.Subcommands {
				names = append(names, sub.Name)
			}
			fmt.Fprintln(os.Stderr, shared.FormatUnknownCommand(args[0], names))
			return flag.ErrHelp
		},
	}

	return root, rt
}
