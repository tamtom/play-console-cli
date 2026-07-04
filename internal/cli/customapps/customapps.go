// Package customapps implements the `gplay custom-apps` command group for
// publishing private apps through the Google Play Custom App Publishing API
// (Managed Google Play). A custom app is an APK distributed only to specific
// organizations rather than the public Play Store.
package customapps

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/googleapi"
	playcustomapp "google.golang.org/api/playcustomapp/v1"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/customappsclient"
)

// CustomAppsCommand returns the Managed Google Play custom-apps command group.
func CustomAppsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("custom-apps", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "custom-apps",
		ShortUsage: "gplay custom-apps <subcommand> [flags]",
		ShortHelp:  "Publish private apps via Managed Google Play (Custom App Publishing API).",
		LongHelp: `Publish private (custom) apps through the Google Play Custom App Publishing API.

Custom apps power Managed Google Play private distribution: an APK is published
to specific organizations instead of the public Play Store. This is the API
behind "private apps" in the Managed Google Play iframe.

Identifiers differ from the rest of gplay. Create takes the numeric developer
account ID (--developer, the ID from the Play Console URL developers/<id>), NOT
the Android package name — the package name is assigned by Google and returned
in the response. Restrict availability with --organizations (comma-separated
organization IDs); omit it to publish to the organization linked to the
developer account.

Custom apps accept an APK only (not an AAB). The service account must have the
developer-account-level permission to publish custom apps.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			CreateCommand(),
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

// CreateCommand returns the `custom-apps create` subcommand.
func CreateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("custom-apps create", flag.ExitOnError)
	developerID := fs.String("developer", "", "Developer account ID (numeric, from the Play Console URL)")
	title := fs.String("title", "", "Title for the custom app")
	language := fs.String("language", "en-US", "Default listing language in BCP 47 format (e.g. en-US)")
	apkPath := fs.String("apk", "", "Path to the .apk file to publish")
	organizations := fs.String("organizations", "", "Comma-separated organization IDs to restrict availability (optional)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "create",
		ShortUsage: "gplay custom-apps create --developer <id> --title <title> --apk <path> [--organizations <ids>]",
		ShortHelp:  "Publish a private APK as a custom app.",
		LongHelp: `Publish a private APK as a custom app for Managed Google Play.

The app is created under the given developer account and the APK is uploaded in
the same request. Google assigns the package name and returns it in the
response.

Examples:
  # Publish to the organization linked to the developer account
  gplay custom-apps create --developer 1234567890 --title "Field Tool" --apk app.apk

  # Restrict to specific organizations
  gplay custom-apps create --developer 1234567890 --title "Field Tool" \
    --language en-US --organizations org-abc,org-def --apk app.apk`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*developerID) == "" {
				return fmt.Errorf("--developer is required")
			}
			account, err := strconv.ParseInt(strings.TrimSpace(*developerID), 10, 64)
			if err != nil {
				return fmt.Errorf("--developer must be a numeric account ID: %w", err)
			}
			if strings.TrimSpace(*title) == "" {
				return fmt.Errorf("--title is required")
			}
			if strings.TrimSpace(*language) == "" {
				return fmt.Errorf("--language is required")
			}
			if strings.TrimSpace(*apkPath) == "" {
				return fmt.Errorf("--apk is required")
			}

			body := &playcustomapp.CustomApp{
				Title:        strings.TrimSpace(*title),
				LanguageCode: strings.TrimSpace(*language),
			}
			for _, org := range strings.Split(*organizations, ",") {
				if id := strings.TrimSpace(org); id != "" {
					body.Organizations = append(body.Organizations, &playcustomapp.Organization{OrganizationId: id})
				}
			}

			service, err := customappsclient.NewService(ctx)
			if err != nil {
				return err
			}

			file, err := os.Open(*apkPath)
			if err != nil {
				return shared.WrapActionable(err, "failed to open APK file", "Check that the file exists and is readable.")
			}
			defer file.Close()

			ctx, cancel := shared.ContextWithUploadTimeout(ctx, service.Cfg)
			defer cancel()

			call := service.API.Accounts.CustomApps.Create(account, body)
			call.Media(file, googleapi.ContentType("application/octet-stream"))
			resp, err := call.Context(ctx).Do()
			if err != nil {
				return shared.WrapGoogleAPIError("failed to create custom app", err)
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}
