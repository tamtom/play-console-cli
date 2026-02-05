package expansion

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/googleapi"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

func ExpansionCommand() *ffcli.Command {
	fs := flag.NewFlagSet("expansion", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "expansion",
		ShortUsage: "gplay expansion <subcommand> [flags]",
		ShortHelp:  "Manage expansion files (OBB files).",
		LongHelp: `Manage expansion files (OBB) for large assets.

Expansion files are used to deliver large assets (up to 2GB each)
that don't fit within the APK size limit.

File types:
  - main: Required, primary expansion file
  - patch: Optional, update to the main file

Note: For new apps, consider using Play Asset Delivery instead,
which provides a better user experience.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			GetCommand(),
			UploadCommand(),
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
	fs := flag.NewFlagSet("expansion get", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	apkVersionCode := fs.Int64("apk-version", 0, "APK version code")
	expansionType := fs.String("type", "main", "Expansion file type: main (default), patch")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "gplay expansion get --package <name> --edit <id> --apk-version <code> --type <type>",
		ShortHelp:  "Get expansion file information.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*editID) == "" {
				return fmt.Errorf("--edit is required")
			}
			if *apkVersionCode == 0 {
				return fmt.Errorf("--apk-version is required")
			}
			expType := strings.ToLower(strings.TrimSpace(*expansionType))
			if expType != "main" && expType != "patch" {
				return fmt.Errorf("--type must be 'main' or 'patch'")
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

			resp, err := service.API.Edits.Expansionfiles.Get(pkg, *editID, *apkVersionCode, expType).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func UploadCommand() *ffcli.Command {
	fs := flag.NewFlagSet("expansion upload", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	apkVersionCode := fs.Int64("apk-version", 0, "APK version code")
	expansionType := fs.String("type", "main", "Expansion file type: main (default), patch")
	filePath := fs.String("file", "", "Path to .obb file")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "upload",
		ShortUsage: "gplay expansion upload --package <name> --edit <id> --apk-version <code> --type <type> --file <path>",
		ShortHelp:  "Upload an expansion file.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*editID) == "" {
				return fmt.Errorf("--edit is required")
			}
			if *apkVersionCode == 0 {
				return fmt.Errorf("--apk-version is required")
			}
			if strings.TrimSpace(*filePath) == "" {
				return fmt.Errorf("--file is required")
			}
			expType := strings.ToLower(strings.TrimSpace(*expansionType))
			if expType != "main" && expType != "patch" {
				return fmt.Errorf("--type must be 'main' or 'patch'")
			}
			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			file, err := os.Open(*filePath)
			if err != nil {
				return shared.WrapActionable(err, "failed to open expansion file", "Check that the file exists and is readable.")
			}
			defer file.Close()

			ctx, cancel := shared.ContextWithUploadTimeout(ctx, service.Cfg)
			defer cancel()

			call := service.API.Edits.Expansionfiles.Upload(pkg, *editID, *apkVersionCode, expType)
			call.Media(file, googleapi.ContentType("application/octet-stream"))
			resp, err := call.Context(ctx).Do()
			if err != nil {
				return shared.WrapGoogleAPIError("failed to upload expansion file", err)
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func PatchCommand() *ffcli.Command {
	fs := flag.NewFlagSet("expansion patch", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	apkVersionCode := fs.Int64("apk-version", 0, "APK version code")
	expansionType := fs.String("type", "main", "Expansion file type: main (default), patch")
	referencesVersion := fs.Int64("references-version", 0, "APK version code that contains the file to reference")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "patch",
		ShortUsage: "gplay expansion patch --package <name> --edit <id> --apk-version <code> --type <type> --references-version <code>",
		ShortHelp:  "Reference an expansion file from another APK version.",
		LongHelp: `Reference an expansion file from a different APK version.

This allows you to reuse an existing expansion file for a new APK
without re-uploading it.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*editID) == "" {
				return fmt.Errorf("--edit is required")
			}
			if *apkVersionCode == 0 {
				return fmt.Errorf("--apk-version is required")
			}
			if *referencesVersion == 0 {
				return fmt.Errorf("--references-version is required")
			}
			expType := strings.ToLower(strings.TrimSpace(*expansionType))
			if expType != "main" && expType != "patch" {
				return fmt.Errorf("--type must be 'main' or 'patch'")
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

			expansionFile := &androidpublisher.ExpansionFile{
				ReferencesVersion: *referencesVersion,
			}

			resp, err := service.API.Edits.Expansionfiles.Patch(pkg, *editID, *apkVersionCode, expType, expansionFile).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}
