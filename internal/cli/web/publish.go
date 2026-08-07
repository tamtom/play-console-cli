package web

import (
	"context"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/output"
	"github.com/tamtom/play-console-cli/internal/webdriver"
	"github.com/tamtom/play-console-cli/internal/websession"
)

// publishBrowser drives the console pages behind the publish-gate commands.
// It exists so the commands can be tested without a real browser.
type publishBrowser interface {
	ReadOverview(ctx context.Context, developerID, appID, account string) (*webdriver.PublishingOverview, error)
	ReadSetup(ctx context.Context, developerID, appID, account string) (*webdriver.AppSetup, error)
	SendForReview(ctx context.Context, timeout time.Duration) error
	OpenCountries(ctx context.Context, developerID, appID, account, track string) error
	ReadCountries(ctx context.Context) (*webdriver.CountriesState, error)
	SetCountries(ctx context.Context, names []string) error
	UnsetCountries(ctx context.Context, names []string) error
	SetAllCountries(ctx context.Context) error
	SubmitCountries(ctx context.Context, timeout time.Duration) error
	DiscardCountries(ctx context.Context) error
	OpenPricing(ctx context.Context, developerID, appID, account string) error
	ReadPricing(ctx context.Context) (*webdriver.AppPricingState, error)
	SetPrice(ctx context.Context, price string) error
	SubmitPricing(ctx context.Context, timeout time.Duration) error
	DiscardPricing(ctx context.Context) error
	OpenProductionReleases(ctx context.Context, developerID, appID, account string) error
	OpenDraftRelease(ctx context.Context) (*webdriver.PrepareState, error)
	ReviewDraftRelease(ctx context.Context) (*webdriver.ReviewState, error)
	SaveReleaseReview(ctx context.Context, timeout time.Duration) error
	ReadDeclarations(ctx context.Context, developerID, appID, account string) (*webdriver.DeclarationsState, error)
	SetPrivacyPolicyURL(ctx context.Context, developerID, appID, account, url string) (bool, error)
	SetRadioDeclaration(ctx context.Context, developerID, appID, account, page string, yes bool) (bool, error)
	OpenQuestionnaire(ctx context.Context, developerID, appID, account, page string) error
	ReadQuestionnaireStep(ctx context.Context) (*webdriver.QuestionnaireStep, error)
	SetStepChoices(ctx context.Context, ids []string) error
	QuestionnaireNext(ctx context.Context) (bool, error)
	QuestionnaireSave(ctx context.Context, timeout time.Duration) error
	QuestionnaireDiscard(ctx context.Context) error
	ReadDistribution(ctx context.Context, developerID, appID, account string) (*webdriver.DistributionState, error)
	AddFormFactor(ctx context.Context, factor string) error
	ImportDataSafetyCSV(ctx context.Context, developerID, appID, account, filePath string) error
	ReadPolicyStatus(ctx context.Context, developerID, appID, account string) (*webdriver.PolicyStatus, error)
	SetManagedPublishing(ctx context.Context, developerID, appID, account string, on bool) error
	PublishApprovedChanges(ctx context.Context, developerID, appID, account string, timeout time.Duration) error
	Close() error
}

type browserPublish struct{ b *webdriver.Browser }

func (p browserPublish) ReadOverview(ctx context.Context, developerID, appID, account string) (*webdriver.PublishingOverview, error) {
	return webdriver.ReadPublishingOverview(ctx, p.b, developerID, appID, account)
}

func (p browserPublish) ReadSetup(ctx context.Context, developerID, appID, account string) (*webdriver.AppSetup, error) {
	return webdriver.ReadAppSetup(ctx, p.b, developerID, appID, account)
}

func (p browserPublish) SendForReview(ctx context.Context, timeout time.Duration) error {
	return webdriver.SendForReview(ctx, p.b, timeout)
}

func (p browserPublish) OpenCountries(ctx context.Context, developerID, appID, account, track string) error {
	return webdriver.OpenCountriesEditor(ctx, p.b, developerID, appID, account, track)
}

func (p browserPublish) ReadCountries(ctx context.Context) (*webdriver.CountriesState, error) {
	return webdriver.ReadCountries(ctx, p.b)
}

func (p browserPublish) SetCountries(ctx context.Context, names []string) error {
	return webdriver.SetCountries(ctx, p.b, names)
}

func (p browserPublish) UnsetCountries(ctx context.Context, names []string) error {
	return webdriver.UnsetCountries(ctx, p.b, names)
}

