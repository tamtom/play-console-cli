package games

import (
	"context"
	"flag"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	gamesconfiguration "google.golang.org/api/gamesconfiguration/v1configuration"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/gamesclient"
)

// LeaderboardsCommand groups leaderboard configuration commands.
func LeaderboardsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games leaderboards", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "leaderboards",
		ShortUsage: "gplay games leaderboards <subcommand> [flags]",
		ShortHelp:  "Manage leaderboard definitions.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			leaderboardsListCommand(),
			leaderboardsGetCommand(),
			leaderboardsCreateCommand(),
			leaderboardsUpdateCommand(),
			leaderboardsDeleteCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

func leaderboardsListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games leaderboards list", flag.ExitOnError)
	appID := fs.String("application-id", "", "Play Games application ID (overrides GPLAY_GAMES_APP_ID/games_application_id)")
	maxResults := fs.Int("max-results", 0, "Maximum leaderboards per page (server default when 0)")
	paginate := fs.Bool("paginate", false, "Fetch all pages")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay games leaderboards list --application-id <id> [flags]",
		ShortHelp:  "List leaderboard configurations for an application.",
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

			call := service.Configuration.LeaderboardConfigurations.List(resolved).Context(ctx)
			if *maxResults > 0 {
				call = call.MaxResults(int64(*maxResults))
			}
			if !*paginate {
				resp, err := call.Do()
				if err != nil {
					return shared.WrapGoogleAPIError("list leaderboard configurations", err)
				}
				return shared.PrintOutput(resp, *outputFlag, *pretty)
			}
			var all []*gamesconfiguration.LeaderboardConfiguration
			err = call.Pages(ctx, func(resp *gamesconfiguration.LeaderboardConfigurationListResponse) error {
				all = append(all, resp.Items...)
				return nil
			})
			if err != nil {
				return shared.WrapGoogleAPIError("list leaderboard configurations", err)
			}
			return shared.PrintOutput(all, *outputFlag, *pretty)
		},
	}
}

func leaderboardsGetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games leaderboards get", flag.ExitOnError)
	id := fs.String("leaderboard-id", "", "Leaderboard ID")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "gplay games leaderboards get --leaderboard-id <id> [flags]",
		ShortHelp:  "Get a single leaderboard configuration.",
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

			resp, err := service.Configuration.LeaderboardConfigurations.Get(*id).Context(ctx).Do()
			if err != nil {
				return shared.WrapGoogleAPIError("get leaderboard configuration", err)
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func leaderboardsCreateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games leaderboards create", flag.ExitOnError)
	appID := fs.String("application-id", "", "Play Games application ID (overrides GPLAY_GAMES_APP_ID/games_application_id)")
	data := fs.String("data", "", "Leaderboard configuration JSON (inline or @file)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "create",
		ShortUsage: "gplay games leaderboards create --application-id <id> --data <json> [flags]",
		ShortHelp:  "Create (insert) a leaderboard configuration.",
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
			if strings.TrimSpace(*data) == "" {
				return shared.UsageError("--data is required (leaderboard configuration JSON, inline or @file)")
			}
			var body gamesconfiguration.LeaderboardConfiguration
			if err := shared.LoadJSONArg(*data, &body); err != nil {
				return shared.UsageErrorf("invalid --data JSON: %v", err)
			}
			service, err := gamesclient.NewService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.Configuration.LeaderboardConfigurations.Insert(resolved, &body).Context(ctx).Do()
			if err != nil {
				return shared.WrapGoogleAPIError("create leaderboard configuration", err)
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func leaderboardsUpdateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games leaderboards update", flag.ExitOnError)
	id := fs.String("leaderboard-id", "", "Leaderboard ID")
	data := fs.String("data", "", "Leaderboard configuration JSON (inline or @file)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "update",
		ShortUsage: "gplay games leaderboards update --leaderboard-id <id> --data <json> [flags]",
		ShortHelp:  "Update a leaderboard configuration (full replace).",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*id) == "" {
				return shared.UsageError("--leaderboard-id is required")
			}
			if strings.TrimSpace(*data) == "" {
				return shared.UsageError("--data is required (leaderboard configuration JSON, inline or @file)")
			}
			var body gamesconfiguration.LeaderboardConfiguration
			if err := shared.LoadJSONArg(*data, &body); err != nil {
				return shared.UsageErrorf("invalid --data JSON: %v", err)
			}
			service, err := gamesclient.NewService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.Configuration.LeaderboardConfigurations.Update(*id, &body).Context(ctx).Do()
			if err != nil {
				return shared.WrapGoogleAPIError("update leaderboard configuration", err)
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func leaderboardsDeleteCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games leaderboards delete", flag.ExitOnError)
	id := fs.String("leaderboard-id", "", "Leaderboard ID")
	confirm := fs.Bool("confirm", false, "Confirm deletion")

	return &ffcli.Command{
		Name:       "delete",
		ShortUsage: "gplay games leaderboards delete --leaderboard-id <id> --confirm",
		ShortHelp:  "Delete a leaderboard configuration.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if strings.TrimSpace(*id) == "" {
				return shared.UsageError("--leaderboard-id is required")
			}
			if !*confirm {
				return shared.UsageError("refusing to delete without --confirm")
			}
			service, err := gamesclient.NewService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			if err := service.Configuration.LeaderboardConfigurations.Delete(*id).Context(ctx).Do(); err != nil {
				return shared.WrapGoogleAPIError("delete leaderboard configuration", err)
			}
			return shared.PrintOutput(map[string]string{"status": "deleted", "leaderboardId": *id}, "json", false)
		},
	}
}
