package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/registry"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

var version = "dev"

func main() {
	var root *ffcli.Command
	root = &ffcli.Command{
		Name:        "gplay",
		ShortUsage:  "gplay <command> [flags]",
		ShortHelp:   "A CLI for Google Play Console.",
		FlagSet:     flag.NewFlagSet("gplay", flag.ExitOnError),
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

	if err := root.ParseAndRun(context.Background(), os.Args[1:]); err != nil {
		if !shared.IsReportedError(err) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
