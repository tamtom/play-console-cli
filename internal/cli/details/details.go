package details

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

func DetailsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("details", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "details",
		ShortUsage: "gplay details <subcommand> [flags]",
		ShortHelp:  "Manage app details (contact info, default language).",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			GetCommand(),
			UpdateCommand(),
			PatchCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			return flag.ErrHelp
		},
	}
}

func GetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("details get", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "gplay details get --package <name> --edit <id>",
		ShortHelp:  "Get app details for an edit.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*editID) == "" {
				return fmt.Errorf("--edit is required")
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
			resp, err := service.API.Edits.Details.Get(pkg, *editID).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func UpdateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("details update", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	contactEmail := fs.String("contact-email", "", "Contact email address")
	contactPhone := fs.String("contact-phone", "", "Contact phone number")
	contactWebsite := fs.String("contact-website", "", "Contact website URL")
	defaultLanguage := fs.String("default-language", "", "Default language (BCP-47 code)")
	jsonFlag := fs.String("json", "", "Full AppDetails JSON (or @file) - overrides other flags")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "update",
		ShortUsage: "gplay details update --package <name> --edit <id> [--contact-email <email>] [--contact-phone <phone>] [--contact-website <url>] [--default-language <lang>] [--json <json>]",
		ShortHelp:  "Update app details (replaces entire resource).",
		LongHelp: `Update app details. Replaces the entire resource.

Use individual flags for simple updates, or --json for full control.
When --json is provided, it overrides all other flags.

JSON format:
{
  "contactEmail": "support@example.com",
  "contactPhone": "+1-555-0100",
  "contactWebsite": "https://example.com",
  "defaultLanguage": "en-US"
}

Examples:
  gplay details update --package com.example --edit EDIT_ID --contact-email support@example.com
  gplay details update --package com.example --edit EDIT_ID --json @details.json`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			return updateDetails(ctx, *packageName, *editID, *contactEmail, *contactPhone, *contactWebsite, *defaultLanguage, *jsonFlag, *outputFlag, *pretty, false)
		},
	}
}

func PatchCommand() *ffcli.Command {
	fs := flag.NewFlagSet("details patch", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	contactEmail := fs.String("contact-email", "", "Contact email address")
	contactPhone := fs.String("contact-phone", "", "Contact phone number")
	contactWebsite := fs.String("contact-website", "", "Contact website URL")
	defaultLanguage := fs.String("default-language", "", "Default language (BCP-47 code)")
	jsonFlag := fs.String("json", "", "Partial AppDetails JSON (or @file) - overrides other flags")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "patch",
		ShortUsage: "gplay details patch --package <name> --edit <id> [--contact-email <email>] [--contact-phone <phone>] [--contact-website <url>] [--default-language <lang>] [--json <json>]",
		ShortHelp:  "Patch app details (partial update).",
		LongHelp: `Patch app details. Only updates provided fields.

Use individual flags for simple updates, or --json for full control.
When --json is provided, it overrides all other flags.

JSON format (include only fields to update):
{
  "contactEmail": "new-support@example.com"
}

Examples:
  gplay details patch --package com.example --edit EDIT_ID --contact-email new@example.com
  gplay details patch --package com.example --edit EDIT_ID --json '{"contactWebsite":"https://new.example.com"}'`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			return updateDetails(ctx, *packageName, *editID, *contactEmail, *contactPhone, *contactWebsite, *defaultLanguage, *jsonFlag, *outputFlag, *pretty, true)
		},
	}
}

func updateDetails(ctx context.Context, packageName, editID, contactEmail, contactPhone, contactWebsite, defaultLanguage, jsonFlag, outputFlag string, pretty, patch bool) error {
	if err := shared.ValidateOutputFlags(outputFlag, pretty); err != nil {
		return err
	}
	if strings.TrimSpace(editID) == "" {
		return fmt.Errorf("--edit is required")
	}

	service, err := playclient.NewService(ctx)
	if err != nil {
		return err
	}
	pkg := shared.ResolvePackageName(packageName, service.Cfg)
	if strings.TrimSpace(pkg) == "" {
		return fmt.Errorf("--package is required")
	}

	var details androidpublisher.AppDetails

	if strings.TrimSpace(jsonFlag) != "" {
		if err := shared.LoadJSONArg(jsonFlag, &details); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
	} else {
		// Build from individual flags
		if strings.TrimSpace(contactEmail) != "" {
			details.ContactEmail = contactEmail
		}
		if strings.TrimSpace(contactPhone) != "" {
			details.ContactPhone = contactPhone
		}
		if strings.TrimSpace(contactWebsite) != "" {
			details.ContactWebsite = contactWebsite
		}
		if strings.TrimSpace(defaultLanguage) != "" {
			details.DefaultLanguage = defaultLanguage
		}
	}

	ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
	defer cancel()

	if patch {
		resp, err := service.API.Edits.Details.Patch(pkg, editID, &details).Context(ctx).Do()
		if err != nil {
			return err
		}
		return shared.PrintOutput(resp, outputFlag, pretty)
	}

	resp, err := service.API.Edits.Details.Update(pkg, editID, &details).Context(ctx).Do()
	if err != nil {
		return err
	}
	return shared.PrintOutput(resp, outputFlag, pretty)
}
