package systemapks

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

func SystemAPKsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("system-apks", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "system-apks",
		ShortUsage: "gplay system-apks <subcommand> [flags]",
		ShortHelp:  "Create APKs for system image inclusion.",
		LongHelp: `Create and manage system APK variants for OEM/system image inclusion.

System APKs are pre-installed on devices by OEMs. Use these commands
to create variants for specific device configurations.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			CreateCommand(),
			ListCommand(),
			GetCommand(),
			DownloadCommand(),
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
	fs := flag.NewFlagSet("system-apks create", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	versionCode := fs.Int64("version-code", 0, "Version code of the app bundle")
	jsonFlag := fs.String("json", "", "SystemApkOptions JSON (or @file)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "create",
		ShortUsage: "gplay system-apks create --package <name> --version-code <code> --json <json>",
		ShortHelp:  "Create a system APK variant.",
		LongHelp: `Create a system APK variant from an app bundle.

JSON format:
{
  "deviceSpec": {
    "screenDensity": 480,
    "supportedAbis": ["arm64-v8a", "armeabi-v7a"],
    "supportedLocales": ["en-US", "es"]
  },
  "options": {
    "rotated": false,
    "uncompressedDexFiles": false,
    "uncompressedNativeLibraries": true
  }
}`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if *versionCode == 0 {
				return fmt.Errorf("--version-code is required")
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

			var variant androidpublisher.Variant
			if err := shared.LoadJSONArg(*jsonFlag, &variant); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.API.Systemapks.Variants.Create(pkg, *versionCode, &variant).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func ListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("system-apks list", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	versionCode := fs.Int64("version-code", 0, "Version code of the app bundle")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay system-apks list --package <name> --version-code <code>",
		ShortHelp:  "List system APK variants.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if *versionCode == 0 {
				return fmt.Errorf("--version-code is required")
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

			resp, err := service.API.Systemapks.Variants.List(pkg, *versionCode).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func GetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("system-apks get", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	versionCode := fs.Int64("version-code", 0, "Version code of the app bundle")
	variantID := fs.Int64("variant-id", 0, "Variant ID")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "gplay system-apks get --package <name> --version-code <code> --variant-id <id>",
		ShortHelp:  "Get a system APK variant.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if *versionCode == 0 {
				return fmt.Errorf("--version-code is required")
			}
			if *variantID == 0 {
				return fmt.Errorf("--variant-id is required")
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

			resp, err := service.API.Systemapks.Variants.Get(pkg, *versionCode, *variantID).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func DownloadCommand() *ffcli.Command {
	fs := flag.NewFlagSet("system-apks download", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	versionCode := fs.Int64("version-code", 0, "Version code of the app bundle")
	variantID := fs.Int64("variant-id", 0, "Variant ID")
	outputDir := fs.String("output", ".", "Output directory for downloaded APK")
	outputFlag := fs.String("format", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "download",
		ShortUsage: "gplay system-apks download --package <name> --version-code <code> --variant-id <id> --output <dir>",
		ShortHelp:  "Download a system APK variant.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if *versionCode == 0 {
				return fmt.Errorf("--version-code is required")
			}
			if *variantID == 0 {
				return fmt.Errorf("--variant-id is required")
			}
			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			ctx, cancel := shared.ContextWithUploadTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.API.Systemapks.Variants.Download(pkg, *versionCode, *variantID).Context(ctx).Download()
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("download failed with status: %s", resp.Status)
			}

			// Create output directory
			if err := os.MkdirAll(*outputDir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}

			// Write file
			fileName := fmt.Sprintf("%s_%d_variant_%d.apk", pkg, *versionCode, *variantID)
			filePath := filepath.Join(*outputDir, fileName)
			file, err := os.Create(filePath)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			defer file.Close()

			written, err := io.Copy(file, resp.Body)
			if err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}

			result := map[string]interface{}{
				"downloaded": true,
				"path":       filePath,
				"size":       written,
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}
