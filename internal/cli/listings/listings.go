package listings

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

func ListingsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("listings", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "listings",
		ShortUsage: "gplay listings <subcommand> [flags]",
		ShortHelp:  "Manage store listings in an edit.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			ListCommand(),
			GetCommand(),
			UpdateCommand(),
			PatchCommand(),
			DeleteCommand(),
			DeleteAllCommand(),
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
	fs := flag.NewFlagSet("listings list", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay listings list --package <name> --edit <id>",
		ShortHelp:  "List listings in an edit.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
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
			if strings.TrimSpace(*editID) == "" {
				return fmt.Errorf("--edit is required")
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			call := service.API.Edits.Listings.List(pkg, *editID).Context(ctx)
			resp, err := call.Do()
			if err != nil {
				return err
			}

			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func GetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("listings get", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	locale := fs.String("locale", "", "Locale (e.g. en-US)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "gplay listings get --package <name> --edit <id> --locale <lang>",
		ShortHelp:  "Get a listing.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*locale) == "" {
				return fmt.Errorf("--locale is required")
			}
			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}
			if strings.TrimSpace(*editID) == "" {
				return fmt.Errorf("--edit is required")
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()
			resp, err := service.API.Edits.Listings.Get(pkg, *editID, *locale).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func UpdateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("listings update", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	locale := fs.String("locale", "", "Locale (e.g. en-US)")
	title := fs.String("title", "", "Listing title")
	fullDescription := fs.String("full-description", "", "Full description")
	shortDescription := fs.String("short-description", "", "Short description")
	video := fs.String("video", "", "Promo video URL")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "update",
		ShortUsage: "gplay listings update --package <name> --edit <id> --locale <lang> [flags]",
		ShortHelp:  "Update or create a listing.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			return updateListing(ctx, *packageName, *editID, *locale, *title, *fullDescription, *shortDescription, *video, *outputFlag, *pretty, false)
		},
	}
}

func PatchCommand() *ffcli.Command {
	fs := flag.NewFlagSet("listings patch", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	locale := fs.String("locale", "", "Locale (e.g. en-US)")
	title := fs.String("title", "", "Listing title")
	fullDescription := fs.String("full-description", "", "Full description")
	shortDescription := fs.String("short-description", "", "Short description")
	video := fs.String("video", "", "Promo video URL")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "patch",
		ShortUsage: "gplay listings patch --package <name> --edit <id> --locale <lang> [flags]",
		ShortHelp:  "Patch a listing.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			return updateListing(ctx, *packageName, *editID, *locale, *title, *fullDescription, *shortDescription, *video, *outputFlag, *pretty, true)
		},
	}
}

func DeleteCommand() *ffcli.Command {
	fs := flag.NewFlagSet("listings delete", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	locale := fs.String("locale", "", "Locale (e.g. en-US)")
	confirm := fs.Bool("confirm", false, "Confirm delete")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "delete",
		ShortUsage: "gplay listings delete --package <name> --edit <id> --locale <lang> --confirm",
		ShortHelp:  "Delete a listing.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*locale) == "" {
				return fmt.Errorf("--locale is required")
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
			if strings.TrimSpace(*editID) == "" {
				return fmt.Errorf("--edit is required")
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()
			err = service.API.Edits.Listings.Delete(pkg, *editID, *locale).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(nil, *outputFlag, *pretty)
		},
	}
}

func DeleteAllCommand() *ffcli.Command {
	fs := flag.NewFlagSet("listings delete-all", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	confirm := fs.Bool("confirm", false, "Confirm delete")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "delete-all",
		ShortUsage: "gplay listings delete-all --package <name> --edit <id> --confirm",
		ShortHelp:  "Delete all listings in an edit.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
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
			if strings.TrimSpace(*editID) == "" {
				return fmt.Errorf("--edit is required")
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()
			err = service.API.Edits.Listings.Deleteall(pkg, *editID).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(nil, *outputFlag, *pretty)
		},
	}
}

func updateListing(ctx context.Context, packageName, editID, locale, title, fullDesc, shortDesc, video, outputFlag string, pretty bool, patch bool) error {
	if err := shared.ValidateOutputFlags(outputFlag, pretty); err != nil {
		return err
	}
	if strings.TrimSpace(locale) == "" {
		return fmt.Errorf("--locale is required")
	}
	service, err := playclient.NewService(ctx)
	if err != nil {
		return err
	}
	pkg := shared.ResolvePackageName(packageName, service.Cfg)
	if strings.TrimSpace(pkg) == "" {
		return fmt.Errorf("--package is required")
	}
	if strings.TrimSpace(editID) == "" {
		return fmt.Errorf("--edit is required")
	}

	listing := &androidpublisher.Listing{
		Title:           title,
		FullDescription: fullDesc,
		ShortDescription: shortDesc,
		Video:           video,
	}

	ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
	defer cancel()
	if patch {
		resp, err := service.API.Edits.Listings.Patch(pkg, editID, locale, listing).Context(ctx).Do()
		if err != nil {
			return err
		}
		return shared.PrintOutput(resp, outputFlag, pretty)
	}
	resp, err := service.API.Edits.Listings.Update(pkg, editID, locale, listing).Context(ctx).Do()
	if err != nil {
		return err
	}
	return shared.PrintOutput(resp, outputFlag, pretty)
}
