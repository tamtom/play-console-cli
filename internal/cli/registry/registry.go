package registry

import (
	"context"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/androidtools"
	"github.com/tamtom/play-console-cli/internal/cli/apks"
	"github.com/tamtom/play-console-cli/internal/cli/apps"
	"github.com/tamtom/play-console-cli/internal/cli/appsigning"
	"github.com/tamtom/play-console-cli/internal/cli/appstores"
	"github.com/tamtom/play-console-cli/internal/cli/auditcmd"
	"github.com/tamtom/play-console-cli/internal/cli/auth"
	"github.com/tamtom/play-console-cli/internal/cli/availability"
	"github.com/tamtom/play-console-cli/internal/cli/baseplans"
	"github.com/tamtom/play-console-cli/internal/cli/bootstrap"
	"github.com/tamtom/play-console-cli/internal/cli/bundles"
	"github.com/tamtom/play-console-cli/internal/cli/capabilities"
	"github.com/tamtom/play-console-cli/internal/cli/checks"
	"github.com/tamtom/play-console-cli/internal/cli/completion"
	"github.com/tamtom/play-console-cli/internal/cli/customapps"
	"github.com/tamtom/play-console-cli/internal/cli/datasafety"
	"github.com/tamtom/play-console-cli/internal/cli/deobfuscation"
	"github.com/tamtom/play-console-cli/internal/cli/details"
	"github.com/tamtom/play-console-cli/internal/cli/devicetiers"
	"github.com/tamtom/play-console-cli/internal/cli/docs"
	"github.com/tamtom/play-console-cli/internal/cli/doctor"
	"github.com/tamtom/play-console-cli/internal/cli/edits"
	"github.com/tamtom/play-console-cli/internal/cli/expansion"
	"github.com/tamtom/play-console-cli/internal/cli/experiments"
	"github.com/tamtom/play-console-cli/internal/cli/externaltx"
	"github.com/tamtom/play-console-cli/internal/cli/games"
	"github.com/tamtom/play-console-cli/internal/cli/generatedapks"
	"github.com/tamtom/play-console-cli/internal/cli/grants"
	"github.com/tamtom/play-console-cli/internal/cli/iap"
	"github.com/tamtom/play-console-cli/internal/cli/images"
	"github.com/tamtom/play-console-cli/internal/cli/initcmd"
	"github.com/tamtom/play-console-cli/internal/cli/insights"
	"github.com/tamtom/play-console-cli/internal/cli/installskills"
	"github.com/tamtom/play-console-cli/internal/cli/integrity"
	"github.com/tamtom/play-console-cli/internal/cli/internalsharing"
	"github.com/tamtom/play-console-cli/internal/cli/listings"
	"github.com/tamtom/play-console-cli/internal/cli/metadata"
	"github.com/tamtom/play-console-cli/internal/cli/migrate"
	"github.com/tamtom/play-console-cli/internal/cli/notify"
	"github.com/tamtom/play-console-cli/internal/cli/offers"
	"github.com/tamtom/play-console-cli/internal/cli/onetimeproducts"
	"github.com/tamtom/play-console-cli/internal/cli/orders"
	"github.com/tamtom/play-console-cli/internal/cli/otpoffers"
	"github.com/tamtom/play-console-cli/internal/cli/preflight"
	"github.com/tamtom/play-console-cli/internal/cli/pricing"
	"github.com/tamtom/play-console-cli/internal/cli/promote"
	"github.com/tamtom/play-console-cli/internal/cli/publish"
	"github.com/tamtom/play-console-cli/internal/cli/purchaseoptions"
	"github.com/tamtom/play-console-cli/internal/cli/purchases"
	"github.com/tamtom/play-console-cli/internal/cli/quota"
	"github.com/tamtom/play-console-cli/internal/cli/recovery"
	"github.com/tamtom/play-console-cli/internal/cli/release"
	releasenotes "github.com/tamtom/play-console-cli/internal/cli/releasenotes"
	"github.com/tamtom/play-console-cli/internal/cli/reports"
	"github.com/tamtom/play-console-cli/internal/cli/reviews"
	"github.com/tamtom/play-console-cli/internal/cli/rollout"
	rtdncmd "github.com/tamtom/play-console-cli/internal/cli/rtdn"
	cliruntime "github.com/tamtom/play-console-cli/internal/cli/runtime"
	"github.com/tamtom/play-console-cli/internal/cli/schema"
	searchcmd "github.com/tamtom/play-console-cli/internal/cli/search"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/cli/snitch"
	"github.com/tamtom/play-console-cli/internal/cli/status"
	"github.com/tamtom/play-console-cli/internal/cli/subscriptions"
	"github.com/tamtom/play-console-cli/internal/cli/sync"
	"github.com/tamtom/play-console-cli/internal/cli/systemapks"
	"github.com/tamtom/play-console-cli/internal/cli/testers"
	"github.com/tamtom/play-console-cli/internal/cli/tracks"
	"github.com/tamtom/play-console-cli/internal/cli/updatecmd"
	"github.com/tamtom/play-console-cli/internal/cli/users"
	"github.com/tamtom/play-console-cli/internal/cli/validate"
	"github.com/tamtom/play-console-cli/internal/cli/verification"
	"github.com/tamtom/play-console-cli/internal/cli/vitals"
	"github.com/tamtom/play-console-cli/internal/cli/web"
	"github.com/tamtom/play-console-cli/internal/cli/workflow"
)

