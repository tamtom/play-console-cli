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

// IssuesCommand returns the `gplay vitals errors issues` subcommand.
func IssuesCommand() *ffcli.Command {
	fs := flag.NewFlagSet("vitals errors issues", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	filter := fs.String("filter", "", "AIP-160 filter expression (e.g. 'errorIssueType = CRASH')")
	orderBy := fs.String("order-by", "", "Order results (e.g. 'errorReportCount desc')")
	from := fs.String("from", "", "Start date, inclusive (UTC, YYYY-MM-DD)")
	to := fs.String("to", "", "End date, inclusive (UTC, YYYY-MM-DD)")
	pageSize := fs.Int64("page-size", 50, "Max results per page (1-1000)")
	paginate := fs.Bool("paginate", false, "Fetch all pages")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "issues",
		ShortUsage: "gplay vitals errors issues --package <name> [flags]",
		ShortHelp:  "Search grouped error issues.",
		LongHelp: `Search all error issues in which reports have been grouped.

Supported --filter fields:
  apiLevel, versionCode, deviceModel, deviceBrand, deviceType,
  errorIssueType (CRASH, ANR, NON_FATAL), appProcessState (FOREGROUND, BACKGROUND),
  isUserPerceived

Supported --order-by fields:
  errorReportCount desc, errorReportCount asc,
  distinctUsers desc, distinctUsers asc

Date range:
  --from and --to accept YYYY-MM-DD dates (both inclusive, interpreted as UTC).
  If neither is set, the API defaults to the last 24 hours.

Examples:
  gplay vitals errors issues --package com.example.app
  gplay vitals errors issues --package com.example.app --filter "errorIssueType = CRASH"
  gplay vitals errors issues --package com.example.app --order-by "errorReportCount desc" --page-size 10
  gplay vitals errors issues --package com.example.app --from 2025-01-01 --to 2025-01-31`,
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
				call := service.API.Vitals.Errors.Issues.Search(parent).
					Context(ctx).
					PageSize(*pageSize)
				if strings.TrimSpace(*filter) != "" {
					call = call.Filter(*filter)
				}
				if strings.TrimSpace(*orderBy) != "" {
					call = call.OrderBy(*orderBy)
				}
				call = applyIssuesInterval(call, interval)
				resp, err := call.Do()
				if err != nil {
					return shared.WrapGoogleAPIError("search error issues", err)
				}
				return shared.PrintOutputContext(ctx, resp, *outputFlag, *pretty)
			}

			var all []*playdeveloperreporting.GooglePlayDeveloperReportingV1beta1ErrorIssue
			call := service.API.Vitals.Errors.Issues.Search(parent).
				Context(ctx).
				PageSize(*pageSize)
			if strings.TrimSpace(*filter) != "" {
				call = call.Filter(*filter)
			}
			if strings.TrimSpace(*orderBy) != "" {
				call = call.OrderBy(*orderBy)
			}
			call = applyIssuesInterval(call, interval)
			err = call.Pages(ctx, func(resp *playdeveloperreporting.GooglePlayDeveloperReportingV1beta1SearchErrorIssuesResponse) error {
				all = append(all, resp.ErrorIssues...)
				return nil
			})
			if err != nil {
				return shared.WrapGoogleAPIError("search error issues (paginate)", err)
			}
			return shared.PrintOutputContext(ctx, all, *outputFlag, *pretty)
		},
	}
}
