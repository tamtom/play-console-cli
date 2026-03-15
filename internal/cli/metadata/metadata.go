package metadata

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// MetadataCommand returns the metadata parent command group.
func MetadataCommand() *ffcli.Command {
	fs := flag.NewFlagSet("metadata", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "metadata",
		ShortUsage: "gplay metadata <subcommand> [flags]",
		ShortHelp:  "File-based metadata sync (pull/push/validate).",
		LongHelp: `Pull, push, and validate store listing metadata as local files.

The metadata directory uses a flat structure with one directory per locale:
  metadata/
    en-US/
      title.txt
      short_description.txt
      full_description.txt
      video_url.txt
    ja-JP/
      title.txt
      ...

Use 'gplay metadata pull' to download current listings to files.
Use 'gplay metadata push' to upload file changes to the store.
Use 'gplay metadata validate' to check metadata locally before pushing.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			PullCommand(),
			PushCommand(),
			ValidateCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}
