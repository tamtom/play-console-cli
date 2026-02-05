package apks

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

func APKsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("apks", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "apks",
		ShortUsage: "gplay apks <subcommand> [flags]",
		ShortHelp:  "Manage APKs in an edit.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			UploadCommand(),
			ListCommand(),
			AddExternallyHostedCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			return flag.ErrHelp
		},
	}
}

func UploadCommand() *ffcli.Command {
	fs := flag.NewFlagSet("apks upload", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	filePath := fs.String("file", "", "Path to .apk file")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "upload",
		ShortUsage: "gplay apks upload --package <name> --edit <id> --file <path>",
		ShortHelp:  "Upload an APK to an edit.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
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
				return shared.WrapActionable(err, "failed to open APK file", "Check that the file exists and is readable.")
			}
			defer file.Close()

			ctx, cancel := shared.ContextWithUploadTimeout(ctx, service.Cfg)
			defer cancel()
			call := service.API.Edits.Apks.Upload(pkg, *editID)
			call.Media(file, googleapi.ContentType("application/octet-stream"))
			resp, err := call.Context(ctx).Do()
			if err != nil {
				return shared.WrapGoogleAPIError("failed to upload APK", err)
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func ListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("apks list", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay apks list --package <name> --edit <id>",
		ShortHelp:  "List APKs in an edit.",
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

			call := service.API.Edits.Apks.List(pkg, *editID).Context(ctx)
			resp, err := call.Do()
			if err != nil {
				return err
			}

			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func AddExternallyHostedCommand() *ffcli.Command {
	fs := flag.NewFlagSet("apks addexternallyhosted", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	jsonFlag := fs.String("json", "", "ExternallyHostedApk JSON (or @file)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "addexternallyhosted",
		ShortUsage: "gplay apks addexternallyhosted --package <name> --edit <id> --json <json>",
		ShortHelp:  "Add an externally hosted APK without uploading.",
		LongHelp: `Add an externally hosted APK without uploading.

Creates an APK entry that references an APK hosted at an external URL.
This is useful for very large APKs that are hosted elsewhere.

JSON format:
{
  "externallyHostedApk": {
    "packageName": "com.example.app",
    "versionCode": 1,
    "versionName": "1.0",
    "applicationLabel": "My App",
    "fileSize": 12345678,
    "fileSha1Base64": "...",
    "fileSha256Base64": "...",
    "externallyHostedUrl": "https://example.com/app.apk",
    "minimumSdk": 21,
    "nativeCodes": ["armeabi-v7a", "arm64-v8a"],
    "usesPermissions": [{"name": "android.permission.INTERNET"}]
  }
}`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
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
			if strings.TrimSpace(*editID) == "" {
				return fmt.Errorf("--edit is required")
			}

			var req androidpublisher.ApksAddExternallyHostedRequest
			if err := shared.LoadJSONArg(*jsonFlag, &req); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.API.Edits.Apks.Addexternallyhosted(pkg, *editID, &req).Context(ctx).Do()
			if err != nil {
				return shared.WrapGoogleAPIError("failed to add externally hosted APK", err)
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}
