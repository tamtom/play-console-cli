package registry

import (
	"context"
	"fmt"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/apks"
	"github.com/tamtom/play-console-cli/internal/cli/apps"
	"github.com/tamtom/play-console-cli/internal/cli/auth"
	"github.com/tamtom/play-console-cli/internal/cli/availability"
	"github.com/tamtom/play-console-cli/internal/cli/baseplans"
	"github.com/tamtom/play-console-cli/internal/cli/bundles"
	"github.com/tamtom/play-console-cli/internal/cli/completion"
	"github.com/tamtom/play-console-cli/internal/cli/datasafety"
	"github.com/tamtom/play-console-cli/internal/cli/deobfuscation"
	"github.com/tamtom/play-console-cli/internal/cli/details"
	"github.com/tamtom/play-console-cli/internal/cli/devicetiers"
	"github.com/tamtom/play-console-cli/internal/cli/docs"
	"github.com/tamtom/play-console-cli/internal/cli/edits"
	"github.com/tamtom/play-console-cli/internal/cli/expansion"
	"github.com/tamtom/play-console-cli/internal/cli/externaltx"
	"github.com/tamtom/play-console-cli/internal/cli/generatedapks"
	// "github.com/tamtom/play-console-cli/internal/cli/grants" // Grants API methods not fully available
	"github.com/tamtom/play-console-cli/internal/cli/iap"
	"github.com/tamtom/play-console-cli/internal/cli/images"
	"github.com/tamtom/play-console-cli/internal/cli/initcmd"
	"github.com/tamtom/play-console-cli/internal/cli/internalsharing"
	"github.com/tamtom/play-console-cli/internal/cli/listings"
	"github.com/tamtom/play-console-cli/internal/cli/offers"
	"github.com/tamtom/play-console-cli/internal/cli/onetimeproducts"
	"github.com/tamtom/play-console-cli/internal/cli/orders"
	"github.com/tamtom/play-console-cli/internal/cli/pricing"
	"github.com/tamtom/play-console-cli/internal/cli/promote"
	"github.com/tamtom/play-console-cli/internal/cli/purchases"
	"github.com/tamtom/play-console-cli/internal/cli/recovery"
	"github.com/tamtom/play-console-cli/internal/cli/release"
	"github.com/tamtom/play-console-cli/internal/cli/reviews"
	"github.com/tamtom/play-console-cli/internal/cli/rollout"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/cli/subscriptions"
	"github.com/tamtom/play-console-cli/internal/cli/sync"
	"github.com/tamtom/play-console-cli/internal/cli/systemapks"
	"github.com/tamtom/play-console-cli/internal/cli/testers"
	"github.com/tamtom/play-console-cli/internal/cli/tracks"
	// "github.com/tamtom/play-console-cli/internal/cli/users" // Users API methods not fully available
	"github.com/tamtom/play-console-cli/internal/cli/validate"
)

// VersionCommand returns a version subcommand.
func VersionCommand(version string) *ffcli.Command {
	return &ffcli.Command{
		Name:       "version",
		ShortUsage: "gplay version",
		ShortHelp:  "Print version information and exit.",
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			fmt.Println(version)
			return nil
		},
	}
}

// Subcommands returns all root subcommands in display order.
func Subcommands(version string) []*ffcli.Command {
	return []*ffcli.Command{
		auth.AuthCommand(),
		apps.AppsCommand(),
		edits.EditsCommand(),
		bundles.BundlesCommand(),
		apks.APKsCommand(),
		tracks.TracksCommand(),
		listings.ListingsCommand(),
		images.ImagesCommand(),
		reviews.ReviewsCommand(),
		details.DetailsCommand(),
		testers.TestersCommand(),
		availability.AvailabilityCommand(),
		deobfuscation.DeobfuscationCommand(),
		release.ReleaseCommand(),
		promote.PromoteCommand(),
		rollout.RolloutCommand(),
		sync.SyncCommand(),
		validate.ValidateCommand(),
		iap.IAPCommand(),
		subscriptions.SubscriptionsCommand(),
		baseplans.BasePlansCommand(),
		offers.OffersCommand(),
		onetimeproducts.OneTimeProductsCommand(),
		pricing.PricingCommand(),
		orders.OrdersCommand(),
		purchases.PurchasesCommand(),
		externaltx.ExternalTxCommand(),
		generatedapks.GeneratedAPKsCommand(),
		internalsharing.InternalSharingCommand(),
		systemapks.SystemAPKsCommand(),
		expansion.ExpansionCommand(),
		recovery.RecoveryCommand(),
		datasafety.DataSafetyCommand(),
		devicetiers.DeviceTiersCommand(),
		docs.DocsCommand(),
		initcmd.InitCommand(),
		completion.CompletionCommand(),
		VersionCommand(version),
	}
}
