package grants

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

// appPermissionsHelp lists the appLevelPermissions values that Play accepts.
// Tests compare this list with the embedded official schema.
const appPermissionsHelp = `Available app permissions:
  - CAN_VIEW_NON_FINANCIAL_DATA: View app information (read-only)
  - CAN_VIEW_FINANCIAL_DATA: View financial data
  - CAN_MANAGE_PERMISSIONS: Admin (all permissions)
  - CAN_REPLY_TO_REVIEWS: Reply to reviews
  - CAN_MANAGE_PUBLIC_APKS: Release to production, exclude devices, and use app signing by Google Play
  - CAN_MANAGE_TRACK_APKS: Release to testing tracks
  - CAN_MANAGE_TRACK_USERS: Manage testing tracks and edit tester lists
  - CAN_MANAGE_PUBLIC_LISTING: Manage store presence
  - CAN_MANAGE_DRAFT_APPS: Edit and delete draft apps
  - CAN_MANAGE_ORDERS: Manage orders and subscriptions
  - CAN_MANAGE_APP_CONTENT: Manage policy pages
  - CAN_VIEW_APP_QUALITY: View app quality data, such as vitals and crashes
  - CAN_MANAGE_DEEPLINKS: Manage the deep link setup of the app

The list does not show deprecated permissions.`

func GrantsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("grants", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "grants",
		ShortUsage: "gplay grants <subcommand> [flags]",
		ShortHelp:  "Manage per-app permission grants.",
		LongHelp: `Manage per-app permission grants for users.

Grants give users specific permissions for individual apps,
as opposed to account-wide permissions.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			CreateCommand(),
			UpdateCommand(),
			DeleteCommand(),
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
	fs := flag.NewFlagSet("grants create", flag.ExitOnError)
	developerID := fs.String("developer", "", "Developer ID")
	email := fs.String("email", "", "User email address")
	packageName := fs.String("package", "", "Package name (applicationId)")
	jsonFlag := fs.String("json", "", "Grant permissions JSON (or @file)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "create",
		ShortUsage: "gplay grants create --developer <id> --email <email> --package <pkg> --json <json>",
		ShortHelp:  "Create a grant for a user on an app.",
		LongHelp: `Create a permission grant for a user on a specific app.

JSON format:
{
  "appLevelPermissions": [
    "CAN_VIEW_NON_FINANCIAL_DATA",
    "CAN_VIEW_APP_QUALITY",
    "CAN_REPLY_TO_REVIEWS"
  ]
}

` + appPermissionsHelp,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*developerID) == "" {
				return fmt.Errorf("--developer is required")
			}
			if strings.TrimSpace(*email) == "" {
				return fmt.Errorf("--email is required")
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

			var grant androidpublisher.Grant
			if err := shared.LoadJSONArg(*jsonFlag, &grant); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			grant.PackageName = pkg

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			parent := fmt.Sprintf("developers/%s/users/%s", *developerID, *email)
			resp, err := service.API.Grants.Create(parent, &grant).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutputContext(ctx, resp, *outputFlag, *pretty)
		},
	}
}

func UpdateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("grants update", flag.ExitOnError)
	developerID := fs.String("developer", "", "Developer ID")
	email := fs.String("email", "", "User email address")
	packageName := fs.String("package", "", "Package name (applicationId)")
	jsonFlag := fs.String("json", "", "Updated grant permissions JSON (or @file)")
	updateMask := fs.String("update-mask", "", "Fields to update (comma-separated)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "update",
		ShortUsage: "gplay grants update --developer <id> --email <email> --package <pkg> --json <json>",
		ShortHelp:  "Update a grant's permissions.",
		LongHelp: `Update permissions for an existing app-level grant.

JSON format:
{
  "appLevelPermissions": [
    "CAN_VIEW_NON_FINANCIAL_DATA",
    "CAN_MANAGE_PUBLIC_LISTING"
  ]
}

` + appPermissionsHelp + `

Use --update-mask to specify which fields to update. If omitted, all
fields in the request body are applied.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*developerID) == "" {
				return fmt.Errorf("--developer is required")
			}
			if strings.TrimSpace(*email) == "" {
				return fmt.Errorf("--email is required")
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

			var grant androidpublisher.Grant
			if err := shared.LoadJSONArg(*jsonFlag, &grant); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			name := fmt.Sprintf("developers/%s/users/%s/grants/%s", *developerID, *email, pkg)
			call := service.API.Grants.Patch(name, &grant).Context(ctx)
			if strings.TrimSpace(*updateMask) != "" {
				call.UpdateMask(*updateMask)
			}
			resp, err := call.Do()
			if err != nil {
				return err
			}
			return shared.PrintOutputContext(ctx, resp, *outputFlag, *pretty)
		},
	}
}

func DeleteCommand() *ffcli.Command {
	fs := flag.NewFlagSet("grants delete", flag.ExitOnError)
	developerID := fs.String("developer", "", "Developer ID")
	email := fs.String("email", "", "User email address")
	packageName := fs.String("package", "", "Package name (applicationId)")
	confirm := fs.Bool("confirm", false, "Confirm deletion")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "delete",
		ShortUsage: "gplay grants delete --developer <id> --email <email> --package <pkg> --confirm",
		ShortHelp:  "Remove a grant from a user.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*developerID) == "" {
				return fmt.Errorf("--developer is required")
			}
			if strings.TrimSpace(*email) == "" {
				return fmt.Errorf("--email is required")
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

			name := fmt.Sprintf("developers/%s/users/%s/grants/%s", *developerID, *email, pkg)
			err = service.API.Grants.Delete(name).Context(ctx).Do()
			if err != nil {
				return err
			}

			result := map[string]interface{}{
				"deleted":     true,
				"email":       *email,
				"packageName": pkg,
			}
			return shared.PrintOutputContext(ctx, result, *outputFlag, *pretty)
		},
	}
}
