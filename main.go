package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/cli/registry"
)

var version = "dev"

func main() {
	rootFS := flag.NewFlagSet("gplay", flag.ExitOnError)
	dryRun := rootFS.Bool("dry-run", false, "Preview write operations without executing them")

	root := &ffcli.Command{
		Name:       "gplay",
		ShortUsage: "gplay <command> [flags]",
		ShortHelp:  "A CLI for Google Play Console.",
		FlagSet:    rootFS,
		Subcommands: registry.Subcommands(version),
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", args[0])
			return flag.ErrHelp
		},
	}

	ctx := context.Background()
	if err := root.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctx = shared.ContextWithDryRun(ctx, *dryRun)

	if err := root.Run(ctx); err != nil {
		if !shared.IsReportedError(err) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
