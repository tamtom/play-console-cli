package testers

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

func TestersCommand() *ffcli.Command {
	fs := flag.NewFlagSet("testers", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "testers",
		ShortUsage: "gplay testers <subcommand> [flags]",
		ShortHelp:  "Manage testers for closed testing tracks.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
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

func GetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("testers get", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	track := fs.String("track", "", "Track name (e.g., internal, alpha, beta, or custom track name)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "gplay testers get --package <name> --edit <id> --track <track>",
		ShortHelp:  "Get testers for a track.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*editID) == "" {
				return fmt.Errorf("--edit is required")
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
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()
			resp, err := service.API.Edits.Testers.Get(pkg, *editID, *track).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func UpdateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("testers update", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	track := fs.String("track", "", "Track name")
	emails := fs.String("emails", "", "Comma-separated list of tester email addresses")
	googleGroups := fs.String("google-groups", "", "Comma-separated list of Google Group email addresses")
	jsonFlag := fs.String("json", "", "Full Testers JSON (or @file) - overrides other flags")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "update",
		ShortUsage: "gplay testers update --package <name> --edit <id> --track <track> [--emails <list>] [--google-groups <list>] [--json <json>]",
		ShortHelp:  "Update testers for a track (replaces entire resource).",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			return updateTesters(ctx, *packageName, *editID, *track, *emails, *googleGroups, *jsonFlag, *outputFlag, *pretty, false)
		},
	}
}

func PatchCommand() *ffcli.Command {
	fs := flag.NewFlagSet("testers patch", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	track := fs.String("track", "", "Track name")
	emails := fs.String("emails", "", "Comma-separated list of tester email addresses")
	googleGroups := fs.String("google-groups", "", "Comma-separated list of Google Group email addresses")
	jsonFlag := fs.String("json", "", "Partial Testers JSON (or @file) - overrides other flags")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "patch",
		ShortUsage: "gplay testers patch --package <name> --edit <id> --track <track> [--emails <list>] [--google-groups <list>] [--json <json>]",
		ShortHelp:  "Patch testers for a track (partial update).",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			return updateTesters(ctx, *packageName, *editID, *track, *emails, *googleGroups, *jsonFlag, *outputFlag, *pretty, true)
		},
	}
}

func updateTesters(ctx context.Context, packageName, editID, track, emails, googleGroups, jsonFlag, outputFlag string, pretty, patch bool) error {
	if err := shared.ValidateOutputFlags(outputFlag, pretty); err != nil {
		return err
	}
	if strings.TrimSpace(editID) == "" {
		return fmt.Errorf("--edit is required")
	}
	if strings.TrimSpace(track) == "" {
		return fmt.Errorf("--track is required")
	}

	service, err := playclient.NewService(ctx)
	if err != nil {
		return err
	}
	pkg := shared.ResolvePackageName(packageName, service.Cfg)
	if strings.TrimSpace(pkg) == "" {
		return fmt.Errorf("--package is required")
	}

	var testers androidpublisher.Testers

	if strings.TrimSpace(jsonFlag) != "" {
		if err := shared.LoadJSONArg(jsonFlag, &testers); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
	} else {
		// Build from individual flags
		if strings.TrimSpace(emails) != "" {
			emailList := strings.Split(emails, ",")
			for i := range emailList {
				emailList[i] = strings.TrimSpace(emailList[i])
			}
			testers.GoogleGroups = nil // Clear if using individual emails
			// Note: The API uses GoogleGroups for both individual testers and groups
			// For individual testers, we need to set them via the Testers resource
		}
		if strings.TrimSpace(googleGroups) != "" {
			groupList := strings.Split(googleGroups, ",")
			for i := range groupList {
				groupList[i] = strings.TrimSpace(groupList[i])
			}
			testers.GoogleGroups = groupList
		}
	}

	ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
	defer cancel()

	if patch {
		resp, err := service.API.Edits.Testers.Patch(pkg, editID, track, &testers).Context(ctx).Do()
		if err != nil {
			return err
		}
		return shared.PrintOutput(resp, outputFlag, pretty)
	}

	resp, err := service.API.Edits.Testers.Update(pkg, editID, track, &testers).Context(ctx).Do()
	if err != nil {
		return err
	}
	return shared.PrintOutput(resp, outputFlag, pretty)
}
