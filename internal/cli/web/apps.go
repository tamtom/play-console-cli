package web

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/output"
	"github.com/tamtom/play-console-cli/internal/webclient"
	"github.com/tamtom/play-console-cli/internal/webdriver"
	"github.com/tamtom/play-console-cli/internal/websession"
)

// WebAppsCommand returns the `gplay web apps` command group.
func WebAppsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web apps", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "apps",
		ShortUsage: "gplay web apps <subcommand> [flags]",
		ShortHelp:  "Manage apps through a Play Console web session.",
		LongHelp: `Manage apps through a Play Console browser session.

These commands use the browser-session auth from "gplay web auth login" for
Play Console capabilities that the official Android Publisher API does not
provide, including listing and creating apps, changing App category, setting
country availability, managing form-factor distribution and promo codes, and
reading content-rating state.

Every subcommand except "list" drives a local Google Chrome window, which gplay
locates automatically on macOS only. On Linux and Windows, set
GPLAY_CHROME_BINARY to the Chrome executable path; those platforms are
untested.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			WebAppsListCommand(),
			WebAppsCreateCommand(),
			WebAppsUpdateCommand(),
			WebAppsStatusCommand(),
			WebAppsAvailabilityCommand(),
			WebAppsPricingCommand(),
			WebAppsReviewCommand(),
			WebAppsRolloutCommand(),
			WebAppsDeclarationsCommand(),
			WebAppsPolicyCommand(),
			WebAppsPublishCommand(),
			WebAppsDistributionCommand(),
			WebAppsPromoCodesCommand(),
			WebAppsRatingCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", args[0])
			return flag.ErrHelp
		},
	}
}

// registerAppTable teaches the output registry how to tabulate app lists so
// --output table/markdown render rows instead of falling back to JSON.
func registerAppTable() {
	output.RegisterType([]webclient.App{},
		[]string{"PACKAGE", "NAME", "APP ID", "LANGUAGE"},
		func(data any) [][]string {
			apps, ok := data.([]webclient.App)
			if !ok {
				return nil
			}
			rows := make([][]string, 0, len(apps))
			for _, a := range apps {
				rows = append(rows, []string{a.PackageName, a.DisplayName, a.AppID, a.Language})
			}
			return rows
		})
}

// WebAppsListCommand returns the `gplay web apps list` subcommand.
func WebAppsListCommand() *ffcli.Command {
	registerAppTable()
	fs := flag.NewFlagSet("web apps list", flag.ExitOnError)
	developerID := fs.String("developer", "", "Developer account ID (numeric; default: auto-discover from the console)")
	account := fs.String("account", "", "Web session account email (default: last used session)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay web apps list [--developer <id>] [--output table]",
		ShortHelp:  "List every app in the developer account via the web session.",
		LongHelp: `List every app in a Play Console developer account.

Unlike "gplay apps list", this sees apps the moment they exist. That command
queries the Play Developer Reporting API, which only returns apps that already
have reporting data, so a freshly created app is missing from it. The official
Publisher API has no list-apps endpoint at all, which is why this uses the web
session instead.

All pages are fetched automatically.

