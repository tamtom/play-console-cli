package checks

import (
	"context"
	"flag"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	checksapi "google.golang.org/api/checks/v1alpha"

	"github.com/tamtom/play-console-cli/internal/checksclient"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

func ReportsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("checks reports", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "reports",
		ShortUsage: "gplay checks reports <subcommand> [flags]",
		ShortHelp:  "List and inspect Checks compliance reports.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			ReportsListCommand(),
			ReportsGetCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

func ReportsListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("checks reports list", flag.ExitOnError)
	account := fs.String("account", "", "Checks account ID (overrides GPLAY_CHECKS_ACCOUNT/checks_account)")
	app := fs.String("app", "", "Checks app ID or resource name")
	filter := fs.String("filter", "", "AIP-160 filter for reports")
	checksFilter := fs.String("checks-filter", "", "AIP-160 filter for checks within reports")
	pageSize := fs.Int("page-size", 10, "Results per page (1-50)")
	paginate := fs.Bool("paginate", false, "Fetch all pages")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay checks reports list --account <account-id> --app <app-id> [flags]",
		ShortHelp:  "List Checks reports for an app.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if err := validatePageSize(*pageSize); err != nil {
				return err
			}
			if strings.TrimSpace(*app) == "" {
				return shared.UsageError("--app is required")
			}
			resolvedAccount, err := resolveAccountForCommand(*account)
			if err != nil {
				return err
			}
			if strings.TrimSpace(resolvedAccount) == "" {
				return shared.UsageError("--account is required (or set GPLAY_CHECKS_ACCOUNT/checks_account)")
			}
			service, err := checksclient.NewService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			call := service.API.Accounts.Apps.Reports.List(appResource(resolvedAccount, *app)).Context(ctx).PageSize(int64(*pageSize))
			if strings.TrimSpace(*filter) != "" {
				call.Filter(*filter)
			}
			if strings.TrimSpace(*checksFilter) != "" {
				call.ChecksFilter(*checksFilter)
			}
			if !*paginate {
				resp, err := call.Do()
				if err != nil {
					return shared.WrapGoogleAPIError("list Checks reports", err)
				}
				return shared.PrintOutputContext(ctx, resp, *outputFlag, *pretty)
			}
			var all []*checksapi.GoogleChecksReportV1alphaReport
			err = call.Pages(ctx, func(resp *checksapi.GoogleChecksReportV1alphaListReportsResponse) error {
				all = append(all, resp.Reports...)
				return nil
			})
			if err != nil {
				return shared.WrapGoogleAPIError("list Checks reports", err)
			}
			return shared.PrintOutputContext(ctx, all, *outputFlag, *pretty)
		},
	}
}

func ReportsGetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("checks reports get", flag.ExitOnError)
	account := fs.String("account", "", "Checks account ID (overrides GPLAY_CHECKS_ACCOUNT/checks_account)")
	app := fs.String("app", "", "Checks app ID or resource name")
	report := fs.String("report", "", "Checks report ID or resource name")
	checksFilter := fs.String("checks-filter", "", "AIP-160 filter for checks within the report")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "gplay checks reports get --account <account-id> --app <app-id> --report <report-id> [flags]",
		ShortHelp:  "Get a Checks report.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*app) == "" {
				return shared.UsageError("--app is required")
			}
			if strings.TrimSpace(*report) == "" {
				return shared.UsageError("--report is required")
			}
			resolvedAccount, err := resolveAccountForCommand(*account)
			if err != nil {
				return err
			}
			if strings.TrimSpace(resolvedAccount) == "" {
				return shared.UsageError("--account is required (or set GPLAY_CHECKS_ACCOUNT/checks_account)")
			}
			service, err := checksclient.NewService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			call := service.API.Accounts.Apps.Reports.Get(reportResource(resolvedAccount, *app, *report)).Context(ctx)
			if strings.TrimSpace(*checksFilter) != "" {
				call.ChecksFilter(*checksFilter)
			}
			resp, err := call.Do()
			if err != nil {
				return shared.WrapGoogleAPIError("get Checks report", err)
			}
			return shared.PrintOutputContext(ctx, resp, *outputFlag, *pretty)
		},
	}
}
