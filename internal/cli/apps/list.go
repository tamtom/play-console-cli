package apps

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/playdeveloperreporting/v1beta1"

	cliruntime "github.com/tamtom/play-console-cli/internal/cli/runtime"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/reportingclient"
)

var newReportingService = reportingclient.NewService

// ListCommand returns the apps list subcommand.
func ListCommand(rt *cliruntime.Runtime) *ffcli.Command {
	fs := flag.NewFlagSet("apps list", flag.ExitOnError)
	pageSize := fs.Int("page-size", 50, "Page size (1-1000)")
	paginate := fs.Bool("paginate", false, "Fetch all pages")
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
			if *pageSize < 1 || *pageSize > 1000 {
				return fmt.Errorf("--page-size must be between 1 and 1000")
			}

			service, err := newReportingService(ctx)
			if err != nil {
				return fmt.Errorf("creating service: %w", err)
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			if !*paginate {
				resp, err := service.API.Apps.Search().Context(ctx).PageSize(int64(*pageSize)).Do()
				if err != nil {
					return shared.WrapGoogleAPIError("list accessible apps", err)
				}
				return shared.PrintOutput(resp, *outputFlag, *pretty)
			}

			var apps []*playdeveloperreporting.GooglePlayDeveloperReportingV1beta1App
			pageToken := ""
			for {
				call := service.API.Apps.Search().Context(ctx).PageSize(int64(*pageSize))
				if strings.TrimSpace(pageToken) != "" {
					call.PageToken(pageToken)
				}
				resp, err := call.Do()
				if err != nil {
					return shared.WrapGoogleAPIError("list accessible apps", err)
				}
				apps = append(apps, resp.Apps...)
				if resp.NextPageToken == "" {
					break
				}
				pageToken = resp.NextPageToken
			}
			return shared.PrintOutput(apps, *outputFlag, *pretty)
		},
	}
}
