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

func AppsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("checks apps", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "apps",
		ShortUsage: "gplay checks apps <subcommand> [flags]",
		ShortHelp:  "List and inspect apps connected to Checks.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			AppsListCommand(),
			AppsGetCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

func AppsListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("checks apps list", flag.ExitOnError)
	account := fs.String("account", "", "Checks account ID (overrides GPLAY_CHECKS_ACCOUNT/checks_account)")
	pageSize := fs.Int("page-size", 10, "Results per page (1-50)")
	paginate := fs.Bool("paginate", false, "Fetch all pages")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay checks apps list --account <account-id> [flags]",
		ShortHelp:  "List apps under a Checks account.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if err := validatePageSize(*pageSize); err != nil {
				return err
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

			call := service.API.Accounts.Apps.List(accountResource(resolvedAccount)).Context(ctx).PageSize(int64(*pageSize))
			if !*paginate {
				resp, err := call.Do()
				if err != nil {
					return shared.WrapGoogleAPIError("list Checks apps", err)
				}
				return shared.PrintOutputContext(ctx, resp, *outputFlag, *pretty)
			}
			var all []*checksapi.GoogleChecksAccountV1alphaApp
			err = call.Pages(ctx, func(resp *checksapi.GoogleChecksAccountV1alphaListAppsResponse) error {
				all = append(all, resp.Apps...)
				return nil
			})
			if err != nil {
				return shared.WrapGoogleAPIError("list Checks apps", err)
			}
			return shared.PrintOutputContext(ctx, all, *outputFlag, *pretty)
		},
	}
}

func AppsGetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("checks apps get", flag.ExitOnError)
	account := fs.String("account", "", "Checks account ID (overrides GPLAY_CHECKS_ACCOUNT/checks_account)")
	app := fs.String("app", "", "Checks app ID or resource name")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "gplay checks apps get --account <account-id> --app <app-id> [flags]",
		ShortHelp:  "Get a Checks app.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
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

			resp, err := service.API.Accounts.Apps.Get(appResource(resolvedAccount, *app)).Context(ctx).Do()
			if err != nil {
				return shared.WrapGoogleAPIError("get Checks app", err)
			}
			return shared.PrintOutputContext(ctx, resp, *outputFlag, *pretty)
		},
	}
}
