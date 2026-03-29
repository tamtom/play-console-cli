package purchaseoptions

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

func PurchaseOptionsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("purchase-options", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "purchase-options",
		ShortUsage: "gplay purchase-options <subcommand> [flags]",
		ShortHelp:  "Manage one-time product purchase options.",
		LongHelp: `Manage purchase options for one-time products.

Purchase options define how a one-time product can be purchased,
including pricing, availability, and regional configurations.

Use "otp-offers" to manage offers within purchase options.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			BatchUpdateStatesCommand(),
			BatchDeleteCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			return flag.ErrHelp
		},
	}
}

func BatchUpdateStatesCommand() *ffcli.Command {
	fs := flag.NewFlagSet("purchase-options batch-update-states", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "One-time product ID")
	jsonFlag := fs.String("json", "", "BatchUpdatePurchaseOptionStatesRequest JSON (or @file)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "batch-update-states",
		ShortUsage: "gplay purchase-options batch-update-states --package <name> --product-id <id> --json <json>",
		ShortHelp:  "Batch activate/deactivate purchase options.",
		LongHelp: `Batch update states for multiple purchase options.

JSON format:
{
  "requests": [
    {
      "activatePurchaseOptionRequest": {
        "packageName": "com.example.app",
        "productId": "premium_item",
        "purchaseOptionId": "default"
      }
    },
    {
      "deactivatePurchaseOptionRequest": {
        "packageName": "com.example.app",
        "productId": "premium_item",
        "purchaseOptionId": "legacy_option"
      }
    }
  ]
}

Each request must contain exactly one of:
  - activatePurchaseOptionRequest
  - deactivatePurchaseOptionRequest`,
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

			var req androidpublisher.BatchUpdatePurchaseOptionStatesRequest
			if err := shared.LoadJSONArg(*jsonFlag, &req); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.API.Monetization.Onetimeproducts.PurchaseOptions.BatchUpdateStates(pkg, *productID, &req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func BatchDeleteCommand() *ffcli.Command {
	fs := flag.NewFlagSet("purchase-options batch-delete", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "One-time product ID")
	jsonFlag := fs.String("json", "", "BatchDeletePurchaseOptionsRequest JSON (or @file)")
	confirm := fs.Bool("confirm", false, "Confirm deletion")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "batch-delete",
		ShortUsage: "gplay purchase-options batch-delete --package <name> --product-id <id> --json <json> --confirm",
		ShortHelp:  "Batch delete purchase options.",
		LongHelp: `Batch delete multiple purchase options in a single request.

JSON format:
{
  "requests": [
    {
      "packageName": "com.example.app",
      "productId": "premium_item",
      "purchaseOptionId": "old_option",
      "force": true
    },
    {
      "packageName": "com.example.app",
      "productId": "basic_item",
      "purchaseOptionId": "legacy_option"
    }
  ]
}

Up to 100 requests per batch. Set "force" to true to also delete
any offers associated with the purchase option. Requires --confirm.`,
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

			var req androidpublisher.BatchDeletePurchaseOptionsRequest
			if err := shared.LoadJSONArg(*jsonFlag, &req); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			err = service.API.Monetization.Onetimeproducts.PurchaseOptions.BatchDelete(pkg, *productID, &req).Context(ctx).Do()
			if err != nil {
				return err
			}

			result := map[string]interface{}{
				"deleted": true,
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}
