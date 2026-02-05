package tracks

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

func TracksCommand() *ffcli.Command {
	fs := flag.NewFlagSet("tracks", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "tracks",
		ShortUsage: "gplay tracks <subcommand> [flags]",
		ShortHelp:  "Manage release tracks in an edit.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			ListCommand(),
			GetCommand(),
			UpdateCommand(),
			PatchCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			return flag.ErrHelp
		},
	}
}

func ListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("tracks list", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay tracks list --package <name> --edit <id>",
		ShortHelp:  "List tracks in an edit.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}
			if strings.TrimSpace(*editID) == "" {
				return fmt.Errorf("--edit is required")
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			call := service.API.Edits.Tracks.List(pkg, *editID).Context(ctx)
			resp, err := call.Do()
			if err != nil {
				return err
			}

			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func GetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("tracks get", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	track := fs.String("track", "", "Track name (production, beta, alpha, internal)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "gplay tracks get --package <name> --edit <id> --track <name>",
		ShortHelp:  "Get a track in an edit.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*track) == "" {
				return fmt.Errorf("--track is required")
			}
			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}
			if strings.TrimSpace(*editID) == "" {
				return fmt.Errorf("--edit is required")
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()
			resp, err := service.API.Edits.Tracks.Get(pkg, *editID, *track).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func UpdateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("tracks update", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	track := fs.String("track", "", "Track name")
	releasesJSON := fs.String("releases", "", "JSON array of track releases (or @file)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "update",
		ShortUsage: "gplay tracks update --package <name> --edit <id> --track <name> --releases <json>",
		ShortHelp:  "Update a track.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			return updateTrack(ctx, *packageName, *editID, *track, *releasesJSON, *outputFlag, *pretty, false)
		},
	}
}

func PatchCommand() *ffcli.Command {
	fs := flag.NewFlagSet("tracks patch", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	track := fs.String("track", "", "Track name")
	releasesJSON := fs.String("releases", "", "JSON array of track releases (or @file)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "patch",
		ShortUsage: "gplay tracks patch --package <name> --edit <id> --track <name> --releases <json>",
		ShortHelp:  "Patch a track.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			return updateTrack(ctx, *packageName, *editID, *track, *releasesJSON, *outputFlag, *pretty, true)
		},
	}
}

func updateTrack(ctx context.Context, packageName, editID, track, releasesJSON, outputFlag string, pretty, patch bool) error {
	if err := shared.ValidateOutputFlags(outputFlag, pretty); err != nil {
		return err
	}
	if strings.TrimSpace(track) == "" {
		return fmt.Errorf("--track is required")
	}
	if strings.TrimSpace(releasesJSON) == "" {
		return fmt.Errorf("--releases is required")
	}

	service, err := playclient.NewService(ctx)
	if err != nil {
		return err
	}
	pkg := shared.ResolvePackageName(packageName, service.Cfg)
	if strings.TrimSpace(pkg) == "" {
		return fmt.Errorf("--package is required")
	}
	if strings.TrimSpace(editID) == "" {
		return fmt.Errorf("--edit is required")
	}

	var releases []*androidpublisher.TrackRelease
	if err := shared.LoadJSONArg(releasesJSON, &releases); err != nil {
		return fmt.Errorf("invalid releases JSON: %w", err)
	}

	trackObj := &androidpublisher.Track{Track: track, Releases: releases}

	ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
	defer cancel()

	if patch {
		resp, err := service.API.Edits.Tracks.Patch(pkg, editID, track, trackObj).Context(ctx).Do()
		if err != nil {
			return err
		}
		return shared.PrintOutput(resp, outputFlag, pretty)
	}

	resp, err := service.API.Edits.Tracks.Update(pkg, editID, track, trackObj).Context(ctx).Do()
	if err != nil {
		return err
	}
	return shared.PrintOutput(resp, outputFlag, pretty)
}
