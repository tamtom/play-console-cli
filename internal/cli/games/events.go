package games

import (
	"context"
	"flag"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/gamesclient"
)

// EventsCommand groups event-progress reset commands (gamesManagement API).
func EventsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games events", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "events",
		ShortUsage: "gplay games events <subcommand> [flags]",
		ShortHelp:  "Reset event progress for testers.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			eventsResetCommand(),
			eventsResetAllCommand(),
			eventsResetForAllCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

func eventsResetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games events reset", flag.ExitOnError)
	id := fs.String("event-id", "", "Event ID")

	return &ffcli.Command{
		Name:       "reset",
		ShortUsage: "gplay games events reset --event-id <id>",
		ShortHelp:  "Reset the calling tester's progress for an event.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if strings.TrimSpace(*id) == "" {
				return shared.UsageError("--event-id is required")
			}
			service, err := gamesclient.NewService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			if err := service.Management.Events.Reset(*id).Context(ctx).Do(); err != nil {
				return shared.WrapGoogleAPIError("reset event progress", err)
			}
			return shared.PrintOutput(map[string]string{"status": "reset", "eventId": *id}, "json", false)
		},
	}
}

func eventsResetAllCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games events reset-all", flag.ExitOnError)
	confirm := fs.Bool("confirm", false, "Confirm resetting all events for the calling tester")

	return &ffcli.Command{
		Name:       "reset-all",
		ShortUsage: "gplay games events reset-all --confirm",
		ShortHelp:  "Reset the calling tester's progress for all events.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if !*confirm {
				return shared.UsageError("refusing to reset all events without --confirm")
			}
			service, err := gamesclient.NewService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			if err := service.Management.Events.ResetAll().Context(ctx).Do(); err != nil {
				return shared.WrapGoogleAPIError("reset all event progress", err)
			}
			return shared.PrintOutput(map[string]string{"status": "reset-all"}, "json", false)
		},
	}
}

func eventsResetForAllCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games events reset-for-all-players", flag.ExitOnError)
	id := fs.String("event-id", "", "Event ID")
	confirm := fs.Bool("confirm", false, "Confirm resetting this event for ALL testers")

	return &ffcli.Command{
		Name:       "reset-for-all-players",
		ShortUsage: "gplay games events reset-for-all-players --event-id <id> --confirm",
		ShortHelp:  "Reset an event for all testers of the application.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if strings.TrimSpace(*id) == "" {
				return shared.UsageError("--event-id is required")
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

			if err := service.Management.Events.ResetForAllPlayers(*id).Context(ctx).Do(); err != nil {
				return shared.WrapGoogleAPIError("reset event for all players", err)
			}
			return shared.PrintOutput(map[string]string{"status": "reset", "eventId": *id}, "json", false)
		},
	}
}
