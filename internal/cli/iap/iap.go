package iap

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

func IAPCommand() *ffcli.Command {
	fs := flag.NewFlagSet("iap", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "iap",
		ShortUsage: "gplay iap <subcommand> [flags]",
		ShortHelp:  "Manage in-app products (managed products).",
		LongHelp: `Manage in-app products including consumables and non-consumables.

Note: This command manages "managed products" (one-time purchases).
For subscriptions, use the "subscriptions" command instead.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			ListCommand(),
			GetCommand(),
			CreateCommand(),
			UpdateCommand(),
			DeleteCommand(),
			BatchGetCommand(),
			BatchUpdateCommand(),
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

func ListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("iap list", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	maxResults := fs.Int("max-results", 100, "Maximum number of results")
	paginate := fs.Bool("paginate", false, "Fetch all pages")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay iap list --package <name> [--max-results <n>] [--paginate]",
		ShortHelp:  "List all in-app products.",
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

			var all []*androidpublisher.InAppProduct
			pageToken := ""
			for {
				call := service.API.Inappproducts.List(pkg).Context(ctx).MaxResults(int64(*maxResults))
				if pageToken != "" {
					call.Token(pageToken)
				}
				resp, err := call.Do()
				if err != nil {
					return err
				}
				if !*paginate {
					return shared.PrintOutput(resp, *outputFlag, *pretty)
				}
				all = append(all, resp.Inappproduct...)
				if resp.TokenPagination == nil || resp.TokenPagination.NextPageToken == "" {
					break
				}
				pageToken = resp.TokenPagination.NextPageToken
			}

			return shared.PrintOutput(all, *outputFlag, *pretty)
		},
	}
}

func GetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("iap get", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	sku := fs.String("sku", "", "Product SKU/ID")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "gplay iap get --package <name> --sku <sku>",
		ShortHelp:  "Get an in-app product.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*sku) == "" {
				return fmt.Errorf("--sku is required")
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

			resp, err := service.API.Inappproducts.Get(pkg, *sku).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func CreateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("iap create", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	jsonFlag := fs.String("json", "", "InAppProduct JSON (or @file)")
	autoConvertPrices := fs.Bool("auto-convert-prices", true, "Auto-convert missing prices to local currencies")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "create",
		ShortUsage: "gplay iap create --package <name> --json <json>",
		ShortHelp:  "Create an in-app product.",
		LongHelp: `Create a new in-app product.

JSON format:
{
  "sku": "premium_upgrade",
  "status": "active",
  "purchaseType": "managedUser",
  "defaultPrice": {
    "priceMicros": "990000",
    "currency": "USD"
  },
  "listings": {
    "en-US": {
      "title": "Premium Upgrade",
      "description": "Unlock all premium features"
    }
  }
}

purchaseType can be:
  - managedUser: One-time purchase
  - subscription: Recurring subscription (use subscriptions command instead)`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
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

			var product androidpublisher.InAppProduct
			if err := shared.LoadJSONArg(*jsonFlag, &product); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			product.PackageName = pkg

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			call := service.API.Inappproducts.Insert(pkg, &product).Context(ctx)
			if *autoConvertPrices {
				call = call.AutoConvertMissingPrices(true)
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
	fs := flag.NewFlagSet("iap update", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	sku := fs.String("sku", "", "Product SKU/ID")
	jsonFlag := fs.String("json", "", "InAppProduct JSON (or @file)")
	autoConvertPrices := fs.Bool("auto-convert-prices", true, "Auto-convert missing prices to local currencies")
	allowMissing := fs.Bool("allow-missing", false, "Create if not exists")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "update",
		ShortUsage: "gplay iap update --package <name> --sku <sku> --json <json>",
		ShortHelp:  "Update an in-app product.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*sku) == "" {
				return fmt.Errorf("--sku is required")
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

			var product androidpublisher.InAppProduct
			if err := shared.LoadJSONArg(*jsonFlag, &product); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			product.PackageName = pkg
			product.Sku = *sku

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			call := service.API.Inappproducts.Update(pkg, *sku, &product).Context(ctx)
			if *autoConvertPrices {
				call = call.AutoConvertMissingPrices(true)
			}
			if *allowMissing {
				call = call.AllowMissing(true)
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
	fs := flag.NewFlagSet("iap delete", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	sku := fs.String("sku", "", "Product SKU/ID")
	confirm := fs.Bool("confirm", false, "Confirm deletion")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "delete",
		ShortUsage: "gplay iap delete --package <name> --sku <sku> --confirm",
		ShortHelp:  "Delete an in-app product.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*sku) == "" {
				return fmt.Errorf("--sku is required")
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

			err = service.API.Inappproducts.Delete(pkg, *sku).Context(ctx).Do()
			if err != nil {
				return err
			}

			result := map[string]interface{}{
				"deleted": true,
				"sku":     *sku,
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}

func BatchGetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("iap batch-get", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	skus := fs.String("skus", "", "Comma-separated list of SKUs")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "batch-get",
		ShortUsage: "gplay iap batch-get --package <name> --skus <sku1,sku2,...>",
		ShortHelp:  "Get multiple in-app products.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*skus) == "" {
				return fmt.Errorf("--skus is required")
			}
			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			skuList := strings.Split(*skus, ",")
			for i := range skuList {
				skuList[i] = strings.TrimSpace(skuList[i])
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.API.Inappproducts.BatchGet(pkg).Sku(skuList...).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func BatchUpdateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("iap batch-update", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	jsonFlag := fs.String("json", "", "Array of InAppProducts JSON (or @file)")
	autoConvertPrices := fs.Bool("auto-convert-prices", true, "Auto-convert missing prices to local currencies")
	allowMissing := fs.Bool("allow-missing", false, "Create if not exists")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "batch-update",
		ShortUsage: "gplay iap batch-update --package <name> --json <json>",
		ShortHelp:  "Update multiple in-app products.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
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

			var products []*androidpublisher.InAppProduct
			if err := shared.LoadJSONArg(*jsonFlag, &products); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			batchReq := &androidpublisher.InappproductsBatchUpdateRequest{
				Requests: make([]*androidpublisher.InappproductsUpdateRequest, 0, len(products)),
			}
			for _, p := range products {
				p.PackageName = pkg
				batchReq.Requests = append(batchReq.Requests, &androidpublisher.InappproductsUpdateRequest{
					AutoConvertMissingPrices: *autoConvertPrices,
					AllowMissing:             *allowMissing,
					Inappproduct:             p,
				})
			}

			resp, err := service.API.Inappproducts.BatchUpdate(pkg, batchReq).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func BatchDeleteCommand() *ffcli.Command {
	fs := flag.NewFlagSet("iap batch-delete", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	skus := fs.String("skus", "", "Comma-separated list of SKUs")
	confirm := fs.Bool("confirm", false, "Confirm deletion")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "batch-delete",
		ShortUsage: "gplay iap batch-delete --package <name> --skus <sku1,sku2,...> --confirm",
		ShortHelp:  "Delete multiple in-app products.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*skus) == "" {
				return fmt.Errorf("--skus is required")
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

			skuList := strings.Split(*skus, ",")
			for i := range skuList {
				skuList[i] = strings.TrimSpace(skuList[i])
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			var requests []*androidpublisher.InappproductsDeleteRequest
			for _, sku := range skuList {
				requests = append(requests, &androidpublisher.InappproductsDeleteRequest{
					Sku:         sku,
					PackageName: pkg,
				})
			}

			req := &androidpublisher.InappproductsBatchDeleteRequest{
				Requests: requests,
			}

			err = service.API.Inappproducts.BatchDelete(pkg, req).Context(ctx).Do()
			if err != nil {
				return err
			}

			result := map[string]interface{}{
				"deleted": true,
				"skus":    skuList,
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}
