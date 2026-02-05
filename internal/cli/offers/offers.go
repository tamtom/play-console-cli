package offers

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

func OffersCommand() *ffcli.Command {
	fs := flag.NewFlagSet("offers", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "offers",
		ShortUsage: "gplay offers <subcommand> [flags]",
		ShortHelp:  "Manage subscription offers (trials, introductory prices).",
		LongHelp: `Manage subscription offers within base plans.

Offers include:
  - Free trials
  - Introductory prices
  - Developer-determined offers

Offers are attached to base plans and provide promotional
pricing for new subscribers.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			ListCommand(),
			GetCommand(),
			CreateCommand(),
			UpdateCommand(),
			ActivateCommand(),
			DeactivateCommand(),
			DeleteCommand(),
			BatchGetCommand(),
			BatchUpdateCommand(),
			BatchUpdateStatesCommand(),
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
	fs := flag.NewFlagSet("offers list", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Subscription product ID")
	basePlanID := fs.String("base-plan-id", "", "Base plan ID")
	pageSize := fs.Int("page-size", 100, "Page size")
	paginate := fs.Bool("paginate", false, "Fetch all pages")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay offers list --package <name> --product-id <id> --base-plan-id <plan>",
		ShortHelp:  "List all offers for a base plan.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
			}
			if strings.TrimSpace(*basePlanID) == "" {
				return fmt.Errorf("--base-plan-id is required")
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

			var all []*androidpublisher.SubscriptionOffer
			pageToken := ""
			for {
				call := service.API.Monetization.Subscriptions.BasePlans.Offers.List(pkg, *productID, *basePlanID).Context(ctx).PageSize(int64(*pageSize))
				if pageToken != "" {
					call.PageToken(pageToken)
				}
				resp, err := call.Do()
				if err != nil {
					return err
				}
				if !*paginate {
					return shared.PrintOutput(resp, *outputFlag, *pretty)
				}
				all = append(all, resp.SubscriptionOffers...)
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
	fs := flag.NewFlagSet("offers get", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Subscription product ID")
	basePlanID := fs.String("base-plan-id", "", "Base plan ID")
	offerID := fs.String("offer-id", "", "Offer ID")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "gplay offers get --package <name> --product-id <id> --base-plan-id <plan> --offer-id <offer>",
		ShortHelp:  "Get an offer.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
			}
			if strings.TrimSpace(*basePlanID) == "" {
				return fmt.Errorf("--base-plan-id is required")
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

			resp, err := service.API.Monetization.Subscriptions.BasePlans.Offers.Get(pkg, *productID, *basePlanID, *offerID).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func CreateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("offers create", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Subscription product ID")
	basePlanID := fs.String("base-plan-id", "", "Base plan ID")
	offerID := fs.String("offer-id", "", "Offer ID")
	jsonFlag := fs.String("json", "", "SubscriptionOffer JSON (or @file)")
	regionsVersion := fs.String("regions-version", "", "Regions version")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "create",
		ShortUsage: "gplay offers create --package <name> --product-id <id> --base-plan-id <plan> --offer-id <offer> --json <json>",
		ShortHelp:  "Create an offer.",
		LongHelp: `Create a new subscription offer.

JSON format for a free trial:
{
  "phases": [
    {
      "recurrenceCount": 1,
      "duration": "P7D",
      "regionConfigs": [
        {
          "regionCode": "US",
          "freeTrialPhase": {}
        }
      ]
    }
  ],
  "targeting": {
    "acquisitionRule": {
      "scope": {
        "anySubscriptionInApp": {}
      }
    }
  },
  "offerTags": [
    {"tag": "trial"}
  ]
}

JSON format for introductory price:
{
  "phases": [
    {
      "recurrenceCount": 3,
      "duration": "P1M",
      "regionConfigs": [
        {
          "regionCode": "US",
          "price": {
            "currencyCode": "USD",
            "units": "4",
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
			if strings.TrimSpace(*basePlanID) == "" {
				return fmt.Errorf("--base-plan-id is required")
			}
			if strings.TrimSpace(*offerID) == "" {
				return fmt.Errorf("--offer-id is required")
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

			var offer androidpublisher.SubscriptionOffer
			if err := shared.LoadJSONArg(*jsonFlag, &offer); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			offer.PackageName = pkg
			offer.ProductId = *productID
			offer.BasePlanId = *basePlanID
			offer.OfferId = *offerID

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			call := service.API.Monetization.Subscriptions.BasePlans.Offers.Create(pkg, *productID, *basePlanID, &offer).Context(ctx).OfferId(*offerID)
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
	fs := flag.NewFlagSet("offers update", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Subscription product ID")
	basePlanID := fs.String("base-plan-id", "", "Base plan ID")
	offerID := fs.String("offer-id", "", "Offer ID")
	jsonFlag := fs.String("json", "", "SubscriptionOffer JSON (or @file)")
	updateMask := fs.String("update-mask", "", "Fields to update (comma-separated)")
	regionsVersion := fs.String("regions-version", "", "Regions version")
	allowMissing := fs.Bool("allow-missing", false, "Create if not exists")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "update",
		ShortUsage: "gplay offers update --package <name> --product-id <id> --base-plan-id <plan> --offer-id <offer> --json <json>",
		ShortHelp:  "Update an offer.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
			}
			if strings.TrimSpace(*basePlanID) == "" {
				return fmt.Errorf("--base-plan-id is required")
			}
			if strings.TrimSpace(*offerID) == "" {
				return fmt.Errorf("--offer-id is required")
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

			var offer androidpublisher.SubscriptionOffer
			if err := shared.LoadJSONArg(*jsonFlag, &offer); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			offer.PackageName = pkg
			offer.ProductId = *productID
			offer.BasePlanId = *basePlanID
			offer.OfferId = *offerID

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			call := service.API.Monetization.Subscriptions.BasePlans.Offers.Patch(pkg, *productID, *basePlanID, *offerID, &offer).Context(ctx)
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

func ActivateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("offers activate", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Subscription product ID")
	basePlanID := fs.String("base-plan-id", "", "Base plan ID")
	offerID := fs.String("offer-id", "", "Offer ID")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "activate",
		ShortUsage: "gplay offers activate --package <name> --product-id <id> --base-plan-id <plan> --offer-id <offer>",
		ShortHelp:  "Activate an offer.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
			}
			if strings.TrimSpace(*basePlanID) == "" {
				return fmt.Errorf("--base-plan-id is required")
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

			req := &androidpublisher.ActivateSubscriptionOfferRequest{}
			resp, err := service.API.Monetization.Subscriptions.BasePlans.Offers.Activate(pkg, *productID, *basePlanID, *offerID, req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func DeactivateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("offers deactivate", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Subscription product ID")
	basePlanID := fs.String("base-plan-id", "", "Base plan ID")
	offerID := fs.String("offer-id", "", "Offer ID")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "deactivate",
		ShortUsage: "gplay offers deactivate --package <name> --product-id <id> --base-plan-id <plan> --offer-id <offer>",
		ShortHelp:  "Deactivate an offer.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
			}
			if strings.TrimSpace(*basePlanID) == "" {
				return fmt.Errorf("--base-plan-id is required")
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

			req := &androidpublisher.DeactivateSubscriptionOfferRequest{}
			resp, err := service.API.Monetization.Subscriptions.BasePlans.Offers.Deactivate(pkg, *productID, *basePlanID, *offerID, req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func DeleteCommand() *ffcli.Command {
	fs := flag.NewFlagSet("offers delete", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Subscription product ID")
	basePlanID := fs.String("base-plan-id", "", "Base plan ID")
	offerID := fs.String("offer-id", "", "Offer ID")
	confirm := fs.Bool("confirm", false, "Confirm deletion")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "delete",
		ShortUsage: "gplay offers delete --package <name> --product-id <id> --base-plan-id <plan> --offer-id <offer> --confirm",
		ShortHelp:  "Delete an offer.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
			}
			if strings.TrimSpace(*basePlanID) == "" {
				return fmt.Errorf("--base-plan-id is required")
			}
			if strings.TrimSpace(*offerID) == "" {
				return fmt.Errorf("--offer-id is required")
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

			err = service.API.Monetization.Subscriptions.BasePlans.Offers.Delete(pkg, *productID, *basePlanID, *offerID).Context(ctx).Do()
			if err != nil {
				return err
			}

			result := map[string]interface{}{
				"deleted":    true,
				"productId":  *productID,
				"basePlanId": *basePlanID,
				"offerId":    *offerID,
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}

func BatchGetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("offers batch-get", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Subscription product ID")
	basePlanID := fs.String("base-plan-id", "", "Base plan ID")
	offerIDs := fs.String("offer-ids", "", "Comma-separated list of offer IDs")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "batch-get",
		ShortUsage: "gplay offers batch-get --package <name> --product-id <id> --base-plan-id <plan> --offer-ids <id1,id2>",
		ShortHelp:  "Get multiple offers.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
			}
			if strings.TrimSpace(*basePlanID) == "" {
				return fmt.Errorf("--base-plan-id is required")
			}
			if strings.TrimSpace(*offerIDs) == "" {
				return fmt.Errorf("--offer-ids is required")
			}
			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			idList := strings.Split(*offerIDs, ",")
			for i := range idList {
				idList[i] = strings.TrimSpace(idList[i])
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			req := &androidpublisher.BatchGetSubscriptionOffersRequest{
				Requests: make([]*androidpublisher.GetSubscriptionOfferRequest, 0, len(idList)),
			}
			for _, id := range idList {
				req.Requests = append(req.Requests, &androidpublisher.GetSubscriptionOfferRequest{
					PackageName: pkg,
					ProductId:   *productID,
					BasePlanId:  *basePlanID,
					OfferId:     id,
				})
			}

			resp, err := service.API.Monetization.Subscriptions.BasePlans.Offers.BatchGet(pkg, *productID, *basePlanID, req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func BatchUpdateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("offers batch-update", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Subscription product ID")
	basePlanID := fs.String("base-plan-id", "", "Base plan ID")
	jsonFlag := fs.String("json", "", "Batch update request JSON (or @file)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "batch-update",
		ShortUsage: "gplay offers batch-update --package <name> --product-id <id> --base-plan-id <plan> --json <json>",
		ShortHelp:  "Batch update multiple offers.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
			}
			if strings.TrimSpace(*basePlanID) == "" {
				return fmt.Errorf("--base-plan-id is required")
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

			var req androidpublisher.BatchUpdateSubscriptionOffersRequest
			if err := shared.LoadJSONArg(*jsonFlag, &req); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.API.Monetization.Subscriptions.BasePlans.Offers.BatchUpdate(pkg, *productID, *basePlanID, &req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func BatchUpdateStatesCommand() *ffcli.Command {
	fs := flag.NewFlagSet("offers batch-update-states", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	productID := fs.String("product-id", "", "Subscription product ID")
	basePlanID := fs.String("base-plan-id", "", "Base plan ID")
	jsonFlag := fs.String("json", "", "Batch update states request JSON (or @file)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "batch-update-states",
		ShortUsage: "gplay offers batch-update-states --package <name> --product-id <id> --base-plan-id <plan> --json <json>",
		ShortHelp:  "Batch activate/deactivate multiple offers.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*productID) == "" {
				return fmt.Errorf("--product-id is required")
			}
			if strings.TrimSpace(*basePlanID) == "" {
				return fmt.Errorf("--base-plan-id is required")
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

			var req androidpublisher.BatchUpdateSubscriptionOfferStatesRequest
			if err := shared.LoadJSONArg(*jsonFlag, &req); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.API.Monetization.Subscriptions.BasePlans.Offers.BatchUpdateStates(pkg, *productID, *basePlanID, &req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}
