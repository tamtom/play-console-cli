package errors

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/playdeveloperreporting/v1beta1"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// ReportsCommand returns the `gplay vitals errors reports` subcommand.
func ReportsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("vitals errors reports", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	filter := fs.String("filter", "", "AIP-160 filter expression (e.g. 'errorIssueType = CRASH')")
	from := fs.String("from", "", "Start date, inclusive (UTC, YYYY-MM-DD)")
	to := fs.String("to", "", "End date, inclusive (UTC, YYYY-MM-DD)")
	pageSize := fs.Int64("page-size", 50, "Max results per page (1-100)")
	paginate := fs.Bool("paginate", false, "Fetch all pages")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "reports",
		ShortUsage: "gplay vitals errors reports --package <name> [flags]",
		ShortHelp:  "Search individual error reports.",
		LongHelp: `Search all error reports received for an app.

Returns individual error reports with stack traces and device info.

Supported --filter fields:
  apiLevel, versionCode, deviceModel, deviceBrand, deviceType,
  errorIssueType (CRASH, ANR, NON_FATAL), errorIssueId, errorReportId,
  appProcessState (FOREGROUND, BACKGROUND), isUserPerceived

Date range:
  --from and --to accept YYYY-MM-DD dates (both inclusive, interpreted as UTC).
  If neither is set, the API defaults to the last 24 hours.

Examples:
  gplay vitals errors reports --package com.example.app
  gplay vitals errors reports --package com.example.app --filter "errorIssueType = CRASH"
  gplay vitals errors reports --package com.example.app --filter "errorIssueId = 12345" --page-size 10
  gplay vitals errors reports --package com.example.app --from 2025-01-01 --to 2025-01-31`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			interval, err := buildSearchInterval(*from, *to)
			if err != nil {
				return err
			}
			service, err := newReportingService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			parent := fmt.Sprintf("apps/%s", pkg)

			if !*paginate {
				call := service.API.Vitals.Errors.Reports.Search(parent).
					Context(ctx).
					PageSize(*pageSize)
				if strings.TrimSpace(*filter) != "" {
					call = call.Filter(*filter)
				}
				call = applyReportsInterval(call, interval)
				resp, err := call.Do()
				if err != nil {
					return shared.WrapGoogleAPIError("search error reports", err)
				}
				return shared.PrintOutputContext(ctx, resp, *outputFlag, *pretty)
			}

			var all []*playdeveloperreporting.GooglePlayDeveloperReportingV1beta1ErrorReport
			call := service.API.Vitals.Errors.Reports.Search(parent).
				Context(ctx).
				PageSize(*pageSize)
			if strings.TrimSpace(*filter) != "" {
				call = call.Filter(*filter)
			}
			call = applyReportsInterval(call, interval)
			err = call.Pages(ctx, func(resp *playdeveloperreporting.GooglePlayDeveloperReportingV1beta1SearchErrorReportsResponse) error {
				all = append(all, resp.ErrorReports...)
				return nil
			})
			if err != nil {
				return shared.WrapGoogleAPIError("search error reports (paginate)", err)
			}
			return shared.PrintOutputContext(ctx, all, *outputFlag, *pretty)
		},
	}
}
