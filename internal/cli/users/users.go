package users

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

// accountPermissionsHelp lists the developerAccountPermissions values that
// Play accepts. Tests compare this list with the embedded official schema.
const accountPermissionsHelp = `Available account permissions:
  - CAN_VIEW_NON_FINANCIAL_DATA_GLOBAL: View app information and download bulk reports (read-only)
  - CAN_VIEW_FINANCIAL_DATA_GLOBAL: View financial data, orders, and cancellation survey responses
  - CAN_MANAGE_PERMISSIONS_GLOBAL: Admin (all permissions)
  - CAN_EDIT_GAMES_GLOBAL: Edit Play Games Services projects
  - CAN_PUBLISH_GAMES_GLOBAL: Publish Play Games Services projects
  - CAN_REPLY_TO_REVIEWS_GLOBAL: Reply to reviews
  - CAN_MANAGE_PUBLIC_APKS_GLOBAL: Release to production, exclude devices, and use app signing by Google Play
  - CAN_MANAGE_TRACK_APKS_GLOBAL: Release to testing tracks
  - CAN_MANAGE_TRACK_USERS_GLOBAL: Manage testing tracks and edit tester lists
  - CAN_MANAGE_PUBLIC_LISTING_GLOBAL: Manage store presence
  - CAN_MANAGE_DRAFT_APPS_GLOBAL: Create, edit, and delete draft apps
  - CAN_CREATE_MANAGED_PLAY_APPS_GLOBAL: Create and publish private apps to your organization
  - CAN_MANAGE_ORDERS_GLOBAL: Manage orders and subscriptions
  - CAN_MANAGE_APP_CONTENT_GLOBAL: Manage policy pages for all apps
  - CAN_VIEW_APP_QUALITY_GLOBAL: View app quality data for all apps
  - CAN_MANAGE_DEEPLINKS_GLOBAL: Manage the deep link setup for all apps
  - CAN_VIEW_CONNECTED_APPS_GLOBAL: View connected apps in Play Console
  - CAN_EDIT_CONNECTED_APPS_GLOBAL: Edit connected apps in Play Console

The list does not show deprecated permissions.`

func UsersCommand() *ffcli.Command {
	fs := flag.NewFlagSet("users", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "users",
		ShortUsage: "gplay users <subcommand> [flags]",
		ShortHelp:  "Manage developer account team members.",
		LongHelp: `Manage users in your Google Play developer account.

Users can be granted access to specific apps or the entire
developer account with various permission levels.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			ListCommand(),
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

func ListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("users list", flag.ExitOnError)
	developerID := fs.String("developer", "", "Developer ID (from Play Console URL)")
	pageSize := fs.Int("page-size", 100, "Page size")
	paginate := fs.Bool("paginate", false, "Fetch all pages")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay users list --developer <id>",
		ShortHelp:  "List all users in the developer account.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*developerID) == "" {
				return fmt.Errorf("--developer is required")
			}
			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			parent := fmt.Sprintf("developers/%s", *developerID)
			var all []*androidpublisher.User
			pageToken := ""
			guard := shared.NewPaginationGuard(0)
			for {
				call := service.API.Users.List(parent).Context(ctx).PageSize(int64(*pageSize))
				if pageToken != "" {
					call.PageToken(pageToken)
				}
				resp, err := call.Do()
				if err != nil {
					return err
				}
				if !*paginate {
					return shared.PrintOutputContext(ctx, resp, *outputFlag, *pretty)
				}
				all = append(all, resp.Users...)
				done, err := guard.Advance(resp.NextPageToken)
				if err != nil {
					return err
				}
				if done {
					break
				}
				pageToken = resp.NextPageToken
			}

			return shared.PrintOutputContext(ctx, all, *outputFlag, *pretty)
		},
	}
}

func CreateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("users create", flag.ExitOnError)
	developerID := fs.String("developer", "", "Developer ID")
	email := fs.String("email", "", "User email address")
	jsonFlag := fs.String("json", "", "User permissions JSON (or @file)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "create",
		ShortUsage: "gplay users create --developer <id> --email <email> --json <json>",
		ShortHelp:  "Create a new user.",
		LongHelp: `Create a new user in the developer account.

JSON format:
{
  "developerAccountPermissions": [
    "CAN_MANAGE_DRAFT_APPS_GLOBAL",
    "CAN_VIEW_FINANCIAL_DATA_GLOBAL"
  ],
  "expirationTime": "2025-12-31T23:59:59Z"
}

` + accountPermissionsHelp,
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

			var user androidpublisher.User
			if err := shared.LoadJSONArg(*jsonFlag, &user); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			user.Email = *email

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			parent := fmt.Sprintf("developers/%s", *developerID)
			resp, err := service.API.Users.Create(parent, &user).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutputContext(ctx, resp, *outputFlag, *pretty)
		},
	}
}

func UpdateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("users update", flag.ExitOnError)
	developerID := fs.String("developer", "", "Developer ID")
	email := fs.String("email", "", "User email address")
	jsonFlag := fs.String("json", "", "Updated user permissions JSON (or @file)")
	updateMask := fs.String("update-mask", "", "Fields to update (comma-separated)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "update",
		ShortUsage: "gplay users update --developer <id> --email <email> --json <json>",
		ShortHelp:  "Update a user's permissions.",
		LongHelp: `Update a developer account user.

JSON format:
{
  "developerAccountPermissions": [
    "CAN_VIEW_NON_FINANCIAL_DATA_GLOBAL",
    "CAN_VIEW_FINANCIAL_DATA_GLOBAL"
  ]
}

` + accountPermissionsHelp + `

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

			var user androidpublisher.User
			if err := shared.LoadJSONArg(*jsonFlag, &user); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			name := fmt.Sprintf("developers/%s/users/%s", *developerID, *email)
			call := service.API.Users.Patch(name, &user).Context(ctx)
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
	fs := flag.NewFlagSet("users delete", flag.ExitOnError)
	developerID := fs.String("developer", "", "Developer ID")
	email := fs.String("email", "", "User email address")
	confirm := fs.Bool("confirm", false, "Confirm deletion")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "delete",
		ShortUsage: "gplay users delete --developer <id> --email <email> --confirm",
		ShortHelp:  "Remove a user from the developer account.",
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

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			name := fmt.Sprintf("developers/%s/users/%s", *developerID, *email)
			err = service.API.Users.Delete(name).Context(ctx).Do()
			if err != nil {
				return err
			}

			result := map[string]interface{}{
				"deleted": true,
				"email":   *email,
			}
			return shared.PrintOutputContext(ctx, result, *outputFlag, *pretty)
		},
	}
}
