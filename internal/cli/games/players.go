package games

import (
	"context"
	"flag"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	gamesmanagement "google.golang.org/api/gamesmanagement/v1management"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/gamesclient"
)

// PlayersCommand groups player hide/unhide/list commands (gamesManagement API).
func PlayersCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games players", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "players",
		ShortUsage: "gplay games players <subcommand> [flags]",
		ShortHelp:  "Hide, unhide, and list hidden players (testers).",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			playersHideCommand(),
			playersUnhideCommand(),
			playersListHiddenCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

func playersHideCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games players hide", flag.ExitOnError)
	appID := fs.String("application-id", "", "Play Games application ID (overrides GPLAY_GAMES_APP_ID/games_application_id)")
	playerID := fs.String("player-id", "", "Player ID to hide from leaderboards")

	return &ffcli.Command{
		Name:       "hide",
		ShortUsage: "gplay games players hide --application-id <id> --player-id <id>",
		ShortHelp:  "Hide a player's scores from an application's leaderboards.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			resolved, err := resolveApplicationID(*appID)
			if err != nil {
				return err
			}
			if resolved == "" {
				return shared.UsageError(missingApplicationIDMsg)
			}
			if strings.TrimSpace(*playerID) == "" {
				return shared.UsageError("--player-id is required")
			}
			service, err := gamesclient.NewService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			if err := service.Management.Players.Hide(resolved, *playerID).Context(ctx).Do(); err != nil {
				return shared.WrapGoogleAPIError("hide player", err)
			}
			return shared.PrintOutput(map[string]string{"status": "hidden", "applicationId": resolved, "playerId": *playerID}, "json", false)
		},
	}
}

func playersUnhideCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games players unhide", flag.ExitOnError)
	appID := fs.String("application-id", "", "Play Games application ID (overrides GPLAY_GAMES_APP_ID/games_application_id)")
	playerID := fs.String("player-id", "", "Player ID to unhide")

	return &ffcli.Command{
		Name:       "unhide",
		ShortUsage: "gplay games players unhide --application-id <id> --player-id <id>",
		ShortHelp:  "Unhide a previously hidden player's scores.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			resolved, err := resolveApplicationID(*appID)
			if err != nil {
				return err
			}
			if resolved == "" {
				return shared.UsageError(missingApplicationIDMsg)
			}
			if strings.TrimSpace(*playerID) == "" {
				return shared.UsageError("--player-id is required")
			}
			service, err := gamesclient.NewService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			if err := service.Management.Players.Unhide(resolved, *playerID).Context(ctx).Do(); err != nil {
				return shared.WrapGoogleAPIError("unhide player", err)
			}
			return shared.PrintOutput(map[string]string{"status": "unhidden", "applicationId": resolved, "playerId": *playerID}, "json", false)
		},
	}
}

func playersListHiddenCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games players list-hidden", flag.ExitOnError)
	appID := fs.String("application-id", "", "Play Games application ID (overrides GPLAY_GAMES_APP_ID/games_application_id)")
	maxResults := fs.Int("max-results", 0, "Maximum hidden players per page (server default when 0)")
	paginate := fs.Bool("paginate", false, "Fetch all pages")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list-hidden",
		ShortUsage: "gplay games players list-hidden --application-id <id> [flags]",
		ShortHelp:  "List players hidden from an application's leaderboards.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			resolved, err := resolveApplicationID(*appID)
			if err != nil {
				return err
			}
			if resolved == "" {
				return shared.UsageError(missingApplicationIDMsg)
			}
			service, err := gamesclient.NewService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			call := service.Management.Applications.ListHidden(resolved).Context(ctx)
			if *maxResults > 0 {
				call = call.MaxResults(int64(*maxResults))
			}
			if !*paginate {
				resp, err := call.Do()
				if err != nil {
					return shared.WrapGoogleAPIError("list hidden players", err)
				}
				return shared.PrintOutput(resp, *outputFlag, *pretty)
			}
			var all []*gamesmanagement.HiddenPlayer
			err = call.Pages(ctx, func(resp *gamesmanagement.HiddenPlayerList) error {
				all = append(all, resp.Items...)
				return nil
			})
			if err != nil {
				return shared.WrapGoogleAPIError("list hidden players", err)
			}
			return shared.PrintOutput(all, *outputFlag, *pretty)
		},
	}
}
