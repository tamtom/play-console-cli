// Package games implements the `gplay games` command group for managing Play
// Games Services content (achievements, leaderboards) and player/progress
// operations via the Play Games Publishing and Management APIs.
package games

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// GamesCommand returns the Play Games command group.
func GamesCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "games",
		ShortUsage: "gplay games <subcommand> [flags]",
		ShortHelp:  "Manage Play Games Services achievements, leaderboards, and player progress.",
		LongHelp: `Manage Play Games Services achievements, leaderboards, and player progress.

Games commands use the https://www.googleapis.com/auth/androidpublisher OAuth
scope (the same service account credentials as the rest of gplay) across two
API surfaces:

  - gamesConfiguration (https://gamesconfiguration.googleapis.com) manages the
    achievement and leaderboard definitions shown in the Play Games section of
    Play Console.
  - gamesManagement (https://gamesmanagement.googleapis.com) resets player
    progress and hides/unhides players. These operations target testers only.

Identifiers differ from the rest of gplay: list/create operations take the
numeric Play Games application ID (--application-id, or GPLAY_GAMES_APP_ID /
games_application_id in config), NOT the Android package name. Individual
resources are addressed by their string achievement/leaderboard/event IDs.

The service account must be linked in the Play Games Services setup for the
game; a normal Play Console grant is not sufficient.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			AchievementsCommand(),
			LeaderboardsCommand(),
			ScoresCommand(),
			EventsCommand(),
			PlayersCommand(),
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