func (p browserPublish) SetAllCountries(ctx context.Context) error {
	return webdriver.SetAllCountries(ctx, p.b)
}

func (p browserPublish) SubmitCountries(ctx context.Context, timeout time.Duration) error {
	return webdriver.SubmitCountries(ctx, p.b, timeout)
}

func (p browserPublish) DiscardCountries(ctx context.Context) error {
	return webdriver.DiscardCountries(ctx, p.b)
}

func (p browserPublish) OpenPricing(ctx context.Context, developerID, appID, account string) error {
	return webdriver.OpenAppPricing(ctx, p.b, developerID, appID, account)
}

func (p browserPublish) ReadPricing(ctx context.Context) (*webdriver.AppPricingState, error) {
	return webdriver.ReadAppPricing(ctx, p.b)
}

func (p browserPublish) SetPrice(ctx context.Context, price string) error {
	return webdriver.SetAppPrice(ctx, p.b, price)
}

func (p browserPublish) SubmitPricing(ctx context.Context, timeout time.Duration) error {
	return webdriver.SubmitAppPricing(ctx, p.b, timeout)
}

func (p browserPublish) DiscardPricing(ctx context.Context) error {
	return webdriver.DiscardAppPricing(ctx, p.b)
}

func (p browserPublish) OpenProductionReleases(ctx context.Context, developerID, appID, account string) error {
	return webdriver.OpenProductionReleases(ctx, p.b, developerID, appID, account)
}

func (p browserPublish) OpenDraftRelease(ctx context.Context) (*webdriver.PrepareState, error) {
	return webdriver.OpenDraftRelease(ctx, p.b)
}

func (p browserPublish) ReviewDraftRelease(ctx context.Context) (*webdriver.ReviewState, error) {
	return webdriver.ReviewDraftRelease(ctx, p.b)
}

func (p browserPublish) SaveReleaseReview(ctx context.Context, timeout time.Duration) error {
	return webdriver.SaveReleaseReview(ctx, p.b, timeout)
}

func (p browserPublish) ReadDeclarations(ctx context.Context, developerID, appID, account string) (*webdriver.DeclarationsState, error) {
	return webdriver.ReadDeclarations(ctx, p.b, developerID, appID, account)
}

func (p browserPublish) SetPrivacyPolicyURL(ctx context.Context, developerID, appID, account, url string) (bool, error) {
	return webdriver.SetPrivacyPolicyURL(ctx, p.b, developerID, appID, account, url)
}

func (p browserPublish) SetRadioDeclaration(ctx context.Context, developerID, appID, account, page string, yes bool) (bool, error) {
	return webdriver.SetRadioDeclaration(ctx, p.b, developerID, appID, account, page, yes)
}

func (p browserPublish) OpenQuestionnaire(ctx context.Context, developerID, appID, account, page string) error {
	return webdriver.OpenQuestionnaire(ctx, p.b, developerID, appID, account, page)
}

func (p browserPublish) ReadQuestionnaireStep(ctx context.Context) (*webdriver.QuestionnaireStep, error) {
	return webdriver.ReadQuestionnaireStep(ctx, p.b)
}

func (p browserPublish) SetStepChoices(ctx context.Context, ids []string) error {
	return webdriver.SetStepChoices(ctx, p.b, ids)
}

func (p browserPublish) QuestionnaireNext(ctx context.Context) (bool, error) {
	return webdriver.QuestionnaireNext(ctx, p.b)
}

func (p browserPublish) QuestionnaireSave(ctx context.Context, timeout time.Duration) error {
	return webdriver.QuestionnaireSave(ctx, p.b, timeout)
}

func (p browserPublish) QuestionnaireDiscard(ctx context.Context) error {
	return webdriver.QuestionnaireDiscard(ctx, p.b)
}

func (p browserPublish) ReadDistribution(ctx context.Context, developerID, appID, account string) (*webdriver.DistributionState, error) {
	return webdriver.ReadDistribution(ctx, p.b, developerID, appID, account)
}

func (p browserPublish) AddFormFactor(ctx context.Context, factor string) error {
	return webdriver.AddFormFactor(ctx, p.b, factor)
}

func (p browserPublish) ImportDataSafetyCSV(ctx context.Context, developerID, appID, account, filePath string) error {
	return webdriver.ImportDataSafetyCSV(ctx, p.b, developerID, appID, account, filePath)
}

