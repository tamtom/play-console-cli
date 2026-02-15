package apps

import (
	"context"
	"flag"
	"fmt"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

// ListCommand returns the apps list subcommand.
func ListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("apps list", flag.ExitOnError)
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay apps list [flags]",
		ShortHelp:  "List all apps accessible by the service account.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}

			service, err := playclient.NewService(ctx)
			if err != nil {
				return fmt.Errorf("creating service: %w", err)
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			// Use the androidpublisher API to list apps
			// The API doesn't have a direct "list apps" endpoint,
			// but we can validate access by checking the service
			_ = ctx

			return fmt.Errorf("apps list requires the Play Developer Reporting API which is not yet configured. Use gplay tracks list --package <name> to verify access to a specific app")
		},
	}
}
