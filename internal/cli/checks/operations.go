package checks

import (
	"context"
	"flag"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
	checksapi "google.golang.org/api/checks/v1alpha"

	"github.com/tamtom/play-console-cli/internal/checksclient"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

func OperationsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("checks operations", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "operations",
		ShortUsage: "gplay checks operations <subcommand> [flags]",
		ShortHelp:  "Manage Checks app analysis operations.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			OperationsGetCommand(),
			OperationsWaitCommand(),
			OperationsCancelCommand(),
			OperationsListCommand(),
			OperationsDeleteCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

func OperationsGetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("checks operations get", flag.ExitOnError)
	account := fs.String("account", "", "Checks account ID (overrides GPLAY_CHECKS_ACCOUNT/checks_account)")
	app := fs.String("app", "", "Checks app ID or resource name")
	operation := fs.String("operation", "", "Operation ID or resource name")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return operationCommand("get", "gplay checks operations get --account <account-id> --app <app-id> --operation <operation-id> [flags]", "Get a Checks operation.", fs, outputFlag, pretty, account, app, operation, func(ctx context.Context, service *checksclient.Service, name string) (any, error) {
		return service.API.Accounts.Apps.Operations.Get(name).Context(ctx).Do()
	})
}

func OperationsWaitCommand() *ffcli.Command {
	fs := flag.NewFlagSet("checks operations wait", flag.ExitOnError)
	account := fs.String("account", "", "Checks account ID (overrides GPLAY_CHECKS_ACCOUNT/checks_account)")
	app := fs.String("app", "", "Checks app ID or resource name")
	operation := fs.String("operation", "", "Operation ID or resource name")
	timeout := fs.Duration("timeout", 10*time.Minute, "Maximum time to wait")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return operationCommand("wait", "gplay checks operations wait --account <account-id> --app <app-id> --operation <operation-id> [flags]", "Wait for a Checks operation.", fs, outputFlag, pretty, account, app, operation, func(ctx context.Context, service *checksclient.Service, name string) (any, error) {
		return service.API.Accounts.Apps.Operations.Wait(name, &checksapi.WaitOperationRequest{
			Timeout: formatWaitTimeout(*timeout),
		}).Context(ctx).Do()
	})
}

func OperationsCancelCommand() *ffcli.Command {
	fs := flag.NewFlagSet("checks operations cancel", flag.ExitOnError)
	account := fs.String("account", "", "Checks account ID (overrides GPLAY_CHECKS_ACCOUNT/checks_account)")
	app := fs.String("app", "", "Checks app ID or resource name")
	operation := fs.String("operation", "", "Operation ID or resource name")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return operationCommand("cancel", "gplay checks operations cancel --account <account-id> --app <app-id> --operation <operation-id> [flags]", "Cancel a Checks operation.", fs, outputFlag, pretty, account, app, operation, func(ctx context.Context, service *checksclient.Service, name string) (any, error) {
		resp, err := service.API.Accounts.Apps.Operations.Cancel(name, &checksapi.CancelOperationRequest{}).Context(ctx).Do()
		if err != nil {
			return nil, err
		}
		return resp, nil
	})
}

func OperationsDeleteCommand() *ffcli.Command {
	fs := flag.NewFlagSet("checks operations delete", flag.ExitOnError)
	account := fs.String("account", "", "Checks account ID (overrides GPLAY_CHECKS_ACCOUNT/checks_account)")
	app := fs.String("app", "", "Checks app ID or resource name")
	operation := fs.String("operation", "", "Operation ID or resource name")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return operationCommand("delete", "gplay checks operations delete --account <account-id> --app <app-id> --operation <operation-id> [flags]", "Delete a Checks operation.", fs, outputFlag, pretty, account, app, operation, func(ctx context.Context, service *checksclient.Service, name string) (any, error) {
		resp, err := service.API.Accounts.Apps.Operations.Delete(name).Context(ctx).Do()
		if err != nil {
			return nil, err
		}
		return resp, nil
	})
}

func OperationsListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("checks operations list", flag.ExitOnError)
	account := fs.String("account", "", "Checks account ID (overrides GPLAY_CHECKS_ACCOUNT/checks_account)")
	app := fs.String("app", "", "Checks app ID or resource name")
	filter := fs.String("filter", "", "AIP-160 filter for operations")
	pageSize := fs.Int("page-size", 10, "Results per page (1-50)")
	paginate := fs.Bool("paginate", false, "Fetch all pages")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay checks operations list --account <account-id> --app <app-id> [flags]",
		ShortHelp:  "List Checks operations for an app.",
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

			call := service.API.Accounts.Apps.Operations.List(appResource(resolvedAccount, *app)).Context(ctx).PageSize(int64(*pageSize))
			if strings.TrimSpace(*filter) != "" {
				call.Filter(*filter)
			}
			if !*paginate {
				resp, err := call.Do()
				if err != nil {
					return shared.WrapGoogleAPIError("list Checks operations", err)
				}
				return shared.PrintOutputContext(ctx, resp, *outputFlag, *pretty)
			}
			var all []*checksapi.Operation
			err = call.Pages(ctx, func(resp *checksapi.ListOperationsResponse) error {
				all = append(all, resp.Operations...)
				return nil
			})
			if err != nil {
				return shared.WrapGoogleAPIError("list Checks operations", err)
			}
			return shared.PrintOutputContext(ctx, all, *outputFlag, *pretty)
		},
	}
}

func operationCommand(name, usage, help string, fs *flag.FlagSet, outputFlag *string, pretty *bool, account *string, app *string, operation *string, run func(context.Context, *checksclient.Service, string) (any, error)) *ffcli.Command {
	return &ffcli.Command{
		Name:       name,
		ShortUsage: usage,
		ShortHelp:  help,
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*app) == "" {
				return shared.UsageError("--app is required")
			}
			if strings.TrimSpace(*operation) == "" {
				return shared.UsageError("--operation is required")
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

			resp, err := run(ctx, service, operationResource(resolvedAccount, *app, *operation))
			if err != nil {
				return shared.WrapGoogleAPIError(name+" Checks operation", err)
			}
			return shared.PrintOutputContext(ctx, resp, *outputFlag, *pretty)
		},
	}
}
