package baseplans

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

func BasePlansCommand() *ffcli.Command {
	fs := flag.NewFlagSet("baseplans", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "baseplans",
		ShortUsage: "gplay baseplans <subcommand> [flags]",
		ShortHelp:  "Manage subscription base plans.",
		LongHelp: `Manage base plans within subscription products.

Base plans define the pricing tiers for a subscription, including:
  - Billing period (monthly, yearly, etc.)
  - Grace period
  - Renewal behavior
  - Regional pricing

Note: Base plans are created as part of the subscription.
Use these commands to activate, deactivate, or delete existing base plans.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			ActivateCommand(),
			DeactivateCommand(),
			DeleteCommand(),
			MigratePricesCommand(),
			BatchUpdateStatesCommand(),
			BatchMigratePricesCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			return flag.ErrHelp
		},
	}
}

func ActivateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("baseplans activate", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Subscription product ID")
	basePlanID := fs.String("base-plan-id", "", "Base plan ID")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "activate",
		ShortUsage: "gplay baseplans activate --package <name> --product-id <id> --base-plan-id <plan>",
		ShortHelp:  "Activate a base plan.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
			}
			if strings.TrimSpace(*basePlanID) == "" {
				return fmt.Errorf("--base-plan-id is required")
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

			req := &androidpublisher.ActivateBasePlanRequest{}
			resp, err := service.API.Monetization.Subscriptions.BasePlans.Activate(pkg, *productID, *basePlanID, req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func DeactivateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("baseplans deactivate", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Subscription product ID")
	basePlanID := fs.String("base-plan-id", "", "Base plan ID")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "deactivate",
		ShortUsage: "gplay baseplans deactivate --package <name> --product-id <id> --base-plan-id <plan>",
		ShortHelp:  "Deactivate a base plan (stops new subscriptions).",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
			}
			if strings.TrimSpace(*basePlanID) == "" {
				return fmt.Errorf("--base-plan-id is required")
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

			req := &androidpublisher.DeactivateBasePlanRequest{}
			resp, err := service.API.Monetization.Subscriptions.BasePlans.Deactivate(pkg, *productID, *basePlanID, req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func DeleteCommand() *ffcli.Command {
	fs := flag.NewFlagSet("baseplans delete", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Subscription product ID")
	basePlanID := fs.String("base-plan-id", "", "Base plan ID")
	confirm := fs.Bool("confirm", false, "Confirm deletion")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "delete",
		ShortUsage: "gplay baseplans delete --package <name> --product-id <id> --base-plan-id <plan> --confirm",
		ShortHelp:  "Delete a base plan (only if never had subscribers).",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
			}
			if strings.TrimSpace(*basePlanID) == "" {
				return fmt.Errorf("--base-plan-id is required")
			}
			if !*confirm {
				return fmt.Errorf("--confirm is required")
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

			err = service.API.Monetization.Subscriptions.BasePlans.Delete(pkg, *productID, *basePlanID).Context(ctx).Do()
			if err != nil {
				return err
			}

			result := map[string]interface{}{
				"deleted":    true,
				"productId":  *productID,
				"basePlanId": *basePlanID,
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}

func MigratePricesCommand() *ffcli.Command {
	fs := flag.NewFlagSet("baseplans migrate-prices", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Subscription product ID")
	basePlanID := fs.String("base-plan-id", "", "Base plan ID")
	jsonFlag := fs.String("json", "", "Migration request JSON (or @file)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "migrate-prices",
		ShortUsage: "gplay baseplans migrate-prices --package <name> --product-id <id> --base-plan-id <plan> --json <json>",
		ShortHelp:  "Migrate subscriber prices to current base plan prices.",
		LongHelp: `Migrate prices for existing subscribers.

JSON format:
{
  "regionalPriceMigrations": [
    {
      "regionCode": "US",
      "oldestAllowedPriceVersionTime": "2024-01-01T00:00:00Z",
      "priceIncreaseType": "OPT_IN_PRICE_INCREASE"
    }
  ],
  "regionsVersion": {
    "version": "2024001"
  }
}

priceIncreaseType values:
  - OPT_IN_PRICE_INCREASE: User must accept
  - OPT_OUT_PRICE_INCREASE: Auto-applied unless user cancels`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
			}
			if strings.TrimSpace(*basePlanID) == "" {
				return fmt.Errorf("--base-plan-id is required")
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

			var req androidpublisher.MigrateBasePlanPricesRequest
			if err := shared.LoadJSONArg(*jsonFlag, &req); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.API.Monetization.Subscriptions.BasePlans.MigratePrices(pkg, *productID, *basePlanID, &req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func BatchUpdateStatesCommand() *ffcli.Command {
	fs := flag.NewFlagSet("baseplans batch-update-states", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Subscription product ID")
	jsonFlag := fs.String("json", "", "Batch update states request JSON (or @file)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "batch-update-states",
		ShortUsage: "gplay baseplans batch-update-states --package <name> --product-id <id> --json <json>",
		ShortHelp:  "Batch activate/deactivate multiple base plans.",
		LongHelp: `Batch update states for multiple base plans.

JSON format:
{
  "requests": [
    {
      "activateBasePlanRequest": {
        "basePlanId": "monthly"
      }
    },
    {
      "deactivateBasePlanRequest": {
        "basePlanId": "yearly"
      }
    }
  ]
}`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
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

			var req androidpublisher.BatchUpdateBasePlanStatesRequest
			if err := shared.LoadJSONArg(*jsonFlag, &req); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.API.Monetization.Subscriptions.BasePlans.BatchUpdateStates(pkg, *productID, &req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func BatchMigratePricesCommand() *ffcli.Command {
	fs := flag.NewFlagSet("baseplans batch-migrate-prices", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Subscription product ID")
	jsonFlag := fs.String("json", "", "Batch migrate prices request JSON (or @file)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "batch-migrate-prices",
		ShortUsage: "gplay baseplans batch-migrate-prices --package <name> --product-id <id> --json <json>",
		ShortHelp:  "Batch migrate prices for multiple base plans.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
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

			var req androidpublisher.BatchMigrateBasePlanPricesRequest
			if err := shared.LoadJSONArg(*jsonFlag, &req); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.API.Monetization.Subscriptions.BasePlans.BatchMigratePrices(pkg, *productID, &req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}
