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
			// ListCommand() is not available - Google Play API v3 Grants service does not expose List method
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

// ListCommand is not implemented because the Google Play Android Publisher API v3
// Grants service does not expose a List method. The available methods are:
// Create, Delete, and Patch.
//
// To view grants, you would need to track them separately or use the
// Google Play Console web interface.
func ListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("grants list", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay grants list (not implemented)",
		ShortHelp:  "List grants (not available in API v3).",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			return fmt.Errorf("grants list is not implemented: the Google Play Android Publisher API v3 does not expose a List method for Grants. Available operations are: create, update (patch), and delete")
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
    "CAN_ACCESS_APP",
    "CAN_VIEW_FINANCIAL_DATA",
    "CAN_MANAGE_PERMISSIONS",
    "CAN_REPLY_TO_REVIEWS",
    "CAN_MANAGE_PUBLIC_APKS",
    "CAN_MANAGE_TRACK_APKS",
    "CAN_MANAGE_TRACK_USERS",
    "CAN_MANAGE_PUBLIC_LISTING",
    "CAN_MANAGE_DRAFT_APPS",
    "CAN_MANAGE_ORDERS"
  ]
}

Available app permissions:
  - CAN_ACCESS_APP: Basic app access
  - CAN_VIEW_FINANCIAL_DATA: View financial reports
  - CAN_MANAGE_PERMISSIONS: Manage user permissions
  - CAN_REPLY_TO_REVIEWS: Reply to user reviews
  - CAN_MANAGE_PUBLIC_APKS: Manage production releases
  - CAN_MANAGE_TRACK_APKS: Manage test tracks
  - CAN_MANAGE_TRACK_USERS: Manage testers
  - CAN_MANAGE_PUBLIC_LISTING: Manage store listing
  - CAN_MANAGE_DRAFT_APPS: Manage draft changes
  - CAN_MANAGE_ORDERS: Manage orders and subscriptions`,
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
			return shared.PrintOutput(resp, *outputFlag, *pretty)
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
			return shared.PrintOutput(resp, *outputFlag, *pretty)
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
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}
