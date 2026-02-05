package externaltx

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

func ExternalTxCommand() *ffcli.Command {
	fs := flag.NewFlagSet("external-transactions", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "external-transactions",
		ShortUsage: "gplay external-transactions <subcommand> [flags]",
		ShortHelp:  "Report external transactions (EU compliance).",
		LongHelp: `Report external transactions for EU Digital Markets Act compliance.

This is used to report transactions that occurred outside of Google Play
(e.g., via your own website payment system) for apps distributed in the EU.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			CreateCommand(),
			GetCommand(),
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

func CreateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("external-transactions create", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	externalTxID := fs.String("external-transaction-id", "", "External transaction ID (your system's ID)")
	jsonFlag := fs.String("json", "", "ExternalTransaction JSON (or @file)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "create",
		ShortUsage: "gplay external-transactions create --package <name> --external-transaction-id <id> --json <json>",
		ShortHelp:  "Report a new external transaction.",
		LongHelp: `Report a new external transaction.

JSON format:
{
  "originalPreTaxAmount": {
    "priceMicros": "9990000",
    "currency": "EUR"
  },
  "originalTaxAmount": {
    "priceMicros": "1990000",
    "currency": "EUR"
  },
  "currentPreTaxAmount": {
    "priceMicros": "9990000",
    "currency": "EUR"
  },
  "currentTaxAmount": {
    "priceMicros": "1990000",
    "currency": "EUR"
  },
  "transactionTime": "2024-01-15T10:30:00Z",
  "oneTimeTransaction": {
    "externalTransactionToken": "your-token-123"
  },
  "userTaxAddress": {
    "regionCode": "DE"
  }
}

For recurring subscriptions, use "recurringTransaction" instead of "oneTimeTransaction".`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*externalTxID) == "" {
				return fmt.Errorf("--external-transaction-id is required")
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

			var tx androidpublisher.ExternalTransaction
			if err := shared.LoadJSONArg(*jsonFlag, &tx); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			tx.PackageName = pkg
			tx.ExternalTransactionId = *externalTxID

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.API.Externaltransactions.Createexternaltransaction(pkg, &tx).Context(ctx).ExternalTransactionId(*externalTxID).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func GetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("external-transactions get", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	externalTxID := fs.String("external-transaction-id", "", "External transaction ID")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "gplay external-transactions get --package <name> --external-transaction-id <id>",
		ShortHelp:  "Get external transaction details.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*externalTxID) == "" {
				return fmt.Errorf("--external-transaction-id is required")
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

			// The resource name format is: applications/{packageName}/externalTransactions/{externalTransactionId}
			name := fmt.Sprintf("applications/%s/externalTransactions/%s", pkg, *externalTxID)
			resp, err := service.API.Externaltransactions.Getexternaltransaction(name).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func RefundCommand() *ffcli.Command {
	fs := flag.NewFlagSet("external-transactions refund", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	externalTxID := fs.String("external-transaction-id", "", "External transaction ID")
	jsonFlag := fs.String("json", "", "Refund JSON (or @file)")
	confirm := fs.Bool("confirm", false, "Confirm refund")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "refund",
		ShortUsage: "gplay external-transactions refund --package <name> --external-transaction-id <id> --json <json> --confirm",
		ShortHelp:  "Report a refund for an external transaction.",
		LongHelp: `Report a refund for an external transaction.

JSON format for full refund:
{
  "refundTime": "2024-01-20T10:30:00Z",
  "fullRefund": {}
}

JSON format for partial refund:
{
  "refundTime": "2024-01-20T10:30:00Z",
  "partialRefund": {
    "refundPreTaxAmount": {
      "priceMicros": "4990000",
      "currency": "EUR"
    }
  }
}`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*externalTxID) == "" {
				return fmt.Errorf("--external-transaction-id is required")
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

			var req androidpublisher.RefundExternalTransactionRequest
			if err := shared.LoadJSONArg(*jsonFlag, &req); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			name := fmt.Sprintf("applications/%s/externalTransactions/%s", pkg, *externalTxID)
			resp, err := service.API.Externaltransactions.Refundexternaltransaction(name, &req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}
