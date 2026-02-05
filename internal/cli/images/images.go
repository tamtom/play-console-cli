package images

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/googleapi"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

func ImagesCommand() *ffcli.Command {
	fs := flag.NewFlagSet("images", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "images",
		ShortUsage: "gplay images <subcommand> [flags]",
		ShortHelp:  "Manage listing images in an edit.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			ListCommand(),
			UploadCommand(),
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
	fs := flag.NewFlagSet("images list", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	locale := fs.String("locale", "", "Locale (e.g. en-US)")
	imageType := fs.String("type", "", "Image type (phoneScreenshots, featureGraphic, etc)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay images list --package <name> --edit <id> --locale <lang> --type <type>",
		ShortHelp:  "List images for a listing.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*locale) == "" {
				return fmt.Errorf("--locale is required")
			}
			if strings.TrimSpace(*imageType) == "" {
				return fmt.Errorf("--type is required")
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

			call := service.API.Edits.Images.List(pkg, *editID, *locale, *imageType).Context(ctx)
			resp, err := call.Do()
			if err != nil {
				return err
			}

			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func UploadCommand() *ffcli.Command {
	fs := flag.NewFlagSet("images upload", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	locale := fs.String("locale", "", "Locale (e.g. en-US)")
	imageType := fs.String("type", "", "Image type (phoneScreenshots, featureGraphic, etc)")
	filePath := fs.String("file", "", "Path to image file")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "upload",
		ShortUsage: "gplay images upload --package <name> --edit <id> --locale <lang> --type <type> --file <path>",
		ShortHelp:  "Upload an image to a listing.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*locale) == "" {
				return fmt.Errorf("--locale is required")
			}
			if strings.TrimSpace(*imageType) == "" {
				return fmt.Errorf("--type is required")
			}
			if strings.TrimSpace(*filePath) == "" {
				return fmt.Errorf("--file is required")
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
			file, err := os.Open(*filePath)
			if err != nil {
				return shared.WrapActionable(err, "failed to open image file", "Check that the file exists and is readable.")
			}
			defer file.Close()

			ctx, cancel := shared.ContextWithUploadTimeout(ctx, service.Cfg)
			defer cancel()
			call := service.API.Edits.Images.Upload(pkg, *editID, *locale, *imageType)
			call.Media(file, googleapi.ContentType("application/octet-stream"))
			resp, err := call.Context(ctx).Do()
			if err != nil {
				return shared.WrapGoogleAPIError("failed to upload image", err)
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func DeleteCommand() *ffcli.Command {
	fs := flag.NewFlagSet("images delete", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	locale := fs.String("locale", "", "Locale (e.g. en-US)")
	imageType := fs.String("type", "", "Image type")
	imageID := fs.String("image", "", "Image ID")
	confirm := fs.Bool("confirm", false, "Confirm delete")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "delete",
		ShortUsage: "gplay images delete --package <name> --edit <id> --locale <lang> --type <type> --image <id> --confirm",
		ShortHelp:  "Delete an image from a listing.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*locale) == "" {
				return fmt.Errorf("--locale is required")
			}
			if strings.TrimSpace(*imageType) == "" {
				return fmt.Errorf("--type is required")
			}
			if strings.TrimSpace(*imageID) == "" {
				return fmt.Errorf("--image is required")
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
			err = service.API.Edits.Images.Delete(pkg, *editID, *locale, *imageType, *imageID).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(nil, *outputFlag, *pretty)
		},
	}
}

func DeleteAllCommand() *ffcli.Command {
	fs := flag.NewFlagSet("images delete-all", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	locale := fs.String("locale", "", "Locale (e.g. en-US)")
	imageType := fs.String("type", "", "Image type")
	confirm := fs.Bool("confirm", false, "Confirm delete")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "delete-all",
		ShortUsage: "gplay images delete-all --package <name> --edit <id> --locale <lang> --type <type> --confirm",
		ShortHelp:  "Delete all images for a listing.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*locale) == "" {
				return fmt.Errorf("--locale is required")
			}
			if strings.TrimSpace(*imageType) == "" {
				return fmt.Errorf("--type is required")
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
			resp, err := service.API.Edits.Images.Deleteall(pkg, *editID, *locale, *imageType).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}