func (p browserPublish) ReadPolicyStatus(ctx context.Context, developerID, appID, account string) (*webdriver.PolicyStatus, error) {
	return webdriver.ReadPolicyStatus(ctx, p.b, developerID, appID, account)
}

func (p browserPublish) SetManagedPublishing(ctx context.Context, developerID, appID, account string, on bool) error {
	return webdriver.SetManagedPublishing(ctx, p.b, developerID, appID, account, on)
}

func (p browserPublish) PublishApprovedChanges(ctx context.Context, developerID, appID, account string, timeout time.Duration) error {
	return webdriver.PublishApprovedChanges(ctx, p.b, developerID, appID, account, timeout)
}
func (p browserPublish) Close() error { return p.b.Close() }

// newPublishBrowser connects to the gplay-managed Chrome profile, starting it
// if needed. Overridden in tests.
var newPublishBrowser = func(ctx context.Context, userDataDir string, timeout time.Duration) (publishBrowser, error) {
	b, err := connectAppBrowser(ctx, userDataDir, timeout)
	if err != nil {
		return nil, err
	}
	return browserPublish{b: b}, nil
}

// appStatusResult is the `web apps status` output.
type appStatusResult struct {
	PackageName string                        `json:"packageName"`
	AppState    string                        `json:"appState,omitempty"`
	Goals       []webdriver.SetupGoal         `json:"goals"`
	Publishing  *webdriver.PublishingOverview `json:"publishing"`
}

func registerAppStatusTable() {
	output.RegisterType(&appStatusResult{},
		[]string{"GOAL", "PROGRESS", "PENDING TASKS"},
		func(data any) [][]string {
			result, ok := data.(*appStatusResult)
			if !ok {
				return nil
			}
			rows := make([][]string, 0, len(result.Goals))
			for _, g := range result.Goals {
				title := g.Title
				if title == "" {
					title = g.ID
				}
				rows = append(rows, []string{title, fmt.Sprintf("%d/%d", g.Complete, g.Total), strings.Join(g.PendingTasks, "; ")})
			}
			return rows
		})
}

// WebAppsStatusCommand returns the `gplay web apps status` subcommand.
func WebAppsStatusCommand() *ffcli.Command {
	registerAppStatusTable()
	fs := flag.NewFlagSet("web apps status", flag.ExitOnError)
	pkg := fs.String("package", "", "Package name (applicationId)")
	developerID := fs.String("developer", "", "Developer account ID (numeric; default: auto-discover)")
	account := fs.String("account", "", "Web session account email (default: last used session)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "status",
		ShortUsage: "gplay web apps status --package <id>",
		ShortHelp:  "Show publishing readiness: setup checklist and pending changes.",
		LongHelp: `Read the app dashboard setup checklist and the Publishing overview.

Reports per-goal progress (including which tasks remain), the app state, the
changes waiting to be sent for review, whether the console allows sending
them, and why not. Read-only: it never changes anything.

Examples:
  gplay web apps status --package com.example.app
  gplay web apps status --package com.example.app --output table`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			packageName := strings.TrimSpace(*pkg)
			if packageName == "" {
				return fmt.Errorf("--package is required")
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, nil)
			defer cancel()
			sess, err := sessionLoad(strings.TrimSpace(*account))
			if err != nil {
				return err
			}
			target, err := resolveWebApp(ctx, newWebClient(sess), *developerID, packageName)
			if err != nil {
				return err
			}

			pb, err := newPublishBrowser(ctx, websession.BrowserProfileDir(), 90*time.Second)
			if err != nil {
				return err
			}
			defer func() { _ = pb.Close() }() //nolint:errcheck // best-effort cleanup

			setup, err := pb.ReadSetup(ctx, target.DeveloperID, target.AppID, sess.UserEmail)
			if err != nil {
				return err
			}
			overview, err := pb.ReadOverview(ctx, target.DeveloperID, target.AppID, sess.UserEmail)
			if err != nil {
				return err
			}
			return shared.PrintOutput(&appStatusResult{
				PackageName: packageName,
				AppState:    setup.AppState,
				Goals:       setup.Goals,
				Publishing:  overview,
			}, *outputFlag, *pretty)
		},
	}
}

// availabilityResult is the `web apps availability` output.
type availabilityResult struct {
	PackageName   string   `json:"packageName"`
	Track         string   `json:"track"`
	Selected      []string `json:"selected"`
	SelectedCount int      `json:"selectedCount"`
	Changed       bool     `json:"changed"`
}

