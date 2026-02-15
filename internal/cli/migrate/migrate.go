package migrate

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// MigrateCommand returns the parent migrate command.
func MigrateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "migrate",
		ShortUsage: "gplay migrate <subcommand> [flags]",
		ShortHelp:  "Migrate metadata from other tools.",
		LongHelp: `Import metadata from other store management tools.

Supported sources:
  fastlane    Import from Fastlane metadata/android/ directory structure`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			FastlaneCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}
