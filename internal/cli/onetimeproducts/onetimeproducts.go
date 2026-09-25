package onetimeproducts

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"

	"github.com/tamtom/play-console-cli/internal/cli/monetizationpricing"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/output"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

// otpMutableFields are the top-level fields on OneTimeProduct that can be
// set via update_mask. Must match the fields the SDK can serialize.
var otpMutableFields = []string{
	"listings",
	"offerTags",
	"purchaseOptions",
	"restrictedPaymentCountries",
	"taxAndComplianceSettings",
}

type BatchOutcome struct {
	ProductID string                           `json:"productId"`
	Status    string                           `json:"status"`
	Error     string                           `json:"error,omitempty"`
	Product   *androidpublisher.OneTimeProduct `json:"product,omitempty"`
}

func init() {
	output.RegisterType(&androidpublisher.OneTimeProduct{}, []string{"PRODUCT ID", "PACKAGE", "LISTINGS", "PURCHASE OPTIONS", "REGIONS VERSION"}, func(data any) [][]string {
		return oneTimeProductRows([]*androidpublisher.OneTimeProduct{data.(*androidpublisher.OneTimeProduct)})
	})
	output.RegisterType([]*androidpublisher.OneTimeProduct{}, []string{"PRODUCT ID", "PACKAGE", "LISTINGS", "PURCHASE OPTIONS", "REGIONS VERSION"}, func(data any) [][]string {
		return oneTimeProductRows(data.([]*androidpublisher.OneTimeProduct))
	})
	output.RegisterType(&androidpublisher.ListOneTimeProductsResponse{}, []string{"PRODUCT ID", "PACKAGE", "LISTINGS", "PURCHASE OPTIONS", "REGIONS VERSION"}, func(data any) [][]string {
		return oneTimeProductRows(data.(*androidpublisher.ListOneTimeProductsResponse).OneTimeProducts)
	})
	output.RegisterType(&androidpublisher.BatchGetOneTimeProductsResponse{}, []string{"PRODUCT ID", "PACKAGE", "LISTINGS", "PURCHASE OPTIONS", "REGIONS VERSION"}, func(data any) [][]string {
		return oneTimeProductRows(data.(*androidpublisher.BatchGetOneTimeProductsResponse).OneTimeProducts)
	})
	output.RegisterType([]BatchOutcome{}, []string{"PRODUCT ID", "STATUS", "ERROR"}, func(data any) [][]string {
		outcomes := data.([]BatchOutcome)
		rows := make([][]string, 0, len(outcomes))
		for _, outcome := range outcomes {
			rows = append(rows, []string{outcome.ProductID, outcome.Status, outcome.Error})
		}
		return rows
	})
}

func oneTimeProductRows(products []*androidpublisher.OneTimeProduct) [][]string {
	rows := make([][]string, 0, len(products))
	for _, product := range products {
		if product == nil {
			continue
		}
		regionsVersion := ""
		if product.RegionsVersion != nil {
			regionsVersion = product.RegionsVersion.Version
		}
		rows = append(rows, []string{
			product.ProductId,
			product.PackageName,
			fmt.Sprint(len(product.Listings)),
			fmt.Sprint(len(product.PurchaseOptions)),
			regionsVersion,
		})
	}
	return rows
}

func OneTimeProductsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("onetime-products", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "onetime-products",
		ShortUsage: "gplay onetime-products <subcommand> [flags]",
		ShortHelp:  "Manage one-time products (monetization).",
		LongHelp: `Manage one-time products in the monetization system.

One-time products are non-subscription purchases that users buy once.
This includes consumables (can be purchased again) and non-consumables.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			ListCommand(),
			GetCommand(),
			CreateCommand(),
			UpdateCommand(),
			shared.DeprecatedAliasLeafCommand(UpdateCommand(), "patch", "gplay onetime-products update"),
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

func LegacyOneTimeProductsCommand() *ffcli.Command {
	return shared.DeprecatedAliasLeafCommand(OneTimeProductsCommand(), "onetimeproducts", "gplay onetime-products")
}

func ListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("onetime-products list", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	pageSize := fs.Int("page-size", 100, "Page size")
	paginate := fs.Bool("paginate", false, "Fetch all pages")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay onetime-products list --package <name>",
		ShortHelp:  "List all one-time products.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if *pageSize < 1 || *pageSize > 1000 {
				return fmt.Errorf("--page-size must be between 1 and 1000")
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

			var all []*androidpublisher.OneTimeProduct
			pageToken := ""
			guard := shared.NewPaginationGuard(0)
			for {
				call := service.API.Monetization.Onetimeproducts.List(pkg).Context(ctx).PageSize(int64(*pageSize))
				if pageToken != "" {
					call = call.PageToken(pageToken)
				}
				resp, err := call.Do()
				if err != nil {
					return err
				}
				if !*paginate {
					return shared.PrintOutputContext(ctx, resp, *outputFlag, *pretty)
				}
				all = append(all, resp.OneTimeProducts...)
				done, err := guard.Advance(resp.NextPageToken)
				if err != nil {
					return err
				}
				if done {
					break
				}
				pageToken = resp.NextPageToken
			}
			return shared.PrintOutputContext(ctx, all, *outputFlag, *pretty)
		},
	}
}

func GetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("onetime-products get", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Product ID")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "gplay onetime-products get --package <name> --product-id <id>",
		ShortHelp:  "Get a one-time product.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if err := validateProductID(strings.TrimSpace(*productID)); err != nil {
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

			resp, err := service.API.Monetization.Onetimeproducts.Get(pkg, *productID).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutputContext(ctx, resp, *outputFlag, *pretty)
		},
	}
}

func CreateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("onetime-products create", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Product ID")
	jsonFlag := fs.String("json", "", "OneTimeProduct JSON (or @file)")
	regionsVersion := fs.String("regions-version", "", "Regions version for price migration")
	latencyTolerance := fs.String("latency-tolerance", "", "Propagation latency tolerance (API default: latency-sensitive)")
	autoConvertRegionalPrices := fs.Bool("auto-convert-regional-prices", false, "Generate regional pricing from --base-price-json")
	basePriceJSON := fs.String("base-price-json", "", "Base Money JSON for --auto-convert-regional-prices (or @file)")
	productTaxCategoryCode := fs.String("product-tax-category-code", "", "Product tax category code for price conversion")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "create",
		ShortUsage: "gplay onetime-products create --package <name> --product-id <id> --json <json>",
		ShortHelp:  "Create a one-time product.",
		LongHelp: `Create a one-time product using patch with allowMissing=true.

The --regions-version flag is required when setting regional pricing.
Use gplay pricing convert to get Google's current regionVersion and
region-specific converted prices.

The --package and --product-id flag values are applied to the request body,
so they do not need to be repeated in the JSON.

Use --auto-convert-regional-prices with --base-price-json to let Google Play
generate valid regionalPricingAndAvailabilityConfigs, newRegionsConfig, and
regionsVersion from one base price. This replaces any regional pricing in the
JSON with the billable regions returned by Google for the current regionVersion.

JSON format:
{
  "listings": [
    {
      "languageCode": "en-US",
      "title": "100 Coins",
      "description": "A pack of 100 coins"
    }
  ],
  "purchaseOptions": [
    {
      "purchaseOptionId": "default",
      "buyOption": {},
      "regionalPricingAndAvailabilityConfigs": [
        {
          "regionCode": "US",
          "availability": "AVAILABLE",
          "price": {
            "currencyCode": "USD",
            "units": "1",
            "nanos": 990000000
          }
        }
      ]
    }
  ],
  "offerTags": [
    {"tag": "coins"}
  ]
}

Examples:
	gplay onetime-products create --package com.example.app --product-id coins_100 --json @product.json --auto-convert-regional-prices --base-price-json '{"currencyCode":"USD","units":"1","nanos":990000000}'
  gplay pricing regions-version --package com.example.app --price-json '{"currencyCode":"USD","units":"1","nanos":990000000}' --output table
	gplay onetime-products create --package com.example.app --product-id coins_100 --json @product.json --regions-version 2025/02`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if err := validateProductID(strings.TrimSpace(*productID)); err != nil {
				return err
			}
			resolvedLatencyTolerance, err := normalizeLatencyTolerance(*latencyTolerance)
			if err != nil {
				return err
			}
			if strings.TrimSpace(*jsonFlag) == "" {
				return fmt.Errorf("--json is required")
			}
			raw, err := shared.LoadJSONArgRaw(*jsonFlag)
			if err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			updateMask, err := shared.DeriveUpdateMask(raw, otpMutableFields)
			if err != nil {
				return err
			}
			var product androidpublisher.OneTimeProduct
			if err := json.Unmarshal(raw, &product); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			var basePrice *androidpublisher.Money
			resolvedRegionsVersion := strings.TrimSpace(*regionsVersion)
			if *autoConvertRegionalPrices {
				if resolvedRegionsVersion != "" {
					return fmt.Errorf("--regions-version cannot be used with --auto-convert-regional-prices; Google Play returns the matching regionVersion")
				}
				if len(product.PurchaseOptions) == 0 {
					return fmt.Errorf("--auto-convert-regional-prices requires at least one purchase option in --json")
				}
				var err error
				basePrice, err = monetizationpricing.LoadMoney(*basePriceJSON)
				if err != nil {
					return fmt.Errorf("--base-price-json is required for --auto-convert-regional-prices: %w", err)
				}
			}
			if !*autoConvertRegionalPrices {
				if err := requireRegionsVersion(&product, resolvedRegionsVersion); err != nil {
					return err
				}
			}

			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}
			product.PackageName = pkg
			product.ProductId = *productID

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			if *autoConvertRegionalPrices {
				converted, err := monetizationpricing.ConvertRegionPrices(ctx, service, pkg, basePrice, *productTaxCategoryCode)
				if err != nil {
					return err
				}
				resolvedRegionsVersion, err = monetizationpricing.RegionVersion(converted)
				if err != nil {
					return err
				}
				regionalConfigs := monetizationpricing.OneTimeProductRegionalConfigs(converted, monetizationpricing.DefaultRegionalAvailability)
				newRegionsConfig := monetizationpricing.OneTimeProductNewRegionsConfig(converted, monetizationpricing.DefaultRegionalAvailability)
				for _, purchaseOption := range product.PurchaseOptions {
					purchaseOption.RegionalPricingAndAvailabilityConfigs = regionalConfigs
					purchaseOption.NewRegionsConfig = newRegionsConfig
				}
			}

			call := service.API.Monetization.Onetimeproducts.Patch(pkg, *productID, &product).Context(ctx).AllowMissing(true).UpdateMask(updateMask)
			if resolvedRegionsVersion != "" {
				call = call.RegionsVersionVersion(resolvedRegionsVersion)
			}
			if resolvedLatencyTolerance != "" {
				call = call.LatencyTolerance(resolvedLatencyTolerance)
			}
			resp, err := call.Do()
			if err != nil {
				return err
			}
			return shared.PrintOutputContext(ctx, resp, *outputFlag, *pretty)
		},
	}
}

func UpdateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("onetime-products update", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Product ID")
	jsonFlag := fs.String("json", "", "OneTimeProduct JSON (or @file)")
	updateMask := fs.String("update-mask", "", "Fields to update (comma-separated)")
	regionsVersion := fs.String("regions-version", "", "Regions version for price migration")
	latencyTolerance := fs.String("latency-tolerance", "", "Propagation latency tolerance (API default: latency-sensitive)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "update",
		ShortUsage: "gplay onetime-products update --package <name> --product-id <id> --json <json> --update-mask <fields>",
		ShortHelp:  "Update an existing one-time product.",
		LongHelp: `Update specific fields of a one-time product.

--update-mask is required. Updates always send allowMissing=false so a typo in
--product-id cannot silently create another product.

Mutable fields: listings, offerTags, purchaseOptions, restrictedPaymentCountries,
taxAndComplianceSettings.

JSON format (partial update):
{
  "listings": [
    {
      "languageCode": "en-US",
      "title": "200 Coins",
      "description": "A pack of 200 coins"
    }
  ]
}

Examples:
	gplay onetime-products update --package com.example.app --product-id coins_100 --json @patch.json --update-mask listings
	gplay onetime-products update --package com.example.app --product-id coins_100 --json @product.json --update-mask purchaseOptions --regions-version 2025/02`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if err := validateProductID(strings.TrimSpace(*productID)); err != nil {
				return err
			}
			if strings.TrimSpace(*jsonFlag) == "" {
				return fmt.Errorf("--json is required")
			}
			mask := strings.TrimSpace(*updateMask)
			if mask == "" {
				return fmt.Errorf("--update-mask is required for update; use create to create a missing product")
			}
			resolvedLatencyTolerance, err := normalizeLatencyTolerance(*latencyTolerance)
			if err != nil {
				return err
			}
			raw, err := shared.LoadJSONArgRaw(*jsonFlag)
			if err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			var product androidpublisher.OneTimeProduct
			if err := json.Unmarshal(raw, &product); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			if err := requireRegionsVersion(&product, *regionsVersion); err != nil {
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
			product.PackageName = pkg
			product.ProductId = *productID

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			call := service.API.Monetization.Onetimeproducts.Patch(pkg, *productID, &product).Context(ctx).AllowMissing(false).UpdateMask(mask)
			if strings.TrimSpace(*regionsVersion) != "" {
				call = call.RegionsVersionVersion(*regionsVersion)
			}
			if resolvedLatencyTolerance != "" {
				call = call.LatencyTolerance(resolvedLatencyTolerance)
			}
			resp, err := call.Do()
			if err != nil {
				return err
			}
			return shared.PrintOutputContext(ctx, resp, *outputFlag, *pretty)
		},
	}
}

// PatchCommand is kept for source compatibility with downstream users of this
// package. The public CLI exposes it only as a hidden deprecated alias for
// update, and it can no longer enable allowMissing.
func PatchCommand() *ffcli.Command {
	return shared.DeprecatedAliasLeafCommand(UpdateCommand(), "patch", "gplay onetime-products update")
}

func DeleteCommand() *ffcli.Command {
	fs := flag.NewFlagSet("onetime-products delete", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Product ID")
	confirm := fs.Bool("confirm", false, "Confirm deletion")
	latencyTolerance := fs.String("latency-tolerance", "", "Propagation latency tolerance (API default: latency-sensitive)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "delete",
		ShortUsage: "gplay onetime-products delete --package <name> --product-id <id> --confirm",
		ShortHelp:  "Delete a one-time product.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if err := validateProductID(strings.TrimSpace(*productID)); err != nil {
				return err
			}
			resolvedLatencyTolerance, err := normalizeLatencyTolerance(*latencyTolerance)
			if err != nil {
				return err
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

			call := service.API.Monetization.Onetimeproducts.Delete(pkg, *productID).Context(ctx)
			if resolvedLatencyTolerance != "" {
				call = call.LatencyTolerance(resolvedLatencyTolerance)
			}
			err = call.Do()
			if err != nil {
				return err
			}

			result := map[string]interface{}{
				"deleted":   true,
				"productId": *productID,
			}
			return shared.PrintOutputContext(ctx, result, *outputFlag, *pretty)
		},
	}
}

func BatchGetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("onetime-products batch-get", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productIDs := fs.String("product-ids", "", "Comma-separated product IDs")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "batch-get",
		ShortUsage: "gplay onetime-products batch-get --package <name> --product-ids <ids>",
		ShortHelp:  "Get multiple one-time products.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productIDs) == "" {
				return fmt.Errorf("--product-ids is required")
			}
			ids := strings.Split(*productIDs, ",")
			if err := validateBatchProductIDs(ids); err != nil {
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

			resp, err := service.API.Monetization.Onetimeproducts.BatchGet(pkg).ProductIds(ids...).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutputContext(ctx, resp, *outputFlag, *pretty)
		},
	}
}

func BatchUpdateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("onetime-products batch-update", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	jsonFlag := fs.String("json", "", "BatchUpdateRequest JSON (or @file)")
	latencyTolerance := fs.String("latency-tolerance", "", "Propagation latency tolerance applied to every item (API default: latency-sensitive)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "batch-update",
		ShortUsage: "gplay onetime-products batch-update --package <name> --json <json>",
		ShortHelp:  "Create or update multiple one-time products.",
		LongHelp: `Create or update multiple one-time products in a single request.

JSON format (BatchUpdateOneTimeProductsRequest):
{
  "requests": [
    {
      "oneTimeProduct": {
        "packageName": "com.example.app",
        "productId": "coins_100",
        "listings": [
          {
            "languageCode": "en-US",
            "title": "100 Coins",
            "description": "A pack of 100 coins"
          }
        ],
        "purchaseOptions": [
          {
            "purchaseOptionId": "default",
            "buyOption": {},
            "regionalPricingAndAvailabilityConfigs": [
              {
                "regionCode": "US",
                "availability": "AVAILABLE",
                "price": {
                  "currencyCode": "USD",
                  "units": "1",
                  "nanos": 990000000
                }
              }
            ]
          }
        ]
      },
      "updateMask": "listings,purchaseOptions",
      "allowMissing": true,
      "regionsVersion": {"version": "2025/02"}
    }
  ]
}

Examples:
	gplay onetime-products batch-update --package com.example.app --json @batch.json
	gplay onetime-products batch-update --package com.example.app --json '{"requests":[...]}'`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*jsonFlag) == "" {
				return fmt.Errorf("--json is required")
			}
			resolvedLatencyTolerance, err := normalizeLatencyTolerance(*latencyTolerance)
			if err != nil {
				return err
			}
			var req androidpublisher.BatchUpdateOneTimeProductsRequest
			if err := shared.LoadJSONArg(*jsonFlag, &req); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			if err := validateBatchUpdateRequest(&req, resolvedLatencyTolerance); err != nil {
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
			for _, item := range req.Requests {
				item.OneTimeProduct.PackageName = pkg
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.API.Monetization.Onetimeproducts.BatchUpdate(pkg, &req).Context(ctx).Do()
			outcomes, complete := batchUpdateOutcomes(req.Requests, resp, err)
			if printErr := shared.PrintOutputContext(ctx, outcomes, *outputFlag, *pretty); printErr != nil {
				return printErr
			}
			if err != nil {
				return shared.NewReportedError(fmt.Errorf("batch update failed: %w", err))
			}
			if !complete {
				return shared.NewReportedError(fmt.Errorf("batch update returned fewer results than requests"))
			}
			return nil
		},
	}
}

func BatchDeleteCommand() *ffcli.Command {
	fs := flag.NewFlagSet("onetime-products batch-delete", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	jsonFlag := fs.String("json", "", "BatchDeleteRequest JSON (or @file)")
	confirm := fs.Bool("confirm", false, "Confirm deletion")
	latencyTolerance := fs.String("latency-tolerance", "", "Propagation latency tolerance applied to every item (API default: latency-sensitive)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "batch-delete",
		ShortUsage: "gplay onetime-products batch-delete --package <name> --json <json> --confirm",
		ShortHelp:  "Delete multiple one-time products.",
		LongHelp: `Delete multiple one-time products in a single request.

Requires --confirm for safety.

JSON format (BatchDeleteOneTimeProductsRequest):
{
  "requests": [
    {"packageName": "com.example.app", "productId": "coins_100"},
    {"packageName": "com.example.app", "productId": "coins_500"}
  ]
}

Examples:
	gplay onetime-products batch-delete --package com.example.app --json @delete.json --confirm
	gplay onetime-products batch-delete --package com.example.app --json '{"requests":[...]}' --confirm`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*jsonFlag) == "" {
				return fmt.Errorf("--json is required")
			}
			if !*confirm {
				return fmt.Errorf("--confirm is required")
			}
			resolvedLatencyTolerance, err := normalizeLatencyTolerance(*latencyTolerance)
			if err != nil {
				return err
			}
			var req androidpublisher.BatchDeleteOneTimeProductsRequest
			if err := shared.LoadJSONArg(*jsonFlag, &req); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			if err := validateBatchDeleteRequest(&req, resolvedLatencyTolerance); err != nil {
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
			for _, item := range req.Requests {
				item.PackageName = pkg
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			err = service.API.Monetization.Onetimeproducts.BatchDelete(pkg, &req).Context(ctx).Do()
			outcomes := batchDeleteOutcomes(req.Requests, err)
			if printErr := shared.PrintOutputContext(ctx, outcomes, *outputFlag, *pretty); printErr != nil {
				return printErr
			}
			if err != nil {
				return shared.NewReportedError(fmt.Errorf("batch delete failed: %w", err))
			}
			return nil
		},
	}
}

func batchUpdateOutcomes(requests []*androidpublisher.UpdateOneTimeProductRequest, response *androidpublisher.BatchUpdateOneTimeProductsResponse, requestErr error) ([]BatchOutcome, bool) {
	outcomes := make([]BatchOutcome, 0, len(requests))
	complete := requestErr == nil && response != nil && len(response.OneTimeProducts) == len(requests)
	for index, request := range requests {
		productID := ""
		if request != nil && request.OneTimeProduct != nil {
			productID = request.OneTimeProduct.ProductId
		}
		outcome := BatchOutcome{ProductID: productID, Status: "failed"}
		switch {
		case requestErr != nil:
			outcome.Error = requestErr.Error()
		case response == nil || index >= len(response.OneTimeProducts) || response.OneTimeProducts[index] == nil:
			outcome.Error = "API returned no result for this item"
			complete = false
		default:
			outcome.Status = "succeeded"
			outcome.Product = response.OneTimeProducts[index]
		}
		outcomes = append(outcomes, outcome)
	}
	return outcomes, complete
}

func batchDeleteOutcomes(requests []*androidpublisher.DeleteOneTimeProductRequest, requestErr error) []BatchOutcome {
	outcomes := make([]BatchOutcome, 0, len(requests))
	for _, request := range requests {
		outcome := BatchOutcome{ProductID: request.ProductId, Status: "succeeded"}
		if requestErr != nil {
			outcome.Status = "failed"
			outcome.Error = requestErr.Error()
		}
		outcomes = append(outcomes, outcome)
	}
	return outcomes
}
