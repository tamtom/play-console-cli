package purchases

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

func PurchasesCommand() *ffcli.Command {
	fs := flag.NewFlagSet("purchases", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "purchases",
		ShortUsage: "gplay purchases <subcommand> [flags]",
		ShortHelp:  "Verify and manage purchases.",
		LongHelp: `Verify and manage in-app purchases and subscriptions.

Use these commands for server-side purchase validation
and subscription management.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			ProductsCommand(),
			ProductsV2Command(),
			SubscriptionsCommand(),
			VoidedCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			return flag.ErrHelp
		},
	}
}

// ProductsCommand handles in-app product purchases
func ProductsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("purchases products", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "products",
		ShortUsage: "gplay purchases products <subcommand> [flags]",
		ShortHelp:  "Verify and manage in-app product purchases.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			ProductsGetCommand(),
			ProductsAcknowledgeCommand(),
			ProductsConsumeCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			return flag.ErrHelp
		},
	}
}

func ProductsGetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("purchases products get", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Product ID (SKU)")
	token := fs.String("token", "", "Purchase token")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "gplay purchases products get --package <name> --product-id <id> --token <token>",
		ShortHelp:  "Get purchase details for verification.",
		LongHelp: `Get purchase details for server-side verification.

The response includes:
  - purchaseState: 0=Purchased, 1=Canceled, 2=Pending
  - consumptionState: 0=Not consumed, 1=Consumed
  - acknowledgementState: 0=Not acknowledged, 1=Acknowledged`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
			}
			if strings.TrimSpace(*token) == "" {
				return fmt.Errorf("--token is required")
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

			resp, err := service.API.Purchases.Products.Get(pkg, *productID, *token).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func ProductsAcknowledgeCommand() *ffcli.Command {
	fs := flag.NewFlagSet("purchases products acknowledge", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Product ID (SKU)")
	token := fs.String("token", "", "Purchase token")
	developerPayload := fs.String("developer-payload", "", "Optional developer payload")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "acknowledge",
		ShortUsage: "gplay purchases products acknowledge --package <name> --product-id <id> --token <token>",
		ShortHelp:  "Acknowledge a purchase.",
		LongHelp: `Acknowledge a purchase.

Purchases must be acknowledged within 3 days or they are
automatically refunded. Use this for server-side acknowledgement.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
			}
			if strings.TrimSpace(*token) == "" {
				return fmt.Errorf("--token is required")
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

			req := &androidpublisher.ProductPurchasesAcknowledgeRequest{}
			if strings.TrimSpace(*developerPayload) != "" {
				req.DeveloperPayload = *developerPayload
			}

			err = service.API.Purchases.Products.Acknowledge(pkg, *productID, *token, req).Context(ctx).Do()
			if err != nil {
				return err
			}

			result := map[string]interface{}{
				"acknowledged": true,
				"productId":    *productID,
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}

func ProductsConsumeCommand() *ffcli.Command {
	fs := flag.NewFlagSet("purchases products consume", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Product ID (SKU)")
	token := fs.String("token", "", "Purchase token")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "consume",
		ShortUsage: "gplay purchases products consume --package <name> --product-id <id> --token <token>",
		ShortHelp:  "Consume a purchase (for consumable products).",
		LongHelp: `Consume a purchase.

Use this for consumable products that can be purchased multiple times.
After consumption, the product can be purchased again.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
			}
			if strings.TrimSpace(*token) == "" {
				return fmt.Errorf("--token is required")
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

			err = service.API.Purchases.Products.Consume(pkg, *productID, *token).Context(ctx).Do()
			if err != nil {
				return err
			}

			result := map[string]interface{}{
				"consumed":  true,
				"productId": *productID,
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}

// ProductsV2Command handles in-app product purchases using v2 API
func ProductsV2Command() *ffcli.Command {
	fs := flag.NewFlagSet("purchases productsv2", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "productsv2",
		ShortUsage: "gplay purchases productsv2 <subcommand> [flags]",
		ShortHelp:  "Verify in-app product purchases (v2 API).",
		LongHelp: `Verify in-app product purchases using the newer v2 API.

The v2 API provides enhanced purchase information including
multi-quantity purchases and improved status fields.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			ProductsV2GetCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			return flag.ErrHelp
		},
	}
}

func ProductsV2GetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("purchases productsv2 get", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	token := fs.String("token", "", "Purchase token")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "gplay purchases productsv2 get --package <name> --token <token>",
		ShortHelp:  "Get purchase details using v2 API.",
		LongHelp: `Get purchase details using the v2 API.

The v2 API returns enhanced purchase information including:
  - productId: The purchased product ID
  - purchaseState: Current state of the purchase
  - consumptionState: Whether the product has been consumed
  - acknowledgementState: Whether acknowledged
  - quantity: Number of items purchased (for multi-quantity)`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*token) == "" {
				return fmt.Errorf("--token is required")
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

			resp, err := service.API.Purchases.Productsv2.Getproductpurchasev2(pkg, *token).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

// SubscriptionsCommand handles subscription purchases
func SubscriptionsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("purchases subscriptions", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "subscriptions",
		ShortUsage: "gplay purchases subscriptions <subcommand> [flags]",
		ShortHelp:  "Verify and manage subscription purchases.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			SubscriptionsGetCommand(),
			SubscriptionsCancelCommand(),
			SubscriptionsDeferCommand(),
			SubscriptionsRevokeCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			return flag.ErrHelp
		},
	}
}

func SubscriptionsGetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("purchases subscriptions get", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	token := fs.String("token", "", "Purchase token")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "gplay purchases subscriptions get --package <name> --token <token>",
		ShortHelp:  "Get subscription purchase details (v2 API).",
		LongHelp: `Get subscription purchase details using the v2 API.

The response includes:
  - subscriptionState: Current state of the subscription
  - lineItems: Details of each subscription item
  - acknowledgementState: Whether the subscription is acknowledged`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*token) == "" {
				return fmt.Errorf("--token is required")
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

			resp, err := service.API.Purchases.Subscriptionsv2.Get(pkg, *token).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func SubscriptionsCancelCommand() *ffcli.Command {
	fs := flag.NewFlagSet("purchases subscriptions cancel", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	subscriptionID := fs.String("subscription-id", "", "Subscription ID")
	token := fs.String("token", "", "Purchase token")
	confirm := fs.Bool("confirm", false, "Confirm cancellation")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "cancel",
		ShortUsage: "gplay purchases subscriptions cancel --package <name> --subscription-id <id> --token <token> --confirm",
		ShortHelp:  "Cancel a subscription.",
		LongHelp: `Cancel a subscription.

The subscription remains active until the end of the current
billing period, then will not renew.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*subscriptionID) == "" {
				return fmt.Errorf("--subscription-id is required")
			}
			if strings.TrimSpace(*token) == "" {
				return fmt.Errorf("--token is required")
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

			err = service.API.Purchases.Subscriptions.Cancel(pkg, *subscriptionID, *token).Context(ctx).Do()
			if err != nil {
				return err
			}

			result := map[string]interface{}{
				"canceled":       true,
				"subscriptionId": *subscriptionID,
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}

func SubscriptionsDeferCommand() *ffcli.Command {
	fs := flag.NewFlagSet("purchases subscriptions defer", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	subscriptionID := fs.String("subscription-id", "", "Subscription ID")
	token := fs.String("token", "", "Purchase token")
	jsonFlag := fs.String("json", "", "DeferralInfo JSON (or @file)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "defer",
		ShortUsage: "gplay purchases subscriptions defer --package <name> --subscription-id <id> --token <token> --json <json>",
		ShortHelp:  "Defer billing for a subscription.",
		LongHelp: `Defer billing for a subscription.

JSON format:
{
  "deferralInfo": {
    "expectedExpiryTimeMillis": 1735689600000,
    "desiredExpiryTimeMillis": 1738368000000
  }
}

The new expiry time must be:
- In the future
- Before the current billing period ends
- No more than one year ahead`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*subscriptionID) == "" {
				return fmt.Errorf("--subscription-id is required")
			}
			if strings.TrimSpace(*token) == "" {
				return fmt.Errorf("--token is required")
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

			var req androidpublisher.SubscriptionPurchasesDeferRequest
			if err := shared.LoadJSONArg(*jsonFlag, &req); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.API.Purchases.Subscriptions.Defer(pkg, *subscriptionID, *token, &req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func SubscriptionsRevokeCommand() *ffcli.Command {
	fs := flag.NewFlagSet("purchases subscriptions revoke", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	subscriptionID := fs.String("subscription-id", "", "Subscription ID")
	token := fs.String("token", "", "Purchase token")
	confirm := fs.Bool("confirm", false, "Confirm revocation")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "revoke",
		ShortUsage: "gplay purchases subscriptions revoke --package <name> --subscription-id <id> --token <token> --confirm",
		ShortHelp:  "Revoke a subscription immediately.",
		LongHelp: `Revoke a subscription immediately.

Unlike cancel, this immediately ends the subscription and
the user loses access right away.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*subscriptionID) == "" {
				return fmt.Errorf("--subscription-id is required")
			}
			if strings.TrimSpace(*token) == "" {
				return fmt.Errorf("--token is required")
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

			err = service.API.Purchases.Subscriptions.Revoke(pkg, *subscriptionID, *token).Context(ctx).Do()
			if err != nil {
				return err
			}

			result := map[string]interface{}{
				"revoked":        true,
				"subscriptionId": *subscriptionID,
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}

// VoidedCommand handles voided purchases
func VoidedCommand() *ffcli.Command {
	fs := flag.NewFlagSet("purchases voided", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "voided",
		ShortUsage: "gplay purchases voided <subcommand> [flags]",
		ShortHelp:  "Track voided purchases (refunds, chargebacks).",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			VoidedListCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			return flag.ErrHelp
		},
	}
}

func VoidedListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("purchases voided list", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	startTime := fs.Int64("start-time", 0, "Start time in milliseconds since epoch")
	endTime := fs.Int64("end-time", 0, "End time in milliseconds since epoch")
	maxResults := fs.Int("max-results", 100, "Maximum results per page")
	voidedType := fs.Int("type", 0, "Voided source type: 0=All, 1=Refund, 2=Chargeback")
	includeQuantity := fs.Bool("include-quantity", false, "Include quantity information")
	paginate := fs.Bool("paginate", false, "Fetch all pages")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay purchases voided list --package <name> [--start-time <ms>] [--end-time <ms>]",
		ShortHelp:  "List voided purchases.",
		LongHelp: `List voided purchases (refunds and chargebacks).

Use this to track:
  - Refunds issued by you or Google
  - Chargebacks from payment processors

The --type flag filters by voided source:
  0 = All voided purchases
  1 = Refunds only
  2 = Chargebacks only`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
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

			var all []*androidpublisher.VoidedPurchase
			pageToken := ""
			for {
				call := service.API.Purchases.Voidedpurchases.List(pkg).Context(ctx).MaxResults(int64(*maxResults))
				if *startTime > 0 {
					call = call.StartTime(*startTime)
				}
				if *endTime > 0 {
					call = call.EndTime(*endTime)
				}
				if *voidedType > 0 {
					call = call.Type(int64(*voidedType))
				}
				if *includeQuantity {
					call = call.IncludeQuantityBasedPartialRefund(true)
				}
				if pageToken != "" {
					call = call.Token(pageToken)
				}
				resp, err := call.Do()
				if err != nil {
					return err
				}
				if !*paginate {
					return shared.PrintOutput(resp, *outputFlag, *pretty)
				}
				all = append(all, resp.VoidedPurchases...)
				if resp.TokenPagination == nil || resp.TokenPagination.NextPageToken == "" {
					break
				}
				pageToken = resp.TokenPagination.NextPageToken
			}

			return shared.PrintOutput(all, *outputFlag, *pretty)
		},
	}
}
