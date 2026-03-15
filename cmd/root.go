package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/registry"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// RootCommand constructs the root CLI command with all subcommands.
func RootCommand(version string) *ffcli.Command {
	rootFS := flag.NewFlagSet("gplay", flag.ExitOnError)
	dryRun := rootFS.Bool("dry-run", false, "Preview write operations without executing them")

	var root *ffcli.Command
	root = &ffcli.Command{
		Name:        "gplay",
		ShortUsage:  "gplay <command> [flags]",
		ShortHelp:   "A CLI for Google Play Console.",
		FlagSet:     rootFS,
		Subcommands: registry.Subcommands(version),
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

	// Store dryRun pointer for use in Run()
	rootDryRun = dryRun

	return root
}

// rootDryRun holds the parsed --dry-run flag value. Set during RootCommand().
var rootDryRun *bool
