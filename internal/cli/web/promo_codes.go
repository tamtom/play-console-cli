package web

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/webdriver"
	"github.com/tamtom/play-console-cli/internal/websession"
)

type promoCodesRunner func(context.Context, string, string, string, string, bool, *webdriver.PaidAppPromotionForm) (*webdriver.PromotionsState, error)

var runPromoCodes promoCodesRunner = func(ctx context.Context, userDataDir, developerID, appID, account string, termsAccepted bool, form *webdriver.PaidAppPromotionForm) (*webdriver.PromotionsState, error) {
	b, err := connectAppBrowser(ctx, userDataDir, 90*time.Second)
	if err != nil {
		return nil, err
	}
	defer func() { _ = b.Close() }() //nolint:errcheck // best-effort cleanup
	var state *webdriver.PromotionsState
	if form == nil {
		state, err = webdriver.ReadPromotions(ctx, b, developerID, appID, account)
	} else {
		state, err = webdriver.CreatePaidAppPromotion(ctx, b, developerID, appID, account, termsAccepted, *form, 2*time.Minute)
	}
	if state != nil {
		state.TermsAcceptanceRequired = !termsAccepted
	}
	return state, err
}

type promoCodesResult struct {
	PackageName             string                `json:"packageName"`
	Promotions              []webdriver.Promotion `json:"promotions"`
	TermsAcceptanceRequired bool                  `json:"termsAcceptanceRequired"`
	Created                 string                `json:"created,omitempty"`
	Changed                 bool                  `json:"changed"`
}

// WebAppsPromoCodesCommand returns the `gplay web apps promo-codes` subcommand.
func WebAppsPromoCodesCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web apps promo-codes", flag.ExitOnError)
	pkg := fs.String("package", "", "Package name (applicationId)")
	create := fs.Bool("create", false, "Create generated one-time codes for the paid app")
	name := fs.String("name", "", "Promotion name (maximum 60 characters)")
	startDate := fs.String("start-date", "", "Promotion start date in GMT (YYYY-MM-DD)")
	endDate := fs.String("end-date", "", "Promotion end date in GMT (YYYY-MM-DD)")
	codeCount := fs.Int("code-count", 0, "Number of codes to generate (1-500)")
	developerID := fs.String("developer", "", "Developer account ID (numeric; default: auto-discover)")
	account := fs.String("account", "", "Web session account email (default: last used session)")
	confirm := fs.Bool("confirm", false, "Confirm promo-code creation")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "promo-codes",
		ShortUsage: "gplay web apps promo-codes --package <id> [--create --name <name> --start-date <YYYY-MM-DD> --end-date <YYYY-MM-DD> --code-count <n> --confirm]",
		ShortHelp:  "List promo-code campaigns or create paid-app codes.",
		LongHelp: `List promo-code campaigns, or create generated one-time-use codes for a paid app.

Creation currently covers the paid-app reward, whose Console form has stable,
verified controls. One-time products, subscriptions, custom codes, campaign
updates, and downloads remain in Play Console.

The promotion may last at most one year and can generate at most 500 codes.
Dates are interpreted in GMT. Promo codes Terms of Service must be reviewed
and accepted manually in Play Console; this command never accepts them.

Examples:
  gplay web apps promo-codes --package com.example.app
  gplay web apps promo-codes --package com.example.app --create --name Launch \
    --start-date 2026-08-02 --end-date 2026-08-31 --code-count 100 --confirm`,
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
			var form *webdriver.PaidAppPromotionForm
			if *create {
				promotionName := strings.TrimSpace(*name)
				if promotionName == "" {
					return fmt.Errorf("--name is required with --create")
				}
				if len([]rune(promotionName)) > 60 {
					return fmt.Errorf("--name must be at most 60 characters")
				}
				start, err := time.Parse("2006-01-02", strings.TrimSpace(*startDate))
				if err != nil {
					return fmt.Errorf("--start-date must use YYYY-MM-DD")
				}
				end, err := time.Parse("2006-01-02", strings.TrimSpace(*endDate))
				if err != nil {
					return fmt.Errorf("--end-date must use YYYY-MM-DD")
				}
				if end.Before(start) {
					return fmt.Errorf("--end-date cannot be before --start-date")
				}
				if end.After(start.AddDate(1, 0, 0)) {
					return fmt.Errorf("a promotion cannot last more than one year")
				}
				if *codeCount < 1 || *codeCount > 500 {
					return fmt.Errorf("--code-count must be between 1 and 500")
				}
				if !*confirm {
					return fmt.Errorf("--confirm is required to create promo codes")
				}
				form = &webdriver.PaidAppPromotionForm{
					Name: promotionName, StartDate: start.Format("2006-01-02"),
					EndDate: end.Format("2006-01-02"), CodeCount: *codeCount,
				}
			} else if strings.TrimSpace(*name) != "" || strings.TrimSpace(*startDate) != "" || strings.TrimSpace(*endDate) != "" || *codeCount != 0 {
				return fmt.Errorf("--create is required with promotion creation flags")
			}

			if shared.IsDryRun(ctx) {
				action := "read promo codes"
				if form != nil {
					action = fmt.Sprintf("create %d paid-app promo codes named %q", form.CodeCount, form.Name)
				}
				fmt.Fprintf(os.Stderr, "[DRY RUN] would %s: package=%s\n", action, packageName)
				fmt.Fprintln(os.Stderr, "[DRY RUN] No changes were made.")
				return nil
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, nil)
			defer cancel()
			sess, err := sessionLoad(strings.TrimSpace(*account))
			if err != nil {
				return err
			}
			client := newWebClient(sess)
			target, err := resolveWebApp(ctx, client, *developerID, packageName)
			if err != nil {
				return err
			}
			termsAccepted, err := client.PromoCodesTermsAccepted(ctx, target.DeveloperID)
			if err != nil {
				return err
			}
			if form != nil && !termsAccepted {
				return fmt.Errorf("the promo codes Terms of Service must be reviewed and accepted manually in Play Console; no promotion was created")
			}
			state, err := runPromoCodes(ctx, websession.BrowserProfileDir(), target.DeveloperID, target.AppID, sess.UserEmail, termsAccepted, form)
			if err != nil {
				return err
			}
			if state.Promotions == nil {
				state.Promotions = []webdriver.Promotion{}
			}
			result := &promoCodesResult{
				PackageName: packageName, Promotions: state.Promotions,
				TermsAcceptanceRequired: state.TermsAcceptanceRequired,
			}
			if form != nil {
				result.Created = form.Name
				result.Changed = true
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}
