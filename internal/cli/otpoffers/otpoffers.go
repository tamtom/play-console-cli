package otpoffers

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

func OTPOffersCommand() *ffcli.Command {
	fs := flag.NewFlagSet("otp-offers", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "otp-offers",
		ShortUsage: "gplay otp-offers <subcommand> [flags]",
		ShortHelp:  "Manage one-time product purchase option offers.",
		LongHelp: `Manage offers within one-time product purchase options.

Offers provide promotional pricing or special terms for purchase options.
Each offer is scoped to a specific purchase option within a one-time product.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			ListCommand(),
			ActivateCommand(),
			DeactivateCommand(),
			CancelCommand(),
			BatchGetCommand(),
			BatchUpdateCommand(),
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

func ListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("otp-offers list", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "One-time product ID")
	purchaseOptionID := fs.String("purchase-option-id", "", "Purchase option ID")
	pageSize := fs.Int("page-size", 100, "Page size")
	paginate := fs.Bool("paginate", false, "Fetch all pages")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay otp-offers list --package <name> --product-id <id> --purchase-option-id <id>",
		ShortHelp:  "List all offers for a purchase option.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
			}
			if strings.TrimSpace(*purchaseOptionID) == "" {
				return fmt.Errorf("--purchase-option-id is required")
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

			var all []*androidpublisher.OneTimeProductOffer
			pageToken := ""
			for {
				call := service.API.Monetization.Onetimeproducts.PurchaseOptions.Offers.List(pkg, *productID, *purchaseOptionID).Context(ctx).PageSize(int64(*pageSize))
				if pageToken != "" {
					call = call.PageToken(pageToken)
				}
				resp, err := call.Do()
				if err != nil {
					return err
				}
				if !*paginate {
					return shared.PrintOutput(resp, *outputFlag, *pretty)
				}
				all = append(all, resp.OneTimeProductOffers...)
				if resp.NextPageToken == "" {
					break
				}
				pageToken = resp.NextPageToken
			}
			return shared.PrintOutput(all, *outputFlag, *pretty)
		},
	}
}

func ActivateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("otp-offers activate", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "One-time product ID")
	purchaseOptionID := fs.String("purchase-option-id", "", "Purchase option ID")
	offerID := fs.String("offer-id", "", "Offer ID")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "activate",
		ShortUsage: "gplay otp-offers activate --package <name> --product-id <id> --purchase-option-id <id> --offer-id <id>",
		ShortHelp:  "Activate an OTP offer.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
			}
			if strings.TrimSpace(*purchaseOptionID) == "" {
				return fmt.Errorf("--purchase-option-id is required")
			}
			if strings.TrimSpace(*offerID) == "" {
				return fmt.Errorf("--offer-id is required")
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

			req := &androidpublisher.ActivateOneTimeProductOfferRequest{}
			resp, err := service.API.Monetization.Onetimeproducts.PurchaseOptions.Offers.Activate(pkg, *productID, *purchaseOptionID, *offerID, req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func DeactivateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("otp-offers deactivate", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "One-time product ID")
	purchaseOptionID := fs.String("purchase-option-id", "", "Purchase option ID")
	offerID := fs.String("offer-id", "", "Offer ID")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "deactivate",
		ShortUsage: "gplay otp-offers deactivate --package <name> --product-id <id> --purchase-option-id <id> --offer-id <id>",
		ShortHelp:  "Deactivate an OTP offer.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
			}
			if strings.TrimSpace(*purchaseOptionID) == "" {
				return fmt.Errorf("--purchase-option-id is required")
			}
			if strings.TrimSpace(*offerID) == "" {
				return fmt.Errorf("--offer-id is required")
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

			req := &androidpublisher.DeactivateOneTimeProductOfferRequest{}
			resp, err := service.API.Monetization.Onetimeproducts.PurchaseOptions.Offers.Deactivate(pkg, *productID, *purchaseOptionID, *offerID, req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func CancelCommand() *ffcli.Command {
	fs := flag.NewFlagSet("otp-offers cancel", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "One-time product ID")
	purchaseOptionID := fs.String("purchase-option-id", "", "Purchase option ID")
	offerID := fs.String("offer-id", "", "Offer ID")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "cancel",
		ShortUsage: "gplay otp-offers cancel --package <name> --product-id <id> --purchase-option-id <id> --offer-id <id>",
		ShortHelp:  "Cancel an OTP offer.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
			}
			if strings.TrimSpace(*purchaseOptionID) == "" {
				return fmt.Errorf("--purchase-option-id is required")
			}
			if strings.TrimSpace(*offerID) == "" {
				return fmt.Errorf("--offer-id is required")
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

			req := &androidpublisher.CancelOneTimeProductOfferRequest{}
			resp, err := service.API.Monetization.Onetimeproducts.PurchaseOptions.Offers.Cancel(pkg, *productID, *purchaseOptionID, *offerID, req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func BatchGetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("otp-offers batch-get", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "One-time product ID")
	purchaseOptionID := fs.String("purchase-option-id", "", "Purchase option ID")
	jsonFlag := fs.String("json", "", "BatchGetOneTimeProductOffersRequest JSON (or @file)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "batch-get",
		ShortUsage: "gplay otp-offers batch-get --package <name> --product-id <id> --purchase-option-id <id> --json <json>",
		ShortHelp:  "Get multiple OTP offers.",
		LongHelp: `Get multiple OTP offers in a single request.

JSON format:
{
  "requests": [
    {
      "packageName": "com.example.app",
      "productId": "premium_item",
      "purchaseOptionId": "default",
      "offerId": "launch_discount"
    },
    {
      "packageName": "com.example.app",
      "productId": "premium_item",
      "purchaseOptionId": "default",
      "offerId": "holiday_sale"
    }
  ]
}

Up to 100 requests per batch. All requests must retrieve
different offers.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
			}
			if strings.TrimSpace(*purchaseOptionID) == "" {
				return fmt.Errorf("--purchase-option-id is required")
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

			var req androidpublisher.BatchGetOneTimeProductOffersRequest
			if err := shared.LoadJSONArg(*jsonFlag, &req); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.API.Monetization.Onetimeproducts.PurchaseOptions.Offers.BatchGet(pkg, *productID, *purchaseOptionID, &req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func BatchUpdateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("otp-offers batch-update", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "One-time product ID")
	purchaseOptionID := fs.String("purchase-option-id", "", "Purchase option ID")
	jsonFlag := fs.String("json", "", "BatchUpdateOneTimeProductOffersRequest JSON (or @file)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "batch-update",
		ShortUsage: "gplay otp-offers batch-update --package <name> --product-id <id> --purchase-option-id <id> --json <json>",
		ShortHelp:  "Batch update multiple OTP offers.",
		LongHelp: `Batch update multiple OTP offers in a single request.

JSON format:
{
  "requests": [
    {
      "oneTimeProductOffer": {
        "packageName": "com.example.app",
        "productId": "premium_item",
        "purchaseOptionId": "default",
        "offerId": "launch_discount"
      },
      "regionsVersion": {
        "version": "2024001"
      },
      "updateMask": "oneTimeProductOffer.offerTags"
    }
  ]
}

Up to 100 requests per batch. All requests must update
different offers. Use updateMask to specify which fields
to update.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
			}
			if strings.TrimSpace(*purchaseOptionID) == "" {
				return fmt.Errorf("--purchase-option-id is required")
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

			var req androidpublisher.BatchUpdateOneTimeProductOffersRequest
			if err := shared.LoadJSONArg(*jsonFlag, &req); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.API.Monetization.Onetimeproducts.PurchaseOptions.Offers.BatchUpdate(pkg, *productID, *purchaseOptionID, &req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func BatchUpdateStatesCommand() *ffcli.Command {
	fs := flag.NewFlagSet("otp-offers batch-update-states", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "One-time product ID")
	purchaseOptionID := fs.String("purchase-option-id", "", "Purchase option ID")
	jsonFlag := fs.String("json", "", "BatchUpdateOneTimeProductOfferStatesRequest JSON (or @file)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "batch-update-states",
		ShortUsage: "gplay otp-offers batch-update-states --package <name> --product-id <id> --purchase-option-id <id> --json <json>",
		ShortHelp:  "Batch activate/deactivate/cancel multiple OTP offers.",
		LongHelp: `Batch update states for multiple OTP offers.

JSON format:
{
  "requests": [
    {
      "activateOneTimeProductOfferRequest": {
        "packageName": "com.example.app",
        "productId": "premium_item",
        "purchaseOptionId": "default",
        "offerId": "launch_discount"
      }
    },
    {
      "deactivateOneTimeProductOfferRequest": {
        "packageName": "com.example.app",
        "productId": "premium_item",
        "purchaseOptionId": "default",
        "offerId": "old_promo"
      }
    },
    {
      "cancelOneTimeProductOfferRequest": {
        "packageName": "com.example.app",
        "productId": "premium_item",
        "purchaseOptionId": "default",
        "offerId": "preorder_deal"
      }
    }
  ]
}

Each request must contain exactly one of:
  - activateOneTimeProductOfferRequest
  - deactivateOneTimeProductOfferRequest (for discounted offers)
  - cancelOneTimeProductOfferRequest (for pre-order offers)`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
			}
			if strings.TrimSpace(*purchaseOptionID) == "" {
				return fmt.Errorf("--purchase-option-id is required")
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

			var req androidpublisher.BatchUpdateOneTimeProductOfferStatesRequest
			if err := shared.LoadJSONArg(*jsonFlag, &req); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.API.Monetization.Onetimeproducts.PurchaseOptions.Offers.BatchUpdateStates(pkg, *productID, *purchaseOptionID, &req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func BatchDeleteCommand() *ffcli.Command {
	fs := flag.NewFlagSet("otp-offers batch-delete", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "One-time product ID")
	purchaseOptionID := fs.String("purchase-option-id", "", "Purchase option ID")
	jsonFlag := fs.String("json", "", "BatchDeleteOneTimeProductOffersRequest JSON (or @file)")
	confirm := fs.Bool("confirm", false, "Confirm deletion")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "batch-delete",
		ShortUsage: "gplay otp-offers batch-delete --package <name> --product-id <id> --purchase-option-id <id> --json <json> --confirm",
		ShortHelp:  "Batch delete OTP offers.",
		LongHelp: `Batch delete multiple OTP offers in a single request.

JSON format:
{
  "requests": [
    {
      "packageName": "com.example.app",
      "productId": "premium_item",
      "purchaseOptionId": "default",
      "offerId": "launch_discount"
    },
    {
      "packageName": "com.example.app",
      "productId": "premium_item",
      "purchaseOptionId": "default",
      "offerId": "old_promo"
    }
  ]
}

Up to 100 requests per batch. All requests must correspond
to different offers. Requires --confirm.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
			}
			if strings.TrimSpace(*purchaseOptionID) == "" {
				return fmt.Errorf("--purchase-option-id is required")
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

			var req androidpublisher.BatchDeleteOneTimeProductOffersRequest
			if err := shared.LoadJSONArg(*jsonFlag, &req); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			err = service.API.Monetization.Onetimeproducts.PurchaseOptions.Offers.BatchDelete(pkg, *productID, *purchaseOptionID, &req).Context(ctx).Do()
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