func registerAvailabilityTable() {
	output.RegisterType(&availabilityResult{},
		[]string{"PACKAGE", "TRACK", "SELECTED", "CHANGED"},
		func(data any) [][]string {
			result, ok := data.(*availabilityResult)
			if !ok {
				return nil
			}
			return [][]string{{result.PackageName, result.Track, fmt.Sprint(result.SelectedCount), fmt.Sprint(result.Changed)}}
		})
}

func parseCountryNames(raw string) []string {
	var names []string
	for _, part := range strings.Split(raw, ",") {
		if name := strings.TrimSpace(part); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := map[string]int{}
	for _, s := range a {
		counts[s]++
	}
	for _, s := range b {
		counts[s]--
		if counts[s] < 0 {
			return false
		}
	}
	return true
}

func missingNames(selected, want []string) []string {
	have := map[string]bool{}
	for _, s := range selected {
		have[s] = true
	}
	var missing []string
	for _, w := range want {
		if !have[w] {
			missing = append(missing, w)
		}
	}
	return missing
}

// WebAppsAvailabilityCommand returns the `gplay web apps availability` subcommand.
func WebAppsAvailabilityCommand() *ffcli.Command {
	registerAvailabilityTable()
	fs := flag.NewFlagSet("web apps availability", flag.ExitOnError)
	pkg := fs.String("package", "", "Package name (applicationId)")
	track := fs.String("track", "production", `Track: "production" or a numeric track ID (console URL after "Manage track")`)
	countries := fs.String("countries", "", `Comma-separated country names as the console shows them, e.g. "Slovenia,United States"`)
	allCountries := fs.Bool("all-countries", false, "Target every country/region")
	developerID := fs.String("developer", "", "Developer account ID (numeric; default: auto-discover)")
	account := fs.String("account", "", "Web session account email (default: last used session)")
	confirm := fs.Bool("confirm", false, "Confirm the country availability change")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "availability",
		ShortUsage: "gplay web apps availability --package <id> [--countries \"A,B\" | --all-countries --confirm]",
		ShortHelp:  "Read or set production country availability.",
		LongHelp: `Read a track's targeted countries/regions, or set them.

Without --countries or --all-countries this only reports the current
selection. Changing the selection requires --confirm. The selection is set to
EXACTLY the given list: named countries are added, previously targeted
countries missing from the list are removed. The form is verified before
saving, and the saved selection is re-read afterwards. Country names must
match the console's display names (e.g. "Slovenia", "United States").

This drives the console's Countries / regions editor because the official
Android Publisher API cannot change country availability.

Use it to unblock paid apps: when the console refuses a pricing change with
"Remove <country> to make your app paid", read the current list, then set it
to the same list minus that country — on every track (production plus each
testing track's numeric ID) — with the user's confirmation.

Examples:
  gplay web apps availability --package com.example.app
  gplay web apps availability --package com.example.app --track 1234567890
  gplay web apps availability --package com.example.app --countries "Slovenia,Austria" --confirm
  gplay web apps availability --package com.example.app --all-countries --confirm
  gplay --dry-run web apps availability --package com.example.app --all-countries --confirm`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			packageName := strings.TrimSpace(*pkg)
			if packageName == "" {
				return fmt.Errorf("--package is required")
			}
			desired := parseCountryNames(*countries)
			if len(desired) > 0 && *allCountries {
				return fmt.Errorf("--countries and --all-countries cannot be used together")
			}
			write := len(desired) > 0 || *allCountries
			if write && !*confirm {
				return fmt.Errorf("--confirm is required to change country availability")
			}

			if shared.IsDryRun(ctx) {
				if write {
					fmt.Fprintf(os.Stderr, "[DRY RUN] would set production countries: package=%s all=%v countries=%q\n", packageName, *allCountries, strings.Join(desired, ", "))
				} else {
					fmt.Fprintf(os.Stderr, "[DRY RUN] would read production countries: package=%s\n", packageName)
				}
				fmt.Fprintln(os.Stderr, "[DRY RUN] No changes were made.")
				return nil
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, nil)
			defer cancel()
			sess, err := sessionLoad(strings.TrimSpace(*account))
			if err != nil {
				return err
			}
			target, err := resolveWebApp(ctx, newWebClient(sess), *developerID, packageName)
			if err != nil {
				return err
			}

			pb, err := newPublishBrowser(ctx, websession.BrowserProfileDir(), 90*time.Second)
			if err != nil {
				return err
			}
			defer func() { _ = pb.Close() }() //nolint:errcheck // best-effort cleanup
			if err := pb.OpenCountries(ctx, target.DeveloperID, target.AppID, sess.UserEmail, strings.TrimSpace(*track)); err != nil {
				return err
			}
			// Leaving the editor without saving is the safe default; after a
			// submit this is a no-op.
			defer func() { _ = pb.DiscardCountries(context.Background()) }() //nolint:errcheck // best-effort cleanup

			current, err := pb.ReadCountries(ctx)
			if err != nil {
				return err
			}
			result := &availabilityResult{
				PackageName:   packageName,
				Track:         strings.TrimSpace(*track),
				Selected:      current.Selected,
				SelectedCount: len(current.Selected),
			}
			if !write {
				return shared.PrintOutput(result, *outputFlag, *pretty)
			}

			if *allCountries {
				if err := pb.SetAllCountries(ctx); err != nil {
					return err
				}
			} else {
				toAdd := missingNames(current.Selected, desired)
				toRemove := missingNames(desired, current.Selected)
				if len(toAdd) > 0 {
					if err := pb.SetCountries(ctx, toAdd); err != nil {
						return err
					}
				}
				if len(toRemove) > 0 {
					if err := pb.UnsetCountries(ctx, toRemove); err != nil {
						return err
					}
				}
			}
			pending, err := pb.ReadCountries(ctx)
			if err != nil {
				return err
			}
			if missing := missingNames(pending.Selected, desired); len(missing) > 0 {
				return fmt.Errorf("the country selection does not match the request (missing: %s); nothing was changed", strings.Join(missing, ", "))
			}
			if !*allCountries {
				if extra := missingNames(desired, pending.Selected); len(extra) > 0 {
					return fmt.Errorf("the country selection does not match the request (still targeted: %s); nothing was changed", strings.Join(extra, ", "))
				}
			}
			if sameStrings(pending.Selected, current.Selected) {
				// The requested selection was already in place.
				return shared.PrintOutput(result, *outputFlag, *pretty)
			}
			if !pending.CanSubmit {
				return fmt.Errorf("the console reports the country selection is not ready to save; nothing was changed")
			}
			if err := pb.SubmitCountries(ctx, 2*time.Minute); err != nil {
				return err
			}
			if err := pb.OpenCountries(ctx, target.DeveloperID, target.AppID, sess.UserEmail, strings.TrimSpace(*track)); err != nil {
				return fmt.Errorf("reopening country availability after save: %w", err)
			}
			saved, err := pb.ReadCountries(ctx)
			if err != nil {
				return err
			}
			if missing := missingNames(saved.Selected, desired); len(missing) > 0 {
				return fmt.Errorf("country availability was not saved (missing: %s)", strings.Join(missing, ", "))
			}
			if !sameStrings(saved.Selected, pending.Selected) {
				return fmt.Errorf("country availability was not saved (saved %d countries, expected %d)", len(saved.Selected), len(pending.Selected))
			}
			result.Selected = saved.Selected
			result.SelectedCount = len(saved.Selected)
			result.Changed = true
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}

// pricingResult is the `web apps pricing` output.
type pricingResult struct {
	PackageName string `json:"packageName"`
	PriceType   string `json:"priceType"`
	PricesSet   bool   `json:"pricesSet"`
	Changed     bool   `json:"changed"`
}

func registerPricingTable() {
	output.RegisterType(&pricingResult{},
		[]string{"PACKAGE", "PRICE TYPE", "PRICES SET", "CHANGED"},
		func(data any) [][]string {
			result, ok := data.(*pricingResult)
			if !ok {
				return nil
			}
			return [][]string{{result.PackageName, result.PriceType, fmt.Sprint(result.PricesSet), fmt.Sprint(result.Changed)}}
		})
}

var pricePattern = regexp.MustCompile(`^\d+(\.\d{1,2})?$`)

// WebAppsPricingCommand returns the `gplay web apps pricing` subcommand.
func WebAppsPricingCommand() *ffcli.Command {
	registerPricingTable()
	fs := flag.NewFlagSet("web apps pricing", flag.ExitOnError)
	pkg := fs.String("package", "", "Package name (applicationId)")
	price := fs.String("price", "", "Price in the merchant account's home currency, e.g. 69.99")
	developerID := fs.String("developer", "", "Developer account ID (numeric; default: auto-discover)")
	account := fs.String("account", "", "Web session account email (default: last used session)")
	confirm := fs.Bool("confirm", false, "Confirm the price change")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "pricing",
		ShortUsage: "gplay web apps pricing --package <id> [--price <amount> --confirm]",
		ShortHelp:  "Read or set the paid app's price.",
		LongHelp: `Read the app pricing state, or set the price for a paid app.

Without --price this only reports the current state. Setting a price requires
--confirm and works for apps already set to Paid. The amount is in the
merchant account's home currency (the console asks e.g. "New price in EUR");
Google converts it to local prices for every targeted country.

This drives the console's App pricing page because the official Android
Publisher API cannot change app pricing. Making a paid app free is NOT
supported: that change cannot be undone once the app is published.

If the console refuses to save with "Remove <country> to make your app paid",
the app targets a country where paid distribution is not allowed (e.g. Sudan,
or the "Rest of world" pseudo-country). Remove it from EVERY track first —
with the user's confirmation — using "gplay web apps availability" on
production and on each testing track (numeric track ID from the console's
Manage track URL), then retry.

Examples:
  gplay web apps pricing --package com.example.app
  gplay web apps pricing --package com.example.app --price 69.99 --confirm
  gplay --dry-run web apps pricing --package com.example.app --price 69.99 --confirm`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			packageName := strings.TrimSpace(*pkg)
			if packageName == "" {
				return fmt.Errorf("--package is required")
			}
			requestedPrice := strings.TrimSpace(*price)
			write := requestedPrice != ""
			if write && !pricePattern.MatchString(requestedPrice) {
				return fmt.Errorf("--price must be a plain amount like 69.99 (the merchant account's home currency)")
			}
			if write && !*confirm {
				return fmt.Errorf("--confirm is required to change the app price")
			}

			if shared.IsDryRun(ctx) {
				if write {
					fmt.Fprintf(os.Stderr, "[DRY RUN] would set app price: package=%s price=%s\n", packageName, requestedPrice)
				} else {
					fmt.Fprintf(os.Stderr, "[DRY RUN] would read app pricing: package=%s\n", packageName)
				}
				fmt.Fprintln(os.Stderr, "[DRY RUN] No changes were made.")
				return nil
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, nil)
			defer cancel()
			sess, err := sessionLoad(strings.TrimSpace(*account))
			if err != nil {
				return err
			}
			target, err := resolveWebApp(ctx, newWebClient(sess), *developerID, packageName)
			if err != nil {
				return err
			}

			pb, err := newPublishBrowser(ctx, websession.BrowserProfileDir(), 90*time.Second)
			if err != nil {
				return err
			}
			defer func() { _ = pb.Close() }() //nolint:errcheck // best-effort cleanup
			if err := pb.OpenPricing(ctx, target.DeveloperID, target.AppID, sess.UserEmail); err != nil {
				return err
			}
			defer func() { _ = pb.DiscardPricing(context.Background()) }() //nolint:errcheck // best-effort cleanup

			current, err := pb.ReadPricing(ctx)
			if err != nil {
				return err
			}
			result := &pricingResult{
				PackageName: packageName,
				PriceType:   current.PriceType,
				PricesSet:   current.PricesSet,
			}
			if !write {
				return shared.PrintOutput(result, *outputFlag, *pretty)
			}
			if current.PriceType != "paid" {
				return fmt.Errorf("the app is not set to Paid in the console; pricing can only be set for a paid app")
			}

			if err := pb.SetPrice(ctx, requestedPrice); err != nil {
				return err
			}
			pending, err := pb.ReadPricing(ctx)
			if err != nil {
				return err
			}
			if !pending.StagedChanges {
				return fmt.Errorf("the pricing change was not staged; nothing was saved")
			}
			if !pending.CanSave {
				return fmt.Errorf("the console reports the pricing form is not ready to save; nothing was changed")
			}
			if err := pb.SubmitPricing(ctx, 2*time.Minute); err != nil {
				return err
			}
			if err := pb.OpenPricing(ctx, target.DeveloperID, target.AppID, sess.UserEmail); err != nil {
				return fmt.Errorf("reopening App pricing after save: %w", err)
			}
			saved, err := pb.ReadPricing(ctx)
			if err != nil {
				return err
			}
			if saved.StagedChanges {
				return fmt.Errorf("the price was not saved (the change is still staged)")
			}
			result.PricesSet = saved.PricesSet
			result.Changed = true
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}

// reviewResult is the `web apps review` output.
type reviewResult struct {
	PackageName string                    `json:"packageName"`
	SentCount   int                       `json:"sentCount"`
	SentChanges []webdriver.PendingChange `json:"sentChanges"`
}

func registerReviewTable() {
	output.RegisterType(&reviewResult{},
		[]string{"PACKAGE", "SENT CHANGES"},
		func(data any) [][]string {
			result, ok := data.(*reviewResult)
			if !ok {
				return nil
			}
			return [][]string{{result.PackageName, fmt.Sprint(result.SentCount)}}
		})
}

// WebAppsReviewCommand returns the `gplay web apps review` subcommand.
func WebAppsReviewCommand() *ffcli.Command {
	registerReviewTable()
	fs := flag.NewFlagSet("web apps review", flag.ExitOnError)
	pkg := fs.String("package", "", "Package name (applicationId)")
	developerID := fs.String("developer", "", "Developer account ID (numeric; default: auto-discover)")
	account := fs.String("account", "", "Web session account email (default: last used session)")
	confirm := fs.Bool("confirm", false, "Confirm sending changes for review")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "review",
		ShortUsage: "gplay web apps review --package <id> --confirm",
		ShortHelp:  "Send pending changes for review from the Publishing overview.",
		LongHelp: `Send the app's pending Play Console changes to Google for review.

Reads the Publishing overview, refuses when there is nothing to send or when
the console reports the app is not ready (incomplete setup steps), sends the
changes, and re-reads the overview to verify the pending list emptied.

This drives the Publishing overview because the official Android Publisher
API cannot send changes for review.

Examples:
  gplay web apps review --package com.example.app --confirm
  gplay --dry-run web apps review --package com.example.app --confirm`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			packageName := strings.TrimSpace(*pkg)
			if packageName == "" {
				return fmt.Errorf("--package is required")
			}
			if !*confirm {
				return fmt.Errorf("--confirm is required to send changes for review")
			}

			if shared.IsDryRun(ctx) {
				fmt.Fprintf(os.Stderr, "[DRY RUN] would send pending changes for review: package=%s\n", packageName)
				fmt.Fprintln(os.Stderr, "[DRY RUN] No changes were made.")
				return nil
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, nil)
			defer cancel()
			sess, err := sessionLoad(strings.TrimSpace(*account))
			if err != nil {
				return err
			}
			target, err := resolveWebApp(ctx, newWebClient(sess), *developerID, packageName)
			if err != nil {
				return err
			}

			pb, err := newPublishBrowser(ctx, websession.BrowserProfileDir(), 90*time.Second)
			if err != nil {
				return err
			}
			defer func() { _ = pb.Close() }() //nolint:errcheck // best-effort cleanup

			overview, err := pb.ReadOverview(ctx, target.DeveloperID, target.AppID, sess.UserEmail)
			if err != nil {
				return err
			}
			pending := len(overview.PendingChanges)
			if pending == 0 {
				// The changes table sometimes renders no rows; the send
				// button's "Submit N changes for review" label still counts.
				pending = overview.SummaryPendingCount
			}
			if pending == 0 {
				return fmt.Errorf("there are no pending changes to send for review")
			}
			if !overview.CanSendForReview {
				reason := overview.SendBlockedReason
				if reason == "" {
					reason = "the console did not say why"
				}
				return fmt.Errorf("the console is not ready to send changes for review: %s", reason)
			}
			if err := pb.SendForReview(ctx, 2*time.Minute); err != nil {
				return err
			}
			after, err := pb.ReadOverview(ctx, target.DeveloperID, target.AppID, sess.UserEmail)
			if err != nil {
				return err
			}
			if remaining := len(after.PendingChanges) + after.SummaryPendingCount; remaining != 0 {
				return fmt.Errorf("%d change(s) were not sent for review; check the Publishing overview", remaining)
			}
			return shared.PrintOutput(&reviewResult{
				PackageName: packageName,
				SentCount:   pending,
				SentChanges: overview.PendingChanges,
			}, *outputFlag, *pretty)
		},
	}
}

