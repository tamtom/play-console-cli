package web

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/webdriver"
	"github.com/tamtom/play-console-cli/internal/websession"
)

var runContentRating = func(ctx context.Context, userDataDir, developerID, appID, account string) (*webdriver.ContentRatingState, error) {
	b, err := connectAppBrowser(ctx, userDataDir, 90*time.Second)
	if err != nil {
		return nil, err
	}
	defer func() { _ = b.Close() }() //nolint:errcheck // best-effort cleanup
	return webdriver.ReadContentRating(ctx, b, developerID, appID, account)
}

type contentRatingResult struct {
	PackageName string `json:"packageName"`
	*webdriver.ContentRatingState
}

// WebAppsRatingCommand returns the read-only `gplay web apps rating` subcommand.
func WebAppsRatingCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web apps rating", flag.ExitOnError)
	pkg := fs.String("package", "", "Package name (applicationId)")
	developerID := fs.String("developer", "", "Developer account ID (numeric; default: auto-discover)")
	account := fs.String("account", "", "Web session account email (default: last used session)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "rating",
		ShortUsage: "gplay web apps rating --package <id>",
		ShortHelp:  "Read the app's IARC content-rating state.",
		LongHelp: `Read the submitted IARC content rating, contact email, certificate,
regional ratings, and any unfinished questionnaire draft.

Read-only. IARC submission is intentionally left in Play Console: its dynamic
questionnaire requires app-specific factual answers and explicit acceptance
of IARC's Terms of Use.

Example:
  gplay web apps rating --package com.example.app`,
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
			state, err := runContentRating(ctx, websession.BrowserProfileDir(), target.DeveloperID, target.AppID, sess.UserEmail)
			if err != nil {
				return err
			}
			return shared.PrintOutput(&contentRatingResult{PackageName: packageName, ContentRatingState: state}, *outputFlag, *pretty)
		},
	}
}
