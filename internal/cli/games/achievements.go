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

// AchievementsCommand groups achievement configuration and progress-reset commands.
func AchievementsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games achievements", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "achievements",
		ShortUsage: "gplay games achievements <subcommand> [flags]",
		ShortHelp:  "Manage achievement definitions and reset achievement progress.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			achievementsListCommand(),
			achievementsGetCommand(),
			achievementsCreateCommand(),
			achievementsUpdateCommand(),
			achievementsDeleteCommand(),
			achievementsResetCommand(),
			achievementsResetAllCommand(),
			achievementsResetForAllCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

func achievementsListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games achievements list", flag.ExitOnError)
	appID := fs.String("application-id", "", "Play Games application ID (overrides GPLAY_GAMES_APP_ID/games_application_id)")
	maxResults := fs.Int("max-results", 0, "Maximum achievements per page (server default when 0)")
	paginate := fs.Bool("paginate", false, "Fetch all pages")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay games achievements list --application-id <id> [flags]",
		ShortHelp:  "List achievement configurations for an application.",
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

			call := service.Configuration.AchievementConfigurations.List(resolved).Context(ctx)
			if *maxResults > 0 {
				call = call.MaxResults(int64(*maxResults))
			}
			if !*paginate {
				resp, err := call.Do()
				if err != nil {
					return shared.WrapGoogleAPIError("list achievement configurations", err)
				}
				return shared.PrintOutput(resp, *outputFlag, *pretty)
			}
			var all []*gamesconfiguration.AchievementConfiguration
			err = call.Pages(ctx, func(resp *gamesconfiguration.AchievementConfigurationListResponse) error {
				all = append(all, resp.Items...)
				return nil
			})
			if err != nil {
				return shared.WrapGoogleAPIError("list achievement configurations", err)
			}
			return shared.PrintOutput(all, *outputFlag, *pretty)
		},
	}
}

func achievementsGetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games achievements get", flag.ExitOnError)
	id := fs.String("achievement-id", "", "Achievement ID")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "gplay games achievements get --achievement-id <id> [flags]",
		ShortHelp:  "Get a single achievement configuration.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*id) == "" {
				return shared.UsageError("--achievement-id is required")
			}
			service, err := gamesclient.NewService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.Configuration.AchievementConfigurations.Get(*id).Context(ctx).Do()
			if err != nil {
				return shared.WrapGoogleAPIError("get achievement configuration", err)
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func achievementsCreateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games achievements create", flag.ExitOnError)
	appID := fs.String("application-id", "", "Play Games application ID (overrides GPLAY_GAMES_APP_ID/games_application_id)")
	data := fs.String("data", "", "Achievement configuration JSON (inline or @file)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "create",
		ShortUsage: "gplay games achievements create --application-id <id> --data <json> [flags]",
		ShortHelp:  "Create (insert) an achievement configuration.",
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
				return shared.UsageError("--data is required (achievement configuration JSON, inline or @file)")
			}
			var body gamesconfiguration.AchievementConfiguration
			if err := shared.LoadJSONArg(*data, &body); err != nil {
				return shared.UsageErrorf("invalid --data JSON: %v", err)
			}
			service, err := gamesclient.NewService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.Configuration.AchievementConfigurations.Insert(resolved, &body).Context(ctx).Do()
			if err != nil {
				return shared.WrapGoogleAPIError("create achievement configuration", err)
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func achievementsUpdateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games achievements update", flag.ExitOnError)
	id := fs.String("achievement-id", "", "Achievement ID")
	data := fs.String("data", "", "Achievement configuration JSON (inline or @file)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "update",
		ShortUsage: "gplay games achievements update --achievement-id <id> --data <json> [flags]",
		ShortHelp:  "Update an achievement configuration (full replace).",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*id) == "" {
				return shared.UsageError("--achievement-id is required")
			}
			if strings.TrimSpace(*data) == "" {
				return shared.UsageError("--data is required (achievement configuration JSON, inline or @file)")
			}
			var body gamesconfiguration.AchievementConfiguration
			if err := shared.LoadJSONArg(*data, &body); err != nil {
				return shared.UsageErrorf("invalid --data JSON: %v", err)
			}
			service, err := gamesclient.NewService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.Configuration.AchievementConfigurations.Update(*id, &body).Context(ctx).Do()
			if err != nil {
				return shared.WrapGoogleAPIError("update achievement configuration", err)
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func achievementsDeleteCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games achievements delete", flag.ExitOnError)
	id := fs.String("achievement-id", "", "Achievement ID")
	confirm := fs.Bool("confirm", false, "Confirm deletion")

	return &ffcli.Command{
		Name:       "delete",
		ShortUsage: "gplay games achievements delete --achievement-id <id> --confirm",
		ShortHelp:  "Delete an achievement configuration.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if strings.TrimSpace(*id) == "" {
				return shared.UsageError("--achievement-id is required")
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

			if err := service.Configuration.AchievementConfigurations.Delete(*id).Context(ctx).Do(); err != nil {
				return shared.WrapGoogleAPIError("delete achievement configuration", err)
			}
			return shared.PrintOutput(map[string]string{"status": "deleted", "achievementId": *id}, "json", false)
		},
	}
}

func achievementsResetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games achievements reset", flag.ExitOnError)
	id := fs.String("achievement-id", "", "Achievement ID")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "reset",
		ShortUsage: "gplay games achievements reset --achievement-id <id> [flags]",
		ShortHelp:  "Reset the calling tester's progress for an achievement.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*id) == "" {
				return shared.UsageError("--achievement-id is required")
			}
			service, err := gamesclient.NewService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.Management.Achievements.Reset(*id).Context(ctx).Do()
			if err != nil {
				return shared.WrapGoogleAPIError("reset achievement progress", err)
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func achievementsResetAllCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games achievements reset-all", flag.ExitOnError)
	confirm := fs.Bool("confirm", false, "Confirm resetting all achievements for the calling tester")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "reset-all",
		ShortUsage: "gplay games achievements reset-all --confirm [flags]",
		ShortHelp:  "Reset the calling tester's progress for all achievements.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if !*confirm {
				return shared.UsageError("refusing to reset all achievements without --confirm")
			}
			service, err := gamesclient.NewService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.Management.Achievements.ResetAll().Context(ctx).Do()
			if err != nil {
				return shared.WrapGoogleAPIError("reset all achievement progress", err)
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func achievementsResetForAllCommand() *ffcli.Command {
	fs := flag.NewFlagSet("games achievements reset-for-all-players", flag.ExitOnError)
	id := fs.String("achievement-id", "", "Achievement ID")
	confirm := fs.Bool("confirm", false, "Confirm resetting this achievement for ALL testers")

	return &ffcli.Command{
		Name:       "reset-for-all-players",
		ShortUsage: "gplay games achievements reset-for-all-players --achievement-id <id> --confirm",
		ShortHelp:  "Reset an achievement for all testers of the application.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if strings.TrimSpace(*id) == "" {
				return shared.UsageError("--achievement-id is required")
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

			if err := service.Management.Achievements.ResetForAllPlayers(*id).Context(ctx).Do(); err != nil {
				return shared.WrapGoogleAPIError("reset achievement for all players", err)
			}
			return shared.PrintOutput(map[string]string{"status": "reset", "achievementId": *id}, "json", false)
		},
	}
}
