package pricing

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"

	"github.com/tamtom/play-console-cli/internal/cli/monetizationpricing"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

func PricingCommand() *ffcli.Command {
	fs := flag.NewFlagSet("pricing", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "pricing",
		ShortUsage: "gplay pricing <subcommand> [flags]",
		ShortHelp:  "Pricing conversion and regions-version discovery.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			ConvertCommand(),
			RegionsVersionCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			return flag.ErrHelp
		},
	}
}

func RegionsVersionCommand() *ffcli.Command {
	fs := flag.NewFlagSet("pricing regions-version", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	priceJSON := fs.String("price-json", "", "Base Money JSON (or @file)")
	productTaxCategoryCode := fs.String("product-tax-category-code", "", "Product tax category code")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "regions-version",
		ShortUsage: "gplay pricing regions-version --package <name> --price-json <money-json>",
		ShortHelp:  "Discover the current regions version and valid currencies.",
		LongHelp: `Discover Google's current regions version and valid region currencies.

This calls pricing convert with a base Money value, then prints the returned
regionVersion plus the billable regions and currencies Google accepts for that
version. Use this output instead of copying regional pricing from list/get
responses when creating subscriptions or one-time products.

For new products, the safer workflow is usually:
  gplay subscriptions create --auto-convert-regional-prices --base-price-json ...
  gplay onetimeproducts create --auto-convert-regional-prices --base-price-json ...

Those create commands call the same Google API and apply the returned
regionVersion automatically.

Money JSON format:
{
  "currencyCode": "USD",
  "units": "9",
  "nanos": 990000000
}

Examples:
  gplay pricing regions-version --package com.example.app --price-json '{"currencyCode":"USD","units":"9","nanos":990000000}'
  gplay pricing regions-version --package com.example.app --price-json @price.json --output table`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*priceJSON) == "" {
				return fmt.Errorf("--price-json is required")
			}
			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}
			price, err := monetizationpricing.LoadMoney(*priceJSON)
			if err != nil {
				return fmt.Errorf("invalid --price-json: %w", err)
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := monetizationpricing.ConvertRegionPrices(ctx, service, pkg, price, *productTaxCategoryCode)
			if err != nil {
				return err
			}
			summary, err := monetizationpricing.Summary(resp)
			if err != nil {
				return err
			}
			return shared.PrintOutput(summary, *outputFlag, *pretty)
		},
	}
}

func ConvertCommand() *ffcli.Command {
	fs := flag.NewFlagSet("pricing convert", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	jsonFlag := fs.String("json", "", "ConvertRegionPricesRequest JSON (or @file)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "convert",
		ShortUsage: "gplay pricing convert --package <name> --json <json>",
		ShortHelp:  "Convert a price to region-specific prices.",
		LongHelp: `Convert a base price to region-specific prices.

This is useful for calculating equivalent prices across regions
while maintaining Google Play's pricing tiers.

The response includes regionVersion, which can be passed to
subscriptions, base plans, offers, and one-time product commands with
--regions-version when setting regional pricing.

JSON format:
{
  "price": {
    "currencyCode": "USD",
    "units": "9",
    "nanos": 990000000
  }
}

This will return prices for all supported regions in their
local currencies, adjusted to Google Play's pricing tiers.`,
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

			var req androidpublisher.ConvertRegionPricesRequest
			if err := shared.LoadJSONArg(*jsonFlag, &req); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.API.Monetization.ConvertRegionPrices(pkg, &req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}
