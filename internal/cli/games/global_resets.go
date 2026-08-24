package games

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	gamesmanagement "google.golang.org/api/gamesmanagement/v1management"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/gamesclient"
)

type globalResetKind string

const (
	globalAchievements globalResetKind = "achievements"
	globalEvents       globalResetKind = "events"
	globalScores       globalResetKind = "scores"
)

var newGlobalResetService = gamesclient.NewService

type globalResetFlags struct {
	applicationID        *string
	confirmApplicationID *string
	confirm              *bool
	output               *string
	pretty               *bool
}

func addGlobalResetFlags(fs *flag.FlagSet) globalResetFlags {
	return globalResetFlags{
		applicationID:        fs.String("application-id", "", "Numeric Play Games application ID (required explicitly for this global operation)"),
		confirmApplicationID: fs.String("confirm-application-id", "", "Repeat the exact Play Games application ID being targeted"),
		confirm:              fs.Bool("confirm", false, "Confirm resetting tester data for every player"),
		output:               fs.String("output", "json", "Output format: json (default), table, markdown"),
		pretty:               fs.Bool("pretty", false, "Pretty-print JSON output"),
	}
}

func (f globalResetFlags) validate() error {
	if err := shared.ValidateOutputFlags(*f.output, *f.pretty); err != nil {
		return err
	}
	appID := strings.TrimSpace(*f.applicationID)
	if appID == "" {
		return fmt.Errorf("--application-id is required explicitly for global resets")
	}
	if strings.TrimSpace(*f.confirmApplicationID) != appID {
		return fmt.Errorf("--confirm-application-id must exactly match --application-id")
	}
	if !*f.confirm {
		return fmt.Errorf("--confirm is required")
	}
	return nil
}

func globalResetAllCommand(kind globalResetKind) *ffcli.Command {
	name := "reset-all-for-all-players"
	fs := flag.NewFlagSet("games "+string(kind)+" "+name, flag.ExitOnError)
	f := addGlobalResetFlags(fs)
	return &ffcli.Command{
		Name: name, ShortUsage: "gplay games " + string(kind) + " " + name + " --application-id <id> --confirm-application-id <id> --confirm", ShortHelp: "Reset all " + string(kind) + " for every tester in the application.", FlagSet: fs, UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			if err := f.validate(); err != nil {
				return err
			}
			service, err := newGlobalResetService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()
			if err := verifyGlobalResetApplication(ctx, service, *f.applicationID); err != nil {
				return err
			}
			switch kind {
			case globalAchievements:
				err = service.Management.Achievements.ResetAllForAllPlayers().Context(ctx).Do()
			case globalEvents:
				err = service.Management.Events.ResetAllForAllPlayers().Context(ctx).Do()
			case globalScores:
				err = service.Management.Scores.ResetAllForAllPlayers().Context(ctx).Do()
			}
			if err != nil {
				return shared.WrapGoogleAPIError("reset all "+string(kind)+" for all players", err)
			}
			return shared.PrintOutputContext(ctx, map[string]any{"status": "reset-all-for-all-players", "resource": kind, "applicationId": *f.applicationID}, *f.output, *f.pretty)
		},
	}
}

func globalResetMultipleCommand(kind globalResetKind) *ffcli.Command {
	name := "reset-multiple-for-all-players"
	fs := flag.NewFlagSet("games "+string(kind)+" "+name, flag.ExitOnError)
	f := addGlobalResetFlags(fs)
	ids := fs.String("ids", "", "Comma-separated achievement, event, or leaderboard IDs")
	return &ffcli.Command{
		Name: name, ShortUsage: "gplay games " + string(kind) + " " + name + " --ids <id,id> --application-id <id> --confirm-application-id <id> --confirm", ShortHelp: "Reset selected " + string(kind) + " for every tester in the application.", FlagSet: fs, UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			if err := f.validate(); err != nil {
				return err
			}
			idList, err := parseResetIDs(*ids)
			if err != nil {
				return err
			}
			service, err := newGlobalResetService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()
			if err := verifyGlobalResetApplication(ctx, service, *f.applicationID); err != nil {
				return err
			}
			switch kind {
			case globalAchievements:
				err = service.Management.Achievements.ResetMultipleForAllPlayers(&gamesmanagement.AchievementResetMultipleForAllRequest{AchievementIds: idList}).Context(ctx).Do()
			case globalEvents:
				err = service.Management.Events.ResetMultipleForAllPlayers(&gamesmanagement.EventsResetMultipleForAllRequest{EventIds: idList}).Context(ctx).Do()
			case globalScores:
				err = service.Management.Scores.ResetMultipleForAllPlayers(&gamesmanagement.ScoresResetMultipleForAllRequest{LeaderboardIds: idList}).Context(ctx).Do()
			}
			if err != nil {
				return shared.WrapGoogleAPIError("reset multiple "+string(kind)+" for all players", err)
			}
			return shared.PrintOutputContext(ctx, map[string]any{"status": "reset-multiple-for-all-players", "resource": kind, "ids": idList, "applicationId": *f.applicationID}, *f.output, *f.pretty)
		},
	}
}

func verifyGlobalResetApplication(ctx context.Context, service *gamesclient.Service, applicationID string) error {
	_, err := service.Configuration.AchievementConfigurations.List(applicationID).MaxResults(1).Context(ctx).Do()
	if err != nil {
		return shared.WrapGoogleAPIError("verify Play Games application before global reset", err)
	}
	return nil
}

func parseResetIDs(value string) ([]string, error) {
	seen := map[string]bool{}
	var ids []string
	for _, raw := range strings.Split(value, ",") {
		id := strings.TrimSpace(raw)
		if id != "" && !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("--ids requires at least one ID")
	}
	return ids, nil
}

func achievementsResetAllForAllCommand() *ffcli.Command {
	return globalResetAllCommand(globalAchievements)
}

func achievementsResetMultipleForAllCommand() *ffcli.Command {
	return globalResetMultipleCommand(globalAchievements)
}
func eventsResetAllForAllCommand() *ffcli.Command { return globalResetAllCommand(globalEvents) }
func eventsResetMultipleForAllCommand() *ffcli.Command {
	return globalResetMultipleCommand(globalEvents)
}
func scoresResetAllForAllCommand() *ffcli.Command { return globalResetAllCommand(globalScores) }
func scoresResetMultipleForAllCommand() *ffcli.Command {
	return globalResetMultipleCommand(globalScores)
}
