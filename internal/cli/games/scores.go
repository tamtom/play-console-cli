package games

import (
	"context"
	"flag"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/gamesclient"
)

// ScoresCommand groups leaderboard-score reset commands (gamesManagement API).
func ScoresCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games scores", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "scores",
		ShortUsage: "gplay games scores <subcommand> [flags]",
		ShortHelp:  "Reset leaderboard scores for testers.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			scoresResetCommand(),
			scoresResetAllCommand(),
			scoresResetForAllCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

func scoresResetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games scores reset", flag.ExitOnError)
	id := fs.String("leaderboard-id", "", "Leaderboard ID")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "reset",
		ShortUsage: "gplay games scores reset --leaderboard-id <id> [flags]",
		ShortHelp:  "Reset the calling tester's scores for a leaderboard.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*id) == "" {
				return shared.UsageError("--leaderboard-id is required")
			}
			service, err := gamesclient.NewService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.Management.Scores.Reset(*id).Context(ctx).Do()
			if err != nil {
				return shared.WrapGoogleAPIError("reset leaderboard scores", err)
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func scoresResetAllCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games scores reset-all", flag.ExitOnError)
	confirm := fs.Bool("confirm", false, "Confirm resetting all leaderboard scores for the calling tester")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "reset-all",
		ShortUsage: "gplay games scores reset-all --confirm [flags]",
		ShortHelp:  "Reset the calling tester's scores for all leaderboards.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if !*confirm {
				return shared.UsageError("refusing to reset all scores without --confirm")
			}
			service, err := gamesclient.NewService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.Management.Scores.ResetAll().Context(ctx).Do()
			if err != nil {
				return shared.WrapGoogleAPIError("reset all leaderboard scores", err)
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func scoresResetForAllCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games scores reset-for-all-players", flag.ExitOnError)
	id := fs.String("leaderboard-id", "", "Leaderboard ID")
	confirm := fs.Bool("confirm", false, "Confirm resetting this leaderboard for ALL testers")

	return &ffcli.Command{
		Name:       "reset-for-all-players",
		ShortUsage: "gplay games scores reset-for-all-players --leaderboard-id <id> --confirm",
		ShortHelp:  "Reset a leaderboard's scores for all testers of the application.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if strings.TrimSpace(*id) == "" {
				return shared.UsageError("--leaderboard-id is required")
			}
			if !*confirm {
				return shared.UsageError("refusing to reset for all players without --confirm")
			}
			service, err := gamesclient.NewService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			if err := service.Management.Scores.ResetForAllPlayers(*id).Context(ctx).Do(); err != nil {
				return shared.WrapGoogleAPIError("reset leaderboard scores for all players", err)
			}
			return shared.PrintOutput(map[string]string{"status": "reset", "leaderboardId": *id}, "json", false)
		},
	}
}