Examples:
  gplay web apps list
  gplay web apps list --output table
  gplay web apps list --developer 1234567890`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			sess, err := sessionLoad(strings.TrimSpace(*account))
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, nil)
			defer cancel()

			list := func(client webRPCClient) ([]webclient.App, error) {
				developer := strings.TrimSpace(*developerID)
				if developer == "" {
					discovered, derr := client.DiscoverDeveloperID(ctx)
					if derr != nil {
						return nil, fmt.Errorf("discovering developer ID (pass --developer to skip): %w", derr)
					}
					developer = discovered
				}
				return client.ListApps(ctx, developer)
			}

			apps, err := list(newWebClient(sess))
			// Web sessions expire roughly daily. When the dedicated browser
			// profile is still signed in, recover silently instead of making
			// the user re-run login for every expiry.
			if errors.Is(err, webclient.ErrAuth) {
				if refreshed := refreshFromBrowserProfile(ctx, sess.UserEmail); refreshed != nil {
					fmt.Fprintln(os.Stderr, "Web session had expired; refreshed it from the gplay browser profile.")
					apps, err = list(newWebClient(refreshed))
				}
			}
			if err != nil {
				return err
			}
			return shared.PrintOutput(apps, *outputFlag, *pretty)
		},
	}
}

// appCreator drives the console's create-app form. It exists so the create
// command can be tested without a real browser.
type appCreator interface {
	Fill(ctx context.Context, developerID string, form webdriver.AppForm) error
	Read(ctx context.Context) (*webdriver.FormState, error)
	Submit(ctx context.Context, timeout time.Duration) (string, error)
	Close() error
}

// browserCreator adapts a live Chrome to appCreator.
type browserCreator struct{ b *webdriver.Browser }

func (c browserCreator) Fill(ctx context.Context, developerID string, form webdriver.AppForm) error {
	return webdriver.FillAppForm(ctx, c.b, developerID, form)
}

func (c browserCreator) Read(ctx context.Context) (*webdriver.FormState, error) {
	return webdriver.ReadForm(ctx, c.b)
}

func (c browserCreator) Submit(ctx context.Context, timeout time.Duration) (string, error) {
	return webdriver.SubmitAppForm(ctx, c.b, timeout)
}
func (c browserCreator) Close() error { return c.b.Close() }

func connectAppBrowser(ctx context.Context, userDataDir string, timeout time.Duration) (*webdriver.Browser, error) {
	if !webdriver.Running(userDataDir) {
		if err := chromeLauncher(ctx, userDataDir, consoleLoginURL); err != nil {
			return nil, err
		}
	}
	b, err := webdriver.Connect(ctx, userDataDir, timeout)
	if err != nil {
		return nil, fmt.Errorf("%w\n\nIf a gplay Chrome window is already open without debugging enabled, quit it and rerun", err)
	}
	return b, nil
}

// newAppCreator connects to the gplay-managed Chrome profile, starting it if
// it is not already running. Overridden in tests.
var newAppCreator = func(ctx context.Context, userDataDir string, timeout time.Duration) (appCreator, error) {
	b, err := connectAppBrowser(ctx, userDataDir, timeout)
	if err != nil {
		return nil, err
	}
	return browserCreator{b: b}, nil
}

// WebAppsCreateCommand returns the `gplay web apps create` subcommand.
func WebAppsCreateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web apps create", flag.ExitOnError)
	name := fs.String("name", "", "App name as shown on Google Play")
	pkg := fs.String("package", "", "Package name (applicationId), e.g. com.example.app")
	language := fs.String("language", "en-US", "Default language in BCP 47 format (e.g. en-US)")
	kind := fs.String("kind", "", "app or game")
	pricing := fs.String("pricing", "", "free or paid")
	developerID := fs.String("developer", "", "Developer account ID (numeric; default: auto-discover)")
	account := fs.String("account", "", "Web session account email (default: last used session)")
	acceptPolicies := fs.Bool("accept-policies", false, "Declare the app meets the Developer Program Policies")
	acceptExport := fs.Bool("accept-us-export-laws", false, "Declare compliance with US export laws")
	confirm := fs.Bool("confirm", false, "Confirm app creation")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "create",
		ShortUsage: "gplay web apps create --name <name> --package <id> --kind <app|game> --pricing <free|paid> --confirm",
		ShortHelp:  "Create a new app in Play Console via the web session.",
		LongHelp: `Create a new app in Play Console using the stored web session.

This mirrors the console's "Create app" dialog exactly: app name, package name,
default language, app or game, free or paid, and the two declarations. The
official Android Publisher API cannot create apps, which is why this uses the
web session.

The package name is checked for availability first; creation is skipped if it
is already taken.

Both declarations are required and must be passed explicitly — they are legal
statements about YOUR app, so this command will not assume them:
  --accept-policies         the app meets the Developer Program Policies
  --accept-us-export-laws   compliance with US export laws

