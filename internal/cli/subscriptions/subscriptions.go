package subscriptions

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

func SubscriptionsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("subscriptions", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "subscriptions",
		ShortUsage: "gplay subscriptions <subcommand> [flags]",
		ShortHelp:  "Manage subscription products.",
		LongHelp: `Manage subscription products.

Subscriptions have a hierarchical structure:
  - Subscription: The product itself
  - Base Plan: A pricing tier within a subscription
  - Offer: Promotional pricing on a base plan (trials, intro prices)

Use the "baseplans" and "offers" commands to manage those resources.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			ListCommand(),
			GetCommand(),
			CreateCommand(),
			UpdateCommand(),
			DeleteCommand(),
			ArchiveCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			return flag.ErrHelp
		},
	}
}

func ListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("subscriptions list", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	pageSize := fs.Int("page-size", 100, "Page size")
	showArchived := fs.Bool("show-archived", false, "Include archived subscriptions")
	paginate := fs.Bool("paginate", false, "Fetch all pages")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay subscriptions list --package <name> [--page-size <n>] [--show-archived]",
		ShortHelp:  "List all subscriptions.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
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

			var all []*androidpublisher.Subscription
			pageToken := ""
			for {
				call := service.API.Monetization.Subscriptions.List(pkg).Context(ctx).PageSize(int64(*pageSize))
				if pageToken != "" {
					call.PageToken(pageToken)
				}
				if *showArchived {
					call.ShowArchived(true)
				}
				resp, err := call.Do()
				if err != nil {
					return err
				}
				if !*paginate {
					return shared.PrintOutput(resp, *outputFlag, *pretty)
				}
				all = append(all, resp.Subscriptions...)
				if resp.NextPageToken == "" {
					break
				}
				pageToken = resp.NextPageToken
			}

			return shared.PrintOutput(all, *outputFlag, *pretty)
		},
	}
}

func GetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("subscriptions get", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Subscription product ID")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "gplay subscriptions get --package <name> --product-id <id>",
		ShortHelp:  "Get a subscription.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
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

			resp, err := service.API.Monetization.Subscriptions.Get(pkg, *productID).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func CreateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("subscriptions create", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Subscription product ID")
	jsonFlag := fs.String("json", "", "Subscription JSON (or @file)")
	regionsVersion := fs.String("regions-version", "", "Regions version for price migration")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "create",
		ShortUsage: "gplay subscriptions create --package <name> --product-id <id> --json <json>",
		ShortHelp:  "Create a subscription.",
		LongHelp: `Create a new subscription product.

JSON format:
{
  "productId": "premium_monthly",
  "listings": {
    "en-US": {
      "title": "Premium Monthly",
      "benefits": ["Feature 1", "Feature 2"],
      "description": "Get premium access"
    }
  },
  "basePlans": [
    {
      "basePlanId": "monthly",
      "state": "ACTIVE",
      "autoRenewingBasePlanType": {
        "billingPeriodDuration": "P1M",
        "gracePeriodDuration": "P7D",
        "resubscribeState": "RESUBSCRIBE_STATE_ACTIVE",
        "prorationMode": "CHARGE_ON_NEXT_BILLING_DATE"
      },
      "regionalConfigs": [
        {
          "regionCode": "US",
          "price": {
            "currencyCode": "USD",
            "units": "9",
            "nanos": 990000000
          }
        }
      ]
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

			var subscription androidpublisher.Subscription
			if err := shared.LoadJSONArg(*jsonFlag, &subscription); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			subscription.PackageName = pkg
			subscription.ProductId = *productID

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			call := service.API.Monetization.Subscriptions.Create(pkg, &subscription).Context(ctx).ProductId(*productID)
			if strings.TrimSpace(*regionsVersion) != "" {
				call.RegionsVersionVersion(*regionsVersion)
			}
			resp, err := call.Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func UpdateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("subscriptions update", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Subscription product ID")
	jsonFlag := fs.String("json", "", "Subscription JSON (or @file)")
	updateMask := fs.String("update-mask", "", "Fields to update (comma-separated, e.g., listings)")
	regionsVersion := fs.String("regions-version", "", "Regions version for price migration")
	allowMissing := fs.Bool("allow-missing", false, "Create if not exists")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "update",
		ShortUsage: "gplay subscriptions update --package <name> --product-id <id> --json <json>",
		ShortHelp:  "Update a subscription.",
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

			var subscription androidpublisher.Subscription
			if err := shared.LoadJSONArg(*jsonFlag, &subscription); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			subscription.PackageName = pkg
			subscription.ProductId = *productID

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			call := service.API.Monetization.Subscriptions.Patch(pkg, *productID, &subscription).Context(ctx)
			if strings.TrimSpace(*updateMask) != "" {
				call.UpdateMask(*updateMask)
			}
			if strings.TrimSpace(*regionsVersion) != "" {
				call.RegionsVersionVersion(*regionsVersion)
			}
			if *allowMissing {
				call.AllowMissing(true)
			}
			resp, err := call.Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func DeleteCommand() *ffcli.Command {
	fs := flag.NewFlagSet("subscriptions delete", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Subscription product ID")
	confirm := fs.Bool("confirm", false, "Confirm deletion")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "delete",
		ShortUsage: "gplay subscriptions delete --package <name> --product-id <id> --confirm",
		ShortHelp:  "Delete a subscription (only if never had subscribers).",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
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

			err = service.API.Monetization.Subscriptions.Delete(pkg, *productID).Context(ctx).Do()
			if err != nil {
				return err
			}

			result := map[string]interface{}{
				"deleted":   true,
				"productId": *productID,
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}

func ArchiveCommand() *ffcli.Command {
	fs := flag.NewFlagSet("subscriptions archive", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Subscription product ID")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "archive",
		ShortUsage: "gplay subscriptions archive --package <name> --product-id <id>",
		ShortHelp:  "Archive a subscription (deprecate without deleting).",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
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

			req := &androidpublisher.ArchiveSubscriptionRequest{}
			resp, err := service.API.Monetization.Subscriptions.Archive(pkg, *productID, req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}
