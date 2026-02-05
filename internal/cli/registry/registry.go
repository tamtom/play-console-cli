package registry

import (
	"context"
	"fmt"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/auth"
	"github.com/tamtom/play-console-cli/internal/cli/apks"
	"github.com/tamtom/play-console-cli/internal/cli/bundles"
	"github.com/tamtom/play-console-cli/internal/cli/edits"
	"github.com/tamtom/play-console-cli/internal/cli/images"
	"github.com/tamtom/play-console-cli/internal/cli/listings"
	"github.com/tamtom/play-console-cli/internal/cli/reviews"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/cli/tracks"
)

// VersionCommand returns a version subcommand.
func VersionCommand(version string) *ffcli.Command {
	return &ffcli.Command{
		Name:       "version",
		ShortUsage: "gplay version",
		ShortHelp:  "Print version information and exit.",
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			fmt.Println(version)
			return nil
		},
	}
}

// Subcommands returns all root subcommands in display order.
func Subcommands(version string) []*ffcli.Command {
	return []*ffcli.Command{
		auth.AuthCommand(),
		edits.EditsCommand(),
		bundles.BundlesCommand(),
		apks.APKsCommand(),
		tracks.TracksCommand(),
		listings.ListingsCommand(),
		images.ImagesCommand(),
		reviews.ReviewsCommand(),
		VersionCommand(version),
	}
}