// rolloutResult is the `web apps rollout` output.
type rolloutResult struct {
	PackageName    string   `json:"packageName"`
	Track          string   `json:"track"`
	ReleaseName    string   `json:"releaseName"`
	Warnings       []string `json:"warnings,omitempty"`
	SentForReview  bool     `json:"sentForReview"`
	PendingChanges int      `json:"pendingChanges"`
}

func registerRolloutTable() {
	output.RegisterType(&rolloutResult{},
		[]string{"PACKAGE", "TRACK", "RELEASE", "SENT FOR REVIEW"},
		func(data any) [][]string {
			result, ok := data.(*rolloutResult)
			if !ok {
				return nil
			}
			return [][]string{{result.PackageName, result.Track, result.ReleaseName, fmt.Sprint(result.SentForReview)}}
		})
}

// WebAppsRolloutCommand returns the `gplay web apps rollout` subcommand.
func WebAppsRolloutCommand() *ffcli.Command {
	registerRolloutTable()
	fs := flag.NewFlagSet("web apps rollout", flag.ExitOnError)
	pkg := fs.String("package", "", "Package name (applicationId)")
	developerID := fs.String("developer", "", "Developer account ID (numeric; default: auto-discover)")
	account := fs.String("account", "", "Web session account email (default: last used session)")
	confirm := fs.Bool("confirm", false, "Confirm rolling out the draft release")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "rollout",
		ShortUsage: "gplay web apps rollout --package <id> --confirm",
		ShortHelp:  "Roll out the production draft release: preview, confirm, send for review.",
		LongHelp: `Roll out the production track's draft release to Google for review.

Drives the console's release wizard: opens the draft release, advances through
Preview and confirm, saves the release into the Publishing overview, then
sends the pending changes for review. Review warnings are reported but do not
block the rollout. This is the public release action for a draft app: once
Google approves, the release goes live (managed publishing off) or can be
published from the Publishing overview.

This drives the console because the official Android Publisher API cannot
roll out the first release of a draft app.

Examples:
  gplay web apps rollout --package com.example.app --confirm
  gplay --dry-run web apps rollout --package com.example.app --confirm`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			packageName := strings.TrimSpace(*pkg)
			if packageName == "" {
				return fmt.Errorf("--package is required")
			}
			if !*confirm {
				return fmt.Errorf("--confirm is required to roll out a release")
			}

			if shared.IsDryRun(ctx) {
				fmt.Fprintf(os.Stderr, "[DRY RUN] would roll out the production draft release: package=%s\n", packageName)
				fmt.Fprintln(os.Stderr, "[DRY RUN] No changes were made.")
				return nil
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, nil)
			defer cancel()
			sess, err := sessionLoad(strings.TrimSpace(*account))
			if err != nil {
				return err
			}
			target, err := resolveWebApp(ctx, newWebClient(sess), *developerID, packageName)
			if err != nil {
				return err
			}

			pb, err := newPublishBrowser(ctx, websession.BrowserProfileDir(), 90*time.Second)
			if err != nil {
				return err
			}
			defer func() { _ = pb.Close() }() //nolint:errcheck // best-effort cleanup

			if err := pb.OpenProductionReleases(ctx, target.DeveloperID, target.AppID, sess.UserEmail); err != nil {
				return err
			}
			prepare, err := pb.OpenDraftRelease(ctx)
			if err != nil {
				return err
			}
			review, err := pb.ReviewDraftRelease(ctx)
			if err != nil {
				return err
			}
			if !review.CanSave {
				return fmt.Errorf("the release review page is not ready to save; nothing was changed")
			}
			if err := pb.SaveReleaseReview(ctx, 2*time.Minute); err != nil {
				return err
			}

			result := &rolloutResult{
				PackageName: packageName,
				Track:       "production",
				ReleaseName: prepare.ReleaseName,
				Warnings:    review.Warnings,
			}
			overview, err := pb.ReadOverview(ctx, target.DeveloperID, target.AppID, sess.UserEmail)
			if err != nil {
				return err
			}
			if !overview.CanSendForReview {
				result.PendingChanges = len(overview.PendingChanges) + overview.SummaryPendingCount
				reason := overview.SendBlockedReason
				if reason == "" {
					reason = "the console did not say why"
				}
				return fmt.Errorf("the release is staged in the Publishing overview, but the console is not ready to send it for review: %s", reason)
			}
			if err := pb.SendForReview(ctx, 2*time.Minute); err != nil {
				return err
			}
			after, err := pb.ReadOverview(ctx, target.DeveloperID, target.AppID, sess.UserEmail)
			if err != nil {
				return err
			}
			result.PendingChanges = len(after.PendingChanges) + after.SummaryPendingCount
			if result.PendingChanges != 0 {
				return fmt.Errorf("%d change(s) were not sent for review; check the Publishing overview", result.PendingChanges)
			}
			result.SentForReview = true
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}