// VersionCommand returns a version subcommand.
func VersionCommand(version string) *ffcli.Command {
	return &ffcli.Command{
		Name:       "version",
		ShortUsage: "gplay version",
		ShortHelp:  "Print version information and exit.",
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			fmt.Fprintln(shared.Stdout(ctx), version)
			return nil
		},
	}
}

// CommandSpec is the lightweight root-command catalog entry shared by root
// help, search, documentation, and completion. Factories remain private so
// callers cannot accidentally bypass lazy command construction.
type CommandSpec struct {
	Path      string   `json:"path"`
	Summary   string   `json:"summary"`
	Intents   []string `json:"intents,omitempty"`
	Stability string   `json:"stability"`
	Provider  string   `json:"provider"`

	newCommand func() *ffcli.Command
}

// Catalog constructs root commands only when a caller selects them.
type Catalog struct {
	specs []CommandSpec
}

// NewCatalog returns the root command metadata and lazy factories in display
// order. Adding a root command anywhere else is intentionally unnecessary.
func NewCatalog(version string, rt *cliruntime.Runtime) *Catalog {
	catalog := &Catalog{}
	catalog.specs = []CommandSpec{
		commandSpec("auth", "Manage Google Play authentication.", auth.AuthCommand),
		commandSpec("setup", "Set up Google Play authentication end-to-end.", auth.SetupCommand),
		commandSpecWith("android", "Run optional local Android build, signing, and screenshot helpers.", "local", nil, androidtools.AndroidCommand),
		commandSpec("apps", "List and manage apps accessible by the service account.", func() *ffcli.Command { return apps.AppsCommand(rt) }),
		commandSpecWith("app-signing", "Manage enterprise self-hosted Cloud KMS Play App Signing.", "android-publisher-api", []string{"app.enterprise_kms_signing"}, appsigning.AppSigningCommand),
		commandSpecWith("app-stores", "Operate official APIs for registered third-party Android app stores.", "android-publisher-api", []string{"app.third_party_store"}, appstores.AppStoresCommand),
		commandSpec("capabilities", "Show policy-aware Google Play workflow capabilities.", capabilities.CapabilitiesCommand),
		commandSpec("search", "Search commands, examples, flags, capabilities, and canonical intents.", func() *ffcli.Command {
			return searchcmd.SearchCommand(catalog.All)
		}),
		commandSpecWith("install-skills", "Install the pinned, verified gplay agent-skill pack.", "local", nil, installskills.InstallSkillsCommand),
		commandSpecWith("schema", "Inspect embedded official Google Play API endpoint and type schemas.", "local", nil, schema.SchemaCommand),
		commandSpecWith("bootstrap", "Plan policy-safe initial app setup.", "local", []string{"app.create", "app.first_artifact_upload", "app.legal_consents", "app.standard_signing_enrollment"}, bootstrap.BootstrapCommand),
		commandSpec("audit", "Query and manage the local command audit log.", auditcmd.AuditCommand),
		commandSpec("quota", "Inspect local API quota usage derived from the audit log.", quota.QuotaCommand),
		commandSpec("doctor", "Diagnose CLI setup, network, credentials, and configuration.", doctor.DoctorCommand),
		commandSpec("preflight", "Run offline compliance and hygiene checks against an AAB/APK.", preflight.PreflightCommand),
		commandSpec("rtdn", "Real-Time Developer Notifications: setup, status, decode.", rtdncmd.RtdnCommand),
		commandSpec("edits", "Manage Google Play app edits.", edits.EditsCommand),
		commandSpec("bundles", "Manage app bundles in an edit.", bundles.BundlesCommand),
		commandSpec("checks", "Analyze app privacy, compliance, and AI safety with the Google Checks API.", checks.ChecksCommand),
		commandSpec("apks", "Manage APKs in an edit.", apks.APKsCommand),
		commandSpec("tracks", "Manage release tracks in an edit.", tracks.TracksCommand),
		commandSpec("users", "Manage developer account team members.", users.UsersCommand),
		commandSpec("listings", "Manage store listings in an edit.", listings.ListingsCommand),
		commandSpecWith("metadata", "File-based metadata sync (pull/push/validate).", "android-publisher-api", []string{"app.store_listing"}, metadata.MetadataCommand),
		commandSpec("images", "Manage listing images and Play media sync.", images.ImagesCommand),
		commandSpecWith("integrity", "Decode Play Integrity tokens and manage restricted Device Recall state.", "play-integrity-api", []string{"app.play_integrity"}, integrity.IntegrityCommand),
		commandSpec("init", "Initialize a .gplay/config.yaml in the current directory.", initcmd.InitCommand),
		commandSpecWith("reviews", "Manage app reviews.", "android-publisher-api", []string{"app.reviews"}, reviews.ReviewsCommand),
		commandSpec("details", "Manage app details (contact info, default language).", details.DetailsCommand),
		commandSpec("testers", "Manage testers for closed testing tracks.", testers.TestersCommand),
		commandSpec("availability", "Check country availability for tracks.", availability.AvailabilityCommand),
		commandSpec("deobfuscation", "Manage deobfuscation files (ProGuard/R8 mapping files).", deobfuscation.DeobfuscationCommand),
		commandSpec("release", "Create a complete release: create edit, upload bundle/apk, configure track, commit.", release.ReleaseCommand),
		commandSpecWith("publish", "Canonical Google Play release workflows.", "android-publisher-api", []string{"app.release"}, publish.PublishCommand),
		commandSpec("promote", "Promote a release from one track to another.", promote.PromoteCommand),
		commandSpec("rollout", "Manage staged rollouts.", rollout.RolloutCommand),
		commandSpec("sync", "Sync metadata between local directory and Play Store.", sync.SyncCommand),
		commandSpecWith("experiments", "Inspect listing-experiment API support and apply a manually selected winner.", "mixed", []string{"app.store_listing_experiments"}, experiments.ExperimentsCommand),
		commandSpec("validate", "Canonical Google Play release-readiness report.", validate.ValidateCommand),
		commandSpecWith("verification", "Check official Android developer package-registration status.", "android-developer-id-status-api", []string{"app.developer_id_status"}, verification.VerificationCommand),
		commandSpec("status", "Show a deterministic release-health snapshot.", status.StatusCommand),
		commandSpecWith("vitals", "App vitals: crashes, performance, and error reports.", "play-developer-reporting-api", []string{"app.vitals", "app.reporting_metric_sets"}, vitals.VitalsCommand),
		commandSpecWith("insights", "Compare trends from official Google Play report exports.", "local-official-reports", []string{"app.insights"}, insights.InsightsCommand),
		commandSpec("iap", "Manage in-app products (managed products).", iap.IAPCommand),
		commandSpec("subscriptions", "Manage subscription products.", subscriptions.SubscriptionsCommand),
		commandSpec("baseplans", "Manage subscription base plans.", baseplans.BasePlansCommand),
		commandSpec("offers", "Manage subscription offers (trials, introductory prices).", offers.OffersCommand),
		commandSpec("onetimeproducts", "Manage one-time products (monetization).", onetimeproducts.OneTimeProductsCommand),
		commandSpec("purchase-options", "Manage one-time product purchase options.", purchaseoptions.PurchaseOptionsCommand),
		commandSpec("otp-offers", "Manage one-time product purchase option offers.", otpoffers.OTPOffersCommand),
		commandSpec("pricing", "Pricing conversion and regions-version discovery.", pricing.PricingCommand),
		commandSpec("orders", "Manage orders.", orders.OrdersCommand),
		commandSpec("purchases", "Verify and manage purchases.", purchases.PurchasesCommand),
		commandSpec("external-transactions", "Report external transactions (EU compliance).", externaltx.ExternalTxCommand),
		commandSpec("games", "Manage Play Games Services achievements, leaderboards, and player progress.", games.GamesCommand),
		commandSpec("custom-apps", "Publish private apps via Managed Google Play (Custom App Publishing API).", customapps.CustomAppsCommand),
		commandSpec("generated-apks", "Download device-specific APKs generated from app bundles.", generatedapks.GeneratedAPKsCommand),
		commandSpec("grants", "Manage per-app permission grants.", grants.GrantsCommand),
		commandSpec("internal-sharing", "Quick internal testing without review.", internalsharing.InternalSharingCommand),
		commandSpec("system-apks", "Create APKs for system image inclusion.", systemapks.SystemAPKsCommand),
		commandSpec("expansion", "Manage expansion files (OBB files).", expansion.ExpansionCommand),
		commandSpec("recovery", "Manage app recovery actions.", recovery.RecoveryCommand),
		commandSpec("data-safety", "Manage data safety declarations.", datasafety.DataSafetyCommand),
		commandSpec("device-tiers", "Manage device tier configurations.", devicetiers.DeviceTiersCommand),
		commandSpec("notify", "Send notifications to Slack, Discord, or HTTP webhooks.", notify.NotifyCommand),
		commandSpec("snitch", "Report CLI friction as a GitHub issue.", func() *ffcli.Command { return snitch.SnitchCommand(version) }),
		commandSpec("migrate", "Migrate metadata from other tools.", migrate.MigrateCommand),
		commandSpec("release-notes", "Generate release notes from git history.", releasenotes.ReleaseNotesCommand),
		commandSpec("reports", "Download and manage Play Console reports.", reports.ReportsCommand),
		commandSpec("workflow", "Run multi-step automation workflows.", workflow.WorkflowCommand),
		commandSpec("docs", "Documentation and help topics.", func() *ffcli.Command {
			return docs.DocsCommandWithRoot(func() *ffcli.Command {
				return &ffcli.Command{
					Name:        "gplay",
					ShortUsage:  "gplay <command> [flags]",
					ShortHelp:   "A CLI for Google Play Console.",
					UsageFunc:   shared.DefaultUsageFunc,
					Subcommands: catalog.All(),
				}
			})
		}),
		commandSpec("web", "Open Google Play Console pages in the browser.", web.WebCommand),
		commandSpec("update", "Update gplay to the latest version.", updatecmd.UpdateCommand),
		commandSpec("completion", "Generate shell completion scripts.", func() *ffcli.Command {
			return completion.CompletionCommandWithCatalog(catalog.All)
		}),
		commandSpec("version", "Print version information and exit.", func() *ffcli.Command { return VersionCommand(version) }),
	}
	return catalog
}

