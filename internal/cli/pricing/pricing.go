package pricing

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

func PricingCommand() *ffcli.Command {
	fs := flag.NewFlagSet("pricing", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "pricing",
		ShortUsage: "gplay pricing <subcommand> [flags]",
		ShortHelp:  "Pricing utilities.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			ConvertCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			return flag.ErrHelp
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
