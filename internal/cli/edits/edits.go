package edits

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

func EditsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("edits", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "edits",
		ShortUsage: "gplay edits <subcommand> [flags]",
		ShortHelp:  "Manage Google Play app edits.",
		LongHelp: `Manage Google Play app edits.

Edits are transactional containers for store changes. The workflow is:
  1. Create an edit (gplay edits create)
  2. Make changes (listings, tracks, APKs, images, etc.)
  3. Validate the edit (gplay edits validate)
  4. Commit to apply changes (gplay edits commit)

Edit IDs expire after a period of inactivity. Use --edit to pass the
edit ID to commands that require it.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			CreateCommand(),
			GetCommand(),
			ValidateCommand(),
			CommitCommand(),
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
	fs := flag.NewFlagSet("edits create", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "create",
		ShortUsage: "gplay edits create --package <name>",
		ShortHelp:  "Create a new edit.",
		LongHelp: `Create a new edit for an app.

Returns an edit ID that must be passed to subsequent commands via --edit.
Only one active edit per app is allowed at a time.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
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
			resp, err := service.API.Edits.Insert(pkg, &androidpublisher.AppEdit{}).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func GetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("edits get", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "gplay edits get --package <name> --edit <id>",
		ShortHelp:  "Get an edit.",
		LongHelp: `Retrieve the details of an existing edit.

Use this to check whether an edit ID is still valid before making changes.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
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
			resp, err := service.API.Edits.Get(pkg, *editID).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func ValidateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("edits validate", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "validate",
		ShortUsage: "gplay edits validate --package <name> --edit <id>",
		ShortHelp:  "Validate an edit.",
		LongHelp: `Validate an edit without committing it.

Run this before commit to catch errors early. Validation checks that all
required fields are present and values are within allowed ranges.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
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
			resp, err := service.API.Edits.Validate(pkg, *editID).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func CommitCommand() *ffcli.Command {
	fs := flag.NewFlagSet("edits commit", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	changesNotSent := fs.Bool("changes-not-sent-for-review", false, "Changes not sent for review")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "commit",
		ShortUsage: "gplay edits commit --package <name> --edit <id>",
		ShortHelp:  "Commit an edit.",
		LongHelp: `Commit an edit to apply all pending changes.

Use --changes-not-sent-for-review to commit changes without submitting
them for review. This is useful for metadata-only updates that don't
require review.

Examples:
  gplay edits commit --package com.example --edit EDIT_ID
  gplay edits commit --package com.example --edit EDIT_ID --changes-not-sent-for-review`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
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
			call := service.API.Edits.Commit(pkg, *editID).Context(ctx)
			if *changesNotSent {
				call = call.ChangesNotSentForReview(true)
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
	fs := flag.NewFlagSet("edits delete", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	confirm := fs.Bool("confirm", false, "Confirm delete")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "delete",
		ShortUsage: "gplay edits delete --package <name> --edit <id> --confirm",
		ShortHelp:  "Delete an edit.",
		LongHelp: `Delete an edit, discarding all pending changes.

Requires --confirm to prevent accidental deletion.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*editID) == "" {
				return fmt.Errorf("--edit is required")
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
			err = service.API.Edits.Delete(pkg, *editID).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(nil, *outputFlag, *pretty)
		},
	}
}
