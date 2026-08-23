package orders

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

var newPlayService = playclient.NewService

func OrdersCommand() *ffcli.Command {
	fs := flag.NewFlagSet("orders", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "orders",
		ShortUsage: "gplay orders <subcommand> [flags]",
		ShortHelp:  "Manage orders.",
		LongHelp: `Manage orders and refunds.

Orders represent completed transactions for in-app products
and subscriptions.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			GetCommand(),
			BatchGetCommand(),
			RefundCommand(),
			ReviewRefundCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			return flag.ErrHelp
		},
	}
}

// ReviewRefundCommand submits the developer's explicit preference for a
// pending refund review. It never chooses a preference automatically.
func ReviewRefundCommand() *ffcli.Command {
	fs := flag.NewFlagSet("orders review-refund", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	orderID := fs.String("order-id", "", "Order ID under refund review")
	jsonFlag := fs.String("json", "", "OrdersReviewRefundRequest JSON (or @file)")
	confirm := fs.Bool("confirm", false, "Confirm submission of the refund preference")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "review-refund",
		ShortUsage: "gplay orders review-refund --package <name> --order-id <id> --json <json> --confirm",
		ShortHelp:  "Submit an explicit preference for a pending refund review.",
		LongHelp: `Submit evidence and an explicit developer preference for a pending refund.

Gplay does not decide whether a refund should be approved. Supply the exact
request received from your own operational review and confirm its submission.

Example:
  gplay orders review-refund --package dev.example.app --order-id GPA.123 --json '{"pendingRefundToken":"token","refundPreference":"DECLINE","sampleContentProvided":true}' --confirm`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*orderID) == "" {
				return fmt.Errorf("--order-id is required")
			}
			if strings.TrimSpace(*jsonFlag) == "" {
				return fmt.Errorf("--json is required")
			}
			if !*confirm {
				return fmt.Errorf("--confirm is required")
			}

			var raw map[string]json.RawMessage
			if err := shared.LoadJSONArg(*jsonFlag, &raw); err != nil {
				return fmt.Errorf("invalid --json: %w", err)
			}
			encoded, err := json.Marshal(raw)
			if err != nil {
				return fmt.Errorf("invalid --json: %w", err)
			}
			var req androidpublisher.OrdersReviewRefundRequest
			if err := json.Unmarshal(encoded, &req); err != nil {
				return fmt.Errorf("invalid --json: %w", err)
			}
			if strings.TrimSpace(req.PendingRefundToken) == "" {
				return fmt.Errorf("pendingRefundToken is required")
			}
			switch req.RefundPreference {
			case "DECLINE", "APPROVE", "NEUTRAL":
			default:
				return fmt.Errorf("refundPreference must be DECLINE, APPROVE, or NEUTRAL")
			}
			if _, ok := raw["sampleContentProvided"]; !ok {
				return fmt.Errorf("sampleContentProvided is required and must be explicitly true or false")
			}
			if req.ConsumptionPercentageMilliunits < 0 || req.ConsumptionPercentageMilliunits > 100000 {
				return fmt.Errorf("consumptionPercentageMilliunits must be between 0 and 100,000")
			}
			if len(req.ConsumptionUsageEvents) > 1000 {
				return fmt.Errorf("consumptionUsageEvents supports at most 1,000 items")
			}
			req.ForceSendFields = append(req.ForceSendFields, "SampleContentProvided")

			service, err := newPlayService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()
			if err := service.API.Orders.Reviewrefund(pkg, *orderID, &req).Context(ctx).Do(); err != nil {
				return err
			}
			return shared.PrintOutput(map[string]any{
				"reviewed":         true,
				"orderId":          *orderID,
				"refundPreference": req.RefundPreference,
			}, *outputFlag, *pretty)
		},
	}
}

func GetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("orders get", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	orderID := fs.String("order-id", "", "Order ID (e.g., GPA.1234-5678-9012-34567)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "gplay orders get --package <name> --order-id <id>",
		ShortHelp:  "Get order details.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*orderID) == "" {
				return fmt.Errorf("--order-id is required")
			}
			service, err := newPlayService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.API.Orders.Get(pkg, *orderID).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func BatchGetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("orders batch-get", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	orderIDs := fs.String("order-ids", "", "Comma-separated list of order IDs")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "batch-get",
		ShortUsage: "gplay orders batch-get --package <name> --order-ids <id1,id2,...>",
		ShortHelp:  "Get multiple orders.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*orderIDs) == "" {
				return fmt.Errorf("--order-ids is required")
			}
			idList, err := parseOrderIDs(*orderIDs)
			if err != nil {
				return err
			}
			service, err := newPlayService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.API.Orders.Batchget(pkg).OrderIds(idList...).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp.Orders, *outputFlag, *pretty)
		},
	}
}

func parseOrderIDs(value string) ([]string, error) {
	seen := make(map[string]bool)
	ids := make([]string, 0)
	for _, raw := range strings.Split(value, ",") {
		id := strings.TrimSpace(raw)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("--order-ids requires at least one order ID")
	}
	if len(ids) > 1000 {
		return nil, fmt.Errorf("--order-ids supports at most 1,000 unique IDs per official batch request")
	}
	return ids, nil
}

func RefundCommand() *ffcli.Command {
	fs := flag.NewFlagSet("orders refund", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	orderID := fs.String("order-id", "", "Order ID to refund")
	revoke := fs.Bool("revoke", false, "Revoke entitlement (user loses access)")
	confirm := fs.Bool("confirm", false, "Confirm refund")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "refund",
		ShortUsage: "gplay orders refund --package <name> --order-id <id> [--revoke] --confirm",
		ShortHelp:  "Refund an order.",
		LongHelp: `Refund an order.

Options:
  --revoke: Also revoke the entitlement (user loses access to the purchased item)
            Without this flag, the user keeps access but the payment is refunded.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*orderID) == "" {
				return fmt.Errorf("--order-id is required")
			}
			if !*confirm {
				return fmt.Errorf("--confirm is required")
			}
			service, err := newPlayService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			call := service.API.Orders.Refund(pkg, *orderID).Context(ctx)
			if *revoke {
				call = call.Revoke(true)
			}
			err = call.Do()
			if err != nil {
				return err
			}

			result := map[string]interface{}{
				"refunded": true,
				"orderId":  *orderID,
				"revoked":  *revoke,
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}
