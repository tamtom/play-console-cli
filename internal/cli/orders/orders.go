package orders

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
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			return flag.ErrHelp
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
			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			idList := strings.Split(*orderIDs, ",")
			for i := range idList {
				idList[i] = strings.TrimSpace(idList[i])
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			req := &androidpublisher.OrdersV2BatchGetRequest{
				OrderIds: idList,
			}

			resp, err := service.API.Orders.BatchGet(pkg, req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
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