func commandSpec(name, summary string, factory func() *ffcli.Command) CommandSpec {
	return commandSpecWith(name, summary, "mixed", nil, factory)
}

func commandSpecWith(name, summary, provider string, intents []string, factory func() *ffcli.Command) CommandSpec {
	return CommandSpec{
		Path:       "gplay " + name,
		Summary:    summary,
		Intents:    append([]string(nil), intents...),
		Stability:  "stable",
		Provider:   provider,
		newCommand: factory,
	}
}

// Specs returns a defensive copy of the lightweight discovery metadata.
func (c *Catalog) Specs() []CommandSpec {
	if c == nil {
		return nil
	}
	result := make([]CommandSpec, len(c.specs))
	copy(result, c.specs)
	for i := range result {
		result[i].Intents = append([]string(nil), result[i].Intents...)
		result[i].newCommand = nil
	}
	return result
}

// MetadataCommands returns lightweight root entries without invoking a command
// factory. Root help therefore remains complete without paying construction
// cost for every command family.
func (c *Catalog) MetadataCommands() []*ffcli.Command {
	if c == nil {
		return nil
	}
	commands := make([]*ffcli.Command, 0, len(c.specs))
	for _, spec := range c.specs {
		commands = append(commands, &ffcli.Command{
			Name:      strings.TrimPrefix(spec.Path, "gplay "),
			ShortHelp: spec.Summary,
			UsageFunc: shared.DefaultUsageFunc,
		})
	}
	return commands
}

