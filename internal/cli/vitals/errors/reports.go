package errors

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/playdeveloperreporting/v1beta1"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/reportingclient"
)

// ReportsCommand returns the `gplay vitals errors reports` subcommand.
func ReportsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("vitals errors reports", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	filter := fs.String("filter", "", "AIP-160 filter expression (e.g. 'errorIssueType = CRASH')")
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

Examples:
  gplay vitals errors reports --package com.example.app
  gplay vitals errors reports --package com.example.app --filter "errorIssueType = CRASH"
  gplay vitals errors reports --package com.example.app --filter "errorIssueId = 12345" --page-size 10`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			service, err := reportingclient.NewService(ctx)
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
				resp, err := call.Do()
				if err != nil {
					return shared.WrapGoogleAPIError("search error reports", err)
				}
				return shared.PrintOutput(resp, *outputFlag, *pretty)
			}

			var all []*playdeveloperreporting.GooglePlayDeveloperReportingV1beta1ErrorReport
			call := service.API.Vitals.Errors.Reports.Search(parent).
				Context(ctx).
				PageSize(*pageSize)
			if strings.TrimSpace(*filter) != "" {
				call = call.Filter(*filter)
			}
			err = call.Pages(ctx, func(resp *playdeveloperreporting.GooglePlayDeveloperReportingV1beta1SearchErrorReportsResponse) error {
				all = append(all, resp.ErrorReports...)
				return nil
			})
			if err != nil {
				return shared.WrapGoogleAPIError("search error reports (paginate)", err)
			}
			return shared.PrintOutput(all, *outputFlag, *pretty)
		},
	}
}
