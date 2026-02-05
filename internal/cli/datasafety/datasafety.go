package datasafety

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

func DataSafetyCommand() *ffcli.Command {
	fs := flag.NewFlagSet("data-safety", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "data-safety",
		ShortUsage: "gplay data-safety <subcommand> [flags]",
		ShortHelp:  "Manage data safety declarations.",
		LongHelp: `Manage data safety declarations for your app.

Data safety declarations inform users about how your app
collects and shares data. These appear on your app's
Play Store listing.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			UpdateCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			return flag.ErrHelp
		},
	}
}

func UpdateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("data-safety update", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	jsonFlag := fs.String("json", "", "SafetyLabelsUpdateRequest JSON (or @file)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "update",
		ShortUsage: "gplay data-safety update --package <name> --json <json>",
		ShortHelp:  "Update data safety declarations.",
		LongHelp: `Update data safety declarations for your app.

JSON format:
{
  "safetyLabels": "... base64 encoded SafetyLabels proto ..."
}

Note: The safety labels are encoded in Protocol Buffer format.
Typically, you would use the Play Console UI or generate this
from your data safety form responses.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*jsonFlag) == "" {
				return fmt.Errorf("--json is required")
			}
			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			var req androidpublisher.SafetyLabelsUpdateRequest
			if err := shared.LoadJSONArg(*jsonFlag, &req); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.API.Applications.DataSafety(pkg, &req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}