// CommandsFor returns complete root metadata with only the named command
// materialized.
func (c *Catalog) CommandsFor(name string) []*ffcli.Command {
	commands := c.MetadataCommands()
	name = strings.TrimSpace(name)
	for i, spec := range c.specs {
		if strings.EqualFold(strings.TrimPrefix(spec.Path, "gplay "), name) {
			commands[i] = materialize(spec)
			break
		}
	}
	return commands
}

// All materializes the complete command tree for explicit full-tree consumers
// such as search, generated docs, tests, and completion generation.
func (c *Catalog) All() []*ffcli.Command {
	if c == nil {
		return nil
	}
	commands := make([]*ffcli.Command, 0, len(c.specs))
	for _, spec := range c.specs {
		commands = append(commands, materialize(spec))
	}
	return commands
}

func materialize(spec CommandSpec) *ffcli.Command {
	if spec.newCommand == nil {
		return &ffcli.Command{
			Name:      strings.TrimPrefix(spec.Path, "gplay "),
			ShortHelp: spec.Summary,
			UsageFunc: shared.DefaultUsageFunc,
		}
	}
	return spec.newCommand()
}

// Subcommands returns all root subcommands in display order.
func Subcommands(version string) []*ffcli.Command {
	return NewCatalog(version, nil).All()
}

// SubcommandsWithRuntime returns all root subcommands in display order using the
// provided runtime for migrated command families.
func SubcommandsWithRuntime(version string, rt *cliruntime.Runtime) []*ffcli.Command {
	return NewCatalog(version, rt).All()
}