Pricing is effectively one-way: once an app is published you cannot change a
free app to paid. The new app starts as a draft.

Examples:
  gplay web apps create --name "Matisse" --package com.example.matisse \
    --kind app --pricing free --accept-policies --accept-us-export-laws --confirm`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			req := webclient.CreateAppRequest{
				Title:           strings.TrimSpace(*name),
				PackageName:     strings.TrimSpace(*pkg),
				DefaultLanguage: strings.TrimSpace(*language),
				Kind:            webclient.AppKind(strings.TrimSpace(*kind)),
				MeetsGuidelines: *acceptPolicies,
				USLawCompliant:  *acceptExport,
			}
			if req.Title == "" {
				return fmt.Errorf("--name is required")
			}
			if req.PackageName == "" {
				return fmt.Errorf("--package is required")
			}
			if req.Kind != webclient.AppKindApp && req.Kind != webclient.AppKindGame {
				return fmt.Errorf("--kind must be app or game")
			}
			switch strings.TrimSpace(*pricing) {
			case "free":
				req.Paid = false
			case "paid":
				req.Paid = true
			default:
				return fmt.Errorf("--pricing must be free or paid")
			}
			if req.DefaultLanguage == "" {
				return fmt.Errorf("--language is required")
			}
			if !*acceptPolicies || !*acceptExport {
				return fmt.Errorf("--accept-policies and --accept-us-export-laws are both required: they are declarations about your app")
			}
			if !*confirm {
				return fmt.Errorf("--confirm is required to create an app")
			}

			if shared.IsDryRun(ctx) {
				fmt.Fprintf(os.Stderr, "[DRY RUN] would create app: name=%q package=%s language=%s kind=%s paid=%t\n",
					req.Title, req.PackageName, req.DefaultLanguage, req.Kind, req.Paid)
				fmt.Fprintln(os.Stderr, "[DRY RUN] No changes were made.")
				return nil
			}

			sess, err := sessionLoad(strings.TrimSpace(*account))
			if err != nil {
				return err
			}
			client := newWebClient(sess)
			ctx, cancel := shared.ContextWithTimeout(ctx, nil)
			defer cancel()

			req.DeveloperID = strings.TrimSpace(*developerID)
			if req.DeveloperID == "" {
				if req.DeveloperID, err = client.DiscoverDeveloperID(ctx); err != nil {
					return fmt.Errorf("discovering developer ID (pass --developer to skip): %w", err)
				}
			}

			// Check before writing: creation is not something to retry blindly.
			availability, err := client.CheckPackageName(ctx, req.DeveloperID, req.PackageName)
			if err != nil {
				return err
			}
			if !availability.Available() {
				return fmt.Errorf("package name %s is not available (%s); choose another --package", req.PackageName, availability)
			}

			// The create request body is undocumented and a hand-built payload
			// is rejected by the server, so drive the console's own form: what
			// the frontend sends is correct by construction.
			creator, err := newAppCreator(ctx, websession.BrowserProfileDir(), 90*time.Second)
			if err != nil {
				return err
			}
			defer func() { _ = creator.Close() }() //nolint:errcheck // best-effort cleanup

			form := webdriver.AppForm{
				Title:       req.Title,
				PackageName: req.PackageName,
				Language:    req.DefaultLanguage,
				Game:        req.Kind == webclient.AppKindGame,
				Paid:        req.Paid,
			}
			if err := creator.Fill(ctx, req.DeveloperID, form); err != nil {
				return err
			}

			// Read the form back before submitting: if the page did not take a
			// value, creating the wrong app is not something you can undo.
			state, err := creator.Read(ctx)
			if err != nil {
				return err
			}
			if state.Title != form.Title || state.PackageName != form.PackageName ||
				state.Game != form.Game || state.Paid != form.Paid ||
				!state.Policies || !state.Export {
				return fmt.Errorf("the create-app form does not match the request (got title=%q package=%q game=%t paid=%t policies=%t export=%t); nothing was created",
					state.Title, state.PackageName, state.Game, state.Paid, state.Policies, state.Export)
			}
			if !state.CanSubmit {
				return fmt.Errorf("the console reports the create-app form is not ready to submit; nothing was created")
			}

			appID, err := creator.Submit(ctx, 3*time.Minute)
			if err != nil {
				return err
			}
			return shared.PrintOutput(&webclient.App{
				AppID:       appID,
				PackageName: req.PackageName,
				DisplayName: req.Title,
				DeveloperID: req.DeveloperID,
				Language:    req.DefaultLanguage,
			}, *outputFlag, *pretty)
		},
	}
}

// resolveWebApp finds the developer and app IDs for a package through the web
// session, discovering the developer account when developerFlag is empty.
func resolveWebApp(ctx context.Context, client webRPCClient, developerFlag, packageName string) (*webclient.App, error) {
	developer := strings.TrimSpace(developerFlag)
	if developer == "" {
		discovered, err := client.DiscoverDeveloperID(ctx)
		if err != nil {
			return nil, fmt.Errorf("discovering developer ID (pass --developer to skip): %w", err)
		}
		developer = discovered
	}
	apps, err := client.ListApps(ctx, developer)
	if err != nil {
		return nil, err
	}
	for i := range apps {
		if apps[i].PackageName != packageName {
			continue
		}
		target := &apps[i]
		if target.DeveloperID == "" {
			target.DeveloperID = developer
		}
		if target.AppID == "" {
			return nil, fmt.Errorf("no app ID returned by Play Console for package %s", packageName)
		}
		return target, nil
	}
	return nil, fmt.Errorf("package %s was not found in developer account %s", packageName, developer)
}

// appUpdater drives the console's Store settings form. It exists so updates
// can be tested without changing a real Play Console app.
type appUpdater interface {
	Open(ctx context.Context, developerID, appID, account string) error
	Read(ctx context.Context) (*webdriver.AppSettingsState, error)
	SetClassification(ctx context.Context, kind, category string) error
	Submit(ctx context.Context, timeout time.Duration) error
	Close() error
}

type browserUpdater struct{ b *webdriver.Browser }

func (u browserUpdater) Open(ctx context.Context, developerID, appID, account string) error {
	return webdriver.OpenAppSettings(ctx, u.b, developerID, appID, account)
}

func (u browserUpdater) Read(ctx context.Context) (*webdriver.AppSettingsState, error) {
	return webdriver.ReadAppSettings(ctx, u.b)
}

func (u browserUpdater) SetClassification(ctx context.Context, kind, category string) error {
	return webdriver.SetAppClassification(ctx, u.b, kind, category)
}

func (u browserUpdater) Submit(ctx context.Context, timeout time.Duration) error {
	return webdriver.SubmitAppSettings(ctx, u.b, timeout)
}
func (u browserUpdater) Close() error { return u.b.Close() }

var newAppUpdater = func(ctx context.Context, userDataDir string, timeout time.Duration) (appUpdater, error) {
	b, err := connectAppBrowser(ctx, userDataDir, timeout)
	if err != nil {
		return nil, err
	}
	return browserUpdater{b: b}, nil
}

type appUpdateResult struct {
	PackageName string `json:"packageName"`
	Kind        string `json:"kind"`
	Category    string `json:"category"`
	Changed     bool   `json:"changed"`
}

func registerAppUpdateTable() {
	output.RegisterType(&appUpdateResult{},
		[]string{"PACKAGE", "KIND", "CATEGORY", "CHANGED"},
		func(data any) [][]string {
			result, ok := data.(*appUpdateResult)
			if !ok {
				return nil
			}
			return [][]string{{result.PackageName, result.Kind, result.Category, fmt.Sprint(result.Changed)}}
		})
}

// WebAppsUpdateCommand returns the `gplay web apps update` subcommand.
func WebAppsUpdateCommand() *ffcli.Command {
	registerAppUpdateTable()
	fs := flag.NewFlagSet("web apps update", flag.ExitOnError)
	pkg := fs.String("package", "", "Package name (applicationId)")
	kind := fs.String("kind", "", "New application type: app or game")
	category := fs.String("category", "", "Google Play category label, e.g. Education")
	developerID := fs.String("developer", "", "Developer account ID (numeric; default: auto-discover)")
	account := fs.String("account", "", "Web session account email (default: last used session)")
	confirm := fs.Bool("confirm", false, "Confirm the App category change")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "update",
		ShortUsage: "gplay web apps update --package <id> --kind <app|game> --category <label> --confirm",
		ShortHelp:  "Update an existing package's App category.",
		LongHelp: `Update an existing Play Console package's application type and required category.

