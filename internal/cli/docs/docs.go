package docs

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// DocsCommand returns the docs command group.
func DocsCommand() *ffcli.Command {
	return DocsCommandWithRoot(nil)
}

// DocsCommandWithRoot returns the docs group backed by a lazy full command
// tree provider for generated reference output.
func DocsCommandWithRoot(root func() *ffcli.Command) *ffcli.Command {
	fs := flag.NewFlagSet("docs", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "docs",
		ShortUsage: "gplay docs <subcommand> [flags]",
		ShortHelp:  "Documentation and help topics.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			GenerateCommandWithRoot(root),
			ListCommand(),
			ShowCommand(),
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