This uses Store settings in the gplay-managed browser because the official
Android Publisher API cannot change App category. Sign in once with:
  gplay web auth login --email <email> --browser

Only App category belongs here. Use "gplay listings patch" for localized app
names and "gplay details patch" for default language or contact details. App
pricing is intentionally excluded because changing a paid app to free can make
a later change back to paid impossible.

Examples:
  gplay web apps update --package com.example.app --kind app --category Education --confirm
  gplay --dry-run web apps update --package com.example.app --kind game --category Educational --confirm`,
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
			requestedKind := strings.TrimSpace(*kind)
			if requestedKind != string(webclient.AppKindApp) && requestedKind != string(webclient.AppKindGame) {
				return fmt.Errorf("--kind must be app or game")
			}
			requestedCategory := strings.TrimSpace(*category)
			if requestedCategory == "" {
				return fmt.Errorf("--category is required")
			}
			if !*confirm {
				return fmt.Errorf("--confirm is required to update an app")
			}

			if shared.IsDryRun(ctx) {
				fmt.Fprintf(os.Stderr, "[DRY RUN] would update app: package=%s kind=%s category=%q\n", packageName, requestedKind, requestedCategory)
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

			updater, err := newAppUpdater(ctx, websession.BrowserProfileDir(), 90*time.Second)
			if err != nil {
				return err
			}
			defer func() { _ = updater.Close() }() //nolint:errcheck // best-effort cleanup

			if err := updater.Open(ctx, target.DeveloperID, target.AppID, sess.UserEmail); err != nil {
				return err
			}
			current, err := updater.Read(ctx)
			if err != nil {
				return err
			}
			if current.Kind == "" {
				return fmt.Errorf("could not read the current application type; nothing was changed")
			}
			result := &appUpdateResult{
				PackageName: packageName,
				Kind:        current.Kind,
				Category:    current.Category,
			}
			if current.Kind == requestedKind && strings.EqualFold(current.Category, requestedCategory) {
				return shared.PrintOutput(result, *outputFlag, *pretty)
			}

			if err := updater.SetClassification(ctx, requestedKind, requestedCategory); err != nil {
				return err
			}
			pending, err := updater.Read(ctx)
			if err != nil {
				return err
			}
			if pending.Kind != requestedKind || !strings.EqualFold(pending.Category, requestedCategory) {
				return fmt.Errorf("the App category form does not match the request (got kind=%q category=%q); nothing was changed", pending.Kind, pending.Category)
			}
			if !pending.CanSubmit {
				return fmt.Errorf("the console reports App category is not ready to save; nothing was changed")
			}
			if err := updater.Submit(ctx, 2*time.Minute); err != nil {
				return err
			}
			if err := updater.Open(ctx, target.DeveloperID, target.AppID, sess.UserEmail); err != nil {
				return fmt.Errorf("reopening App category after save: %w", err)
			}
			saved, err := updater.Read(ctx)
			if err != nil {
				return err
			}
			if saved.Kind != requestedKind || !strings.EqualFold(saved.Category, requestedCategory) {
				return fmt.Errorf("the App category was not saved (got kind=%q category=%q, want kind=%q category=%q)",
					saved.Kind, saved.Category, requestedKind, requestedCategory)
			}
			result.Kind = saved.Kind
			result.Category = saved.Category
			result.Changed = true
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}
